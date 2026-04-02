package adapter

import (
	tidbparser "github.com/fatballfish/uniquedialect/internal/parser/tidb"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
	_ "github.com/fatballfish/uniquedialect/internal/parser/tidb/test_driver"
)

// ParseTiDBStatements parses SQL through the forked TiDB parser.
func ParseTiDBStatements(sql string) ([]tidbast.StmtNode, error) {
	parser := tidbparser.New()
	nodes, _, err := parser.ParseSQL(sql)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}
