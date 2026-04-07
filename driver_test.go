package uniquedialect_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatballfish/uniquedialect"
)

func TestOpenWithOptionsRunsQueriesAgainstSQLite(t *testing.T) {
	t.Parallel()

	db, err := uniquedialect.OpenWithOptions(uniquedialect.Options{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
		Driver:        uniquedialect.DriverSQLite,
		Connection: uniquedialect.ConnectionOptions{
			Database: "file:ud_driver_test?mode=memory&cache=shared",
		},
	})
	if err != nil {
		t.Fatalf("OpenWithOptions() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "CREATE TABLE `users` (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)"); err != nil {
		t.Fatalf("ExecContext(create table) error = %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO `users` (`name`) VALUES (?)", "alice"); err != nil {
		t.Fatalf("ExecContext(insert) error = %v", err)
	}

	var name string
	if err := db.QueryRowContext(ctx, "SELECT `name` FROM `users` WHERE id = ? LIMIT 1", 1).Scan(&name); err != nil {
		t.Fatalf("QueryRowContext().Scan() error = %v", err)
	}
	if name != "alice" {
		t.Fatalf("QueryRowContext() name = %q, want alice", name)
	}
}

func TestOpenWithOptionsLogsTranslationFailures(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	translateLogPath := filepath.Join(tempDir, "translate-failures.jsonl")
	executionLogPath := filepath.Join(tempDir, "execution-failures.jsonl")

	db, err := uniquedialect.OpenWithOptions(uniquedialect.Options{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
		Driver:        uniquedialect.DriverSQLite,
		Connection: uniquedialect.ConnectionOptions{
			Database: "file:ud_driver_translate_log_test?mode=memory&cache=shared",
		},
		Logging: uniquedialect.LoggingOptions{
			LogFailedSQL:            true,
			TranslateFailureLogPath: translateLogPath,
			ExecutionFailureLogPath: executionLogPath,
		},
	})
	if err != nil {
		t.Fatalf("OpenWithOptions() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	rows, err := db.QueryContext(ctx, "SELECT `name` FROM `users` LIMIT nope")
	if rows != nil {
		defer rows.Close()
	}
	if err == nil {
		t.Fatalf("QueryContext() error = nil, want translation error")
	}

	content, readErr := os.ReadFile(translateLogPath)
	if readErr != nil {
		t.Fatalf("ReadFile(translate log) error = %v", readErr)
	}
	logText := string(content)
	if !strings.Contains(logText, `"kind":"translation_failure"`) {
		t.Fatalf("translate log = %q, want translation_failure entry", logText)
	}
	if !strings.Contains(logText, `"stage":"query"`) {
		t.Fatalf("translate log = %q, want query stage", logText)
	}
	if !strings.Contains(logText, "\"input_sql\":\"SELECT `name` FROM `users` LIMIT nope\"") {
		t.Fatalf("translate log = %q, want original SQL", logText)
	}

	if _, statErr := os.Stat(executionLogPath); !os.IsNotExist(statErr) {
		t.Fatalf("execution log stat error = %v, want not exist", statErr)
	}
}

func TestOpenWithOptionsLogsExecutionFailures(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	translateLogPath := filepath.Join(tempDir, "translate-failures.jsonl")
	executionLogPath := filepath.Join(tempDir, "execution-failures.jsonl")

	db, err := uniquedialect.OpenWithOptions(uniquedialect.Options{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
		Driver:        uniquedialect.DriverSQLite,
		Connection: uniquedialect.ConnectionOptions{
			Database: "file:ud_driver_execution_log_test?mode=memory&cache=shared",
		},
		Logging: uniquedialect.LoggingOptions{
			LogFailedSQL:            true,
			TranslateFailureLogPath: translateLogPath,
			ExecutionFailureLogPath: executionLogPath,
		},
	})
	if err != nil {
		t.Fatalf("OpenWithOptions() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO `missing_users` (`name`) VALUES (?)", "alice"); err == nil {
		t.Fatalf("ExecContext() error = nil, want execution error")
	}

	content, readErr := os.ReadFile(executionLogPath)
	if readErr != nil {
		t.Fatalf("ReadFile(execution log) error = %v", readErr)
	}
	logText := string(content)
	if !strings.Contains(logText, `"kind":"execution_failure"`) {
		t.Fatalf("execution log = %q, want execution_failure entry", logText)
	}
	if !strings.Contains(logText, `"stage":"exec"`) {
		t.Fatalf("execution log = %q, want exec stage", logText)
	}
	if !strings.Contains(logText, "\"input_sql\":\"INSERT INTO `missing_users` (`name`) VALUES (?)\"") {
		t.Fatalf("execution log = %q, want original SQL", logText)
	}
	if !strings.Contains(logText, "\"translated_sql\":\"INSERT INTO `missing_users` (`name`) VALUES (?)\"") {
		t.Fatalf("execution log = %q, want translated SQL", logText)
	}

	if _, statErr := os.Stat(translateLogPath); !os.IsNotExist(statErr) {
		t.Fatalf("translate log stat error = %v, want not exist", statErr)
	}
}
