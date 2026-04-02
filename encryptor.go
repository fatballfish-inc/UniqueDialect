package uniquedialect

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// Encryptor encrypts and decrypts string values.
type Encryptor interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

type aesEncryptor struct {
	aead cipher.AEAD
}

// NewDefaultEncryptor constructs the built-in AES-GCM encryptor.
func NewDefaultEncryptor(key string) (Encryptor, error) {
	sum := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &aesEncryptor{aead: aead}, nil
}

func (e *aesEncryptor) Encrypt(plain string) (string, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	ciphertext := e.aead.Seal(nonce, nonce, []byte(plain), nil)
	return DefaultCipherPrefix + base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func (e *aesEncryptor) Decrypt(cipherText string) (string, error) {
	if !strings.HasPrefix(cipherText, DefaultCipherPrefix) {
		return cipherText, nil
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(cipherText, DefaultCipherPrefix))
	if err != nil {
		return "", fmt.Errorf("decode cipher text: %w", err)
	}
	nonceSize := e.aead.NonceSize()
	if len(payload) < nonceSize {
		return "", fmt.Errorf("cipher text too short")
	}
	nonce := payload[:nonceSize]
	body := payload[nonceSize:]
	plain, err := e.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt field: %w", err)
	}
	return string(plain), nil
}

type gormEncryptionPlugin struct {
	encryptor Encryptor
}

func newEncryptionPlugin(encryptor Encryptor) *gormEncryptionPlugin {
	return &gormEncryptionPlugin{encryptor: encryptor}
}

func (p *gormEncryptionPlugin) Name() string {
	return "uniquedialect:encryption"
}

func (p *gormEncryptionPlugin) Initialize(db *gorm.DB) error {
	if db.Callback().Create().Get("uniquedialect:encrypt_create") == nil {
		if err := db.Callback().Create().Before("gorm:create").Register("uniquedialect:encrypt_create", p.encryptFields); err != nil {
			return err
		}
	}
	if db.Callback().Update().Get("uniquedialect:encrypt_update") == nil {
		if err := db.Callback().Update().Before("gorm:update").Register("uniquedialect:encrypt_update", p.encryptFields); err != nil {
			return err
		}
	}
	if db.Callback().Query().Get("uniquedialect:decrypt_query") == nil {
		if err := db.Callback().Query().After("gorm:after_query").Register("uniquedialect:decrypt_query", p.decryptFields); err != nil {
			return err
		}
	}
	return nil
}

func (p *gormEncryptionPlugin) encryptFields(db *gorm.DB) {
	p.walkModel(db, func(fieldValue string) (string, error) {
		if strings.HasPrefix(fieldValue, DefaultCipherPrefix) {
			return fieldValue, nil
		}
		return p.encryptor.Encrypt(fieldValue)
	})
}

func (p *gormEncryptionPlugin) decryptFields(db *gorm.DB) {
	p.walkModel(db, func(fieldValue string) (string, error) {
		if !strings.HasPrefix(fieldValue, DefaultCipherPrefix) {
			return fieldValue, nil
		}
		value, err := p.encryptor.Decrypt(fieldValue)
		if err != nil {
			return fieldValue, nil
		}
		return value, nil
	})
}

func (p *gormEncryptionPlugin) walkModel(db *gorm.DB, convert func(string) (string, error)) {
	if db.Statement == nil || db.Statement.Schema == nil {
		return
	}

	targets := reflectValues(db.Statement.ReflectValue)
	for _, target := range targets {
		for _, field := range db.Statement.Schema.Fields {
			if _, ok := field.TagSettings["ENCRYPT"]; !ok {
				continue
			}
			value, zero := field.ValueOf(db.Statement.Context, target)
			if zero {
				continue
			}
			text, ok := value.(string)
			if !ok {
				continue
			}
			converted, err := convert(text)
			if err != nil {
				db.AddError(err)
				continue
			}
			if err := field.Set(db.Statement.Context, target, converted); err != nil {
				db.AddError(err)
			}
		}
	}
}

func reflectValues(value reflect.Value) []reflect.Value {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]reflect.Value, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			for item.Kind() == reflect.Pointer {
				if item.IsNil() {
					break
				}
				item = item.Elem()
			}
			out = append(out, item)
		}
		return out
	case reflect.Struct:
		return []reflect.Value{value}
	default:
		return nil
	}
}
