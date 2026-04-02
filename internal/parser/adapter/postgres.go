package adapter

import (
	"github.com/fatballfish/uniquedialect/internal/syntax"
	postgressyntax "github.com/fatballfish/uniquedialect/internal/syntax/postgres"
)

// ParsePostgresStatements parses SQL through the transitional PostgreSQL syntax backend.
func ParsePostgresStatements(sql string) ([]syntax.Statement, error) {
	stmt, err := postgressyntax.Parse(sql)
	if err != nil {
		return nil, err
	}
	return []syntax.Statement{stmt}, nil
}
