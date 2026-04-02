package mysql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatballfish/uniquedialect/internal/syntax"
)

// Parse parses a minimal MySQL syntax subset into syntax nodes.
func Parse(sql string) (syntax.Statement, error) {
	sql = strings.TrimSpace(sql)
	trimmed := syntaxTrim(sql)
	switch {
	case strings.HasPrefix(strings.ToUpper(trimmed), "SELECT "):
		return parseSelect(trimmed)
	case strings.HasPrefix(strings.ToUpper(trimmed), "INSERT INTO "):
		return parseInsert(trimmed)
	case strings.HasPrefix(strings.ToUpper(trimmed), "UPDATE "):
		return parseUpdate(trimmed)
	case strings.HasPrefix(strings.ToUpper(trimmed), "DELETE FROM "):
		return parseDelete(trimmed)
	default:
		return syntax.RawStatement{SQL: sql}, nil
	}
}

func parseSelect(sql string) (syntax.Statement, error) {
	body := strings.TrimSpace(sql[len("SELECT "):])
	fromIdx := syntaxFind(body, " FROM ")
	if fromIdx < 0 {
		return syntax.RawStatement{SQL: sql}, nil
	}

	stmt := syntax.SelectStatement{
		Columns: syntaxSplitCSV(body[:fromIdx]),
	}
	afterFrom := body[fromIdx+len(" FROM "):]
	whereIdx := syntaxFind(afterFrom, " WHERE ")
	orderIdx := syntaxFind(afterFrom, " ORDER BY ")
	limitIdx := syntaxFind(afterFrom, " LIMIT ")

	fromEnd := earliestPositive(whereIdx, orderIdx, limitIdx)
	if fromEnd < 0 {
		fromEnd = len(afterFrom)
	}
	fromPart := strings.TrimSpace(afterFrom[:fromEnd])
	stmt.From, stmt.Joins = parseFromAndJoins(fromPart)

	if whereIdx >= 0 {
		whereEnd := len(afterFrom)
		for _, idx := range []int{orderIdx, limitIdx} {
			if idx > whereIdx && idx < whereEnd {
				whereEnd = idx
			}
		}
		stmt.Where = strings.TrimSpace(afterFrom[whereIdx+len(" WHERE ") : whereEnd])
	}
	if orderIdx >= 0 {
		orderEnd := len(afterFrom)
		if limitIdx > orderIdx {
			orderEnd = limitIdx
		}
		stmt.OrderBy = syntaxSplitCSV(afterFrom[orderIdx+len(" ORDER BY ") : orderEnd])
	}
	if limitIdx >= 0 {
		limitClause, err := parseMySQLLimit(afterFrom[limitIdx+len(" LIMIT "):])
		if err != nil {
			return nil, err
		}
		stmt.Limit = limitClause
	}
	return stmt, nil
}

func parseInsert(sql string) (syntax.Statement, error) {
	body := strings.TrimSpace(sql[len("INSERT INTO "):])
	openCols := strings.Index(body, "(")
	if openCols < 0 {
		return syntax.RawStatement{SQL: sql}, nil
	}
	closeCols := syntaxMatchParen(body, openCols)
	if closeCols < 0 {
		return nil, fmt.Errorf("invalid insert columns")
	}

	stmt := syntax.InsertStatement{
		Table:   strings.TrimSpace(body[:openCols]),
		Columns: normalizeIdentifiers(syntaxSplitCSV(body[openCols+1 : closeCols])),
	}
	afterCols := strings.TrimSpace(body[closeCols+1:])
	valuesIdx := syntaxFind(afterCols, "VALUES")
	if valuesIdx < 0 {
		return syntax.RawStatement{SQL: sql}, nil
	}
	valuesBody := strings.TrimSpace(afterCols[valuesIdx+len("VALUES"):])
	openVals := strings.Index(valuesBody, "(")
	if openVals < 0 {
		return nil, fmt.Errorf("invalid insert values")
	}
	closeVals := syntaxMatchParen(valuesBody, openVals)
	if closeVals < 0 {
		return nil, fmt.Errorf("invalid insert values")
	}
	stmt.Values = syntaxSplitCSV(valuesBody[openVals+1 : closeVals])

	rest := strings.TrimSpace(valuesBody[closeVals+1:])
	if rest == "" {
		return stmt, nil
	}

	upsertPrefix := "ON DUPLICATE KEY UPDATE "
	if strings.HasPrefix(strings.ToUpper(rest), upsertPrefix) {
		assignments, err := parseAssignments(rest[len(upsertPrefix):])
		if err != nil {
			return nil, err
		}
		targetColumns := []string{}
		if len(stmt.Columns) > 0 {
			targetColumns = append(targetColumns, stmt.Columns[0])
		}
		stmt.Conflict = &syntax.ConflictClause{
			Style:         syntax.ConflictStyleMySQL,
			TargetColumns: targetColumns,
			Assignments:   assignments,
		}
	}
	return stmt, nil
}

func parseUpdate(sql string) (syntax.Statement, error) {
	body := strings.TrimSpace(sql[len("UPDATE "):])
	setIdx := syntaxFind(body, " SET ")
	if setIdx < 0 {
		return syntax.RawStatement{SQL: sql}, nil
	}

	stmt := syntax.UpdateStatement{
		Table: strings.TrimSpace(body[:setIdx]),
	}
	afterSet := body[setIdx+len(" SET "):]
	whereIdx := syntaxFind(afterSet, " WHERE ")

	assignPart := afterSet
	if whereIdx >= 0 {
		assignPart = afterSet[:whereIdx]
		stmt.Where = strings.TrimSpace(afterSet[whereIdx+len(" WHERE "):])
	}
	assignments, err := parseAssignments(assignPart)
	if err != nil {
		return nil, err
	}
	stmt.Assignments = assignments
	return stmt, nil
}

func parseDelete(sql string) (syntax.Statement, error) {
	body := strings.TrimSpace(sql[len("DELETE FROM "):])
	whereIdx := syntaxFind(body, " WHERE ")
	stmt := syntax.DeleteStatement{
		Table: strings.TrimSpace(body),
	}
	if whereIdx >= 0 {
		stmt.Table = strings.TrimSpace(body[:whereIdx])
		stmt.Where = strings.TrimSpace(body[whereIdx+len(" WHERE "):])
	}
	return stmt, nil
}

func parseMySQLLimit(input string) (*syntax.LimitClause, error) {
	parts := syntaxSplitCSV(strings.TrimSpace(input))
	if len(parts) == 1 {
		count, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		return &syntax.LimitClause{Count: count}, nil
	}
	if len(parts) == 2 {
		offset, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		return &syntax.LimitClause{Offset: &offset, Count: count}, nil
	}
	return nil, fmt.Errorf("invalid mysql limit clause")
}

func parseFromAndJoins(input string) (string, []syntax.Join) {
	base := strings.TrimSpace(input)
	var joins []syntax.Join
	for {
		index, keyword := findJoin(base)
		if index < 0 {
			if len(joins) == 0 {
				return strings.TrimSpace(base), nil
			}
			return strings.TrimSpace(base), joins
		}
		baseTable := strings.TrimSpace(base[:index])
		remainder := strings.TrimSpace(base[index+len(keyword):])
		joinTable, joinOn, tail := splitJoinRemainder(remainder)
		joins = append(joins, syntax.Join{
			Kind:  strings.TrimSpace(keyword),
			Table: strings.TrimSpace(joinTable),
			On:    strings.TrimSpace(joinOn),
		})
		base = strings.TrimSpace(tail)
		if baseTable != "" {
			return baseTable, appendJoins(base, joins)
		}
	}
}

func appendJoins(remaining string, joins []syntax.Join) []syntax.Join {
	if strings.TrimSpace(remaining) == "" {
		return joins
	}
	index, keyword := findJoin(remaining)
	if index < 0 {
		return joins
	}
	remainder := strings.TrimSpace(remaining[index+len(keyword):])
	joinTable, joinOn, tail := splitJoinRemainder(remainder)
	joins = append(joins, syntax.Join{
		Kind:  strings.TrimSpace(keyword),
		Table: strings.TrimSpace(joinTable),
		On:    strings.TrimSpace(joinOn),
	})
	return appendJoins(strings.TrimSpace(tail), joins)
}

func splitJoinRemainder(input string) (string, string, string) {
	onIdx := syntaxFind(input, " ON ")
	if onIdx < 0 {
		return input, "", ""
	}
	table := strings.TrimSpace(input[:onIdx])
	afterOn := strings.TrimSpace(input[onIdx+len(" ON "):])
	nextIdx, _ := findJoin(afterOn)
	if nextIdx < 0 {
		return table, afterOn, ""
	}
	return table, strings.TrimSpace(afterOn[:nextIdx]), strings.TrimSpace(afterOn[nextIdx:])
}

func findJoin(input string) (int, string) {
	keywords := []string{" LEFT JOIN ", " INNER JOIN ", " JOIN "}
	best := -1
	bestKeyword := ""
	for _, keyword := range keywords {
		index := syntaxFind(input, keyword)
		if index >= 0 && (best < 0 || index < best) {
			best = index
			bestKeyword = keyword
		}
	}
	return best, bestKeyword
}

func parseAssignments(input string) ([]syntax.Assignment, error) {
	parts := syntaxSplitCSV(input)
	assignments := make([]syntax.Assignment, 0, len(parts))
	for _, part := range parts {
		pieces := strings.SplitN(part, "=", 2)
		if len(pieces) != 2 {
			return nil, fmt.Errorf("invalid assignment %q", part)
		}
		assignments = append(assignments, syntax.Assignment{
			Column: strings.TrimSpace(pieces[0]),
			Value:  strings.TrimSpace(pieces[1]),
		})
	}
	return assignments, nil
}

func earliestPositive(values ...int) int {
	best := -1
	for _, value := range values {
		if value >= 0 && (best < 0 || value < best) {
			best = value
		}
	}
	return best
}

func normalizeIdentifiers(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, syntaxStrip(value))
	}
	return out
}

func syntaxTrim(sql string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
}
func syntaxFind(input, keyword string) int        { return syntax.FindTopLevelKeyword(input, keyword) }
func syntaxSplitCSV(input string) []string        { return syntax.SplitTopLevelCSV(input) }
func syntaxMatchParen(input string, open int) int { return syntax.FindMatchingParen(input, open) }
func syntaxStrip(input string) string             { return syntax.StripIdentifier(input) }
