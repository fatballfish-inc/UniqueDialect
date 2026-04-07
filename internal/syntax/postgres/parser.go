package postgres

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatballfish-inc/UniqueDialect/internal/syntax"
)

// Parse parses a minimal PostgreSQL syntax subset into syntax nodes.
func Parse(sql string) (syntax.Statement, error) {
	sql = strings.TrimSpace(sql)
	trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
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
	fromIdx := syntax.FindTopLevelKeyword(body, " FROM ")
	if fromIdx < 0 {
		return syntax.RawStatement{SQL: sql}, nil
	}

	stmt := syntax.SelectStatement{
		Columns: syntax.SplitTopLevelCSV(body[:fromIdx]),
	}
	afterFrom := body[fromIdx+len(" FROM "):]
	whereIdx := syntax.FindTopLevelKeyword(afterFrom, " WHERE ")
	orderIdx := syntax.FindTopLevelKeyword(afterFrom, " ORDER BY ")
	limitIdx := syntax.FindTopLevelKeyword(afterFrom, " LIMIT ")

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
		stmt.OrderBy = syntax.SplitTopLevelCSV(afterFrom[orderIdx+len(" ORDER BY ") : orderEnd])
	}
	if limitIdx >= 0 {
		limitClause, err := parsePostgresLimit(afterFrom[limitIdx+len(" LIMIT "):])
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
	closeCols := syntax.FindMatchingParen(body, openCols)
	if closeCols < 0 {
		return nil, fmt.Errorf("invalid insert columns")
	}

	stmt := syntax.InsertStatement{
		Table:   strings.TrimSpace(body[:openCols]),
		Columns: normalizeIdentifiers(syntax.SplitTopLevelCSV(body[openCols+1 : closeCols])),
	}
	afterCols := strings.TrimSpace(body[closeCols+1:])
	valuesIdx := syntax.FindTopLevelKeyword(strings.ToUpper(afterCols), "VALUES")
	if valuesIdx < 0 {
		return syntax.RawStatement{SQL: sql}, nil
	}
	valuesBody := strings.TrimSpace(afterCols[valuesIdx+len("VALUES"):])
	openVals := strings.Index(valuesBody, "(")
	if openVals < 0 {
		return nil, fmt.Errorf("invalid insert values")
	}
	closeVals := syntax.FindMatchingParen(valuesBody, openVals)
	if closeVals < 0 {
		return nil, fmt.Errorf("invalid insert values")
	}
	stmt.Values = syntax.SplitTopLevelCSV(valuesBody[openVals+1 : closeVals])

	rest := strings.TrimSpace(valuesBody[closeVals+1:])
	if rest == "" {
		return stmt, nil
	}

	conflictPrefix := "ON CONFLICT "
	if strings.HasPrefix(strings.ToUpper(rest), conflictPrefix) {
		conflictBody := strings.TrimSpace(rest[len(conflictPrefix):])
		openTarget := strings.Index(conflictBody, "(")
		closeTarget := syntax.FindMatchingParen(conflictBody, openTarget)
		if openTarget < 0 || closeTarget < 0 {
			return nil, fmt.Errorf("invalid conflict target")
		}
		targetColumns := normalizeIdentifiers(syntax.SplitTopLevelCSV(conflictBody[openTarget+1 : closeTarget]))
		afterTarget := strings.TrimSpace(conflictBody[closeTarget+1:])
		updatePrefix := "DO UPDATE SET "
		if !strings.HasPrefix(strings.ToUpper(afterTarget), updatePrefix) {
			return nil, fmt.Errorf("unsupported conflict action")
		}
		assignments, err := parseAssignments(afterTarget[len(updatePrefix):])
		if err != nil {
			return nil, err
		}
		stmt.Conflict = &syntax.ConflictClause{
			Style:         syntax.ConflictStylePostgres,
			TargetColumns: targetColumns,
			Assignments:   assignments,
		}
	}
	return stmt, nil
}

func parseUpdate(sql string) (syntax.Statement, error) {
	body := strings.TrimSpace(sql[len("UPDATE "):])
	setIdx := syntax.FindTopLevelKeyword(body, " SET ")
	if setIdx < 0 {
		return syntax.RawStatement{SQL: sql}, nil
	}
	stmt := syntax.UpdateStatement{
		Table: strings.TrimSpace(body[:setIdx]),
	}
	afterSet := body[setIdx+len(" SET "):]
	whereIdx := syntax.FindTopLevelKeyword(afterSet, " WHERE ")
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
	whereIdx := syntax.FindTopLevelKeyword(body, " WHERE ")
	stmt := syntax.DeleteStatement{
		Table: strings.TrimSpace(body),
	}
	if whereIdx >= 0 {
		stmt.Table = strings.TrimSpace(body[:whereIdx])
		stmt.Where = strings.TrimSpace(body[whereIdx+len(" WHERE "):])
	}
	return stmt, nil
}

func parsePostgresLimit(input string) (*syntax.LimitClause, error) {
	upper := strings.ToUpper(strings.TrimSpace(input))
	offsetIdx := strings.Index(upper, " OFFSET ")
	if offsetIdx < 0 {
		count, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			return nil, err
		}
		return &syntax.LimitClause{Count: count}, nil
	}
	count, err := strconv.Atoi(strings.TrimSpace(input[:offsetIdx]))
	if err != nil {
		return nil, err
	}
	offset, err := strconv.Atoi(strings.TrimSpace(input[offsetIdx+len(" OFFSET "):]))
	if err != nil {
		return nil, err
	}
	return &syntax.LimitClause{Offset: &offset, Count: count}, nil
}

func parseFromAndJoins(input string) (string, []syntax.Join) {
	base := strings.TrimSpace(input)
	index, keyword := findJoin(base)
	if index < 0 {
		return base, nil
	}
	baseTable := strings.TrimSpace(base[:index])
	var joins []syntax.Join
	remainder := strings.TrimSpace(base[index:])
	for remainder != "" {
		_, keyword = findJoin(" " + remainder)
		keyword = strings.TrimSpace(keyword)
		remainder = strings.TrimSpace(strings.TrimPrefix(remainder, keyword))
		onIdx := syntax.FindTopLevelKeyword(remainder, " ON ")
		if onIdx < 0 {
			break
		}
		table := strings.TrimSpace(remainder[:onIdx])
		afterOn := strings.TrimSpace(remainder[onIdx+len(" ON "):])
		nextIdx, nextKeyword := findJoin(" " + afterOn)
		if nextIdx < 0 {
			joins = append(joins, syntax.Join{Kind: keyword, Table: table, On: strings.TrimSpace(afterOn)})
			break
		}
		onExpr := strings.TrimSpace(afterOn[:nextIdx])
		joins = append(joins, syntax.Join{Kind: keyword, Table: table, On: onExpr})
		remainder = strings.TrimSpace(afterOn[nextIdx:])
		_ = nextKeyword
	}
	return baseTable, joins
}

func findJoin(input string) (int, string) {
	keywords := []string{" LEFT JOIN ", " INNER JOIN ", " JOIN "}
	best := -1
	bestKeyword := ""
	for _, keyword := range keywords {
		index := syntax.FindTopLevelKeyword(input, keyword)
		if index >= 0 && (best < 0 || index < best) {
			best = index
			bestKeyword = keyword
		}
	}
	return best, bestKeyword
}

func parseAssignments(input string) ([]syntax.Assignment, error) {
	parts := syntax.SplitTopLevelCSV(input)
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
		out = append(out, syntax.StripIdentifier(value))
	}
	return out
}
