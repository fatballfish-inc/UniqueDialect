package uniquedialect_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fatballfish-inc/UniqueDialect"
)

func TestBootstrapPlannerPlansPostgresArtifactsForMySQLOnUpdateTimestamp(t *testing.T) {
	t.Parallel()

	planner := uniquedialect.NewBootstrapPlanner(uniquedialect.BootstrapOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
		Enabled:       true,
	})

	artifacts, err := planner.Plan(context.Background(), "CREATE TABLE `users` (`id` bigint primary key, `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("len(artifacts) = %d, want 2", len(artifacts))
	}
	if artifacts[0].Kind != uniquedialect.BootstrapKindFunction {
		t.Fatalf("artifacts[0].Kind = %s, want %s", artifacts[0].Kind, uniquedialect.BootstrapKindFunction)
	}
	if artifacts[1].Kind != uniquedialect.BootstrapKindTrigger {
		t.Fatalf("artifacts[1].Kind = %s, want %s", artifacts[1].Kind, uniquedialect.BootstrapKindTrigger)
	}
	if !artifacts[0].Idempotent || !artifacts[1].Idempotent {
		t.Fatalf("artifacts idempotent = %#v, want true for both", artifacts)
	}
	if !strings.Contains(artifacts[0].SQL, "CREATE OR REPLACE FUNCTION") {
		t.Fatalf("function SQL = %q, want CREATE OR REPLACE FUNCTION", artifacts[0].SQL)
	}
	if !strings.Contains(artifacts[1].SQL, "CREATE TRIGGER") {
		t.Fatalf("trigger SQL = %q, want CREATE TRIGGER", artifacts[1].SQL)
	}
}

func TestBootstrapPlannerReturnsNothingWhenDisabled(t *testing.T) {
	t.Parallel()

	planner := uniquedialect.NewBootstrapPlanner(uniquedialect.BootstrapOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
		Enabled:       false,
	})

	artifacts, err := planner.Plan(context.Background(), "CREATE TABLE `users` (`updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("len(artifacts) = %d, want 0", len(artifacts))
	}
}

func TestBootstrapPlannerPlansFromCreateTableASTWithQuotedNames(t *testing.T) {
	t.Parallel()

	planner := uniquedialect.NewBootstrapPlanner(uniquedialect.BootstrapOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
		Enabled:       true,
	})

	sql := "CREATE TABLE IF NOT EXISTS `audit_logs` (\n  `id` bigint primary key,\n  `modified_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP\n)"
	artifacts, err := planner.Plan(context.Background(), sql)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("len(artifacts) = %d, want 2", len(artifacts))
	}
	if !strings.Contains(artifacts[0].SQL, "NEW.modified_at = CURRENT_TIMESTAMP;") {
		t.Fatalf("function SQL = %q, want modified_at assignment", artifacts[0].SQL)
	}
	if !strings.Contains(artifacts[1].SQL, "ON audit_logs") {
		t.Fatalf("trigger SQL = %q, want target table audit_logs", artifacts[1].SQL)
	}
}
