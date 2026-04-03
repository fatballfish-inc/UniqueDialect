package normalize

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fatballfish/uniquedialect/internal/ir"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
)

var (
	normalizeSetSessionTransactionReadModePattern  = regexp.MustCompile(`(?is)^\s*SET\s+SESSION\s+TRANSACTION\s+READ\s+(ONLY|WRITE)\s*;?\s*$`)
	normalizeSetTransactionReadModePattern         = regexp.MustCompile(`(?is)^\s*SET\s+TRANSACTION\s+READ\s+(ONLY|WRITE)\s*;?\s*$`)
	normalizeSetSessionTxReadOnlyAssignmentPattern = regexp.MustCompile(`(?is)^\s*SET\s+SESSION\s+tx_read_only\s*=\s*[01]\s*;?\s*$`)
)

func normalizeTiDBSet(sql string, stmt *tidbast.SetStmt) (ir.Statement, error) {
	if stmt == nil || len(stmt.Variables) != 1 || stmt.Variables[0] == nil {
		return nil, fmt.Errorf("unsupported SET variant")
	}

	variable := stmt.Variables[0]
	if variable.ExtendValue != nil {
		return nil, fmt.Errorf("unsupported SET variant")
	}

	switch variable.Name {
	case tidbast.SetNames:
		charset, err := normalizeTiDBSetCharset(variable.Value)
		if err != nil {
			return nil, err
		}
		return ir.SetStatement{Kind: "names", Charset: charset}, nil
	case tidbast.SetCharset:
		charset, err := normalizeTiDBSetCharset(variable.Value)
		if err != nil {
			return nil, err
		}
		return ir.SetStatement{Kind: "charset", Charset: charset}, nil
	case "tx_isolation_one_shot":
		level, err := normalizeTiDBTransactionIsolationLevel(variable.Value)
		if err != nil {
			return nil, err
		}
		return ir.SetTransactionStatement{
			Scope:          "transaction",
			IsolationLevel: level,
		}, nil
	case "tx_isolation":
		if variable.IsGlobal {
			return nil, fmt.Errorf("unsupported SET variant")
		}
		level, err := normalizeTiDBTransactionIsolationLevel(variable.Value)
		if err != nil {
			return nil, err
		}
		return ir.SetTransactionStatement{
			Scope:          "session",
			IsolationLevel: level,
		}, nil
	case "tx_read_only":
		scope, err := normalizeTiDBTransactionReadModeScope(sql)
		if err != nil {
			return nil, err
		}
		mode, err := normalizeTiDBTransactionReadMode(variable.Value)
		if err != nil {
			return nil, err
		}
		return ir.SetTransactionStatement{
			Scope:      scope,
			AccessMode: mode,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported SET variant")
	}
}

func normalizeTiDBSetCharset(value tidbast.ExprNode) (string, error) {
	if value == nil {
		return "", fmt.Errorf("unsupported SET charset")
	}

	charset := strings.ToLower(strings.TrimSpace(tiDBNodeText(value)))
	charset = strings.Trim(charset, "`'\" ")
	switch charset {
	case "utf8", "utf8mb4":
		return charset, nil
	default:
		return "", fmt.Errorf("unsupported SET charset %s", charset)
	}
}

func normalizeTiDBTransactionIsolationLevel(value tidbast.ExprNode) (string, error) {
	if value == nil {
		return "", fmt.Errorf("unsupported SET transaction isolation level")
	}

	level := ""
	if valueExpr, ok := value.(tidbast.ValueExpr); ok {
		level = valueExpr.GetString()
	} else {
		level = tiDBNodeText(value)
	}

	level = strings.ToUpper(strings.TrimSpace(level))
	level = strings.Trim(level, "`'\" ")
	level = strings.ReplaceAll(level, "-", " ")
	switch level {
	case "READ UNCOMMITTED", "READ COMMITTED", "REPEATABLE READ", "SERIALIZABLE":
		return level, nil
	default:
		return "", fmt.Errorf("unsupported SET transaction isolation level %s", level)
	}
}

func normalizeTiDBTransactionReadModeScope(sql string) (string, error) {
	switch {
	case normalizeSetSessionTxReadOnlyAssignmentPattern.MatchString(sql):
		return "session", nil
	case normalizeSetSessionTransactionReadModePattern.MatchString(sql):
		return "session", nil
	case normalizeSetTransactionReadModePattern.MatchString(sql):
		return "transaction", nil
	default:
		return "", fmt.Errorf("unsupported SET variant")
	}
}

func normalizeTiDBTransactionReadMode(value tidbast.ExprNode) (string, error) {
	if value == nil {
		return "", fmt.Errorf("unsupported SET transaction access mode")
	}

	raw := ""
	if valueExpr, ok := value.(tidbast.ValueExpr); ok {
		if direct := valueExpr.GetValue(); direct != nil {
			raw = fmt.Sprint(direct)
		} else {
			raw = valueExpr.GetString()
		}
	} else {
		raw = tiDBNodeText(value)
	}

	mode := strings.Trim(strings.TrimSpace(raw), "`'\" ")
	switch mode {
	case "0":
		return "READ WRITE", nil
	case "1":
		return "READ ONLY", nil
	default:
		return "", fmt.Errorf("unsupported SET transaction access mode %s", mode)
	}
}
