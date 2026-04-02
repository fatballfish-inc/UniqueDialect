package normalize

import (
	"fmt"

	"github.com/fatballfish/uniquedialect/internal/ir"
	internalparser "github.com/fatballfish/uniquedialect/internal/parser"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
	"github.com/fatballfish/uniquedialect/internal/syntax"
)

// ParsedStatement converts a unified parser result into normalized IR.
func ParsedStatement(parsed *internalparser.ParsedStatement) (ir.Statement, error) {
	if parsed == nil {
		return nil, fmt.Errorf("parsed statement is nil")
	}

	switch node := parsed.NativeAST.(type) {
	case syntax.Statement:
		return Statement(node), nil
	case *internalparser.MariaDBDropForeignKeyStmt:
		return ir.AlterTableStatement{
			Name: node.Table,
			Specs: []ir.AlterTableSpec{
				{
					Kind:     ir.AlterTableSpecDropForeignKey,
					Name:     node.Name,
					IfExists: node.IfExists,
				},
			},
		}, nil
	case *tidbast.SelectStmt:
		if node.With != nil {
			return normalizeTiDBRawComposable(parsed.SQL)
		}
		return normalizeTiDBSelect(parsed.SQL, node)
	case *tidbast.SetOprStmt:
		return normalizeTiDBRawComposable(parsed.SQL)
	case *tidbast.SetStmt:
		return normalizeTiDBSet(node)
	case *tidbast.ExplainStmt:
		return normalizeTiDBRawComposable(parsed.SQL)
	case *tidbast.ShowStmt:
		return normalizeTiDBShowStmt(node)
	case *tidbast.CreateTableStmt:
		return normalizeTiDBCreateTable(node)
	case *tidbast.CreateDatabaseStmt:
		return normalizeTiDBRawDDL(parsed.SQL)
	case *tidbast.RenameTableStmt:
		return normalizeTiDBRenameTable(parsed.SQL, node)
	case *tidbast.CreateViewStmt:
		return normalizeTiDBRawDDL(parsed.SQL)
	case *tidbast.AlterTableStmt:
		return normalizeTiDBAlterTable(parsed.SQL, node)
	case *tidbast.DropTableStmt:
		return normalizeTiDBRawDDL(parsed.SQL)
	case *tidbast.DropDatabaseStmt:
		return normalizeTiDBRawDDL(parsed.SQL)
	case *tidbast.TruncateTableStmt:
		return normalizeTiDBRawDDL(parsed.SQL)
	case *tidbast.CreateIndexStmt:
		return normalizeTiDBCreateIndex(node)
	case *tidbast.DropIndexStmt:
		return normalizeTiDBDropIndex(node)
	case *tidbast.BeginStmt:
		return normalizeTiDBRawDDL(parsed.SQL)
	case *tidbast.CommitStmt:
		return normalizeTiDBRawDDL(parsed.SQL)
	case *tidbast.RollbackStmt:
		return normalizeTiDBRollback(node, parsed.SQL)
	case *tidbast.SavepointStmt:
		return normalizeTiDBSavepoint(node)
	case *tidbast.ReleaseSavepointStmt:
		return normalizeTiDBReleaseSavepoint(node)
	case *tidbast.UseStmt:
		return ir.UseStatement{Database: node.DBName}, nil
	case *tidbast.InsertStmt:
		return normalizeTiDBInsert(parsed.SQL, node)
	case *tidbast.UpdateStmt:
		return normalizeTiDBUpdate(parsed.SQL, node)
	case *tidbast.DeleteStmt:
		return normalizeTiDBDelete(parsed.SQL, node)
	default:
		return nil, fmt.Errorf("unsupported parsed statement type %T", parsed.NativeAST)
	}
}
