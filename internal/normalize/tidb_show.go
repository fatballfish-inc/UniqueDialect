package normalize

import (
	"fmt"
	"strings"

	"github.com/fatballfish/uniquedialect/internal/ir"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
)

func normalizeTiDBShowStmt(stmt *tidbast.ShowStmt) (ir.Statement, error) {
	if stmt == nil {
		return ir.RawStatement{}, nil
	}

	switch stmt.Tp {
	case tidbast.ShowTables:
		if stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW TABLES variant")
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW TABLES")
		if err != nil {
			return nil, err
		}
		return ir.ShowTablesStatement{
			Database: strings.TrimSpace(stmt.DBName),
			Full:     stmt.Full,
			Pattern:  pattern,
		}, nil
	case tidbast.ShowColumns:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW COLUMNS table")
		}
		if stmt.Extended || stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW COLUMNS variant")
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW COLUMNS")
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
		}, nil
	case tidbast.ShowIndex:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW INDEX table")
		}
		if stmt.Pattern != nil || stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW INDEX variant")
		}
		database := strings.TrimSpace(stmt.DBName)
		if database == "" {
			database = strings.TrimSpace(stmt.Table.Schema.O)
		}
		return ir.ShowIndexStatement{
			Table:    strings.TrimSpace(stmt.Table.Name.O),
			Database: database,
		}, nil
	case tidbast.ShowTableStatus:
		if stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW TABLE STATUS variant")
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW TABLE STATUS")
		if err != nil {
			return nil, err
		}
		return ir.ShowTableStatusStatement{
			Database: strings.TrimSpace(stmt.DBName),
			Pattern:  pattern,
		}, nil
	case tidbast.ShowDatabases:
		if stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW DATABASES variant")
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW DATABASES")
		if err != nil {
			return nil, err
		}
		return ir.ShowDatabasesStatement{Pattern: pattern}, nil
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
		if stmt.GlobalScope || stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW VARIABLES variant")
		}
		pattern, err := normalizeTiDBShowLikePattern(stmt.Pattern, "SHOW VARIABLES")
		if err != nil {
			return nil, err
		}
		return ir.ShowVariablesStatement{Pattern: pattern}, nil
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
