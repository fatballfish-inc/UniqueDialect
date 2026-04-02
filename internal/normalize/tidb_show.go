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
		if stmt.Full || stmt.Pattern != nil || stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW TABLES variant")
		}
		return ir.ShowTablesStatement{Database: strings.TrimSpace(stmt.DBName)}, nil
	case tidbast.ShowColumns:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW COLUMNS table")
		}
		if stmt.Full || stmt.Extended || stmt.Pattern != nil || stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW COLUMNS variant")
		}
		database := strings.TrimSpace(stmt.DBName)
		if database == "" {
			database = strings.TrimSpace(stmt.Table.Schema.O)
		}
		return ir.ShowColumnsStatement{
			Table:    strings.TrimSpace(stmt.Table.Name.O),
			Database: database,
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
		if stmt.Pattern != nil || stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW TABLE STATUS variant")
		}
		return ir.ShowTableStatusStatement{
			Database: strings.TrimSpace(stmt.DBName),
		}, nil
	case tidbast.ShowDatabases:
		if stmt.Pattern != nil || stmt.Where != nil {
			return nil, fmt.Errorf("unsupported SHOW DATABASES variant")
		}
		return ir.ShowDatabasesStatement{}, nil
	case tidbast.ShowCreateDatabase:
		if strings.TrimSpace(stmt.DBName) == "" {
			return nil, fmt.Errorf("missing SHOW CREATE DATABASE name")
		}
		return ir.ShowCreateDatabaseStatement{
			Name:        strings.TrimSpace(stmt.DBName),
			IfNotExists: stmt.IfNotExists,
		}, nil
	case tidbast.ShowCreateView:
		if stmt.Table == nil {
			return nil, fmt.Errorf("missing SHOW CREATE VIEW table")
		}
		return ir.ShowCreateViewStatement{Name: normalizeTiDBTableName(stmt.Table)}, nil
	default:
		return nil, fmt.Errorf("unsupported SHOW statement type %v", stmt.Tp)
	}
}
