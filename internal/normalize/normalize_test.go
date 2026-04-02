package normalize_test

import (
	"testing"

	"github.com/fatballfish/uniquedialect"
	"github.com/fatballfish/uniquedialect/internal/ir"
	internalnormalize "github.com/fatballfish/uniquedialect/internal/normalize"
	internalparser "github.com/fatballfish/uniquedialect/internal/parser"
)

func TestParsedStatementNormalizesSelectFromTiDBAST(t *testing.T) {
	parsed, err := internalparser.ParseOne(
		"SELECT `name` FROM `users` WHERE id = ? LIMIT 5, 10",
		uniquedialect.DialectMySQL,
	)
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}

	stmt, err := internalnormalize.ParsedStatement(parsed)
	if err != nil {
		t.Fatalf("ParsedStatement() error = %v", err)
	}

	selectStmt, ok := stmt.(ir.SelectStatement)
	if !ok {
		t.Fatalf("ParsedStatement() type = %T, want ir.SelectStatement", stmt)
	}
	if got, want := selectStmt.From, "`users`"; got != want {
		t.Fatalf("From = %q, want %q", got, want)
	}
	if len(selectStmt.Columns) != 1 || selectStmt.Columns[0] != "`name`" {
		t.Fatalf("Columns = %#v, want [`name`]", selectStmt.Columns)
	}
	if selectStmt.Where != "id = ?" {
		t.Fatalf("Where = %q, want %q", selectStmt.Where, "id = ?")
	}
	if selectStmt.Limit == nil || selectStmt.Limit.Count != 10 || selectStmt.Limit.Offset == nil || *selectStmt.Limit.Offset != 5 {
		t.Fatalf("Limit = %#v, want offset=5 count=10", selectStmt.Limit)
	}
}
