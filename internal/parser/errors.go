package parser

import "fmt"

// ErrUnsupportedDialect indicates there is no parser backend for the requested dialect yet.
type ErrUnsupportedDialect struct {
	Dialect string
}

func (e ErrUnsupportedDialect) Error() string {
	return fmt.Sprintf("unsupported parser dialect %q", e.Dialect)
}

// ErrExpectedSingleStatement indicates ParseOne received zero or multiple statements.
type ErrExpectedSingleStatement struct {
	Count int
}

func (e ErrExpectedSingleStatement) Error() string {
	return fmt.Sprintf("expected exactly one statement, got %d", e.Count)
}
