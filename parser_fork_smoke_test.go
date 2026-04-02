package uniquedialect_test

import (
	"testing"

	tidbparser "github.com/fatballfish/uniquedialect/internal/parser/tidb"
	_ "github.com/fatballfish/uniquedialect/internal/parser/tidb/test_driver"
)

func TestForkedTiDBParserParsesSimpleSelect(t *testing.T) {
	p := tidbparser.New()
	stmt, _, err := p.ParseSQL("SELECT 1")
	if err != nil {
		t.Fatalf("ParseSQL() error = %v", err)
	}
	if len(stmt) != 1 {
		t.Fatalf("len(stmt) = %d, want 1", len(stmt))
	}
}
