package uniquedialect_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fatballfish-inc/UniqueDialect"
)

func TestOpenWithOptionsNormalizesStructuredArgsForSQLite(t *testing.T) {
	t.Parallel()

	db, err := uniquedialect.OpenWithOptions(uniquedialect.Options{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
		Driver:        uniquedialect.DriverSQLite,
		Connection: uniquedialect.ConnectionOptions{
			Database: "file:ud_param_test?mode=memory&cache=shared",
		},
	})
	if err != nil {
		t.Fatalf("OpenWithOptions() error = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "CREATE TABLE `events` (id INTEGER PRIMARY KEY AUTOINCREMENT, payload TEXT, enabled BOOLEAN, created_at TEXT)"); err != nil {
		t.Fatalf("ExecContext(create table) error = %v", err)
	}

	payload := map[string]any{
		"name": "deploy",
		"meta": map[string]any{
			"source": "api",
		},
	}
	createdAt := time.Date(2026, 3, 12, 14, 30, 0, 0, time.UTC)

	if _, err := db.ExecContext(ctx, "INSERT INTO `events` (`payload`, `enabled`, `created_at`) VALUES (?, ?, ?)", payload, true, createdAt); err != nil {
		t.Fatalf("ExecContext(insert) error = %v", err)
	}

	var (
		rawPayload string
		enabled    bool
		rawCreated string
	)
	if err := db.QueryRowContext(ctx, "SELECT `payload`, `enabled`, `created_at` FROM `events` WHERE id = ? LIMIT 1", 1).Scan(&rawPayload, &enabled, &rawCreated); err != nil {
		t.Fatalf("QueryRowContext().Scan() error = %v", err)
	}
	if !enabled {
		t.Fatalf("enabled = %v, want true", enabled)
	}
	if rawCreated == "" {
		t.Fatalf("created_at = empty, want formatted timestamp")
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(rawPayload), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if decoded["name"] != "deploy" {
		t.Fatalf("decoded[name] = %#v, want deploy", decoded["name"])
	}
}
