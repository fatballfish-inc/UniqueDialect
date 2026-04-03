package normalize

import (
	"fmt"
	"strings"

	"github.com/fatballfish/uniquedialect/internal/ir"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
)

func normalizeTiDBSet(stmt *tidbast.SetStmt) (ir.Statement, error) {
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
