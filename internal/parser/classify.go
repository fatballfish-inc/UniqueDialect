package parser

import (
	"fmt"

	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
)

func nativeNodeType(node any) string {
	return fmt.Sprintf("%T", node)
}

func classifyTiDBStatement(node tidbast.StmtNode) (StatementKind, SupportStatus) {
	switch value := node.(type) {
	case *tidbast.SelectStmt:
		if value.With != nil {
			return StatementKindWith, SupportStatusSupported
		}
		return StatementKindSelect, SupportStatusSupported
	case *tidbast.SetOprStmt:
		return StatementKindSetOp, SupportStatusSupported
	case *tidbast.SetStmt:
		return classifyTiDBSetStatement(value)
	case *tidbast.InsertStmt:
		return StatementKindInsert, SupportStatusSupported
	case *tidbast.UpdateStmt:
		return StatementKindUpdate, SupportStatusSupported
	case *tidbast.DeleteStmt:
		return StatementKindDelete, SupportStatusSupported
	case *tidbast.RenameTableStmt:
		return StatementKindRenameTable, SupportStatusSupported

	case *tidbast.CreateTableStmt:
		return StatementKindCreateTable, SupportStatusSupported
	case *tidbast.CreateDatabaseStmt:
		return StatementKindCreateDatabase, SupportStatusSupported
	case *tidbast.ExplainStmt:
		return StatementKindExplain, SupportStatusSupported
	case *tidbast.CreateViewStmt:
		return StatementKindCreateView, SupportStatusSupported
	case *tidbast.ShowStmt:
		switch value.Tp {
		case tidbast.ShowTables, tidbast.ShowColumns, tidbast.ShowIndex, tidbast.ShowTableStatus, tidbast.ShowCreateTable, tidbast.ShowCreateView, tidbast.ShowDatabases, tidbast.ShowCreateDatabase:
			return StatementKindShow, SupportStatusSupported
		case tidbast.ShowVariables:
			if value.GlobalScope {
				return StatementKindShow, SupportStatusRecognizedUnadapted
			}
			return StatementKindShow, SupportStatusSupported
		default:
			return StatementKindShow, SupportStatusRecognizedUnadapted
		}
	case *tidbast.AlterTableStmt:
		return StatementKindAlterTable, SupportStatusSupported
	case *tidbast.DropTableStmt:
		if value.IsView {
			return StatementKindDropView, SupportStatusSupported
		}
		return StatementKindDropTable, SupportStatusSupported
	case *tidbast.DropDatabaseStmt:
		return StatementKindDropDatabase, SupportStatusSupported
	case *tidbast.TruncateTableStmt:
		return StatementKindTruncate, SupportStatusSupported
	case *tidbast.CreateIndexStmt:
		return StatementKindCreateIndex, SupportStatusSupported
	case *tidbast.DropIndexStmt:
		return StatementKindDropIndex, SupportStatusSupported
	case *tidbast.BeginStmt:
		return StatementKindBegin, SupportStatusSupported
	case *tidbast.CommitStmt:
		return StatementKindCommit, SupportStatusSupported
	case *tidbast.RollbackStmt:
		return StatementKindRollback, SupportStatusSupported
	case *tidbast.UseStmt:
		return StatementKindUse, SupportStatusSupported
	default:
		return StatementKindOther, SupportStatusUnsupported
	}
}

func classifyTiDBSetStatement(stmt *tidbast.SetStmt) (StatementKind, SupportStatus) {
	if stmt == nil || len(stmt.Variables) != 1 || stmt.Variables[0] == nil {
		return StatementKindSet, SupportStatusRecognizedUnadapted
	}

	switch stmt.Variables[0].Name {
	case tidbast.SetNames, tidbast.SetCharset:
		return StatementKindSet, SupportStatusSupported
	default:
		return StatementKindSet, SupportStatusRecognizedUnadapted
	}
}
