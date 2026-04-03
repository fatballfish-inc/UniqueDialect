package parser

import (
	"regexp"
	"strings"

	"github.com/fatballfish/uniquedialect/internal/parser/adapter"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
)

var mariaDBDropForeignKeyIfExistsPattern = regexp.MustCompile(`(?is)^\s*ALTER\s+TABLE\s+(` + "`" + `[^` + "`" + `]+` + "`" + `|\"[^\"]+\"|[A-Za-z0-9_\.]+)\s+DROP\s+FOREIGN\s+KEY\s+IF\s+EXISTS\s+(` + "`" + `[^` + "`" + `]+` + "`" + `|\"[^\"]+\"|[A-Za-z0-9_]+)\s*;?\s*$`)
var mariaDBDropForeignKeyIfExistsInlinePattern = regexp.MustCompile(`(?i)\bDROP\s+FOREIGN\s+KEY\s+IF\s+EXISTS\b`)
var mariaDBDropPrimaryKeyIfExistsInlinePattern = regexp.MustCompile(`(?i)\bDROP\s+PRIMARY\s+KEY\s+IF\s+EXISTS\b`)
var mariaDBDropIndexIfExistsInlinePattern = regexp.MustCompile(`(?i)\bDROP\s+(?:INDEX|KEY)\s+IF\s+EXISTS\b`)
var mariaDBAddUniqueKeyIfNotExistsInlinePattern = regexp.MustCompile(`(?i)\bADD\s+UNIQUE\s+(?:KEY|INDEX)\s+IF\s+NOT\s+EXISTS\b`)
var showDatabasesWhereDatabasePattern = regexp.MustCompile(`(?is)^\s*SHOW\s+DATABASES\s+WHERE\s+DATABASE\s*=`)
var showDatabasesWhereDatabaseInlinePattern = regexp.MustCompile(`(?i)\bWHERE\s+DATABASE\b`)

// MariaDBDropForeignKeyStmt captures a MariaDB-only ALTER TABLE extension not accepted by TiDB parser.
type MariaDBDropForeignKeyStmt struct {
	Table    string
	Name     string
	IfExists bool
}

func parseMariaDBExtensionStatement(sql, dialect string) ([]*ParsedStatement, bool) {
	if parsed, ok := parseShowDatabasesWhereDatabaseExtension(sql, dialect); ok {
		return parsed, true
	}
	if parsed, ok := parseStandaloneMariaDBDropForeignKeyIfExists(sql, dialect); ok {
		return parsed, true
	}
	return parseMariaDBDropForeignKeyIfExistsAlterTableBatch(sql, dialect)
}

func parseShowDatabasesWhereDatabaseExtension(sql, dialect string) ([]*ParsedStatement, bool) {
	if !showDatabasesWhereDatabasePattern.MatchString(sql) {
		return nil, false
	}

	sanitized := showDatabasesWhereDatabaseInlinePattern.ReplaceAllString(sql, "WHERE `Database`")
	nodes, err := adapter.ParseTiDBStatements(sanitized)
	if err != nil || len(nodes) != 1 {
		return nil, false
	}

	showStmt, ok := nodes[0].(*tidbast.ShowStmt)
	if !ok || showStmt.Tp != tidbast.ShowDatabases || showStmt.Where == nil {
		return nil, false
	}

	return wrapTiDBStatements(sql, dialect, nodes), true
}

func parseStandaloneMariaDBDropForeignKeyIfExists(sql, dialect string) ([]*ParsedStatement, bool) {
	matches := mariaDBDropForeignKeyIfExistsPattern.FindStringSubmatch(sql)
	if len(matches) != 3 {
		return nil, false
	}

	stmt := &MariaDBDropForeignKeyStmt{
		Table:    stripWrappedIdentifier(matches[1]),
		Name:     stripWrappedIdentifier(matches[2]),
		IfExists: true,
	}

	return []*ParsedStatement{
		{
			SQL:            sql,
			SourceDialect:  dialect,
			Kind:           StatementKindAlterTable,
			Status:         SupportStatusSupported,
			NativeNodeType: nativeNodeType(stmt),
			NativeAST:      stmt,
		},
	}, true
}

func parseMariaDBDropForeignKeyIfExistsAlterTableBatch(sql, dialect string) ([]*ParsedStatement, bool) {
	dropForeignKeySpecIndexes, dropPrimaryKeySpecIndexes, dropIndexSpecIndexes, addUniqueKeySpecIndexes, ok := mariaDBExtensionSpecIndexes(sql)
	if !ok {
		return nil, false
	}

	sanitized := mariaDBDropForeignKeyIfExistsInlinePattern.ReplaceAllString(sql, "DROP FOREIGN KEY")
	sanitized = mariaDBDropPrimaryKeyIfExistsInlinePattern.ReplaceAllString(sanitized, "DROP PRIMARY KEY")
	sanitized = mariaDBDropIndexIfExistsInlinePattern.ReplaceAllString(sanitized, "DROP INDEX")
	sanitized = mariaDBAddUniqueKeyIfNotExistsInlinePattern.ReplaceAllString(sanitized, "ADD UNIQUE KEY")
	nodes, err := adapter.ParseTiDBStatements(sanitized)
	if err != nil || len(nodes) != 1 {
		return nil, false
	}

	alterStmt, ok := nodes[0].(*tidbast.AlterTableStmt)
	if !ok || !markAlterTableExtensionSpecs(alterStmt, dropForeignKeySpecIndexes, dropPrimaryKeySpecIndexes, dropIndexSpecIndexes, addUniqueKeySpecIndexes) {
		return nil, false
	}

	return wrapTiDBStatements(sql, dialect, nodes), true
}

func mariaDBExtensionSpecIndexes(sql string) ([]int, []int, []int, []int, bool) {
	if !mariaDBDropForeignKeyIfExistsInlinePattern.MatchString(sql) &&
		!mariaDBDropPrimaryKeyIfExistsInlinePattern.MatchString(sql) &&
		!mariaDBDropIndexIfExistsInlinePattern.MatchString(sql) &&
		!mariaDBAddUniqueKeyIfNotExistsInlinePattern.MatchString(sql) {
		return nil, nil, nil, nil, false
	}

	specs, ok := splitAlterTableSpecClauses(sql)
	if !ok {
		return nil, nil, nil, nil, false
	}

	dropForeignKeyIndexes := make([]int, 0, len(specs))
	dropPrimaryKeyIndexes := make([]int, 0, len(specs))
	dropIndexIndexes := make([]int, 0, len(specs))
	addUniqueKeyIndexes := make([]int, 0, len(specs))
	for i, spec := range specs {
		if mariaDBDropForeignKeyIfExistsInlinePattern.MatchString(spec) {
			dropForeignKeyIndexes = append(dropForeignKeyIndexes, i)
		}
		if mariaDBDropPrimaryKeyIfExistsInlinePattern.MatchString(spec) {
			dropPrimaryKeyIndexes = append(dropPrimaryKeyIndexes, i)
		}
		if mariaDBDropIndexIfExistsInlinePattern.MatchString(spec) {
			dropIndexIndexes = append(dropIndexIndexes, i)
		}
		if mariaDBAddUniqueKeyIfNotExistsInlinePattern.MatchString(spec) {
			addUniqueKeyIndexes = append(addUniqueKeyIndexes, i)
		}
	}
	if len(dropForeignKeyIndexes) == 0 &&
		len(dropPrimaryKeyIndexes) == 0 &&
		len(dropIndexIndexes) == 0 &&
		len(addUniqueKeyIndexes) == 0 {
		return nil, nil, nil, nil, false
	}
	return dropForeignKeyIndexes, dropPrimaryKeyIndexes, dropIndexIndexes, addUniqueKeyIndexes, true
}

func splitAlterTableSpecClauses(sql string) ([]string, bool) {
	start, ok := alterTableSpecStart(sql)
	if !ok {
		return nil, false
	}

	body := strings.TrimSpace(sql[start:])
	if body == "" {
		return nil, false
	}
	if strings.HasSuffix(body, ";") {
		body = strings.TrimSpace(strings.TrimSuffix(body, ";"))
	}
	if body == "" {
		return nil, false
	}

	specs := make([]string, 0, 4)
	depth := 0
	quote := byte(0)
	startIndex := 0
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if quote != 0 {
			if ch == quote && !isEscapedQuote(body, i, quote) {
				quote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				spec := strings.TrimSpace(body[startIndex:i])
				if spec != "" {
					specs = append(specs, spec)
				}
				startIndex = i + 1
			}
		}
	}

	last := strings.TrimSpace(body[startIndex:])
	if last != "" {
		specs = append(specs, last)
	}
	if len(specs) == 0 {
		return nil, false
	}
	return specs, true
}

func alterTableSpecStart(sql string) (int, bool) {
	trimmed := strings.TrimSpace(sql)
	if len(trimmed) < len("ALTER TABLE") || !strings.EqualFold(trimmed[:len("ALTER TABLE")], "ALTER TABLE") {
		return 0, false
	}

	index := len("ALTER TABLE")
	for index < len(trimmed) && isSQLSpace(trimmed[index]) {
		index++
	}
	if index >= len(trimmed) {
		return 0, false
	}

	next, ok := scanQualifiedIdentifier(trimmed, index)
	if !ok {
		return 0, false
	}
	for next < len(trimmed) && isSQLSpace(trimmed[next]) {
		next++
	}
	if next >= len(trimmed) {
		return 0, false
	}
	return len(sql) - len(trimmed) + next, true
}

func scanQualifiedIdentifier(sql string, start int) (int, bool) {
	index := start
	for {
		next, ok := scanIdentifierAtom(sql, index)
		if !ok {
			return 0, false
		}
		index = next
		for index < len(sql) && isSQLSpace(sql[index]) {
			index++
		}
		if index >= len(sql) || sql[index] != '.' {
			return index, true
		}
		index++
		for index < len(sql) && isSQLSpace(sql[index]) {
			index++
		}
		if index >= len(sql) {
			return 0, false
		}
	}
}

func scanIdentifierAtom(sql string, start int) (int, bool) {
	if start >= len(sql) {
		return 0, false
	}

	switch sql[start] {
	case '`':
		end := strings.IndexByte(sql[start+1:], '`')
		if end < 0 {
			return 0, false
		}
		return start + end + 2, true
	case '"':
		end := strings.IndexByte(sql[start+1:], '"')
		if end < 0 {
			return 0, false
		}
		return start + end + 2, true
	default:
		index := start
		for index < len(sql) && isIdentifierChar(sql[index]) {
			index++
		}
		if index == start {
			return 0, false
		}
		return index, true
	}
}

func isIdentifierChar(ch byte) bool {
	return ch == '_' || ch == '$' ||
		(ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z')
}

func isSQLSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func isEscapedQuote(sql string, index int, quote byte) bool {
	if quote != '\'' || index == 0 {
		return false
	}
	return sql[index-1] == '\\'
}

func markAlterTableExtensionSpecs(stmt *tidbast.AlterTableStmt, dropForeignKeySpecIndexes, dropPrimaryKeySpecIndexes, dropIndexSpecIndexes, addUniqueKeySpecIndexes []int) bool {
	if stmt == nil ||
		(len(dropForeignKeySpecIndexes) == 0 &&
			len(dropPrimaryKeySpecIndexes) == 0 &&
			len(dropIndexSpecIndexes) == 0 &&
			len(addUniqueKeySpecIndexes) == 0) {
		return false
	}

	pendingDropForeignKey := make(map[int]struct{}, len(dropForeignKeySpecIndexes))
	for _, index := range dropForeignKeySpecIndexes {
		pendingDropForeignKey[index] = struct{}{}
	}
	pendingDropPrimaryKey := make(map[int]struct{}, len(dropPrimaryKeySpecIndexes))
	for _, index := range dropPrimaryKeySpecIndexes {
		pendingDropPrimaryKey[index] = struct{}{}
	}
	pendingDropIndex := make(map[int]struct{}, len(dropIndexSpecIndexes))
	for _, index := range dropIndexSpecIndexes {
		pendingDropIndex[index] = struct{}{}
	}
	pendingAddUniqueKey := make(map[int]struct{}, len(addUniqueKeySpecIndexes))
	for _, index := range addUniqueKeySpecIndexes {
		pendingAddUniqueKey[index] = struct{}{}
	}

	for i, spec := range stmt.Specs {
		if _, ok := pendingDropForeignKey[i]; ok {
			if spec == nil || spec.Tp != tidbast.AlterTableDropForeignKey {
				return false
			}
			spec.IfExists = true
			delete(pendingDropForeignKey, i)
		}
		if _, ok := pendingDropPrimaryKey[i]; ok {
			if spec == nil || spec.Tp != tidbast.AlterTableDropPrimaryKey {
				return false
			}
			spec.IfExists = true
			delete(pendingDropPrimaryKey, i)
		}
		if _, ok := pendingDropIndex[i]; ok {
			if spec == nil || spec.Tp != tidbast.AlterTableDropIndex {
				return false
			}
			spec.IfExists = true
			delete(pendingDropIndex, i)
		}
		if _, ok := pendingAddUniqueKey[i]; ok {
			if spec == nil || spec.Tp != tidbast.AlterTableAddConstraint || spec.Constraint == nil {
				return false
			}
			switch spec.Constraint.Tp {
			case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
				spec.Constraint.IfNotExists = true
				delete(pendingAddUniqueKey, i)
			default:
				return false
			}
		}
	}
	return len(pendingDropForeignKey) == 0 &&
		len(pendingDropPrimaryKey) == 0 &&
		len(pendingDropIndex) == 0 &&
		len(pendingAddUniqueKey) == 0
}

func stripWrappedIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.Trim(trimmed, "`\"")
}
