package parser_test

import (
	"testing"

	"github.com/fatballfish/uniquedialect"
	internalparser "github.com/fatballfish/uniquedialect/internal/parser"
)

func TestParseOneClassifiesCreateTable(t *testing.T) {
	parsed, err := internalparser.ParseOne("CREATE TABLE `users` (`id` bigint primary key)", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindCreateTable {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindCreateTable)
	}
}

func TestParseOnePostgresReturnsUnifiedParsedStatement(t *testing.T) {
	parsed, err := internalparser.ParseOne(`SELECT "name" FROM "users"`, uniquedialect.DialectPostgres)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSelect {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSelect)
	}
}

func TestParseOneClassifiesWithAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"WITH recent AS (SELECT `id` FROM `users`) SELECT `id` FROM recent",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindWith {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindWith)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesSetOperationAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SELECT `id` FROM `users` UNION ALL SELECT `id` FROM `admins`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSetOp {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSetOp)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneSupportsMariaDBDropForeignKeyIfExistsExtension(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"ALTER TABLE `users` DROP FOREIGN KEY IF EXISTS `fk_users_org_id`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindAlterTable {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindAlterTable)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneSupportsMariaDBDropForeignKeyIfExistsWithinAlterTableBatch(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"ALTER TABLE `users` DROP FOREIGN KEY IF EXISTS `fk_users_org_id`, DROP INDEX IF EXISTS `idx_users_org`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindAlterTable {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindAlterTable)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneSupportsMariaDBAddUniqueKeyIfNotExistsExtension(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"ALTER TABLE `users` ADD UNIQUE KEY IF NOT EXISTS `uq_users_email` (`email`)",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindAlterTable {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindAlterTable)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneSupportsMariaDBAddUniqueKeyIfNotExistsWithinAlterTableBatch(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"ALTER TABLE `users` ADD COLUMN `nickname` varchar(32), ADD UNIQUE KEY IF NOT EXISTS `uq_users_nickname` (`nickname`)",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindAlterTable {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindAlterTable)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesTruncateTableAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"TRUNCATE TABLE `users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindTruncate {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindTruncate)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesCreateViewAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"CREATE VIEW `active_users` AS SELECT `id` FROM `users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindCreateView {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindCreateView)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesDropViewAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"DROP VIEW IF EXISTS `active_users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindDropView {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindDropView)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesBeginAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne("BEGIN", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindBegin {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindBegin)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesCommitAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne("COMMIT", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindCommit {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindCommit)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesRollbackAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne("ROLLBACK", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindRollback {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindRollback)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesExplainAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"EXPLAIN SELECT `id` FROM `users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindExplain {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindExplain)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesRenameTableAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"RENAME TABLE `users` TO `customers`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindRenameTable {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindRenameTable)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesCreateDatabaseAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"CREATE DATABASE `appdb`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindCreateDatabase {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindCreateDatabase)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesDropDatabaseAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"DROP DATABASE IF EXISTS `appdb`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindDropDatabase {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindDropDatabase)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesUseAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"USE `appdb`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindUse {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindUse)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowTablesAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW TABLES",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowCreateViewAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW CREATE VIEW `active_users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowDatabasesAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW DATABASES",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowCreateDatabaseAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW CREATE DATABASE `appdb`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowColumnsAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW COLUMNS FROM `users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowIndexAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW INDEX FROM `users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowTableStatusAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW TABLE STATUS",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesShowCreateTableAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW CREATE TABLE `users`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneSupportsMariaDBDropPrimaryKeyIfExistsWithinAlterTableBatch(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"ALTER TABLE `users` DROP PRIMARY KEY IF EXISTS, DROP INDEX IF EXISTS `idx_users_org`",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindAlterTable {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindAlterTable)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}
