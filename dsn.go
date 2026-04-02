package uniquedialect

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ParseDSN decodes a UniqueDialect DSN into canonical options.
func ParseDSN(raw string) (Options, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return Options{}, fmt.Errorf("parse uniquedialect dsn: %w", err)
	}
	if parsed.Scheme != Scheme {
		return Options{}, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}

	opts := Options{
		InputDialect:  Dialect(parsed.Query().Get("input")),
		TargetDialect: Dialect(parsed.Query().Get("target")),
		Driver:        DriverName(parsed.Query().Get("driver")),
		Connection: ConnectionOptions{
			Host:       parsed.Hostname(),
			Parameters: map[string]string{},
		},
		Encryption: EncryptionOptions{
			Enabled: parsed.Query().Get("encrypt_enabled") == "true",
			Key:     parsed.Query().Get("encrypt_key"),
		},
		Logging: LoggingOptions{
			LogFailedSQL:            parsed.Query().Get("log_failed_sql") == "true",
			TranslateFailureLogPath: parsed.Query().Get("translate_failure_log_path"),
			ExecutionFailureLogPath: parsed.Query().Get("execution_failure_log_path"),
		},
	}

	if parsed.User != nil {
		opts.Connection.Username = parsed.User.Username()
		password, _ := parsed.User.Password()
		opts.Connection.Password = password
	}

	if port := parsed.Port(); port != "" {
		value, convErr := strconv.Atoi(port)
		if convErr != nil {
			return Options{}, fmt.Errorf("invalid port %q: %w", port, convErr)
		}
		opts.Connection.Port = value
	} else if value := parsed.Query().Get("port"); value != "" {
		port, convErr := strconv.Atoi(value)
		if convErr != nil {
			return Options{}, fmt.Errorf("invalid port %q: %w", value, convErr)
		}
		opts.Connection.Port = port
	}

	if database := strings.TrimPrefix(parsed.EscapedPath(), "/"); database != "" {
		dbValue, unescapeErr := url.PathUnescape(database)
		if unescapeErr != nil {
			return Options{}, fmt.Errorf("unescape database: %w", unescapeErr)
		}
		opts.Connection.Database = dbValue
	}
	if database := parsed.Query().Get("database"); database != "" {
		opts.Connection.Database = database
	}

	for key, values := range parsed.Query() {
		switch strings.ToLower(key) {
		case "input", "target", "driver", "database", "port", "encrypt_enabled", "encrypt_key", "log_failed_sql", "translate_failure_log_path", "execution_failure_log_path":
			continue
		default:
			if len(values) > 0 {
				opts.Connection.Parameters[key] = values[len(values)-1]
			}
		}
	}

	if opts.Connection.Host == "" && parsed.Host != "" {
		host, _, splitErr := net.SplitHostPort(parsed.Host)
		if splitErr == nil {
			opts.Connection.Host = host
		}
	}

	return opts.normalized(), nil
}
