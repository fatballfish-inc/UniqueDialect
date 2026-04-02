package uniquedialect_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fatballfish/uniquedialect"
)

func TestTranslatorTranslatesMySQLToPostgresAndCaches(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	input := "SELECT `name` FROM `users` WHERE id = ? LIMIT 5, 10"

	first, err := translator.Translate(context.Background(), input, 42)
	if err != nil {
		t.Fatalf("Translate() first call error = %v", err)
	}
	if got, want := first.SQL, `SELECT "name" FROM "users" WHERE id = $1 LIMIT 10 OFFSET 5`; got != want {
		t.Fatalf("Translate() SQL = %q, want %q", got, want)
	}
	if len(first.Args) != 1 || first.Args[0] != 42 {
		t.Fatalf("Translate() args = %#v, want [42]", first.Args)
	}

	second, err := translator.Translate(context.Background(), input, 42)
	if err != nil {
		t.Fatalf("Translate() second call error = %v", err)
	}
	if second.SQL != first.SQL {
		t.Fatalf("Translate() second SQL = %q, want %q", second.SQL, first.SQL)
	}

	stats := translator.Stats()
	if stats.Misses != 1 {
		t.Fatalf("Stats().Misses = %d, want 1", stats.Misses)
	}
	if stats.Hits != 1 {
		t.Fatalf("Stats().Hits = %d, want 1", stats.Hits)
	}
}

func TestTranslatorTranslatesPostgresToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectPostgres,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		`SELECT "name" FROM "users" WHERE id = $1 ILIKE $2 LIMIT 10 OFFSET 5`,
		42,
		"AL%",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got, want := result.SQL, "SELECT `name` FROM `users` WHERE id = ? LIKE ? LIMIT 5, 10"; got != want {
		t.Fatalf("Translate() SQL = %q, want %q", got, want)
	}
}

func TestTranslatorTranslatesMySQLJoinOrderByToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"SELECT `users`.`name`, `orders`.`amount` FROM `users` LEFT JOIN `orders` ON `users`.`id` = `orders`.`user_id` WHERE `users`.`status` = ? ORDER BY `orders`.`created_at` DESC LIMIT 10",
		"active",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT "users"."name", "orders"."amount" FROM "users" LEFT JOIN "orders" ON "users"."id" = "orders"."user_id" WHERE "users"."status" = $1 ORDER BY "orders"."created_at" DESC LIMIT 10`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesPostgresOnConflictToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectPostgres,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		`INSERT INTO "users" ("id", "name") VALUES ($1, $2) ON CONFLICT ("id") DO UPDATE SET "name" = EXCLUDED."name"`,
		1,
		"alice",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "INSERT INTO `users` (`id`, `name`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`)"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLOnDuplicateKeyToSQLiteUpsert(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"INSERT INTO `users` (`id`, `name`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`)",
		1,
		"alice",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "INSERT INTO `users` (`id`, `name`) VALUES (?, ?) ON CONFLICT (`id`) DO UPDATE SET `name` = excluded.`name`"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLUpdateToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"UPDATE `users` SET `name` = ?, `updated_at` = NOW() WHERE `id` = ?",
		"alice",
		1,
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `UPDATE "users" SET "name" = $1, "updated_at" = CURRENT_TIMESTAMP WHERE "id" = $2`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesPostgresDeleteToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectPostgres,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		`DELETE FROM "users" WHERE "deleted_at" IS NOT NULL AND "status" = $1`,
		"inactive",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "DELETE FROM `users` WHERE `deleted_at` IS NOT NULL AND `status` = ?"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLWithOffsetLimitToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"WITH recent AS (SELECT `id` FROM `users`) SELECT `id` FROM recent LIMIT 5, 10",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `WITH recent AS (SELECT "id" FROM "users") SELECT "id" FROM recent LIMIT 10 OFFSET 5`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLWithToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"WITH recent AS (SELECT `id` FROM `users` WHERE `status` = ?) SELECT recent.`id` FROM recent",
		"active",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `WITH recent AS (SELECT "id" FROM "users" WHERE "status" = $1) SELECT recent."id" FROM recent`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLUnionAllToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"SELECT `id` FROM `users` UNION ALL SELECT `id` FROM `admins`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT "id" FROM "users" UNION ALL SELECT "id" FROM "admins"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLUnionAllOffsetLimitToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"SELECT `id` FROM `users` UNION ALL SELECT `id` FROM `admins` LIMIT 2, 3",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT "id" FROM "users" UNION ALL SELECT "id" FROM "admins" LIMIT 3 OFFSET 2`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE IF NOT EXISTS `users` (`id` bigint, `name` varchar(255))",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE IF NOT EXISTS "users" ("id" bigint, "name" varchar(255))`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableToPostgresSemantically(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE IF NOT EXISTS `users` (`id` bigint NOT NULL AUTO_INCREMENT, `name` varchar(255) NOT NULL, `created_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE IF NOT EXISTS "users" ("id" bigint GENERATED BY DEFAULT AS IDENTITY NOT NULL, "name" varchar(255) NOT NULL, "created_at" timestamp DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY ("id"))`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLDropTableToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"DROP TABLE IF EXISTS `users`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `DROP TABLE IF EXISTS "users"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLTruncateTableToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"TRUNCATE TABLE `users`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `TRUNCATE TABLE "users"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateViewToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE VIEW `active_users` AS SELECT `id` FROM `users`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE VIEW "active_users" AS SELECT "id" FROM "users"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLDropViewToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"DROP VIEW IF EXISTS `active_users`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `DROP VIEW IF EXISTS "active_users"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLBeginToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "BEGIN")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "BEGIN" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "BEGIN")
	}
}

func TestTranslatorTranslatesMySQLCommitToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "COMMIT")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "COMMIT" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "COMMIT")
	}
}

func TestTranslatorTranslatesMySQLRollbackToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "ROLLBACK")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "ROLLBACK" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "ROLLBACK")
	}
}

func TestTranslatorTranslatesMySQLExplainToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"EXPLAIN SELECT `id` FROM `users`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `EXPLAIN SELECT "id" FROM "users"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLRenameTableToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"RENAME TABLE `users` TO `customers`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" RENAME TO "customers"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLRenameTableBatchToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"RENAME TABLE `users` TO `customers`, `orders` TO `purchases`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" RENAME TO "customers"; ALTER TABLE "orders" RENAME TO "purchases"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLRenameTableToMySQLViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"RENAME TABLE `users` TO `customers`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "RENAME TABLE `users` TO `customers`"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLQualifiedRenameTableToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"RENAME TABLE `app`.`users` TO `app`.`customers`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "app"."users" RENAME TO "customers"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateDatabaseToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE DATABASE `appdb`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE DATABASE "appdb"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLDropDatabaseToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"DROP DATABASE IF EXISTS `appdb`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `DROP DATABASE IF EXISTS "appdb"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLUseToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"USE `appdb`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SET search_path TO "appdb"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLSetNamesToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SET NAMES utf8mb4")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SET client_encoding TO 'UTF8'" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SET client_encoding TO 'UTF8'")
	}
}

func TestTranslatorTranslatesMySQLSetCharacterSetToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SET CHARACTER SET utf8")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SET client_encoding TO 'UTF8'" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SET client_encoding TO 'UTF8'")
	}
}

func TestTranslatorRejectsUnsupportedMySQLSetNamesCharsetToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SET NAMES latin1")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SET charset error")
	}
	if !strings.Contains(err.Error(), "unsupported SET charset") {
		t.Fatalf("Translate() error = %v, want unsupported SET charset error", err)
	}
}

func TestTranslatorRejectsUnsupportedMySQLSetNamesCollateToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SET variant error")
	}
	if !strings.Contains(err.Error(), "unsupported SET variant") {
		t.Fatalf("Translate() error = %v, want unsupported SET variant error", err)
	}
}

func TestTranslatorRejectsMySQLSetNamesToSQLiteViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SET NAMES utf8mb4")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SET target dialect error")
	}
	if !strings.Contains(err.Error(), "unsupported SET target dialect sqlite") {
		t.Fatalf("Translate() error = %v, want unsupported SET target dialect sqlite", err)
	}
}

func TestTranslatorRejectsMySQLSetNamesToOracleViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectOracle,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SET NAMES utf8mb4")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SET target dialect error")
	}
	if !strings.Contains(err.Error(), "unsupported SET target dialect oracle") {
		t.Fatalf("Translate() error = %v, want unsupported SET target dialect oracle", err)
	}
}

func TestTranslatorBlocksGenericMySQLSetToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SET autocommit = 1")
	if err == nil {
		t.Fatalf("Translate() error = nil, want parser adaptation error")
	}

	var adaptationErr *uniquedialect.ParserAdaptationError
	if !errors.As(err, &adaptationErr) {
		t.Fatalf("Translate() error = %T, want *ParserAdaptationError", err)
	}
	if adaptationErr.StatementKind != "set" {
		t.Fatalf("ParserAdaptationError.StatementKind = %q, want %q", adaptationErr.StatementKind, "set")
	}
	if adaptationErr.Status != "recognized_unadapted" {
		t.Fatalf("ParserAdaptationError.Status = %q, want %q", adaptationErr.Status, "recognized_unadapted")
	}
}

func TestTranslatorTranslatesMySQLShowTablesToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW TABLES")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT table_name AS "Tables_in_current_schema" FROM information_schema.tables WHERE table_schema = current_schema() AND table_type IN ('BASE TABLE', 'VIEW') ORDER BY table_name`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowCreateViewToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE VIEW `active_users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT c.relname AS "View", 'CREATE VIEW ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname) || ' AS ' || pg_get_viewdef(c.oid, true) AS "Create View" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'v' AND c.relname = 'active_users' AND n.nspname = current_schema()`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowCreateViewWithSchemaToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE VIEW `appdb`.`active_users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT c.relname AS "View", 'CREATE VIEW ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname) || ' AS ' || pg_get_viewdef(c.oid, true) AS "Create View" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'v' AND c.relname = 'active_users' AND n.nspname = 'appdb'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowCreateViewWithDottedIdentifierToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE VIEW `my.view`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if !strings.Contains(result.SQL, "c.relname = 'my.view'") {
		t.Fatalf("Translate() SQL = %q, want dotted identifier preserved as view name", result.SQL)
	}
	if strings.Contains(result.SQL, "n.nspname = 'my'") {
		t.Fatalf("Translate() SQL = %q, want no mistaken schema split for dotted identifier", result.SQL)
	}
}

func TestTranslatorTranslatesMySQLShowCreateViewWithSchemaToMySQLViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE VIEW `appdb`.`active_users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SHOW CREATE VIEW `appdb`.`active_users`" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SHOW CREATE VIEW `appdb`.`active_users`")
	}
}

func TestTranslatorTranslatesMySQLShowCreateViewWithDottedIdentifierToMySQLViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE VIEW `my.view`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SHOW CREATE VIEW `my.view`" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SHOW CREATE VIEW `my.view`")
	}
}

func TestTranslatorTranslatesMySQLShowVariablesToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW VARIABLES")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT name AS "Variable_name", setting AS "Value" FROM pg_catalog.pg_settings ORDER BY name`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowSessionVariablesLikeToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW SESSION VARIABLES LIKE 'client_%'")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT name AS "Variable_name", setting AS "Value" FROM pg_catalog.pg_settings WHERE name ILIKE 'client_%' ORDER BY name`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorBlocksUnadaptedMySQLShowGlobalVariablesToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW GLOBAL VARIABLES")
	if err == nil {
		t.Fatalf("Translate() error = nil, want parser adaptation error")
	}

	var adaptationErr *uniquedialect.ParserAdaptationError
	if !errors.As(err, &adaptationErr) {
		t.Fatalf("Translate() error = %T, want *ParserAdaptationError", err)
	}
	if adaptationErr.StatementKind != "show" {
		t.Fatalf("ParserAdaptationError.StatementKind = %q, want %q", adaptationErr.StatementKind, "show")
	}
	if adaptationErr.Status != "recognized_unadapted" {
		t.Fatalf("ParserAdaptationError.Status = %q, want %q", adaptationErr.Status, "recognized_unadapted")
	}
}

func TestTranslatorRejectsUnsupportedMySQLShowVariablesWhereToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW VARIABLES WHERE Variable_name = 'client_encoding'")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SHOW VARIABLES variant error")
	}
	if !strings.Contains(err.Error(), "unsupported SHOW VARIABLES variant") {
		t.Fatalf("Translate() error = %v, want unsupported SHOW VARIABLES variant error", err)
	}
}

func TestTranslatorTranslatesMySQLShowDatabasesToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW DATABASES")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT datname AS "Database" FROM pg_database WHERE datistemplate = false ORDER BY datname`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowCreateDatabaseToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE DATABASE `appdb`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT datname AS "Database", 'CREATE DATABASE ' || quote_ident(datname) || ' ENCODING = ' || quote_literal(pg_encoding_to_char(encoding)) || ' LC_COLLATE = ' || quote_literal(datcollate) || ' LC_CTYPE = ' || quote_literal(datctype) AS "Create Database" FROM pg_database WHERE datname = 'appdb'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowColumnsToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW COLUMNS FROM `users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT a.attname AS "Field", format_type(a.atttypid, a.atttypmod) AS "Type", CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END AS "Null", CASE WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisprimary AND a.attnum = ANY(ix.indkey)) THEN 'PRI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisunique AND ix.indnkeyatts = 1 AND a.attnum = ANY(ix.indkey)) THEN 'UNI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND a.attnum = ANY(ix.indkey)) THEN 'MUL' ELSE '' END AS "Key", pg_get_expr(ad.adbin, ad.adrelid) AS "Default", CASE WHEN a.attidentity IN ('a', 'd') THEN 'auto_increment' WHEN a.attgenerated = 's' THEN 'STORED GENERATED' ELSE '' END AS "Extra" FROM pg_attribute a JOIN pg_class t ON t.oid = a.attrelid JOIN pg_namespace n ON n.oid = t.relnamespace LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum WHERE t.relkind IN ('r', 'p', 'v', 'm', 'f') AND t.relname = 'users' AND pg_table_is_visible(t.oid) AND a.attnum > 0 AND NOT a.attisdropped ORDER BY a.attnum`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowColumnsInDatabaseToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW COLUMNS FROM `users` IN `appdb`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT a.attname AS "Field", format_type(a.atttypid, a.atttypmod) AS "Type", CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END AS "Null", CASE WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisprimary AND a.attnum = ANY(ix.indkey)) THEN 'PRI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisunique AND ix.indnkeyatts = 1 AND a.attnum = ANY(ix.indkey)) THEN 'UNI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND a.attnum = ANY(ix.indkey)) THEN 'MUL' ELSE '' END AS "Key", pg_get_expr(ad.adbin, ad.adrelid) AS "Default", CASE WHEN a.attidentity IN ('a', 'd') THEN 'auto_increment' WHEN a.attgenerated = 's' THEN 'STORED GENERATED' ELSE '' END AS "Extra" FROM pg_attribute a JOIN pg_class t ON t.oid = a.attrelid JOIN pg_namespace n ON n.oid = t.relnamespace LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum WHERE t.relkind IN ('r', 'p', 'v', 'm', 'f') AND t.relname = 'users' AND n.nspname = 'appdb' AND a.attnum > 0 AND NOT a.attisdropped ORDER BY a.attnum`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowColumnsWithDottedIdentifierToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW COLUMNS FROM `my.table`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT a.attname AS "Field", format_type(a.atttypid, a.atttypmod) AS "Type", CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END AS "Null", CASE WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisprimary AND a.attnum = ANY(ix.indkey)) THEN 'PRI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisunique AND ix.indnkeyatts = 1 AND a.attnum = ANY(ix.indkey)) THEN 'UNI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND a.attnum = ANY(ix.indkey)) THEN 'MUL' ELSE '' END AS "Key", pg_get_expr(ad.adbin, ad.adrelid) AS "Default", CASE WHEN a.attidentity IN ('a', 'd') THEN 'auto_increment' WHEN a.attgenerated = 's' THEN 'STORED GENERATED' ELSE '' END AS "Extra" FROM pg_attribute a JOIN pg_class t ON t.oid = a.attrelid JOIN pg_namespace n ON n.oid = t.relnamespace LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum WHERE t.relkind IN ('r', 'p', 'v', 'm', 'f') AND t.relname = 'my.table' AND pg_table_is_visible(t.oid) AND a.attnum > 0 AND NOT a.attisdropped ORDER BY a.attnum`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowIndexToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW INDEX FROM `users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT t.relname AS "Table", CASE WHEN ix.indisunique THEN 0 ELSE 1 END AS "Non_unique", i.relname AS "Key_name", k.ordinality AS "Seq_in_index", CASE WHEN k.attnum = 0 THEN NULL ELSE a.attname END AS "Column_name", 'A' AS "Collation", NULL AS "Cardinality", NULL AS "Sub_part", NULL AS "Packed", CASE WHEN a.attnotnull THEN '' ELSE 'YES' END AS "Null", am.amname AS "Index_type", '' AS "Comment", '' AS "Index_comment", 'YES' AS "Visible", CASE WHEN k.attnum = 0 THEN pg_get_indexdef(ix.indexrelid, k.ordinality, true) ELSE NULL END AS "Expression" FROM pg_index ix JOIN pg_class t ON t.oid = ix.indrelid JOIN pg_class i ON i.oid = ix.indexrelid JOIN pg_am am ON am.oid = i.relam JOIN pg_namespace n ON n.oid = t.relnamespace JOIN LATERAL unnest(string_to_array(ix.indkey::text, ' ')::smallint[]) WITH ORDINALITY AS k(attnum, ordinality) ON TRUE LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum WHERE t.relkind IN ('r', 'p') AND t.relname = 'users' AND pg_table_is_visible(t.oid) AND k.ordinality <= ix.indnkeyatts ORDER BY i.relname, k.ordinality`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowIndexToPostgresExcludesIncludeColumns(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW INDEX FROM `users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if !strings.Contains(result.SQL, "k.ordinality <= ix.indnkeyatts") {
		t.Fatalf("Translate() SQL = %q, want predicate excluding INCLUDE columns", result.SQL)
	}
}

func TestTranslatorTranslatesMySQLShowIndexToPostgresAlignsExpressionPerKeyPart(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW INDEX FROM `users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if !strings.Contains(result.SQL, `CASE WHEN k.attnum = 0 THEN NULL ELSE a.attname END AS "Column_name"`) {
		t.Fatalf("Translate() SQL = %q, want Column_name guarded for expression key parts", result.SQL)
	}
	if !strings.Contains(result.SQL, `CASE WHEN k.attnum = 0 THEN pg_get_indexdef(ix.indexrelid, k.ordinality, true) ELSE NULL END AS "Expression"`) {
		t.Fatalf("Translate() SQL = %q, want Expression aligned per key part", result.SQL)
	}
}

func TestTranslatorTranslatesMySQLShowIndexWithDottedIdentifierToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW INDEX FROM `my.table`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if !strings.Contains(result.SQL, "t.relname = 'my.table'") {
		t.Fatalf("Translate() SQL = %q, want dotted identifier preserved as table name", result.SQL)
	}
	if strings.Contains(result.SQL, "n.nspname = 'my'") {
		t.Fatalf("Translate() SQL = %q, want no mistaken schema split for dotted identifier", result.SQL)
	}
}

func TestTranslatorTranslatesMySQLShowTableStatusToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW TABLE STATUS")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT c.relname AS "Name", COALESCE(am.amname, 'heap') AS "Engine", NULL AS "Version", NULL AS "Row_format", CASE WHEN c.reltuples < 0 THEN NULL ELSE c.reltuples::bigint END AS "Rows", CASE WHEN c.reltuples > 0 THEN pg_relation_size(c.oid) / NULLIF(c.reltuples::bigint, 0) ELSE NULL END AS "Avg_row_length", pg_relation_size(c.oid) AS "Data_length", NULL AS "Max_data_length", pg_indexes_size(c.oid) AS "Index_length", NULL AS "Data_free", NULL AS "Auto_increment", NULL AS "Create_time", NULL AS "Update_time", NULL AS "Check_time", NULL AS "Collation", NULL AS "Checksum", NULL AS "Create_options", COALESCE(obj_description(c.oid, 'pg_class'), '') AS "Comment" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace LEFT JOIN pg_am am ON am.oid = c.relam WHERE c.relkind IN ('r', 'p') AND pg_table_is_visible(c.oid) ORDER BY c.relname`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowTableStatusInDatabaseToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW TABLE STATUS IN `appdb`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT c.relname AS "Name", COALESCE(am.amname, 'heap') AS "Engine", NULL AS "Version", NULL AS "Row_format", CASE WHEN c.reltuples < 0 THEN NULL ELSE c.reltuples::bigint END AS "Rows", CASE WHEN c.reltuples > 0 THEN pg_relation_size(c.oid) / NULLIF(c.reltuples::bigint, 0) ELSE NULL END AS "Avg_row_length", pg_relation_size(c.oid) AS "Data_length", NULL AS "Max_data_length", pg_indexes_size(c.oid) AS "Index_length", NULL AS "Data_free", NULL AS "Auto_increment", NULL AS "Create_time", NULL AS "Update_time", NULL AS "Check_time", NULL AS "Collation", NULL AS "Checksum", NULL AS "Create_options", COALESCE(obj_description(c.oid, 'pg_class'), '') AS "Comment" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace LEFT JOIN pg_am am ON am.oid = c.relam WHERE c.relkind IN ('r', 'p') AND n.nspname = 'appdb' ORDER BY c.relname`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowTableStatusToMySQLViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW TABLE STATUS IN `appdb`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SHOW TABLE STATUS IN `appdb`" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SHOW TABLE STATUS IN `appdb`")
	}
}

func TestTranslatorRejectsUnsupportedMySQLShowTableStatusLikeToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW TABLE STATUS LIKE 'user%'")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SHOW TABLE STATUS variant error")
	}
	if !strings.Contains(err.Error(), "unsupported SHOW TABLE STATUS variant") {
		t.Fatalf("Translate() error = %v, want unsupported SHOW TABLE STATUS variant error", err)
	}
}

func TestTranslatorRejectsUnsupportedMySQLShowTableStatusWhereToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW TABLE STATUS WHERE Name = 'users'")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SHOW TABLE STATUS variant error")
	}
	if !strings.Contains(err.Error(), "unsupported SHOW TABLE STATUS variant") {
		t.Fatalf("Translate() error = %v, want unsupported SHOW TABLE STATUS variant error", err)
	}
}

func TestTranslatorRejectsMySQLShowTableStatusToSQLiteViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW TABLE STATUS")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SHOW target dialect error")
	}
	if !strings.Contains(err.Error(), "unsupported SHOW target dialect sqlite") {
		t.Fatalf("Translate() error = %v, want unsupported SHOW target dialect sqlite", err)
	}
}

func TestTranslatorRejectsMySQLShowColumnsToSQLiteViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW COLUMNS FROM `users`")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SHOW target dialect error")
	}
	if !strings.Contains(err.Error(), "unsupported SHOW target dialect sqlite") {
		t.Fatalf("Translate() error = %v, want unsupported SHOW target dialect sqlite", err)
	}
}

func TestTranslatorRejectsUnsupportedMySQLShowFullColumnsToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW FULL COLUMNS FROM `users`")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SHOW COLUMNS variant error")
	}
	if !strings.Contains(err.Error(), "unsupported SHOW COLUMNS variant") {
		t.Fatalf("Translate() error = %v, want unsupported SHOW COLUMNS variant error", err)
	}
}

func TestTranslatorTranslatesMySQLShowCreateTableToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE TABLE `users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT c.relname AS "Table", 'CREATE TABLE ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname) || E' (\n' || cols.columns || COALESCE(E',\n' || cons.constraints, '') || E'\n)' AS "Create Table" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace JOIN LATERAL (SELECT string_agg('  ' || quote_ident(a.attname) || ' ' || format_type(a.atttypid, a.atttypmod) || CASE WHEN a.attidentity = 'a' THEN ' GENERATED ALWAYS AS IDENTITY' WHEN a.attidentity = 'd' THEN ' GENERATED BY DEFAULT AS IDENTITY' WHEN a.attgenerated = 's' THEN ' GENERATED ALWAYS AS (' || pg_get_expr(ad.adbin, ad.adrelid) || ') STORED' ELSE '' END || CASE WHEN a.attgenerated = '' AND ad.adbin IS NOT NULL THEN ' DEFAULT ' || pg_get_expr(ad.adbin, ad.adrelid) ELSE '' END || CASE WHEN a.attnotnull THEN ' NOT NULL' ELSE '' END, E',\n' ORDER BY a.attnum) AS columns FROM pg_attribute a LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped) cols ON TRUE LEFT JOIN LATERAL (SELECT string_agg('  CONSTRAINT ' || quote_ident(con.conname) || ' ' || pg_get_constraintdef(con.oid, true), E',\n' ORDER BY con.conname) AS constraints FROM pg_constraint con WHERE con.conrelid = c.oid) cons ON TRUE WHERE c.relkind IN ('r', 'p') AND c.relname = 'users' AND pg_table_is_visible(c.oid)`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowCreateTableWithSchemaToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE TABLE `appdb`.`users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `SELECT c.relname AS "Table", 'CREATE TABLE ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname) || E' (\n' || cols.columns || COALESCE(E',\n' || cons.constraints, '') || E'\n)' AS "Create Table" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace JOIN LATERAL (SELECT string_agg('  ' || quote_ident(a.attname) || ' ' || format_type(a.atttypid, a.atttypmod) || CASE WHEN a.attidentity = 'a' THEN ' GENERATED ALWAYS AS IDENTITY' WHEN a.attidentity = 'd' THEN ' GENERATED BY DEFAULT AS IDENTITY' WHEN a.attgenerated = 's' THEN ' GENERATED ALWAYS AS (' || pg_get_expr(ad.adbin, ad.adrelid) || ') STORED' ELSE '' END || CASE WHEN a.attgenerated = '' AND ad.adbin IS NOT NULL THEN ' DEFAULT ' || pg_get_expr(ad.adbin, ad.adrelid) ELSE '' END || CASE WHEN a.attnotnull THEN ' NOT NULL' ELSE '' END, E',\n' ORDER BY a.attnum) AS columns FROM pg_attribute a LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped) cols ON TRUE LEFT JOIN LATERAL (SELECT string_agg('  CONSTRAINT ' || quote_ident(con.conname) || ' ' || pg_get_constraintdef(con.oid, true), E',\n' ORDER BY con.conname) AS constraints FROM pg_constraint con WHERE con.conrelid = c.oid) cons ON TRUE WHERE c.relkind IN ('r', 'p') AND c.relname = 'users' AND n.nspname = 'appdb'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLShowCreateTableWithDottedIdentifierToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE TABLE `my.table`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if !strings.Contains(result.SQL, "c.relname = 'my.table'") {
		t.Fatalf("Translate() SQL = %q, want dotted identifier preserved as table name", result.SQL)
	}
	if strings.Contains(result.SQL, "n.nspname = 'my'") {
		t.Fatalf("Translate() SQL = %q, want no mistaken schema split for dotted identifier", result.SQL)
	}
}

func TestTranslatorTranslatesMySQLShowCreateTableToMySQLViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE TABLE `users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SHOW CREATE TABLE `users`" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SHOW CREATE TABLE `users`")
	}
}

func TestTranslatorTranslatesMySQLShowCreateTableWithSchemaToMySQLViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE TABLE `appdb`.`users`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SHOW CREATE TABLE `appdb`.`users`" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SHOW CREATE TABLE `appdb`.`users`")
	}
}

func TestTranslatorTranslatesMySQLShowCreateTableWithDottedIdentifierToMySQLViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(context.Background(), "SHOW CREATE TABLE `my.table`")
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if result.SQL != "SHOW CREATE TABLE `my.table`" {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, "SHOW CREATE TABLE `my.table`")
	}
}

func TestTranslatorRejectsMySQLShowCreateTableToSQLiteViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectSQLite,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	_, err = translator.Translate(context.Background(), "SHOW CREATE TABLE `users`")
	if err == nil {
		t.Fatalf("Translate() error = nil, want unsupported SHOW target dialect error")
	}
	if !strings.Contains(err.Error(), "unsupported SHOW target dialect sqlite") {
		t.Fatalf("Translate() error = %v, want unsupported SHOW target dialect sqlite", err)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddColumnToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `email` varchar(255) NOT NULL",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "email" varchar(255) NOT NULL`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddAutoIncrementColumnToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `external_id` bigint NOT NULL AUTO_INCREMENT",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "external_id" bigint GENERATED BY DEFAULT AS IDENTITY NOT NULL`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropColumnToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP COLUMN `legacy_name`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP COLUMN "legacy_name"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableSetDefaultToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ALTER COLUMN `status` SET DEFAULT 'active'",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ALTER COLUMN "status" SET DEFAULT 'active'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableModifyColumnToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` MODIFY COLUMN `status` varchar(32) NOT NULL",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ALTER COLUMN "status" TYPE varchar(32), ALTER COLUMN "status" SET NOT NULL`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableChangeColumnToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` CHANGE COLUMN `full_name` `display_name` varchar(255) NOT NULL",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" RENAME COLUMN "full_name" TO "display_name", ALTER COLUMN "display_name" TYPE varchar(255), ALTER COLUMN "display_name" SET NOT NULL`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddUniqueKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD UNIQUE KEY `uq_users_email` (`email`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD CONSTRAINT "uq_users_email" UNIQUE ("email")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddUniqueKeyWithColumnAddToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `nickname` varchar(32), ADD UNIQUE KEY `uq_users_nickname` (`nickname`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "nickname" varchar(32); ALTER TABLE "users" ADD CONSTRAINT "uq_users_nickname" UNIQUE ("nickname")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddUniqueKeyMultiIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD UNIQUE KEY `uq_users_email` (`email`), ADD UNIQUE KEY `uq_users_ref` (`ref_id`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD CONSTRAINT "uq_users_email" UNIQUE ("email"); ALTER TABLE "users" ADD CONSTRAINT "uq_users_ref" UNIQUE ("ref_id")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddUniqueKeyIfNotExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD UNIQUE KEY IF NOT EXISTS `uq_users_email` (`email`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD CONSTRAINT "uq_users_email" UNIQUE ("email")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddColumnAndAddUniqueKeyIfNotExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `nickname` varchar(32), ADD UNIQUE KEY IF NOT EXISTS `uq_users_nickname` (`nickname`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "nickname" varchar(32); ALTER TABLE "users" ADD CONSTRAINT "uq_users_nickname" UNIQUE ("nickname")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddPrimaryKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD PRIMARY KEY (`id`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD PRIMARY KEY ("id")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropPrimaryKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP PRIMARY KEY",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP CONSTRAINT "users_pkey"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateIndexToPostgresViaParserHooks(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" ("name")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateUniquePrefixIndexToPostgresSemantically(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE UNIQUE INDEX `idx_users_name` ON `users` (`name`(20))",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE UNIQUE INDEX "idx_users_name" ON "users" ("name")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableWithTableLevelUniqueKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE `users` (`id` bigint NOT NULL, `email` varchar(255) NOT NULL, UNIQUE KEY `uq_users_email` (`email`))",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE "users" ("id" bigint NOT NULL, "email" varchar(255) NOT NULL, CONSTRAINT "uq_users_email" UNIQUE ("email"))`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableWithTableLevelUniqueIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE `users` (`id` bigint NOT NULL, `email` varchar(255) NOT NULL, UNIQUE INDEX `uq_users_email` (`email`))",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE "users" ("id" bigint NOT NULL, "email" varchar(255) NOT NULL, CONSTRAINT "uq_users_email" UNIQUE ("email"))`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableWithAnonymousForeignKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE `users` (`org_id` bigint, FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE "users" ("org_id" bigint, FOREIGN KEY ("org_id") REFERENCES "orgs" ("id") ON DELETE CASCADE)`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableWithNamedForeignKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE `users` (`org_id` bigint, CONSTRAINT `fk_users_org_id` FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE "users" ("org_id" bigint, CONSTRAINT "fk_users_org_id" FOREIGN KEY ("org_id") REFERENCES "orgs" ("id") ON DELETE CASCADE ON UPDATE RESTRICT)`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableWithCheckConstraintToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE `users` (`age` int, CHECK (`age` > 0))",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE "users" ("age" int, CHECK ("age" > 0))`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateIndexUsingHashToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`) USING HASH",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" USING HASH ("name")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateIndexWithWhereToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`) WHERE `name` IS NOT NULL",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" ("name") WHERE "name" IS NOT NULL`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateIndexWithCommentToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`) COMMENT 'lookup by name'",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" ("name"); COMMENT ON INDEX "idx_users_name" IS 'lookup by name'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateIndexSafelyDropsMySQLOptionsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`) USING HASH WITH PARSER `ngram` VISIBLE WHERE `name` IS NOT NULL ALGORITHM = INPLACE LOCK = NONE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" USING HASH ("name") WHERE "name" IS NOT NULL`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesSQLiteCompatCreateIndexWithInvisibleAndLockAlgorithmToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`) INVISIBLE ALGORITHM = INPLACE LOCK = NONE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "CREATE INDEX `idx_users_name` ON `users` (`name`) INVISIBLE ALGORITHM = INPLACE LOCK = NONE"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesSQLiteCompatCreateIndexIfNotExistsAndVisibleToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX IF NOT EXISTS `idx_users_name` ON `users` (`name`) VISIBLE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "CREATE INDEX IF NOT EXISTS `idx_users_name` ON `users` (`name`) VISIBLE"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesSQLiteCompatCreateIndexWithCommentVisibilityWhereAndLockAlgorithmToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`) COMMENT 'hot path' VISIBLE WHERE `name` IS NOT NULL ALGORITHM = INPLACE LOCK = NONE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "CREATE INDEX `idx_users_name` ON `users` (`name`) COMMENT 'hot path' VISIBLE WHERE `name` IS NOT NULL ALGORITHM = INPLACE LOCK = NONE"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesSQLiteCompatCreateIndexWithParserCommentVisibilityAndWhereToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE INDEX `idx_users_name` ON `users` (`name`) WITH PARSER `ngram` COMMENT 'hot path' VISIBLE WHERE `name` IS NOT NULL",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "CREATE INDEX `idx_users_name` ON `users` (`name`) WITH PARSER `ngram` COMMENT 'hot path' VISIBLE WHERE `name` IS NOT NULL"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableConstraintForeignKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE `orders` (`id` bigint primary key, `user_id` bigint, CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE "orders" ("id" bigint PRIMARY KEY, "user_id" bigint, CONSTRAINT "fk_orders_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE)`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLCreateTableAnonymousCheckToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"CREATE TABLE `products` (`id` bigint primary key, `price` numeric, CHECK (`price` > 0))",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE TABLE "products" ("id" bigint PRIMARY KEY, "price" decimal(10,0), CHECK ("price" > 0))`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLDropIndexToPostgresSemantically(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"DROP INDEX IF EXISTS `idx_users_name` ON `users`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `DROP INDEX IF EXISTS "idx_users_name"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP INDEX `idx_users_name`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `DROP INDEX "idx_users_name"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropIndexIfExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP INDEX IF EXISTS `idx_users_name`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `DROP INDEX IF EXISTS "idx_users_name"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD INDEX `idx_users_name` (`name`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" ("name")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddIndexIfNotExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD INDEX IF NOT EXISTS `idx_users_name` (`name`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX IF NOT EXISTS "idx_users_name" ON "users" ("name")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesSQLiteCompatAlterTableAddIndexIfNotExistsAndVisibleToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD INDEX IF NOT EXISTS `idx_users_name` (`name`) VISIBLE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "CREATE INDEX IF NOT EXISTS `idx_users_name` ON `users` (`name`) VISIBLE"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesSQLiteCompatAlterTableAddIndexWithParserAndInvisibleToMySQL(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectSQLite,
		TargetDialect: uniquedialect.DialectMySQL,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD INDEX `idx_users_name` (`name`) WITH PARSER `ngram` INVISIBLE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := "CREATE INDEX `idx_users_name` ON `users` (`name`) WITH PARSER `ngram` INVISIBLE"
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddColumnAndAddIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `nickname` varchar(32), ADD INDEX `idx_users_name` (`name`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "nickname" varchar(32); CREATE INDEX "idx_users_name" ON "users" ("name")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddTwoIndexesToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD INDEX `idx_users_name` (`name`), ADD INDEX `idx_users_email` (`email`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" ("name"); CREATE INDEX "idx_users_email" ON "users" ("email")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropColumnAndDropIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP COLUMN `legacy_name`, DROP INDEX IF EXISTS `idx_users_name`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP COLUMN "legacy_name"; DROP INDEX IF EXISTS "idx_users_name"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddColumnAndAddIndexUsingHashToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `nickname` varchar(32) NOT NULL, ADD INDEX `idx_users_name` (`name`) USING HASH",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "nickname" varchar(32) NOT NULL; CREATE INDEX "idx_users_name" ON "users" USING HASH ("name")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddColumnAndAddIndexWithCommentToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `nickname` varchar(32) NOT NULL, ADD INDEX `idx_users_name` (`name`) COMMENT 'lookup by name'",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "nickname" varchar(32) NOT NULL; CREATE INDEX "idx_users_name" ON "users" ("name"); COMMENT ON INDEX "idx_users_name" IS 'lookup by name'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddIndexWithCommentToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD INDEX `idx_users_name` (`name`) COMMENT 'hot path'",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" ("name"); COMMENT ON INDEX "idx_users_name" IS 'hot path'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddIndexSafelyDropsMySQLOptionsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD INDEX `idx_users_name` (`name`) WITH PARSER `ngram` COMMENT 'lookup by name' INVISIBLE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" ("name"); COMMENT ON INDEX "idx_users_name" IS 'lookup by name'`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddForeignKeyAndAddIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD CONSTRAINT `fk_users_org_id` FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`), ADD INDEX `idx_users_org` (`org_id`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD CONSTRAINT "fk_users_org_id" FOREIGN KEY ("org_id") REFERENCES "orgs" ("id"); CREATE INDEX "idx_users_org" ON "users" ("org_id")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddKeyUsingHashThenAddColumnToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD KEY `idx_users_name` (`name`) USING HASH, ADD COLUMN `nickname` varchar(32)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `CREATE INDEX "idx_users_name" ON "users" USING HASH ("name"); ALTER TABLE "users" ADD COLUMN "nickname" varchar(32)`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropForeignKeyAndDropIndexToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP FOREIGN KEY `fk_users_org_id`, DROP INDEX `idx_users_org`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP CONSTRAINT "fk_users_org_id"; DROP INDEX "idx_users_org"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropIndexThenDropForeignKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP INDEX `idx_users_org`, DROP FOREIGN KEY `fk_users_org_id`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `DROP INDEX "idx_users_org"; ALTER TABLE "users" DROP CONSTRAINT "fk_users_org_id"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropForeignKeyIfExistsAndDropIndexIfExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP FOREIGN KEY IF EXISTS `fk_users_org_id`, DROP INDEX IF EXISTS `idx_users_org`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "fk_users_org_id"; DROP INDEX IF EXISTS "idx_users_org"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropPrimaryKeyIfExistsAndDropIndexIfExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP PRIMARY KEY IF EXISTS, DROP INDEX IF EXISTS `idx_users_org`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "users_pkey"; DROP INDEX IF EXISTS "idx_users_org"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropForeignKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP FOREIGN KEY `fk_users_org_id`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP CONSTRAINT "fk_users_org_id"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableDropForeignKeyIfExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` DROP FOREIGN KEY IF EXISTS `fk_users_org_id`",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" DROP CONSTRAINT IF EXISTS "fk_users_org_id"`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddForeignKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD CONSTRAINT `fk_users_org_id` FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD CONSTRAINT "fk_users_org_id" FOREIGN KEY ("org_id") REFERENCES "orgs" ("id") ON DELETE CASCADE ON UPDATE RESTRICT`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddForeignKeyIfNotExistsToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD CONSTRAINT `fk_users_org_id` FOREIGN KEY IF NOT EXISTS (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD CONSTRAINT "fk_users_org_id" FOREIGN KEY ("org_id") REFERENCES "orgs" ("id") ON DELETE CASCADE`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}

func TestTranslatorTranslatesMySQLAlterTableAddColumnAndAddForeignKeyToPostgres(t *testing.T) {
	t.Parallel()

	translator, err := uniquedialect.NewTranslator(uniquedialect.TranslatorOptions{
		InputDialect:  uniquedialect.DialectMySQL,
		TargetDialect: uniquedialect.DialectPostgres,
	})
	if err != nil {
		t.Fatalf("NewTranslator() error = %v", err)
	}

	result, err := translator.Translate(
		context.Background(),
		"ALTER TABLE `users` ADD COLUMN `org_id` bigint, ADD CONSTRAINT `fk_users_org_id` FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`)",
	)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	want := `ALTER TABLE "users" ADD COLUMN "org_id" bigint, ADD CONSTRAINT "fk_users_org_id" FOREIGN KEY ("org_id") REFERENCES "orgs" ("id")`
	if result.SQL != want {
		t.Fatalf("Translate() SQL = %q, want %q", result.SQL, want)
	}
}
