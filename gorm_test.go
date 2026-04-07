package uniquedialect_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/fatballfish/uniquedialect"
	"gorm.io/gorm"
)

type encryptedUser struct {
	ID     int64  `gorm:"primaryKey"`
	Name   string `gorm:"column:name"`
	Secret string `gorm:"column:secret;encrypt"`
}

func TestOpenGORMWithOptionsEncryptsAndDecryptsTaggedFields(t *testing.T) {
	t.Parallel()

	targetDSN := "file:ud_gorm_encrypt?mode=memory&cache=shared"

	dialector, err := uniquedialect.OpenGORMWithOptions(uniquedialect.Options{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectSQLite,
		Driver:        uniquedialect.DriverSQLite,
		Connection: uniquedialect.ConnectionOptions{
			Database: targetDSN,
		},
		Encryption: uniquedialect.EncryptionOptions{
			Enabled: true,
			Key:     "0123456789abcdef0123456789abcdef",
		},
	}, uniquedialect.GORMConfig{})
	if err != nil {
		t.Fatalf("OpenGORMWithOptions() error = %v", err)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	if err := db.AutoMigrate(&encryptedUser{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	if err := db.Create(&encryptedUser{Name: "alice", Secret: "top-secret"}).Error; err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	rawDB, err := sql.Open("sqlite", targetDSN)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer rawDB.Close()

	var stored string
	if err := rawDB.QueryRow("SELECT secret FROM encrypted_users WHERE id = 1").Scan(&stored); err != nil {
		t.Fatalf("raw QueryRow().Scan() error = %v", err)
	}
	if !strings.HasPrefix(stored, uniquedialect.DefaultCipherPrefix) {
		t.Fatalf("stored secret = %q, want prefix %q", stored, uniquedialect.DefaultCipherPrefix)
	}

	var got encryptedUser
	if err := db.First(&got, 1).Error; err != nil {
		t.Fatalf("First() error = %v", err)
	}
	if got.Secret != "top-secret" {
		t.Fatalf("First() secret = %q, want top-secret", got.Secret)
	}
}

func TestOpenGORMWithOptionsLeavesPlaintextHistoricalDataReadable(t *testing.T) {
	t.Parallel()

	targetDSN := "file:ud_gorm_plain?mode=memory&cache=shared"

	rawDB, err := sql.Open("sqlite", targetDSN)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer rawDB.Close()

	if _, err := rawDB.Exec(`CREATE TABLE encrypted_users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, secret TEXT)`); err != nil {
		t.Fatalf("raw Exec(create table) error = %v", err)
	}
	if _, err := rawDB.Exec(`INSERT INTO encrypted_users (name, secret) VALUES ('alice', 'legacy-plain')`); err != nil {
		t.Fatalf("raw Exec(insert) error = %v", err)
	}

	dialector, err := uniquedialect.OpenGORMWithOptions(uniquedialect.Options{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectSQLite,
		Driver:        uniquedialect.DriverSQLite,
		Connection: uniquedialect.ConnectionOptions{
			Database: targetDSN,
		},
		Encryption: uniquedialect.EncryptionOptions{
			Enabled: true,
			Key:     "0123456789abcdef0123456789abcdef",
		},
	}, uniquedialect.GORMConfig{})
	if err != nil {
		t.Fatalf("OpenGORMWithOptions() error = %v", err)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	var got encryptedUser
	if err := db.First(&got, 1).Error; err != nil {
		t.Fatalf("First() error = %v", err)
	}
	if got.Secret != "legacy-plain" {
		t.Fatalf("First() secret = %q, want legacy-plain", got.Secret)
	}
}
