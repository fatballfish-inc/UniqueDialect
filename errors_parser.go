package uniquedialect

import "fmt"

// ParserAdaptationError indicates the SQL was parsed successfully but is not yet adapted by the translation pipeline.
type ParserAdaptationError struct {
	SourceDialect string
	StatementKind string
	NativeNodeType string
	Status        string
}

func (e *ParserAdaptationError) Error() string {
	return fmt.Sprintf(
		"parser adaptation error: dialect=%s kind=%s node=%s status=%s",
		e.SourceDialect,
		e.StatementKind,
		e.NativeNodeType,
		e.Status,
	)
}
