package uniquedialect_test

import (
	"strings"
	"testing"

	"github.com/fatballfish/uniquedialect"
)

func TestParseDSNBuildsCanonicalOptions(t *testing.T) {
	t.Parallel()

	opts, err := uniquedialect.ParseDSN("uniquedialect://tester:secret@127.0.0.1:5432/app?input=mysql&target=postgres&driver=postgres&sslmode=disable&timezone=Asia%2FShanghai&search_path=public")
	if err != nil {
		t.Fatalf("ParseDSN() error = %v", err)
	}

	if opts.InputDialect != uniquedialect.DialectMySQL {
		t.Fatalf("InputDialect = %s, want %s", opts.InputDialect, uniquedialect.DialectMySQL)
	}
	if opts.TargetDialect != uniquedialect.DialectPostgres {
		t.Fatalf("TargetDialect = %s, want %s", opts.TargetDialect, uniquedialect.DialectPostgres)
	}
	if opts.Driver != uniquedialect.DriverPostgres {
		t.Fatalf("Driver = %s, want %s", opts.Driver, uniquedialect.DriverPostgres)
	}
	if opts.Connection.Host != "127.0.0.1" {
		t.Fatalf("Connection.Host = %q, want 127.0.0.1", opts.Connection.Host)
	}
	if opts.Connection.Port != 5432 {
		t.Fatalf("Connection.Port = %d, want 5432", opts.Connection.Port)
	}
	if opts.Connection.Username != "tester" {
		t.Fatalf("Connection.Username = %q, want tester", opts.Connection.Username)
	}
	if opts.Connection.Password != "secret" {
		t.Fatalf("Connection.Password = %q, want secret", opts.Connection.Password)
	}
	if opts.Connection.Database != "app" {
		t.Fatalf("Connection.Database = %q, want app", opts.Connection.Database)
	}
	if opts.Connection.Parameters["sslmode"] != "disable" {
		t.Fatalf("Connection.Parameters[sslmode] = %q, want disable", opts.Connection.Parameters["sslmode"])
	}
	if opts.Connection.Parameters["search_path"] != "public" {
		t.Fatalf("Connection.Parameters[search_path] = %q, want public", opts.Connection.Parameters["search_path"])
	}
}

func TestOptionsBuildTargetDSNForPostgres(t *testing.T) {
	t.Parallel()

	opts := uniquedialect.Options{
		TargetDialect: uniquedialect.DialectPostgres,
		Driver:        uniquedialect.DriverPostgres,
		Connection: uniquedialect.ConnectionOptions{
			Host:     "127.0.0.1",
			Port:     5432,
			Username: "tester",
			Password: "secret",
			Database: "app",
			Parameters: map[string]string{
				"sslmode":  "disable",
				"timezone": "Asia/Shanghai",
			},
		},
	}

	dsn, err := opts.BuildDriverDSN()
	if err != nil {
		t.Fatalf("BuildDriverDSN() error = %v", err)
	}

	for _, fragment := range []string{
		"host=127.0.0.1",
		"port=5432",
		"user=tester",
		"password=secret",
		"dbname=app",
		"sslmode=disable",
		"TimeZone=Asia/Shanghai",
	} {
		if !strings.Contains(dsn, fragment) {
			t.Fatalf("BuildDriverDSN() = %q, want fragment %q", dsn, fragment)
		}
	}
}

func TestFormatDSNIncludesSQLFailureLoggingOptions(t *testing.T) {
	t.Parallel()

	dsn := uniquedialect.Options{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
		Driver:        uniquedialect.DriverPostgres,
		Logging: uniquedialect.LoggingOptions{
			LogFailedSQL:            true,
			TranslateFailureLogPath: "/tmp/translate.jsonl",
			ExecutionFailureLogPath: "/tmp/execute.jsonl",
		},
	}.FormatDSN()

	for _, fragment := range []string{
		"log_failed_sql=true",
		"translate_failure_log_path=%2Ftmp%2Ftranslate.jsonl",
		"execution_failure_log_path=%2Ftmp%2Fexecute.jsonl",
	} {
		if !strings.Contains(dsn, fragment) {
			t.Fatalf("FormatDSN() = %q, want fragment %q", dsn, fragment)
		}
	}
}

func TestParseDSNBuildsSQLFailureLoggingOptions(t *testing.T) {
	t.Parallel()

	opts, err := uniquedialect.ParseDSN("uniquedialect://tester@127.0.0.1/app?input=mysql&target=postgres&driver=postgres&log_failed_sql=true&translate_failure_log_path=%2Ftmp%2Ftranslate.jsonl&execution_failure_log_path=%2Ftmp%2Fexecute.jsonl")
	if err != nil {
		t.Fatalf("ParseDSN() error = %v", err)
	}

	if !opts.Logging.LogFailedSQL {
		t.Fatalf("Logging.LogFailedSQL = false, want true")
	}
	if got, want := opts.Logging.TranslateFailureLogPath, "/tmp/translate.jsonl"; got != want {
		t.Fatalf("Logging.TranslateFailureLogPath = %q, want %q", got, want)
	}
	if got, want := opts.Logging.ExecutionFailureLogPath, "/tmp/execute.jsonl"; got != want {
		t.Fatalf("Logging.ExecutionFailureLogPath = %q, want %q", got, want)
	}
}
