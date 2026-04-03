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

func TestParseOneClassifiesSetNamesAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET NAMES utf8mb4",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesSetCharacterSetAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET CHARACTER SET utf8",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesSetTransactionIsolationLevelAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET TRANSACTION ISOLATION LEVEL REPEATABLE READ",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesSetSessionTransactionIsolationLevelAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesSetTransactionReadOnlyAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET TRANSACTION READ ONLY",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesSetSessionTransactionReadWriteAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET SESSION TRANSACTION READ WRITE",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesSetGlobalTransactionReadOnlyAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET GLOBAL TRANSACTION READ ONLY",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
	}
}

func TestParseOneClassifiesSetGlobalTransactionIsolationLevelAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET GLOBAL TRANSACTION ISOLATION LEVEL READ COMMITTED",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
	}
}

func TestParseOneClassifiesGenericSetAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SET autocommit = 1",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSet {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSet)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
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

func TestParseOneClassifiesStartTransactionReadOnlyAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne("START TRANSACTION READ ONLY", uniquedialect.DialectMySQL)
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

func TestParseOneClassifiesBeginPessimisticAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne("BEGIN PESSIMISTIC", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindBegin {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindBegin)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
	}
}

func TestParseOneClassifiesStartTransactionWithConsistentSnapshotAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne("START TRANSACTION WITH CONSISTENT SNAPSHOT", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindBegin {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindBegin)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
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

func TestParseOneClassifiesCommitAndNoChainAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne("COMMIT AND NO CHAIN", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindCommit {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindCommit)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
	}
}

func TestParseOneClassifiesCommitReleaseAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne("COMMIT RELEASE", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindCommit {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindCommit)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
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

func TestParseOneClassifiesRollbackNoReleaseAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne("ROLLBACK NO RELEASE", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindRollback {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindRollback)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
	}
}

func TestParseOneClassifiesRollbackToSavepointAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne("ROLLBACK TO SAVEPOINT sp1", uniquedialect.DialectMySQL)
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

func TestParseOneClassifiesSavepointAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne("SAVEPOINT sp1", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindSavepoint {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindSavepoint)
	}
	if parsed.Status != internalparser.SupportStatusSupported {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusSupported)
	}
}

func TestParseOneClassifiesReleaseSavepointAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne("RELEASE SAVEPOINT sp1", uniquedialect.DialectMySQL)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindReleaseSavepoint {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindReleaseSavepoint)
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

func TestParseOneClassifiesShowFullTablesAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW FULL TABLES",
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

func TestParseOneClassifiesShowVariablesAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW VARIABLES",
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

func TestParseOneClassifiesShowTablesLikeAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW TABLES LIKE 'user%'",
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

func TestParseOneClassifiesShowFullTablesLikeAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW FULL TABLES LIKE 'user%'",
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

func TestParseOneClassifiesShowDatabasesLikeAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW DATABASES LIKE 'app%'",
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

func TestParseOneClassifiesShowSessionVariablesLikeAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW SESSION VARIABLES LIKE 'client_%'",
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

func TestParseOneClassifiesShowGlobalVariablesAsRecognizedUnadapted(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW GLOBAL VARIABLES",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if parsed.Kind != internalparser.StatementKindShow {
		t.Fatalf("Kind = %s, want %s", parsed.Kind, internalparser.StatementKindShow)
	}
	if parsed.Status != internalparser.SupportStatusRecognizedUnadapted {
		t.Fatalf("Status = %s, want %s", parsed.Status, internalparser.SupportStatusRecognizedUnadapted)
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

func TestParseOneClassifiesShowFullColumnsAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW FULL COLUMNS FROM `users`",
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

func TestParseOneClassifiesShowColumnsLikeAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW COLUMNS FROM `users` LIKE 'id%'",
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

func TestParseOneClassifiesShowFullColumnsLikeAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW FULL COLUMNS FROM `users` LIKE 'id%'",
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

func TestParseOneClassifiesShowTableStatusLikeAsSupported(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SHOW TABLE STATUS LIKE 'user%'",
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
