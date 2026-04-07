package uniquedialect_test

import (
	"testing"

	uniquedialect "github.com/fatballfish-inc/UniqueDialect"
)

func TestModulePathMatchesRepositoryImportPath(t *testing.T) {
	if uniquedialect.DialectMySQL == "" {
		t.Fatal("expected exported dialect constant to be available from repository module path")
	}
}
