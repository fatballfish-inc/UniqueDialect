package uniquedialect

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Dialect identifies a SQL dialect.
type Dialect string

const (
	// DialectMySQL is the MySQL dialect.
	DialectMySQL Dialect = "mysql"
	// DialectPostgres is the PostgreSQL dialect.
	DialectPostgres Dialect = "postgres"
	// DialectSQLite is the SQLite dialect.
	DialectSQLite Dialect = "sqlite"
	// DialectOracle is the Oracle dialect.
	DialectOracle Dialect = "oracle"
)

// DriverName identifies an underlying database/sql driver family.
type DriverName string

const (
	// DriverMySQL is the MySQL database/sql driver family.
	DriverMySQL DriverName = "mysql"
	// DriverPostgres is the PostgreSQL database/sql driver family.
	DriverPostgres DriverName = "postgres"
	// DriverSQLite is the SQLite database/sql driver family.
	DriverSQLite DriverName = "sqlite"
	// DriverOracle is the Oracle database/sql driver family.
	DriverOracle DriverName = "oracle"
)

const (
	// Scheme is the custom DSN scheme for UniqueDialect.
	Scheme = "uniquedialect"
	// DefaultSQLDriverName is the registered database/sql driver name.
	DefaultSQLDriverName = "uniquedialect"
	// DefaultCipherPrefix prefixes encrypted values for downgrade-safe reads.
	DefaultCipherPrefix = "enc:v1:"
	// DefaultTranslateFailureLogPath is the default JSONL log path for translation failures.
	DefaultTranslateFailureLogPath = "uniquedialect-translate-failures.jsonl"
	// DefaultExecutionFailureLogPath is the default JSONL log path for execution failures.
	DefaultExecutionFailureLogPath = "uniquedialect-execution-failures.jsonl"
)

// Options configures the virtual driver.
type Options struct {
	InputDialect  Dialect
	TargetDialect Dialect
	Driver        DriverName
	Connection    ConnectionOptions
	Translator    TranslatorOptions
	Logging       LoggingOptions
	Encryption    EncryptionOptions
}

// ConnectionOptions is the canonical connection model shared across targets.
type ConnectionOptions struct {
	Host       string
	Port       int
	Username   string
	Password   string
	Database   string
	Parameters map[string]string
}

// TranslatorOptions configures SQL translation.
type TranslatorOptions struct {
	InputDialect  Dialect
	TargetDialect Dialect
}

// LoggingOptions configures execution logging.
type LoggingOptions struct {
	Level                   string
	LogFailedSQL            bool
	TranslateFailureLogPath string
	ExecutionFailureLogPath string
}

// EncryptionOptions configures field encryption for GORM.
type EncryptionOptions struct {
	Enabled  bool
	Key      string
	Provider Encryptor
}

// GORMConfig configures the GORM dialector wrapper.
type GORMConfig struct {
	PreferSimpleProtocol      bool
	SkipInitializeWithVersion bool
}

func (o Options) normalized() Options {
	cp := o
	if cp.TargetDialect == "" {
		cp.TargetDialect = cp.InputDialect
	}
	if cp.Driver == "" {
		cp.Driver = driverForDialect(cp.TargetDialect)
	}
	if cp.Translator.InputDialect == "" {
		cp.Translator.InputDialect = cp.InputDialect
	}
	if cp.Translator.TargetDialect == "" {
		cp.Translator.TargetDialect = cp.TargetDialect
	}
	if cp.Connection.Parameters == nil {
		cp.Connection.Parameters = map[string]string{}
	}
	if cp.Logging.LogFailedSQL {
		if cp.Logging.TranslateFailureLogPath == "" {
			cp.Logging.TranslateFailureLogPath = DefaultTranslateFailureLogPath
		}
		if cp.Logging.ExecutionFailureLogPath == "" {
			cp.Logging.ExecutionFailureLogPath = DefaultExecutionFailureLogPath
		}
	}
	return cp
}

// FormatDSN encodes options into a custom UniqueDialect DSN.
func (o Options) FormatDSN() string {
	o = o.normalized()

	values := url.Values{}
	if o.InputDialect != "" {
		values.Set("input", string(o.InputDialect))
	}
	if o.TargetDialect != "" {
		values.Set("target", string(o.TargetDialect))
	}
	if o.Driver != "" {
		values.Set("driver", string(o.Driver))
	}
	if o.Connection.Database != "" {
		values.Set("database", o.Connection.Database)
	}
	if o.Connection.Port != 0 {
		values.Set("port", strconv.Itoa(o.Connection.Port))
	}
	if o.Encryption.Enabled {
		values.Set("encrypt_enabled", "true")
	}
	if o.Encryption.Key != "" {
		values.Set("encrypt_key", o.Encryption.Key)
	}
	if o.Logging.LogFailedSQL {
		values.Set("log_failed_sql", "true")
	}
	if o.Logging.TranslateFailureLogPath != "" {
		values.Set("translate_failure_log_path", o.Logging.TranslateFailureLogPath)
	}
	if o.Logging.ExecutionFailureLogPath != "" {
		values.Set("execution_failure_log_path", o.Logging.ExecutionFailureLogPath)
	}

	keys := make([]string, 0, len(o.Connection.Parameters))
	for key := range o.Connection.Parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values.Set(key, o.Connection.Parameters[key])
	}

	host := o.Connection.Host
	if host == "" {
		host = "localhost"
	}
	user := url.UserPassword(o.Connection.Username, o.Connection.Password)
	return (&url.URL{
		Scheme:   Scheme,
		User:     user,
		Host:     host,
		RawQuery: values.Encode(),
	}).String()
}

// BuildDriverDSN maps the canonical connection options to the target driver DSN.
func (o Options) BuildDriverDSN() (string, error) {
	o = o.normalized()

	switch o.Driver {
	case DriverSQLite:
		if o.Connection.Database == "" {
			return "", errors.New("sqlite database is required")
		}
		return o.Connection.Database, nil
	case DriverMySQL:
		if o.Connection.Host == "" {
			o.Connection.Host = "127.0.0.1"
		}
		if o.Connection.Port == 0 {
			o.Connection.Port = 3306
		}
		builder := strings.Builder{}
		builder.WriteString(o.Connection.Username)
		if o.Connection.Password != "" {
			builder.WriteString(":")
			builder.WriteString(o.Connection.Password)
		}
		builder.WriteString("@tcp(")
		builder.WriteString(o.Connection.Host)
		builder.WriteString(":")
		builder.WriteString(strconv.Itoa(o.Connection.Port))
		builder.WriteString(")/")
		builder.WriteString(o.Connection.Database)
		if len(o.Connection.Parameters) > 0 {
			params := url.Values{}
			keys := make([]string, 0, len(o.Connection.Parameters))
			for key := range o.Connection.Parameters {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				params.Set(key, o.Connection.Parameters[key])
			}
			builder.WriteString("?")
			builder.WriteString(params.Encode())
		}
		return builder.String(), nil
	case DriverPostgres:
		host := o.Connection.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := o.Connection.Port
		if port == 0 {
			port = 5432
		}
		parts := []string{
			"host=" + host,
			"port=" + strconv.Itoa(port),
		}
		if o.Connection.Username != "" {
			parts = append(parts, "user="+o.Connection.Username)
		}
		if o.Connection.Password != "" {
			parts = append(parts, "password="+o.Connection.Password)
		}
		if o.Connection.Database != "" {
			parts = append(parts, "dbname="+o.Connection.Database)
		}

		keys := make([]string, 0, len(o.Connection.Parameters))
		for key := range o.Connection.Parameters {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			valueKey := key
			if strings.EqualFold(key, "timezone") {
				valueKey = "TimeZone"
			}
			parts = append(parts, fmt.Sprintf("%s=%s", valueKey, o.Connection.Parameters[key]))
		}
		return strings.Join(parts, " "), nil
	case DriverOracle:
		return "", errors.New("oracle driver execution is not implemented in MVP")
	default:
		return "", fmt.Errorf("unsupported driver %q", o.Driver)
	}
}

func driverForDialect(dialect Dialect) DriverName {
	switch dialect {
	case DialectPostgres:
		return DriverPostgres
	case DialectSQLite:
		return DriverSQLite
	case DialectOracle:
		return DriverOracle
	default:
		return DriverMySQL
	}
}
