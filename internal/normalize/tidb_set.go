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

	charset, err := normalizeTiDBSetCharset(variable.Value)
	if err != nil {
		return nil, err
	}

	switch variable.Name {
	case tidbast.SetNames:
		return ir.SetStatement{Kind: "names", Charset: charset}, nil
	case tidbast.SetCharset:
		return ir.SetStatement{Kind: "charset", Charset: charset}, nil
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
