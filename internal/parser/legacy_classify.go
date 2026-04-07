package parser

import (
	"fmt"

	"github.com/fatballfish-inc/UniqueDialect/internal/syntax"
)

func classifyLegacyStatement(node syntax.Statement) (StatementKind, SupportStatus, string) {
	switch value := node.(type) {
	case syntax.SelectStatement:
		return StatementKindSelect, SupportStatusSupported, fmt.Sprintf("%T", value)
	case syntax.InsertStatement:
		return StatementKindInsert, SupportStatusSupported, fmt.Sprintf("%T", value)
	case syntax.UpdateStatement:
		return StatementKindUpdate, SupportStatusSupported, fmt.Sprintf("%T", value)
	case syntax.DeleteStatement:
		return StatementKindDelete, SupportStatusSupported, fmt.Sprintf("%T", value)
	case syntax.RawStatement:
		return StatementKindOther, SupportStatusUnsupported, fmt.Sprintf("%T", value)
	default:
		return StatementKindOther, SupportStatusUnsupported, fmt.Sprintf("%T", value)
	}
}
