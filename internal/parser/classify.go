package parser

import (
	"fmt"
	"regexp"

	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
)

var (
	setSessionTransactionReadModePattern          = regexp.MustCompile(`(?is)^\s*SET\s+SESSION\s+TRANSACTION\s+READ\s+(ONLY|WRITE)\s*;?\s*$`)
	setTransactionReadModePattern                 = regexp.MustCompile(`(?is)^\s*SET\s+TRANSACTION\s+READ\s+(ONLY|WRITE)\s*;?\s*$`)
	setGlobalTransactionReadModePattern           = regexp.MustCompile(`(?is)^\s*SET\s+GLOBAL\s+TRANSACTION\s+READ\s+(ONLY|WRITE)\s*;?\s*$`)
	setSessionTxReadOnlyAssignmentPattern         = regexp.MustCompile(`(?is)^\s*SET\s+(?:(?:SESSION\s+|@@session\.))?(?:tx_read_only|transaction_read_only)\s*=\s*(?:[01]|ON|OFF|TRUE|FALSE)\s*;?\s*$`)
	startTransactionWithConsistentSnapshotPattern = regexp.MustCompile(`(?is)^\s*START\s+TRANSACTION\s+WITH\s+CONSISTENT\s+SNAPSHOT\s*;?\s*$`)
	commitDefaultCompletionVariantPattern         = regexp.MustCompile(`(?is)^\s*COMMIT\s+(AND\s+NO\s+CHAIN(?:\s+NO\s+RELEASE)?|NO\s+RELEASE)\s*;?\s*$`)
	rollbackDefaultCompletionVariantPattern       = regexp.MustCompile(`(?is)^\s*ROLLBACK\s+(AND\s+NO\s+CHAIN(?:\s+NO\s+RELEASE)?|NO\s+RELEASE)\s*;?\s*$`)
)

func nativeNodeType(node any) string {
	return fmt.Sprintf("%T", node)
}

func classifyTiDBStatement(sql string, node tidbast.StmtNode) (StatementKind, SupportStatus) {
	switch value := node.(type) {
	case *tidbast.SelectStmt:
		if value.With != nil {
			return StatementKindWith, SupportStatusSupported
		}
		return StatementKindSelect, SupportStatusSupported
	case *tidbast.SetOprStmt:
		return StatementKindSetOp, SupportStatusSupported
	case *tidbast.SetStmt:
		return classifyTiDBSetStatement(sql, value)
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
		return classifyTiDBBeginStatement(sql, value)
	case *tidbast.CommitStmt:
		if commitDefaultCompletionVariantPattern.MatchString(sql) {
			return StatementKindCommit, SupportStatusRecognizedUnadapted
		}
		if value.CompletionType != tidbast.CompletionTypeDefault {
			return StatementKindCommit, SupportStatusRecognizedUnadapted
		}
		return StatementKindCommit, SupportStatusSupported
	case *tidbast.RollbackStmt:
		if rollbackDefaultCompletionVariantPattern.MatchString(sql) {
			return StatementKindRollback, SupportStatusRecognizedUnadapted
		}
		if value.CompletionType != tidbast.CompletionTypeDefault {
			return StatementKindRollback, SupportStatusRecognizedUnadapted
		}
		return StatementKindRollback, SupportStatusSupported
	case *tidbast.SavepointStmt:
		return StatementKindSavepoint, SupportStatusSupported
	case *tidbast.ReleaseSavepointStmt:
		return StatementKindReleaseSavepoint, SupportStatusSupported
	case *tidbast.UseStmt:
		return StatementKindUse, SupportStatusSupported
	default:
		return StatementKindOther, SupportStatusUnsupported
	}
}

func classifyTiDBSetStatement(sql string, stmt *tidbast.SetStmt) (StatementKind, SupportStatus) {
	if stmt == nil || len(stmt.Variables) != 1 || stmt.Variables[0] == nil {
		return StatementKindSet, SupportStatusRecognizedUnadapted
	}

	switch stmt.Variables[0].Name {
	case tidbast.SetNames, tidbast.SetCharset:
		return StatementKindSet, SupportStatusSupported
	case "tx_isolation_one_shot":
		return StatementKindSet, SupportStatusSupported
	case "tx_isolation", "transaction_isolation":
		if stmt.Variables[0].IsGlobal {
			return StatementKindSet, SupportStatusRecognizedUnadapted
		}
		return StatementKindSet, SupportStatusSupported
	case "tx_read_only", "transaction_read_only":
		switch {
		case setGlobalTransactionReadModePattern.MatchString(sql):
			return StatementKindSet, SupportStatusRecognizedUnadapted
		case setSessionTxReadOnlyAssignmentPattern.MatchString(sql):
			return StatementKindSet, SupportStatusSupported
		case setSessionTransactionReadModePattern.MatchString(sql), setTransactionReadModePattern.MatchString(sql):
			return StatementKindSet, SupportStatusSupported
		default:
			return StatementKindSet, SupportStatusRecognizedUnadapted
		}
	default:
		return StatementKindSet, SupportStatusRecognizedUnadapted
	}
}

func classifyTiDBBeginStatement(sql string, stmt *tidbast.BeginStmt) (StatementKind, SupportStatus) {
	if stmt == nil {
		return StatementKindBegin, SupportStatusSupported
	}
	if startTransactionWithConsistentSnapshotPattern.MatchString(sql) {
		return StatementKindBegin, SupportStatusRecognizedUnadapted
	}
	if stmt.Mode != "" || stmt.CausalConsistencyOnly || stmt.AsOf != nil {
		return StatementKindBegin, SupportStatusRecognizedUnadapted
	}
	return StatementKindBegin, SupportStatusSupported
}
