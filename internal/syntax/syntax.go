package syntax

import (
	"strings"
)

// Statement is the syntax-layer statement node.
type Statement interface {
	statementNode()
}

// RawStatement preserves SQL that is not yet structurally parsed.
type RawStatement struct {
	SQL string
}

func (RawStatement) statementNode() {}

// SelectStatement is a minimal SELECT AST.
type SelectStatement struct {
	Columns []string
	From    string
	Joins   []Join
	Where   string
	OrderBy []string
	Limit   *LimitClause
}

func (SelectStatement) statementNode() {}

// InsertStatement is a minimal INSERT AST.
type InsertStatement struct {
	Table    string
	Columns  []string
	Values   []string
	Conflict *ConflictClause
}

func (InsertStatement) statementNode() {}

// UpdateStatement is a minimal UPDATE AST.
type UpdateStatement struct {
	Table       string
	Assignments []Assignment
	Where       string
}

func (UpdateStatement) statementNode() {}

// DeleteStatement is a minimal DELETE AST.
type DeleteStatement struct {
	Table string
	Where string
}

func (DeleteStatement) statementNode() {}

// Join is a parsed join clause.
type Join struct {
	Kind  string
	Table string
	On    string
}

// LimitClause is a parsed LIMIT/OFFSET pair.
type LimitClause struct {
	Offset *int
	Count  int
}

// Assignment is a parsed assignment pair.
type Assignment struct {
	Column string
	Value  string
}

// ConflictStyle identifies the source upsert form.
type ConflictStyle string

const (
	// ConflictStyleMySQL represents ON DUPLICATE KEY UPDATE.
	ConflictStyleMySQL ConflictStyle = "mysql"
	// ConflictStylePostgres represents ON CONFLICT ... DO UPDATE.
	ConflictStylePostgres ConflictStyle = "postgres"
)

// ConflictClause is a parsed upsert clause.
type ConflictClause struct {
	Style         ConflictStyle
	TargetColumns []string
	Assignments   []Assignment
}

func trimStatement(sql string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
}

func splitTopLevelCSV(input string) []string {
	var (
		parts    []string
		start    int
		depth    int
		inSingle bool
		inDouble bool
		inBack   bool
	)

	for index := 0; index < len(input); index++ {
		switch input[index] {
		case '\'':
			if !inDouble && !inBack {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBack {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBack = !inBack
			}
		case '(':
			if !inSingle && !inDouble && !inBack {
				depth++
			}
		case ')':
			if !inSingle && !inDouble && !inBack && depth > 0 {
				depth--
			}
		case ',':
			if !inSingle && !inDouble && !inBack && depth == 0 {
				parts = append(parts, strings.TrimSpace(input[start:index]))
				start = index + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(input[start:]))
	return parts
}

func findTopLevelKeyword(input, keyword string) int {
	upperInput := strings.ToUpper(input)
	upperKeyword := strings.ToUpper(keyword)

	var (
		depth    int
		inSingle bool
		inDouble bool
		inBack   bool
	)

	for index := 0; index <= len(input)-len(keyword); index++ {
		switch input[index] {
		case '\'':
			if !inDouble && !inBack {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBack {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBack = !inBack
			}
		case '(':
			if !inSingle && !inDouble && !inBack {
				depth++
			}
		case ')':
			if !inSingle && !inDouble && !inBack && depth > 0 {
				depth--
			}
		}

		if inSingle || inDouble || inBack || depth != 0 {
			continue
		}
		if upperInput[index:index+len(keyword)] == upperKeyword {
			return index
		}
	}
	return -1
}

func findMatchingParen(input string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(input) || input[openIndex] != '(' {
		return -1
	}

	var (
		depth    int
		inSingle bool
		inDouble bool
		inBack   bool
	)

	for index := openIndex; index < len(input); index++ {
		switch input[index] {
		case '\'':
			if !inDouble && !inBack {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBack {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBack = !inBack
			}
		case '(':
			if !inSingle && !inDouble && !inBack {
				depth++
			}
		case ')':
			if !inSingle && !inDouble && !inBack {
				depth--
				if depth == 0 {
					return index
				}
			}
		}
	}
	return -1
}

func stripIdentifier(input string) string {
	value := strings.TrimSpace(input)
	value = strings.TrimPrefix(value, `"`)
	value = strings.TrimSuffix(value, `"`)
	value = strings.TrimPrefix(value, "`")
	value = strings.TrimSuffix(value, "`")
	return value
}

// TrimStatement removes surrounding whitespace and a trailing semicolon.
func TrimStatement(sql string) string {
	return trimStatement(sql)
}

// SplitTopLevelCSV splits a comma-separated string while ignoring nested commas.
func SplitTopLevelCSV(input string) []string {
	return splitTopLevelCSV(input)
}

// FindTopLevelKeyword finds a keyword outside nested structures.
func FindTopLevelKeyword(input, keyword string) int {
	return findTopLevelKeyword(input, keyword)
}

// FindMatchingParen finds the matching closing paren for an opening paren index.
func FindMatchingParen(input string, openIndex int) int {
	return findMatchingParen(input, openIndex)
}

// StripIdentifier removes surrounding quote markers from a simple identifier.
func StripIdentifier(input string) string {
	return stripIdentifier(input)
}
