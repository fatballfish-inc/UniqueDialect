package render

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatballfish/uniquedialect/internal/ir"
	"github.com/fatballfish/uniquedialect/internal/syntax"
)

var (
	backtickIdentifierPattern  = regexp.MustCompile("`([^`]*)`")
	doubleIdentifierPattern    = regexp.MustCompile(`"([^"]*)"`)
	postgresPlaceholderPattern = regexp.MustCompile(`\$[0-9]+`)
	ilikePattern               = regexp.MustCompile(`(?i)\s+ILIKE\s+`)
	mysqlOffsetLimitPattern    = regexp.MustCompile(`^\s*(\d+)\s*,\s*(\d+)\s*$`)
	mysqlIntegerWidthPattern   = regexp.MustCompile(`(?i)\b(tinyint|smallint|mediumint|int|integer|bigint)\(\d+\)`)
	mysqlCharsetLiteralPattern = regexp.MustCompile(`(?i)_[a-z0-9]+\s*'`)
)

type state struct {
	placeholder int
}

// Statement renders normalized IR into the target dialect.
func Statement(stmt ir.Statement, from, to string) (string, error) {
	renderState := &state{}

	if batch, ok := stmt.(ir.BatchStatement); ok {
		parts := make([]string, 0, len(batch.Statements))
		for _, substmt := range batch.Statements {
			rendered, err := renderStatement(substmt, from, to, renderState)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(rendered) == "" {
				continue
			}
			parts = append(parts, rendered)
		}
		return strings.Join(parts, "; "), nil
	}

	return renderStatement(stmt, from, to, renderState)
}

func renderStatement(stmt ir.Statement, from, to string, renderState *state) (string, error) {
	switch value := stmt.(type) {
	case ir.RawStatement:
		return rewriteRaw(value.SQL, from, to, renderState), nil
	case ir.SelectStatement:
		return renderSelect(value, from, to, renderState), nil
	case ir.InsertStatement:
		return renderInsert(value, from, to, renderState), nil
	case ir.UpdateStatement:
		return renderUpdate(value, from, to, renderState), nil
	case ir.DeleteStatement:
		return renderDelete(value, from, to, renderState), nil
	case ir.SetStatement:
		return renderSet(value, to)
	case ir.SetTransactionStatement:
		return renderSetTransaction(value, to)
	case ir.SavepointStatement:
		return renderSavepoint(value), nil
	case ir.ReleaseSavepointStatement:
		return renderReleaseSavepoint(value), nil
	case ir.RollbackToSavepointStatement:
		return renderRollbackToSavepoint(value), nil
	case ir.UseStatement:
		return renderUse(value, to), nil
	case ir.ShowTablesStatement:
		return renderShowTables(value, to)
	case ir.ShowColumnsStatement:
		return renderShowColumns(value, to)
	case ir.ShowIndexStatement:
		return renderShowIndex(value, to)
	case ir.ShowTableStatusStatement:
		return renderShowTableStatus(value, to)
	case ir.ShowDatabasesStatement:
		return renderShowDatabases(value, to)
	case ir.ShowCreateDatabaseStatement:
		return renderShowCreateDatabase(value, to)
	case ir.ShowCreateTableStatement:
		return renderShowCreateTable(value, to)
	case ir.ShowCreateViewStatement:
		return renderShowCreateView(value, to)
	case ir.ShowVariablesStatement:
		return renderShowVariables(value, to)
	case ir.CreateTableStatement:
		return renderCreateTable(value, from, to, renderState), nil
	case ir.AlterTableStatement:
		return renderAlterTable(value, from, to, renderState), nil
	case ir.DropIndexStatement:
		return renderDropIndex(value, to), nil
	case ir.CreateIndexStatement:
		return renderCreateIndex(value, from, to, renderState), nil
	case ir.RenameTableStatement:
		return renderRenameTable(value, to), nil
	default:
		return "", fmt.Errorf("unsupported statement type %T", stmt)
	}
}

func renderSelect(stmt ir.SelectStatement, from, to string, st *state) string {
	columns := make([]string, 0, len(stmt.Columns))
	for _, column := range stmt.Columns {
		columns = append(columns, rewriteRaw(column, from, to, st))
	}

	builder := strings.Builder{}
	builder.WriteString("SELECT ")
	builder.WriteString(strings.Join(columns, ", "))
	builder.WriteString(" FROM ")
	builder.WriteString(rewriteRaw(stmt.From, from, to, st))

	for _, join := range stmt.Joins {
		builder.WriteString(" ")
		builder.WriteString(strings.TrimSpace(join.Kind))
		builder.WriteString(" ")
		builder.WriteString(rewriteRaw(join.Table, from, to, st))
		if join.On != "" {
			builder.WriteString(" ON ")
			builder.WriteString(rewriteRaw(join.On, from, to, st))
		}
	}

	if stmt.Where != "" {
		builder.WriteString(" WHERE ")
		builder.WriteString(rewriteRaw(stmt.Where, from, to, st))
	}
	if len(stmt.OrderBy) > 0 {
		parts := make([]string, 0, len(stmt.OrderBy))
		for _, part := range stmt.OrderBy {
			parts = append(parts, rewriteRaw(part, from, to, st))
		}
		builder.WriteString(" ORDER BY ")
		builder.WriteString(strings.Join(parts, ", "))
	}
	if stmt.Limit != nil {
		builder.WriteString(renderLimit(stmt.Limit, to))
	}
	return builder.String()
}

func renderInsert(stmt ir.InsertStatement, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("INSERT INTO ")
	builder.WriteString(quoteIdentifierChain(stmt.Table, to))
	builder.WriteString(" (")
	columns := make([]string, 0, len(stmt.Columns))
	for _, column := range stmt.Columns {
		columns = append(columns, quoteIdentifierChain(column, to))
	}
	builder.WriteString(strings.Join(columns, ", "))
	builder.WriteString(") VALUES (")
	values := make([]string, 0, len(stmt.Values))
	for _, value := range stmt.Values {
		values = append(values, rewriteRaw(value, from, to, st))
	}
	builder.WriteString(strings.Join(values, ", "))
	builder.WriteString(")")

	if stmt.Conflict != nil {
		builder.WriteString(renderConflict(stmt.Conflict, to))
	}
	return builder.String()
}

func renderUpdate(stmt ir.UpdateStatement, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("UPDATE ")
	builder.WriteString(quoteIdentifierChain(stmt.Table, to))
	builder.WriteString(" SET ")
	parts := make([]string, 0, len(stmt.Assignments))
	for _, assignment := range stmt.Assignments {
		parts = append(parts, quoteIdentifierChain(assignment.Column, to)+" = "+rewriteRaw(assignment.Value, from, to, st))
	}
	builder.WriteString(strings.Join(parts, ", "))
	if stmt.Where != "" {
		builder.WriteString(" WHERE ")
		builder.WriteString(rewriteRaw(stmt.Where, from, to, st))
	}
	return builder.String()
}

func renderDelete(stmt ir.DeleteStatement, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("DELETE FROM ")
	builder.WriteString(quoteIdentifierChain(stmt.Table, to))
	if stmt.Where != "" {
		builder.WriteString(" WHERE ")
		builder.WriteString(rewriteRaw(stmt.Where, from, to, st))
	}
	return builder.String()
}

func renderSet(stmt ir.SetStatement, to string) (string, error) {
	switch to {
	case "postgres":
		return "SET client_encoding TO " + quoteStringLiteral(renderPostgresClientEncoding(stmt.Charset)), nil
	case "mysql":
		if stmt.Kind == "charset" {
			return "SET CHARACTER SET " + stmt.Charset, nil
		}
		return "SET NAMES " + stmt.Charset, nil
	default:
		return "", fmt.Errorf("unsupported SET target dialect %s", to)
	}
}

func renderSetTransaction(stmt ir.SetTransactionStatement, to string) (string, error) {
	switch to {
	case "postgres":
		if stmt.Scope == "transaction" {
			if stmt.AccessMode != "" {
				return "SET TRANSACTION " + stmt.AccessMode, nil
			}
			return "SET TRANSACTION ISOLATION LEVEL " + stmt.IsolationLevel, nil
		}
		if stmt.AccessMode != "" {
			return "SET SESSION CHARACTERISTICS AS TRANSACTION " + stmt.AccessMode, nil
		}
		return "SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL " + stmt.IsolationLevel, nil
	case "mysql":
		if stmt.Scope == "transaction" {
			if stmt.AccessMode != "" {
				return "SET TRANSACTION " + stmt.AccessMode, nil
			}
			return "SET TRANSACTION ISOLATION LEVEL " + stmt.IsolationLevel, nil
		}
		if stmt.AccessMode != "" {
			return "SET SESSION TRANSACTION " + stmt.AccessMode, nil
		}
		return "SET SESSION TRANSACTION ISOLATION LEVEL " + stmt.IsolationLevel, nil
	default:
		return "", fmt.Errorf("unsupported SET target dialect %s", to)
	}
}

func renderSavepoint(stmt ir.SavepointStatement) string {
	return "SAVEPOINT " + strings.TrimSpace(stmt.Name)
}

func renderReleaseSavepoint(stmt ir.ReleaseSavepointStatement) string {
	return "RELEASE SAVEPOINT " + strings.TrimSpace(stmt.Name)
}

func renderRollbackToSavepoint(stmt ir.RollbackToSavepointStatement) string {
	return "ROLLBACK TO SAVEPOINT " + strings.TrimSpace(stmt.Name)
}

func renderPostgresClientEncoding(charset string) string {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "utf8", "utf8mb4":
		return "UTF8"
	default:
		return charset
	}
}

func renderUse(stmt ir.UseStatement, to string) string {
	switch to {
	case "postgres":
		return "SET search_path TO " + quoteIdentifierChain(stmt.Database, to)
	default:
		return "USE " + quoteIdentifierChain(stmt.Database, to)
	}
}

func renderShowTables(stmt ir.ShowTablesStatement, to string) (string, error) {
	switch to {
	case "postgres":
		schemaExpr := "current_schema()"
		alias := "Tables_in_current_schema"
		if strings.TrimSpace(stmt.Database) != "" {
			schemaExpr = quoteStringLiteral(strings.TrimSpace(stmt.Database))
			alias = "Tables_in_" + strings.TrimSpace(stmt.Database)
		}
		selectClause := "SELECT table_name AS " + quoteIdentifierChain(alias, to)
		if stmt.Full {
			selectClause += ", CASE WHEN table_type = 'VIEW' THEN 'VIEW' ELSE 'BASE TABLE' END AS " + quoteIdentifierChain("Table_type", to)
		}
		return selectClause +
			" FROM information_schema.tables WHERE table_schema = " + schemaExpr +
			" AND table_type IN ('BASE TABLE', 'VIEW')" + renderShowTablesPattern(stmt.Pattern) +
			" ORDER BY table_name", nil
	case "mysql":
		sql := "SHOW "
		if stmt.Full {
			sql += "FULL "
		}
		sql += "TABLES"
		if strings.TrimSpace(stmt.Database) != "" {
			sql += " IN " + quoteIdentifierChain(stmt.Database, to)
		}
		if stmt.Pattern != "" {
			sql += " LIKE " + quoteStringLiteral(stmt.Pattern)
		}
		return sql, nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowColumns(stmt ir.ShowColumnsStatement, to string) (string, error) {
	switch to {
	case "postgres":
		schema := strings.TrimSpace(stmt.Database)
		table := strings.TrimSpace(stmt.Table)
		schemaPredicate := "pg_table_is_visible(t.oid)"
		if schema != "" {
			schemaPredicate = "n.nspname = " + quoteStringLiteral(schema)
		}
		selectColumns := "a.attname AS " + quoteIdentifierChain("Field", to) +
			", format_type(a.atttypid, a.atttypmod) AS " + quoteIdentifierChain("Type", to)
		if stmt.Full {
			selectColumns += ", coll.collname AS " + quoteIdentifierChain("Collation", to)
		}
		selectColumns += ", CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END AS " + quoteIdentifierChain("Null", to) +
			", CASE WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisprimary AND a.attnum = ANY(ix.indkey)) THEN 'PRI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND ix.indisunique AND ix.indnkeyatts = 1 AND a.attnum = ANY(ix.indkey)) THEN 'UNI' WHEN EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid = t.oid AND a.attnum = ANY(ix.indkey)) THEN 'MUL' ELSE '' END AS " + quoteIdentifierChain("Key", to) +
			", pg_get_expr(ad.adbin, ad.adrelid) AS " + quoteIdentifierChain("Default", to) +
			", CASE WHEN a.attidentity IN ('a', 'd') THEN 'auto_increment' WHEN a.attgenerated = 's' THEN 'STORED GENERATED' ELSE '' END AS " + quoteIdentifierChain("Extra", to)
		if stmt.Full {
			selectColumns += ", '' AS " + quoteIdentifierChain("Privileges", to) +
				", COALESCE(col_description(t.oid, a.attnum), '') AS " + quoteIdentifierChain("Comment", to)
		}
		sql := "SELECT " + selectColumns +
			" FROM pg_attribute a JOIN pg_class t ON t.oid = a.attrelid JOIN pg_namespace n ON n.oid = t.relnamespace LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum"
		if stmt.Full {
			sql += " LEFT JOIN pg_collation coll ON coll.oid = a.attcollation AND a.attcollation <> 0"
		}
		sql += " WHERE t.relkind IN ('r', 'p', 'v', 'm', 'f') AND t.relname = " +
			quoteStringLiteral(strings.TrimSpace(table)) + " AND " + schemaPredicate
		if stmt.Pattern != "" {
			sql += " AND a.attname ILIKE " + quoteStringLiteral(stmt.Pattern)
		}
		sql +=
			" AND a.attnum > 0 AND NOT a.attisdropped ORDER BY a.attnum"
		return sql, nil
	case "mysql":
		sql := "SHOW "
		if stmt.Full {
			sql += "FULL "
		}
		sql += "COLUMNS FROM " + quoteIdentifierChain(stmt.Table, to)
		if strings.TrimSpace(stmt.Database) != "" {
			sql += " IN " + quoteIdentifierChain(stmt.Database, to)
		}
		if stmt.Pattern != "" {
			sql += " LIKE " + quoteStringLiteral(stmt.Pattern)
		}
		return sql, nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowIndex(stmt ir.ShowIndexStatement, to string) (string, error) {
	switch to {
	case "postgres":
		schema := strings.TrimSpace(stmt.Database)
		visibilityPredicate := "pg_table_is_visible(t.oid)"
		table := strings.TrimSpace(stmt.Table)
		if schema != "" {
			visibilityPredicate = "n.nspname = " + quoteStringLiteral(schema)
		}
		return "SELECT t.relname AS " + quoteIdentifierChain("Table", to) +
			", CASE WHEN ix.indisunique THEN 0 ELSE 1 END AS " + quoteIdentifierChain("Non_unique", to) +
			", i.relname AS " + quoteIdentifierChain("Key_name", to) +
			", k.ordinality AS " + quoteIdentifierChain("Seq_in_index", to) +
			", CASE WHEN k.attnum = 0 THEN NULL ELSE a.attname END AS " + quoteIdentifierChain("Column_name", to) +
			", 'A' AS " + quoteIdentifierChain("Collation", to) +
			", NULL AS " + quoteIdentifierChain("Cardinality", to) +
			", NULL AS " + quoteIdentifierChain("Sub_part", to) +
			", NULL AS " + quoteIdentifierChain("Packed", to) +
			", CASE WHEN a.attnotnull THEN '' ELSE 'YES' END AS " + quoteIdentifierChain("Null", to) +
			", am.amname AS " + quoteIdentifierChain("Index_type", to) +
			", '' AS " + quoteIdentifierChain("Comment", to) +
			", '' AS " + quoteIdentifierChain("Index_comment", to) +
			", 'YES' AS " + quoteIdentifierChain("Visible", to) +
			", CASE WHEN k.attnum = 0 THEN pg_get_indexdef(ix.indexrelid, k.ordinality, true) ELSE NULL END AS " + quoteIdentifierChain("Expression", to) +
			" FROM pg_index ix JOIN pg_class t ON t.oid = ix.indrelid JOIN pg_class i ON i.oid = ix.indexrelid JOIN pg_am am ON am.oid = i.relam JOIN pg_namespace n ON n.oid = t.relnamespace JOIN LATERAL unnest(string_to_array(ix.indkey::text, ' ')::smallint[]) WITH ORDINALITY AS k(attnum, ordinality) ON TRUE LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum WHERE t.relkind IN ('r', 'p') AND t.relname = " + quoteStringLiteral(strings.TrimSpace(table)) +
			" AND " + visibilityPredicate +
			" AND k.ordinality <= ix.indnkeyatts" +
			" ORDER BY i.relname, k.ordinality", nil
	case "mysql":
		sql := "SHOW INDEX FROM " + quoteIdentifierChain(stmt.Table, to)
		if strings.TrimSpace(stmt.Database) != "" {
			sql += " IN " + quoteIdentifierChain(stmt.Database, to)
		}
		return sql, nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowTableStatus(stmt ir.ShowTableStatusStatement, to string) (string, error) {
	switch to {
	case "postgres":
		schemaPredicate := "pg_table_is_visible(c.oid)"
		if strings.TrimSpace(stmt.Database) != "" {
			schemaPredicate = "n.nspname = " + quoteStringLiteral(strings.TrimSpace(stmt.Database))
		}
		return "SELECT c.relname AS " + quoteIdentifierChain("Name", to) +
			", COALESCE(am.amname, 'heap') AS " + quoteIdentifierChain("Engine", to) +
			", NULL AS " + quoteIdentifierChain("Version", to) +
			", NULL AS " + quoteIdentifierChain("Row_format", to) +
			", CASE WHEN c.reltuples < 0 THEN NULL ELSE c.reltuples::bigint END AS " + quoteIdentifierChain("Rows", to) +
			", CASE WHEN c.reltuples > 0 THEN pg_relation_size(c.oid) / NULLIF(c.reltuples::bigint, 0) ELSE NULL END AS " + quoteIdentifierChain("Avg_row_length", to) +
			", pg_relation_size(c.oid) AS " + quoteIdentifierChain("Data_length", to) +
			", NULL AS " + quoteIdentifierChain("Max_data_length", to) +
			", pg_indexes_size(c.oid) AS " + quoteIdentifierChain("Index_length", to) +
			", NULL AS " + quoteIdentifierChain("Data_free", to) +
			", NULL AS " + quoteIdentifierChain("Auto_increment", to) +
			", NULL AS " + quoteIdentifierChain("Create_time", to) +
			", NULL AS " + quoteIdentifierChain("Update_time", to) +
			", NULL AS " + quoteIdentifierChain("Check_time", to) +
			", NULL AS " + quoteIdentifierChain("Collation", to) +
			", NULL AS " + quoteIdentifierChain("Checksum", to) +
			", NULL AS " + quoteIdentifierChain("Create_options", to) +
			", COALESCE(obj_description(c.oid, 'pg_class'), '') AS " + quoteIdentifierChain("Comment", to) +
			" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace LEFT JOIN pg_am am ON am.oid = c.relam WHERE c.relkind IN ('r', 'p') AND " + schemaPredicate +
			renderShowTableStatusName(stmt.Name) +
			renderShowTableStatusPattern(stmt.Pattern) +
			" ORDER BY c.relname", nil
	case "mysql":
		sql := "SHOW TABLE STATUS"
		if strings.TrimSpace(stmt.Database) != "" {
			sql += " IN " + quoteIdentifierChain(stmt.Database, to)
		}
		if stmt.Name != "" {
			sql += " WHERE Name = " + quoteStringLiteral(stmt.Name)
		}
		if stmt.Pattern != "" {
			sql += " LIKE " + quoteStringLiteral(stmt.Pattern)
		}
		return sql, nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowDatabases(stmt ir.ShowDatabasesStatement, to string) (string, error) {
	switch to {
	case "postgres":
		sql := "SELECT datname AS " + quoteIdentifierChain("Database", to) +
			" FROM pg_database WHERE datistemplate = false"
		if stmt.Pattern != "" {
			sql += " AND datname ILIKE " + quoteStringLiteral(stmt.Pattern)
		}
		sql += " ORDER BY datname"
		return sql, nil
	case "mysql":
		if stmt.Pattern == "" {
			return "SHOW DATABASES", nil
		}
		return "SHOW DATABASES LIKE " + quoteStringLiteral(stmt.Pattern), nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowTablesPattern(pattern string) string {
	if pattern == "" {
		return ""
	}
	return " AND table_name ILIKE " + quoteStringLiteral(pattern)
}

func renderShowTableStatusPattern(pattern string) string {
	if pattern == "" {
		return ""
	}
	return " AND c.relname ILIKE " + quoteStringLiteral(pattern)
}

func renderShowTableStatusName(name string) string {
	if name == "" {
		return ""
	}
	return " AND c.relname = " + quoteStringLiteral(name)
}

func renderShowCreateDatabase(stmt ir.ShowCreateDatabaseStatement, to string) (string, error) {
	switch to {
	case "postgres":
		return "SELECT datname AS " + quoteIdentifierChain("Database", to) +
			", 'CREATE DATABASE ' || quote_ident(datname) || ' ENCODING = ' || quote_literal(pg_encoding_to_char(encoding)) || ' LC_COLLATE = ' || quote_literal(datcollate) || ' LC_CTYPE = ' || quote_literal(datctype) AS " +
			quoteIdentifierChain("Create Database", to) +
			" FROM pg_database WHERE datname = " + quoteStringLiteral(strings.TrimSpace(stmt.Name)), nil
	case "mysql":
		if stmt.IfNotExists {
			return "SHOW CREATE DATABASE IF NOT EXISTS " + quoteIdentifierChain(stmt.Name, to), nil
		}
		return "SHOW CREATE DATABASE " + quoteIdentifierChain(stmt.Name, to), nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowCreateTable(stmt ir.ShowCreateTableStatement, to string) (string, error) {
	switch to {
	case "postgres":
		schemaPredicate := "pg_table_is_visible(c.oid)"
		if strings.TrimSpace(stmt.Schema) != "" {
			schemaPredicate = "n.nspname = " + quoteStringLiteral(strings.TrimSpace(stmt.Schema))
		}
		return "SELECT c.relname AS " + quoteIdentifierChain("Table", to) +
			", 'CREATE TABLE ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname) || E' (\\n' || cols.columns || COALESCE(E',\\n' || cons.constraints, '') || E'\\n)' AS " + quoteIdentifierChain("Create Table", to) +
			" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace" +
			" JOIN LATERAL (SELECT string_agg('  ' || quote_ident(a.attname) || ' ' || format_type(a.atttypid, a.atttypmod) || CASE WHEN a.attidentity = 'a' THEN ' GENERATED ALWAYS AS IDENTITY' WHEN a.attidentity = 'd' THEN ' GENERATED BY DEFAULT AS IDENTITY' WHEN a.attgenerated = 's' THEN ' GENERATED ALWAYS AS (' || pg_get_expr(ad.adbin, ad.adrelid) || ') STORED' ELSE '' END || CASE WHEN a.attgenerated = '' AND ad.adbin IS NOT NULL THEN ' DEFAULT ' || pg_get_expr(ad.adbin, ad.adrelid) ELSE '' END || CASE WHEN a.attnotnull THEN ' NOT NULL' ELSE '' END, E',\\n' ORDER BY a.attnum) AS columns FROM pg_attribute a LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped) cols ON TRUE" +
			" LEFT JOIN LATERAL (SELECT string_agg('  CONSTRAINT ' || quote_ident(con.conname) || ' ' || pg_get_constraintdef(con.oid, true), E',\\n' ORDER BY con.conname) AS constraints FROM pg_constraint con WHERE con.conrelid = c.oid) cons ON TRUE" +
			" WHERE c.relkind IN ('r', 'p') AND c.relname = " + quoteStringLiteral(strings.TrimSpace(stmt.Name)) + " AND " + schemaPredicate, nil
	case "mysql":
		if strings.TrimSpace(stmt.Schema) != "" {
			return "SHOW CREATE TABLE " + quoteIdentifierPart(stmt.Schema, to) + "." + quoteIdentifierPart(stmt.Name, to), nil
		}
		return "SHOW CREATE TABLE " + quoteIdentifierPart(stmt.Name, to), nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowCreateView(stmt ir.ShowCreateViewStatement, to string) (string, error) {
	switch to {
	case "postgres":
		schemaPredicate := "current_schema()"
		if strings.TrimSpace(stmt.Schema) != "" {
			schemaPredicate = quoteStringLiteral(strings.TrimSpace(stmt.Schema))
		}
		return "SELECT c.relname AS " + quoteIdentifierChain("View", to) +
			", 'CREATE VIEW ' || quote_ident(n.nspname) || '.' || quote_ident(c.relname) || ' AS ' || pg_get_viewdef(c.oid, true) AS " +
			quoteIdentifierChain("Create View", to) +
			" FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'v' AND c.relname = " +
			quoteStringLiteral(strings.TrimSpace(stmt.Name)) + " AND n.nspname = " + schemaPredicate, nil
	case "mysql":
		if strings.TrimSpace(stmt.Schema) != "" {
			return "SHOW CREATE VIEW " + quoteIdentifierPart(stmt.Schema, to) + "." + quoteIdentifierPart(stmt.Name, to), nil
		}
		return "SHOW CREATE VIEW " + quoteIdentifierPart(stmt.Name, to), nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func renderShowVariables(stmt ir.ShowVariablesStatement, to string) (string, error) {
	switch to {
	case "postgres":
		sql := "SELECT name AS " + quoteIdentifierChain("Variable_name", to) +
			", setting AS " + quoteIdentifierChain("Value", to) +
			" FROM pg_catalog.pg_settings"
		if stmt.Name != "" {
			sql += " WHERE name = " + quoteStringLiteral(stmt.Name)
		} else if stmt.Pattern != "" {
			sql += " WHERE name ILIKE " + quoteStringLiteral(stmt.Pattern)
		}
		sql += " ORDER BY name"
		return sql, nil
	case "mysql":
		if stmt.Name != "" {
			return "SHOW VARIABLES WHERE Variable_name = " + quoteStringLiteral(stmt.Name), nil
		}
		if stmt.Pattern == "" {
			return "SHOW VARIABLES", nil
		}
		return "SHOW VARIABLES LIKE " + quoteStringLiteral(stmt.Pattern), nil
	default:
		return "", unsupportedShowTargetDialect(to)
	}
}

func unsupportedShowTargetDialect(to string) error {
	return fmt.Errorf("unsupported SHOW target dialect %s", to)
}

func renderCreateTable(stmt ir.CreateTableStatement, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("CREATE TABLE ")
	if stmt.IfNotExists {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString(quoteIdentifierChain(stmt.Name, to))
	builder.WriteString(" (")

	parts := make([]string, 0, len(stmt.Columns)+len(stmt.Constraints))
	for _, column := range stmt.Columns {
		parts = append(parts, renderCreateTableColumn(column, from, to, st))
	}
	for _, constraint := range stmt.Constraints {
		parts = append(parts, rewriteCreateTableConstraint(constraint, from, to, st))
	}

	builder.WriteString(strings.Join(parts, ", "))
	builder.WriteString(")")
	return builder.String()
}

func renderAlterTable(stmt ir.AlterTableStatement, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("ALTER TABLE ")
	builder.WriteString(quoteIdentifierChain(stmt.Name, to))

	parts := make([]string, 0, len(stmt.Specs))
	for _, spec := range stmt.Specs {
		switch spec.Kind {
		case ir.AlterTableSpecAddColumn:
			parts = append(parts, "ADD COLUMN "+renderCreateTableColumn(spec.Column, from, to, st))
		case ir.AlterTableSpecDropColumn:
			parts = append(parts, "DROP COLUMN "+quoteIdentifierChain(spec.Name, to))
		case ir.AlterTableSpecSetDefault:
			parts = append(parts, "ALTER COLUMN "+quoteIdentifierChain(spec.Name, to)+" SET DEFAULT "+rewriteRaw(spec.Default, from, to, st))
		case ir.AlterTableSpecDropDefault:
			parts = append(parts, "ALTER COLUMN "+quoteIdentifierChain(spec.Name, to)+" DROP DEFAULT")
		case ir.AlterTableSpecAlterType:
			parts = append(parts, "ALTER COLUMN "+quoteIdentifierChain(spec.Name, to)+" TYPE "+rewriteCreateTableType(spec.Column.Type, spec.Column.AutoIncrement, to))
		case ir.AlterTableSpecSetNotNull:
			parts = append(parts, "ALTER COLUMN "+quoteIdentifierChain(spec.Name, to)+" SET NOT NULL")
		case ir.AlterTableSpecDropNotNull:
			parts = append(parts, "ALTER COLUMN "+quoteIdentifierChain(spec.Name, to)+" DROP NOT NULL")
		case ir.AlterTableSpecRenameColumn:
			parts = append(parts, "RENAME COLUMN "+quoteIdentifierChain(spec.Name, to)+" TO "+quoteIdentifierChain(spec.NewName, to))
		case ir.AlterTableSpecAddPrimaryKey:
			parts = append(parts, renderAlterTableAddConstraint("PRIMARY KEY", spec, from, to, st))
		case ir.AlterTableSpecAddUniqueKey:
			parts = append(parts, renderAlterTableAddConstraint("UNIQUE", spec, from, to, st))
		case ir.AlterTableSpecAddForeignKey:
			parts = append(parts, renderAlterTableAddForeignKey(spec, from, to, st))
		case ir.AlterTableSpecDropPrimaryKey:
			parts = append(parts, renderAlterTableDropPrimaryKey(stmt.Name, spec.IfExists, to))
		case ir.AlterTableSpecDropForeignKey:
			parts = append(parts, renderAlterTableDropForeignKey(spec.Name, spec.IfExists, to))
		}
	}

	if len(parts) > 0 {
		builder.WriteString(" ")
		builder.WriteString(strings.Join(parts, ", "))
	}
	return builder.String()
}

func renderAlterTableAddConstraint(kind string, spec ir.AlterTableSpec, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("ADD ")
	if to == "mysql" && kind == "UNIQUE" {
		builder.WriteString("UNIQUE KEY ")
		if spec.IfNotExists {
			builder.WriteString("IF NOT EXISTS ")
		}
		if spec.ConstraintName != "" {
			builder.WriteString(quoteIdentifierChain(spec.ConstraintName, to))
			builder.WriteString(" ")
		}
	} else {
		if spec.ConstraintName != "" {
			builder.WriteString("CONSTRAINT ")
			builder.WriteString(quoteIdentifierChain(spec.ConstraintName, to))
			builder.WriteString(" ")
		}
		builder.WriteString(kind)
		builder.WriteString(" (")
		parts := make([]string, 0, len(spec.Parts))
		for _, part := range spec.Parts {
			rendered := renderIndexPart(part, from, to, st)
			if rendered == "" {
				continue
			}
			parts = append(parts, rendered)
		}
		builder.WriteString(strings.Join(parts, ", "))
		builder.WriteString(")")
		return builder.String()
	}

	builder.WriteString(" (")
	parts := make([]string, 0, len(spec.Parts))
	for _, part := range spec.Parts {
		rendered := renderIndexPart(part, from, to, st)
		if rendered == "" {
			continue
		}
		parts = append(parts, rendered)
	}
	builder.WriteString(strings.Join(parts, ", "))
	builder.WriteString(")")
	return builder.String()
}

func renderAlterTableAddForeignKey(spec ir.AlterTableSpec, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("ADD ")
	if spec.ConstraintName != "" {
		builder.WriteString("CONSTRAINT ")
		builder.WriteString(quoteIdentifierChain(spec.ConstraintName, to))
		builder.WriteString(" ")
	}
	builder.WriteString("FOREIGN KEY ")
	if spec.IfNotExists && to == "mysql" {
		builder.WriteString("IF NOT EXISTS ")
	}
	builder.WriteString("(")

	parts := make([]string, 0, len(spec.Parts))
	for _, part := range spec.Parts {
		rendered := renderIndexPart(part, from, to, st)
		if rendered == "" {
			continue
		}
		parts = append(parts, rendered)
	}
	builder.WriteString(strings.Join(parts, ", "))
	builder.WriteString(")")
	builder.WriteString(" REFERENCES ")
	builder.WriteString(quoteIdentifierChain(spec.Reference.Table, to))
	builder.WriteString(" (")

	refParts := make([]string, 0, len(spec.Reference.Parts))
	for _, part := range spec.Reference.Parts {
		rendered := renderIndexPart(part, from, to, st)
		if rendered == "" {
			continue
		}
		refParts = append(refParts, rendered)
	}
	builder.WriteString(strings.Join(refParts, ", "))
	builder.WriteString(")")

	if spec.Reference.Match != "" {
		builder.WriteString(" MATCH ")
		builder.WriteString(spec.Reference.Match)
	}
	if spec.Reference.OnDelete != "" {
		builder.WriteString(" ON DELETE ")
		builder.WriteString(spec.Reference.OnDelete)
	}
	if spec.Reference.OnUpdate != "" {
		builder.WriteString(" ON UPDATE ")
		builder.WriteString(spec.Reference.OnUpdate)
	}
	return builder.String()
}

func renderAlterTableDropPrimaryKey(tableName string, ifExists bool, to string) string {
	switch to {
	case "postgres", "oracle":
		prefix := "DROP CONSTRAINT "
		if ifExists {
			prefix = "DROP CONSTRAINT IF EXISTS "
		}
		return prefix + quoteIdentifierChain(tableName+"_pkey", to)
	default:
		return "DROP PRIMARY KEY"
	}
}

func renderAlterTableDropForeignKey(name string, ifExists bool, to string) string {
	switch to {
	case "postgres", "oracle":
		prefix := "DROP CONSTRAINT "
		if ifExists {
			prefix = "DROP CONSTRAINT IF EXISTS "
		}
		return prefix + quoteIdentifierChain(name, to)
	default:
		prefix := "DROP FOREIGN KEY "
		if ifExists {
			prefix = "DROP FOREIGN KEY IF EXISTS "
		}
		return prefix + quoteIdentifierChain(name, to)
	}
}

func renderDropIndex(stmt ir.DropIndexStatement, to string) string {
	builder := strings.Builder{}
	builder.WriteString("DROP INDEX ")
	if stmt.IfExists {
		builder.WriteString("IF EXISTS ")
	}
	builder.WriteString(quoteIdentifierChain(stmt.Name, to))

	switch to {
	case "mysql", "sqlite":
		if stmt.Table != "" {
			builder.WriteString(" ON ")
			builder.WriteString(quoteIdentifierChain(stmt.Table, to))
		}
	}

	return builder.String()
}

func renderRenameTable(stmt ir.RenameTableStatement, to string) string {
	_, newBaseName := splitIdentifierSchemaAndName(stmt.NewName)
	if newBaseName == "" {
		newBaseName = strings.TrimSpace(stmt.NewName)
	}

	switch to {
	case "postgres", "sqlite":
		return "ALTER TABLE " + quoteIdentifierChain(stmt.OldName, to) + " RENAME TO " + quoteIdentifierChain(newBaseName, to)
	case "oracle":
		return "RENAME " + quoteIdentifierChain(stmt.OldName, to) + " TO " + quoteIdentifierChain(newBaseName, to)
	default:
		return "RENAME TABLE " + quoteIdentifierChain(stmt.OldName, to) + " TO " + quoteIdentifierChain(stmt.NewName, to)
	}
}

func renderCreateIndex(stmt ir.CreateIndexStatement, from, to string, st *state) string {
	builder := strings.Builder{}
	builder.WriteString("CREATE ")
	if stmt.Unique {
		builder.WriteString("UNIQUE ")
	}
	builder.WriteString("INDEX ")
	if stmt.IfNotExists {
		switch to {
		case "mysql", "postgres", "sqlite":
			builder.WriteString("IF NOT EXISTS ")
		}
	}
	builder.WriteString(quoteIdentifierChain(stmt.Name, to))
	builder.WriteString(" ON ")
	builder.WriteString(quoteIdentifierChain(stmt.Table, to))
	if stmt.Using != "" {
		builder.WriteString(" USING ")
		builder.WriteString(stmt.Using)
	}
	builder.WriteString(" (")

	parts := make([]string, 0, len(stmt.Parts))
	for _, part := range stmt.Parts {
		rendered := renderIndexPart(part, from, to, st)
		if rendered == "" {
			continue
		}
		parts = append(parts, rendered)
	}
	builder.WriteString(strings.Join(parts, ", "))
	builder.WriteString(")")

	switch to {
	case "mysql":
		if stmt.ParserName != "" {
			builder.WriteString(" WITH PARSER ")
			builder.WriteString(quoteIdentifierChain(stmt.ParserName, to))
		}
		if stmt.Comment != "" {
			builder.WriteString(" COMMENT ")
			builder.WriteString(quoteStringLiteral(stmt.Comment))
		}
		if stmt.Visibility != "" {
			builder.WriteString(" ")
			builder.WriteString(stmt.Visibility)
		}
		if stmt.Where != "" {
			builder.WriteString(" WHERE ")
			builder.WriteString(rewriteRaw(stmt.Where, from, to, st))
		}
		if stmt.Algorithm != "" {
			builder.WriteString(" ALGORITHM = ")
			builder.WriteString(stmt.Algorithm)
		}
		if stmt.Lock != "" {
			builder.WriteString(" LOCK = ")
			builder.WriteString(stmt.Lock)
		}
	case "postgres", "oracle":
		if stmt.Where != "" {
			builder.WriteString(" WHERE ")
			builder.WriteString(rewriteRaw(stmt.Where, from, to, st))
		}
		if stmt.Comment != "" {
			return builder.String() + "; COMMENT ON INDEX " + quoteIdentifierChain(stmt.Name, to) + " IS " + quoteStringLiteral(stmt.Comment)
		}
	default:
		if stmt.Where != "" {
			builder.WriteString(" WHERE ")
			builder.WriteString(rewriteRaw(stmt.Where, from, to, st))
		}
	}

	return builder.String()
}

func renderIndexPart(part ir.IndexPart, from, to string, st *state) string {
	var value string
	switch {
	case part.Column != "":
		value = quoteIdentifierChain(part.Column, to)
	case part.Expr != "":
		value = rewriteRaw(part.Expr, from, to, st)
	default:
		return ""
	}

	if part.Length > 0 {
		switch to {
		case "mysql", "sqlite":
			value += "(" + strconv.Itoa(part.Length) + ")"
		}
	}
	if part.Desc {
		value += " DESC"
	}
	return value
}

func renderCreateTableColumn(column ir.CreateTableColumn, from, to string, st *state) string {
	parts := []string{
		quoteIdentifierChain(column.Name, to),
		rewriteCreateTableType(column.Type, column.AutoIncrement, to),
	}

	if column.AutoIncrement {
		if to == "postgres" {
			parts = append(parts, "GENERATED BY DEFAULT AS IDENTITY")
		}
	}

	if column.NotNull {
		parts = append(parts, "NOT NULL")
	}
	if column.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if column.Unique {
		parts = append(parts, "UNIQUE")
	}
	if column.Default != "" {
		parts = append(parts, "DEFAULT "+rewriteRaw(column.Default, from, to, st))
	}

	return strings.Join(parts, " ")
}

func rewriteCreateTableConstraint(input string, from, to string, st *state) string {
	rewritten := rewriteRaw(input, from, to, st)
	if to != "postgres" && to != "oracle" {
		return rewritten
	}

	upper := strings.ToUpper(rewritten)
	switch {
	case strings.HasPrefix(upper, "UNIQUE "):
		openParen := strings.Index(rewritten, "(")
		if openParen <= len("UNIQUE ") {
			return rewritten
		}
		constraintName := strings.TrimSpace(rewritten[len("UNIQUE "):openParen])
		if constraintName == "" {
			return rewritten
		}
		rewritten = "CONSTRAINT " + constraintName + " UNIQUE " + rewritten[openParen:]
	case strings.HasPrefix(upper, "CONSTRAINT FOREIGN KEY"):
		rewritten = "FOREIGN KEY" + rewritten[len("CONSTRAINT FOREIGN KEY"):]
	}

	rewritten = strings.ReplaceAll(rewritten, "FOREIGN KEY(", "FOREIGN KEY (")
	rewritten = strings.ReplaceAll(rewritten, ")REFERENCES", ") REFERENCES")
	rewritten = strings.ReplaceAll(rewritten, "CHECK(", "CHECK (")
	rewritten = strings.ReplaceAll(rewritten, "\"(", "\" (")
	rewritten = strings.ReplaceAll(rewritten, "\">", "\" > ")
	rewritten = strings.ReplaceAll(rewritten, ">0", "> 0")
	rewritten = strings.ReplaceAll(rewritten, " NOT ENFORCED", "")
	rewritten = strings.ReplaceAll(rewritten, " ENFORCED", "")

	return strings.Join(strings.Fields(strings.TrimSpace(rewritten)), " ")
}

func quoteStringLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func rewriteCreateTableType(typeName string, autoIncrement bool, to string) string {
	normalized := strings.TrimSpace(typeName)
	if normalized == "" {
		return normalized
	}

	if to == "postgres" {
		normalized = mysqlIntegerWidthPattern.ReplaceAllStringFunc(normalized, func(match string) string {
			openIdx := strings.IndexByte(match, '(')
			if openIdx < 0 {
				return match
			}
			return match[:openIdx]
		})
		if strings.EqualFold(normalized, "datetime") {
			return "timestamp"
		}
		if autoIncrement && strings.EqualFold(normalized, "integer") {
			return "integer"
		}
	}

	return normalized
}

func renderConflict(conflict *ir.ConflictClause, to string) string {
	if conflict == nil {
		return ""
	}
	switch to {
	case "mysql":
		parts := make([]string, 0, len(conflict.Assignments))
		for _, assignment := range conflict.Assignments {
			parts = append(parts, quoteIdentifierChain(assignment.Column, to)+" = VALUES("+quoteIdentifierChain(assignment.Column, to)+")")
		}
		return " ON DUPLICATE KEY UPDATE " + strings.Join(parts, ", ")
	case "postgres":
		fallthrough
	case "sqlite":
		targets := make([]string, 0, len(conflict.TargetColumns))
		for _, target := range conflict.TargetColumns {
			targets = append(targets, quoteIdentifierChain(target, to))
		}
		parts := make([]string, 0, len(conflict.Assignments))
		for _, assignment := range conflict.Assignments {
			excluded := "EXCLUDED."
			if to == "sqlite" {
				excluded = "excluded."
			}
			parts = append(parts, quoteIdentifierChain(assignment.Column, to)+" = "+excluded+quoteIdentifierChain(assignment.Column, to))
		}
		return " ON CONFLICT (" + strings.Join(targets, ", ") + ") DO UPDATE SET " + strings.Join(parts, ", ")
	default:
		return ""
	}
}

func renderLimit(limit *ir.LimitClause, to string) string {
	if limit == nil {
		return ""
	}
	switch to {
	case "mysql", "sqlite":
		if limit.Offset != nil {
			return " LIMIT " + strconv.Itoa(*limit.Offset) + ", " + strconv.Itoa(limit.Count)
		}
		return " LIMIT " + strconv.Itoa(limit.Count)
	case "postgres":
		if limit.Offset != nil {
			return " LIMIT " + strconv.Itoa(limit.Count) + " OFFSET " + strconv.Itoa(*limit.Offset)
		}
		return " LIMIT " + strconv.Itoa(limit.Count)
	case "oracle":
		if limit.Offset != nil {
			return " OFFSET " + strconv.Itoa(*limit.Offset) + " ROWS FETCH NEXT " + strconv.Itoa(limit.Count) + " ROWS ONLY"
		}
		return " FETCH FIRST " + strconv.Itoa(limit.Count) + " ROWS ONLY"
	default:
		return ""
	}
}

func rewriteRaw(input string, from, to string, st *state) string {
	output := strings.TrimSpace(input)
	if from == to {
		return output
	}

	if from == "mysql" {
		output = backtickIdentifierPattern.ReplaceAllStringFunc(output, func(match string) string {
			identifier := strings.Trim(match, "`")
			return quoteIdentifierChain(identifier, to)
		})
		output = mysqlCharsetLiteralPattern.ReplaceAllString(output, `'`)
		output = strings.ReplaceAll(output, "NOW()", "CURRENT_TIMESTAMP")
		output = strings.ReplaceAll(output, "CURRENT_TIMESTAMP()", "CURRENT_TIMESTAMP")
		output = strings.ReplaceAll(output, "PRIMARY KEY(", "PRIMARY KEY (")
		output = rewriteMySQLOffsetLimit(output, to)
		output = replaceQuestionPlaceholders(output, to, st)
		return output
	}

	if from == "postgres" {
		output = doubleIdentifierPattern.ReplaceAllStringFunc(output, func(match string) string {
			identifier := strings.Trim(match, `"`)
			return quoteIdentifierChain(identifier, to)
		})
		output = ilikePattern.ReplaceAllString(output, " LIKE ")
		if to == "mysql" || to == "sqlite" {
			output = postgresPlaceholderPattern.ReplaceAllString(output, "?")
		}
		return output
	}

	return output
}

func rewriteMySQLOffsetLimit(input, to string) string {
	switch to {
	case "postgres", "oracle":
	default:
		return input
	}

	limitIdx := syntax.FindTopLevelKeyword(input, " LIMIT ")
	if limitIdx < 0 {
		return input
	}

	limitClause := strings.TrimSpace(input[limitIdx+len(" LIMIT "):])
	matches := mysqlOffsetLimitPattern.FindStringSubmatch(limitClause)
	if len(matches) != 3 {
		return input
	}

	offset, count := matches[1], matches[2]
	prefix := strings.TrimSpace(input[:limitIdx])

	switch to {
	case "postgres":
		return prefix + " LIMIT " + count + " OFFSET " + offset
	case "oracle":
		return prefix + " OFFSET " + offset + " ROWS FETCH NEXT " + count + " ROWS ONLY"
	default:
		return input
	}
}

func replaceQuestionPlaceholders(input string, to string, st *state) string {
	if to != "postgres" && to != "oracle" {
		return input
	}
	builder := strings.Builder{}
	for _, r := range input {
		if r != '?' {
			builder.WriteRune(r)
			continue
		}
		st.placeholder++
		if to == "postgres" {
			builder.WriteString("$")
		} else {
			builder.WriteString(":")
		}
		builder.WriteString(strconv.Itoa(st.placeholder))
	}
	return builder.String()
}

func quoteIdentifierChain(identifier string, dialect string) string {
	parts := strings.Split(strings.TrimSpace(identifier), ".")
	for index := range parts {
		parts[index] = quoteIdentifierPart(parts[index], dialect)
	}
	return strings.Join(parts, ".")
}

func quoteIdentifierPart(identifier string, dialect string) string {
	part := strings.Trim(identifier, "`\" ")
	switch dialect {
	case "postgres", "oracle":
		return `"` + part + `"`
	default:
		return "`" + part + "`"
	}
}

func splitIdentifierSchemaAndName(identifier string) (schema, name string) {
	value := strings.TrimSpace(identifier)
	if value == "" {
		return "", ""
	}
	parts := strings.Split(value, ".")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}
