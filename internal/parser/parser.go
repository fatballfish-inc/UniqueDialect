package parser

import (
	"strings"

	"github.com/fatballfish-inc/UniqueDialect/internal/parser/adapter"
	tidbast "github.com/fatballfish-inc/UniqueDialect/internal/parser/tidb/ast"
	"github.com/fatballfish-inc/UniqueDialect/internal/syntax"
)

// ParseOne parses a single SQL statement and classifies it for downstream adaptation.
func ParseOne[D ~string](sql string, dialect D) (*ParsedStatement, error) {
	statements, err := ParseMulti(sql, dialect)
	if err != nil {
		return nil, err
	}
	if len(statements) != 1 {
		return nil, ErrExpectedSingleStatement{Count: len(statements)}
	}
	return statements[0], nil
}

// ParseMulti parses one or more SQL statements and returns project-owned parse results.
func ParseMulti[D ~string](sql string, dialect D) ([]*ParsedStatement, error) {
	dialectValue := string(dialect)
	switch strings.ToLower(strings.TrimSpace(dialectValue)) {
	case "mysql", "sqlite", "oracle":
		nodes, err := adapter.ParseTiDBStatements(sql)
		if err != nil {
			if extended, ok := parseMariaDBExtensionStatement(sql, dialectValue); ok {
				return extended, nil
			}
			return nil, err
		}
		return wrapTiDBStatements(sql, dialectValue, nodes), nil
	case "postgres":
		nodes, err := adapter.ParsePostgresStatements(sql)
		if err != nil {
			return nil, err
		}
		return wrapLegacyStatements(sql, dialectValue, nodes), nil
	default:
		return nil, ErrUnsupportedDialect{Dialect: dialectValue}
	}
}

func wrapTiDBStatements(sql, dialect string, nodes []tidbast.StmtNode) []*ParsedStatement {
	statements := make([]*ParsedStatement, 0, len(nodes))
	for _, node := range nodes {
		kind, status := classifyTiDBStatement(sql, node)
		statements = append(statements, &ParsedStatement{
			SQL:            sql,
			SourceDialect:  dialect,
			Kind:           kind,
			Status:         status,
			NativeNodeType: nativeNodeType(node),
			NativeAST:      node,
		})
	}
	return statements
}

func wrapLegacyStatements(sql, dialect string, nodes []syntax.Statement) []*ParsedStatement {
	statements := make([]*ParsedStatement, 0, len(nodes))
	for _, node := range nodes {
		kind, status, nativeType := classifyLegacyStatement(node)
		statements = append(statements, &ParsedStatement{
			SQL:            sql,
			SourceDialect:  dialect,
			Kind:           kind,
			Status:         status,
			NativeNodeType: nativeType,
			NativeAST:      node,
		})
	}
	return statements
}
