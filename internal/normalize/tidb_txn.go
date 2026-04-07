package normalize

import (
	"fmt"
	"strings"

	"github.com/fatballfish-inc/UniqueDialect/internal/ir"
	tidbast "github.com/fatballfish-inc/UniqueDialect/internal/parser/tidb/ast"
)

func normalizeTiDBSavepoint(stmt *tidbast.SavepointStmt) (ir.Statement, error) {
	if stmt == nil || strings.TrimSpace(stmt.Name) == "" {
		return nil, fmt.Errorf("missing SAVEPOINT name")
	}
	return ir.SavepointStatement{Name: strings.TrimSpace(stmt.Name)}, nil
}

func normalizeTiDBReleaseSavepoint(stmt *tidbast.ReleaseSavepointStmt) (ir.Statement, error) {
	if stmt == nil || strings.TrimSpace(stmt.Name) == "" {
		return nil, fmt.Errorf("missing RELEASE SAVEPOINT name")
	}
	return ir.ReleaseSavepointStatement{Name: strings.TrimSpace(stmt.Name)}, nil
}

func normalizeTiDBRollback(stmt *tidbast.RollbackStmt, sql string) (ir.Statement, error) {
	if stmt == nil {
		return normalizeTiDBRawDDL(sql)
	}
	if strings.TrimSpace(stmt.SavepointName) != "" {
		return ir.RollbackToSavepointStatement{Name: strings.TrimSpace(stmt.SavepointName)}, nil
	}
	return normalizeTiDBRawDDL(sql)
}
