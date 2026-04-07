package normalize

import (
	"strings"

	"github.com/fatballfish-inc/UniqueDialect/internal/ir"
	"github.com/fatballfish-inc/UniqueDialect/internal/syntax"
)

// Statement converts syntax-layer statements into normalized IR.
func Statement(stmt syntax.Statement) ir.Statement {
	switch value := stmt.(type) {
	case syntax.RawStatement:
		return ir.RawStatement{SQL: value.SQL}
	case syntax.SelectStatement:
		joins := make([]ir.Join, 0, len(value.Joins))
		for _, join := range value.Joins {
			joins = append(joins, ir.Join{
				Kind:  strings.TrimSpace(join.Kind),
				Table: strings.TrimSpace(join.Table),
				On:    strings.TrimSpace(join.On),
			})
		}
		var limit *ir.LimitClause
		if value.Limit != nil {
			limit = &ir.LimitClause{Offset: value.Limit.Offset, Count: value.Limit.Count}
		}
		return ir.SelectStatement{
			Columns: cloneStrings(value.Columns),
			From:    strings.TrimSpace(value.From),
			Joins:   joins,
			Where:   strings.TrimSpace(value.Where),
			OrderBy: cloneStrings(value.OrderBy),
			Limit:   limit,
		}
	case syntax.InsertStatement:
		var conflict *ir.ConflictClause
		if value.Conflict != nil {
			assignments := make([]ir.Assignment, 0, len(value.Conflict.Assignments))
			for _, assignment := range value.Conflict.Assignments {
				assignments = append(assignments, ir.Assignment{
					Column: stripIdentifier(assignment.Column),
					Value:  strings.TrimSpace(assignment.Value),
				})
			}
			style := ir.ConflictStyle(value.Conflict.Style)
			conflict = &ir.ConflictClause{
				Style:         style,
				TargetColumns: cloneStrings(value.Conflict.TargetColumns),
				Assignments:   assignments,
			}
		}
		return ir.InsertStatement{
			Table:    stripIdentifier(value.Table),
			Columns:  cloneStrings(value.Columns),
			Values:   cloneStrings(value.Values),
			Conflict: conflict,
		}
	case syntax.UpdateStatement:
		assignments := make([]ir.Assignment, 0, len(value.Assignments))
		for _, assignment := range value.Assignments {
			assignments = append(assignments, ir.Assignment{
				Column: stripIdentifier(assignment.Column),
				Value:  strings.TrimSpace(assignment.Value),
			})
		}
		return ir.UpdateStatement{
			Table:       stripIdentifier(value.Table),
			Assignments: assignments,
			Where:       strings.TrimSpace(value.Where),
		}
	case syntax.DeleteStatement:
		return ir.DeleteStatement{
			Table: stripIdentifier(value.Table),
			Where: strings.TrimSpace(value.Where),
		}
	default:
		return ir.RawStatement{}
	}
}

func stripIdentifier(input string) string {
	value := strings.TrimSpace(input)
	parts := strings.Split(value, ".")
	for index := range parts {
		parts[index] = strings.Trim(parts[index], "`\" ")
	}
	return strings.Join(parts, ".")
}

func cloneStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}
