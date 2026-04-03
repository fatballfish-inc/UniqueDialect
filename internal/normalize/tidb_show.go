package normalize

import (
	"fmt"
	"strings"

	"github.com/fatballfish/uniquedialect/internal/ir"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
	tidbopcode "github.com/fatballfish/uniquedialect/internal/parser/tidb/opcode"
)

func normalizeTiDBShowStmt(stmt *tidbast.ShowStmt) (ir.Statement, error) {
	if stmt == nil {
		return ir.RawStatement{}, nil
	}

	switch stmt.Tp {
	case tidbast.ShowTables:
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW TABLES")
		if err != nil {
			return nil, err
		}
		name, tableType, err := normalizeTiDBShowTablesWhere(strings.TrimSpace(stmt.DBName), stmt.Full, stmt.Where)
		if err != nil {
			return nil, err
		}
		return ir.ShowTablesStatement{
			Database:  strings.TrimSpace(stmt.DBName),
			Full:      stmt.Full,
			Pattern:   pattern,
			Name:      name,
			TableType: tableType,
		}, nil
	case tidbast.ShowColumns:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW COLUMNS table")
		}
		if stmt.Extended {
			return nil, fmt.Errorf("unsupported SHOW COLUMNS variant")
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW COLUMNS")
		if err != nil {
			return nil, err
		}
		field, err := normalizeTiDBShowColumnsWhere(stmt.Where)
		if err != nil {
			return nil, err
		}
		database := strings.TrimSpace(stmt.DBName)
		if database == "" {
			database = strings.TrimSpace(stmt.Table.Schema.O)
		}
		return ir.ShowColumnsStatement{
			Table:    strings.TrimSpace(stmt.Table.Name.O),
			Database: database,
			Full:     stmt.Full,
			Pattern:  pattern,
			Field:    field,
		}, nil
	case tidbast.ShowIndex:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW INDEX table")
		}
		if stmt.Pattern != nil {
			return nil, fmt.Errorf("unsupported SHOW INDEX variant")
		}
		keyName, columnName, indexType, err := normalizeTiDBShowIndexWhere(stmt.Where)
		if err != nil {
			return nil, err
		}
		database := strings.TrimSpace(stmt.DBName)
		if database == "" {
			database = strings.TrimSpace(stmt.Table.Schema.O)
		}
		return ir.ShowIndexStatement{
			Table:     strings.TrimSpace(stmt.Table.Name.O),
			Database:  database,
			KeyName:   keyName,
			Column:    columnName,
			IndexType: indexType,
		}, nil
	case tidbast.ShowTableStatus:
		name, comment, err := normalizeTiDBShowTableStatusWhere(stmt.Where)
		if err != nil {
			return nil, err
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW TABLE STATUS")
		if err != nil {
			return nil, err
		}
		return ir.ShowTableStatusStatement{
			Database: strings.TrimSpace(stmt.DBName),
			Pattern:  pattern,
			Name:     name,
			Comment:  comment,
		}, nil
	case tidbast.ShowDatabases:
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW DATABASES")
		if err != nil {
			return nil, err
		}
		name, err := normalizeTiDBShowDatabasesWhere(stmt.Where)
		if err != nil {
			return nil, err
		}
		return ir.ShowDatabasesStatement{Pattern: pattern, Name: name}, nil
	case tidbast.ShowCreateDatabase:
		if strings.TrimSpace(stmt.DBName) == "" {
			return nil, fmt.Errorf("missing SHOW CREATE DATABASE name")
		}
		return ir.ShowCreateDatabaseStatement{
			Name:        strings.TrimSpace(stmt.DBName),
			IfNotExists: stmt.IfNotExists,
		}, nil
	case tidbast.ShowCreateTable:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW CREATE TABLE table")
		}
		return ir.ShowCreateTableStatement{
			Schema: strings.TrimSpace(stmt.Table.Schema.O),
			Name:   strings.TrimSpace(stmt.Table.Name.O),
		}, nil
	case tidbast.ShowCreateView:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW CREATE VIEW table")
		}
		return ir.ShowCreateViewStatement{
			Schema: strings.TrimSpace(stmt.Table.Schema.O),
			Name:   strings.TrimSpace(stmt.Table.Name.O),
		}, nil
	case tidbast.ShowVariables:
		if stmt.GlobalScope {
			return nil, fmt.Errorf("unsupported SHOW VARIABLES variant")
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW VARIABLES")
		if err != nil {
			return nil, err
		}
		name, err := normalizeTiDBShowVariablesWhere(stmt.Where)
		if err != nil {
			return nil, err
		}
		return ir.ShowVariablesStatement{Pattern: pattern, Name: name}, nil
	default:
		return nil, fmt.Errorf("unsupported SHOW statement type %v", stmt.Tp)
	}
}

func normalizeTiDBShowLikePattern(pattern *tidbast.PatternLikeOrIlikeExpr, label string) (string, error) {
	if pattern == nil || pattern.Pattern == nil {
		return "", nil
	}
	if pattern.Not || !pattern.IsLike || (pattern.Escape != 0 && pattern.Escape != '\\') {
		return "", fmt.Errorf("unsupported %s variant", label)
	}

	valueExpr, ok := pattern.Pattern.(tidbast.ValueExpr)
	if !ok {
		return "", fmt.Errorf("unsupported %s variant", label)
	}

	return valueExpr.GetString(), nil
}

func normalizeTiDBShowTableStatusWhere(where tidbast.ExprNode) (string, string, error) {
	if where == nil {
		return "", "", nil
	}

	binary, ok := where.(*tidbast.BinaryOperationExpr)
	if !ok || binary.Op != tidbopcode.EQ {
		return "", "", fmt.Errorf("unsupported SHOW TABLE STATUS variant")
	}

	columnExpr, ok := binary.L.(*tidbast.ColumnNameExpr)
	if !ok || columnExpr.Name == nil {
		return "", "", fmt.Errorf("unsupported SHOW TABLE STATUS variant")
	}

	valueExpr, ok := binary.R.(tidbast.ValueExpr)
	if !ok {
		return "", "", fmt.Errorf("unsupported SHOW TABLE STATUS variant")
	}

	value := strings.TrimSpace(valueExpr.GetString())

	switch {
	case strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Name"):
		return value, "", nil
	case strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Comment"):
		return "", value, nil
	default:
		return "", "", fmt.Errorf("unsupported SHOW TABLE STATUS variant")
	}
}

func normalizeTiDBShowIndexWhere(where tidbast.ExprNode) (string, string, string, error) {
	if where == nil {
		return "", "", "", nil
	}

	binary, ok := where.(*tidbast.BinaryOperationExpr)
	if !ok || binary.Op != tidbopcode.EQ {
		return "", "", "", fmt.Errorf("unsupported SHOW INDEX variant")
	}

	columnExpr, ok := binary.L.(*tidbast.ColumnNameExpr)
	if !ok || columnExpr.Name == nil {
		return "", "", "", fmt.Errorf("unsupported SHOW INDEX variant")
	}

	valueExpr, ok := binary.R.(tidbast.ValueExpr)
	if !ok {
		return "", "", "", fmt.Errorf("unsupported SHOW INDEX variant")
	}

	value := strings.TrimSpace(valueExpr.GetString())

	switch {
	case strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Key_name"):
		return value, "", "", nil
	case strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Column_name"):
		return "", value, "", nil
	case strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Index_type"):
		return "", "", value, nil
	default:
		return "", "", "", fmt.Errorf("unsupported SHOW INDEX variant")
	}
}

func normalizeTiDBShowVariablesWhere(where tidbast.ExprNode) (string, error) {
	if where == nil {
		return "", nil
	}

	binary, ok := where.(*tidbast.BinaryOperationExpr)
	if !ok || binary.Op != tidbopcode.EQ {
		return "", fmt.Errorf("unsupported SHOW VARIABLES variant")
	}

	columnExpr, ok := binary.L.(*tidbast.ColumnNameExpr)
	if !ok || columnExpr.Name == nil || !strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Variable_name") {
		return "", fmt.Errorf("unsupported SHOW VARIABLES variant")
	}

	valueExpr, ok := binary.R.(tidbast.ValueExpr)
	if !ok {
		return "", fmt.Errorf("unsupported SHOW VARIABLES variant")
	}

	return strings.TrimSpace(valueExpr.GetString()), nil
}

func normalizeTiDBShowDatabasesWhere(where tidbast.ExprNode) (string, error) {
	if where == nil {
		return "", nil
	}

	binary, ok := where.(*tidbast.BinaryOperationExpr)
	if !ok || binary.Op != tidbopcode.EQ {
		return "", fmt.Errorf("unsupported SHOW DATABASES variant")
	}

	columnExpr, ok := binary.L.(*tidbast.ColumnNameExpr)
	if !ok || columnExpr.Name == nil || !strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Database") {
		return "", fmt.Errorf("unsupported SHOW DATABASES variant")
	}

	valueExpr, ok := binary.R.(tidbast.ValueExpr)
	if !ok {
		return "", fmt.Errorf("unsupported SHOW DATABASES variant")
	}

	return strings.TrimSpace(valueExpr.GetString()), nil
}

func normalizeTiDBShowColumnsWhere(where tidbast.ExprNode) (string, error) {
	if where == nil {
		return "", nil
	}

	binary, ok := where.(*tidbast.BinaryOperationExpr)
	if !ok || binary.Op != tidbopcode.EQ {
		return "", fmt.Errorf("unsupported SHOW COLUMNS variant")
	}

	columnExpr, ok := binary.L.(*tidbast.ColumnNameExpr)
	if !ok || columnExpr.Name == nil || !strings.EqualFold(strings.TrimSpace(columnExpr.Name.Name.O), "Field") {
		return "", fmt.Errorf("unsupported SHOW COLUMNS variant")
	}

	valueExpr, ok := binary.R.(tidbast.ValueExpr)
	if !ok {
		return "", fmt.Errorf("unsupported SHOW COLUMNS variant")
	}

	return strings.TrimSpace(valueExpr.GetString()), nil
}

func normalizeTiDBShowTablesWhere(database string, full bool, where tidbast.ExprNode) (string, string, error) {
	if where == nil {
		return "", "", nil
	}

	binary, ok := where.(*tidbast.BinaryOperationExpr)
	if !ok || binary.Op != tidbopcode.EQ {
		return "", "", fmt.Errorf("unsupported SHOW TABLES variant")
	}

	columnExpr, ok := binary.L.(*tidbast.ColumnNameExpr)
	if !ok || columnExpr.Name == nil {
		return "", "", fmt.Errorf("unsupported SHOW TABLES variant")
	}

	valueExpr, ok := binary.R.(tidbast.ValueExpr)
	if !ok {
		return "", "", fmt.Errorf("unsupported SHOW TABLES variant")
	}

	columnName := strings.TrimSpace(columnExpr.Name.Name.O)
	if full && strings.EqualFold(columnName, "Table_type") {
		return "", strings.TrimSpace(valueExpr.GetString()), nil
	}

	expectedNameColumn := "Tables_in_current_schema"
	if strings.TrimSpace(database) != "" {
		expectedNameColumn = "Tables_in_" + strings.TrimSpace(database)
	}
	if strings.EqualFold(columnName, expectedNameColumn) {
		return strings.TrimSpace(valueExpr.GetString()), "", nil
	}

	return "", "", fmt.Errorf("unsupported SHOW TABLES variant")
}
