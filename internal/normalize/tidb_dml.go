package normalize

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatballfish-inc/UniqueDialect/internal/ir"
	tidbast "github.com/fatballfish-inc/UniqueDialect/internal/parser/tidb/ast"
	tidbformat "github.com/fatballfish-inc/UniqueDialect/internal/parser/tidb/format"
	"github.com/fatballfish-inc/UniqueDialect/internal/syntax"
)

func normalizeTiDBSelect(sql string, stmt *tidbast.SelectStmt) (ir.Statement, error) {
	selectParts, err := extractSelectParts(sql)
	if err != nil {
		return nil, err
	}

	var limit *ir.LimitClause
	if stmt.Limit != nil {
		count, err := parseTiDBIntExpr(stmt.Limit.Count)
		if err != nil {
			return nil, err
		}
		limit = &ir.LimitClause{Count: count}
		if stmt.Limit.Offset != nil {
			offset, err := parseTiDBIntExpr(stmt.Limit.Offset)
			if err != nil {
				return nil, err
			}
			limit.Offset = &offset
		}
	}

	return ir.SelectStatement{
		Columns: selectParts.Columns,
		From:    selectParts.From,
		Where:   selectParts.Where,
		OrderBy: selectParts.OrderBy,
		Limit:   limit,
	}, nil
}

func normalizeTiDBInsert(sql string, stmt *tidbast.InsertStmt) (ir.Statement, error) {
	insertParts, err := extractInsertParts(sql)
	if err != nil {
		return nil, err
	}

	var conflict *ir.ConflictClause
	if len(stmt.OnDuplicate) > 0 {
		assignments := make([]ir.Assignment, 0, len(stmt.OnDuplicate))
		for _, assignment := range stmt.OnDuplicate {
			assignments = append(assignments, ir.Assignment{
				Column: stripIdentifier(tiDBNodeText(assignment.Column)),
				Value:  normalizeWhitespace(tiDBNodeText(assignment.Expr)),
			})
		}
		targetColumns := []string{}
		if len(insertParts.Columns) > 0 {
			targetColumns = append(targetColumns, insertParts.Columns[0])
		}
		conflict = &ir.ConflictClause{
			Style:         ir.ConflictStyleMySQL,
			TargetColumns: targetColumns,
			Assignments:   assignments,
		}
	}

	return ir.InsertStatement{
		Table:    insertParts.Table,
		Columns:  insertParts.Columns,
		Values:   insertParts.Values,
		Conflict: conflict,
	}, nil
}

func normalizeTiDBUpdate(sql string, stmt *tidbast.UpdateStmt) (ir.Statement, error) {
	updateParts, err := extractUpdateParts(sql)
	if err != nil {
		return nil, err
	}

	return ir.UpdateStatement{
		Table:       updateParts.Table,
		Assignments: updateParts.Assignments,
		Where:       updateParts.Where,
	}, nil
}

func normalizeTiDBDelete(sql string, stmt *tidbast.DeleteStmt) (ir.Statement, error) {
	deleteParts, err := extractDeleteParts(sql)
	if err != nil {
		return nil, err
	}

	return ir.DeleteStatement{
		Table: deleteParts.Table,
		Where: deleteParts.Where,
	}, nil
}

func restoreTiDBNode(node tidbast.Node) string {
	var builder strings.Builder
	ctx := tidbformat.NewRestoreCtx(tidbformat.DefaultRestoreFlags, &builder)
	if err := node.Restore(ctx); err != nil {
		return ""
	}
	return builder.String()
}

func tiDBNodeText(node tidbast.Node) string {
	if original := strings.TrimSpace(node.OriginalText()); original != "" {
		return original
	}
	return restoreTiDBNode(node)
}

func parseTiDBIntExpr(expr tidbast.ExprNode) (int, error) {
	value := strings.TrimSpace(restoreTiDBNode(expr))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse limit integer %q: %w", value, err)
	}
	return parsed, nil
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeTiDBRawComposable(sql string) (ir.Statement, error) {
	return ir.RawStatement{SQL: syntax.TrimStatement(sql)}, nil
}

func normalizeTiDBRawDDL(sql string) (ir.Statement, error) {
	return ir.RawStatement{SQL: syntax.TrimStatement(sql)}, nil
}

type selectParts struct {
	Columns []string
	From    string
	Where   string
	OrderBy []string
}

func extractSelectParts(sql string) (selectParts, error) {
	trimmed := syntax.TrimStatement(sql)
	body := strings.TrimSpace(trimmed[len("SELECT "):])
	fromIdx := syntax.FindTopLevelKeyword(body, " FROM ")
	if fromIdx < 0 {
		return selectParts{}, fmt.Errorf("missing FROM clause")
	}

	parts := selectParts{
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
	parts.From = strings.TrimSpace(afterFrom[:fromEnd])

	if whereIdx >= 0 {
		whereEnd := len(afterFrom)
		for _, idx := range []int{orderIdx, limitIdx} {
			if idx > whereIdx && idx < whereEnd {
				whereEnd = idx
			}
		}
		parts.Where = strings.TrimSpace(afterFrom[whereIdx+len(" WHERE ") : whereEnd])
	}
	if orderIdx >= 0 {
		orderEnd := len(afterFrom)
		if limitIdx > orderIdx {
			orderEnd = limitIdx
		}
		parts.OrderBy = syntax.SplitTopLevelCSV(afterFrom[orderIdx+len(" ORDER BY ") : orderEnd])
	}
	return parts, nil
}

type insertParts struct {
	Table   string
	Columns []string
	Values  []string
}

func extractInsertParts(sql string) (insertParts, error) {
	trimmed := syntax.TrimStatement(sql)
	body := strings.TrimSpace(trimmed[len("INSERT INTO "):])
	openCols := strings.Index(body, "(")
	if openCols < 0 {
		return insertParts{}, fmt.Errorf("invalid insert columns")
	}
	closeCols := syntax.FindMatchingParen(body, openCols)
	if closeCols < 0 {
		return insertParts{}, fmt.Errorf("invalid insert columns")
	}
	parts := insertParts{
		Table:   stripIdentifier(strings.TrimSpace(body[:openCols])),
		Columns: normalizeIdentifiers(syntax.SplitTopLevelCSV(body[openCols+1 : closeCols])),
	}
	afterCols := strings.TrimSpace(body[closeCols+1:])
	valuesIdx := syntax.FindTopLevelKeyword(strings.ToUpper(afterCols), "VALUES")
	if valuesIdx < 0 {
		return parts, nil
	}
	valuesBody := strings.TrimSpace(afterCols[valuesIdx+len("VALUES"):])
	openVals := strings.Index(valuesBody, "(")
	if openVals < 0 {
		return insertParts{}, fmt.Errorf("invalid insert values")
	}
	closeVals := syntax.FindMatchingParen(valuesBody, openVals)
	if closeVals < 0 {
		return insertParts{}, fmt.Errorf("invalid insert values")
	}
	parts.Values = syntax.SplitTopLevelCSV(valuesBody[openVals+1 : closeVals])
	return parts, nil
}

type updateParts struct {
	Table       string
	Assignments []ir.Assignment
	Where       string
}

func extractUpdateParts(sql string) (updateParts, error) {
	trimmed := syntax.TrimStatement(sql)
	body := strings.TrimSpace(trimmed[len("UPDATE "):])
	setIdx := syntax.FindTopLevelKeyword(body, " SET ")
	if setIdx < 0 {
		return updateParts{}, fmt.Errorf("missing SET clause")
	}
	parts := updateParts{
		Table: stripIdentifier(strings.TrimSpace(body[:setIdx])),
	}
	afterSet := body[setIdx+len(" SET "):]
	whereIdx := syntax.FindTopLevelKeyword(afterSet, " WHERE ")
	assignPart := afterSet
	if whereIdx >= 0 {
		assignPart = afterSet[:whereIdx]
		parts.Where = strings.TrimSpace(afterSet[whereIdx+len(" WHERE "):])
	}
	assignments, err := parseAssignments(assignPart)
	if err != nil {
		return updateParts{}, err
	}
	parts.Assignments = assignments
	return parts, nil
}

type deleteParts struct {
	Table string
	Where string
}

func extractDeleteParts(sql string) (deleteParts, error) {
	trimmed := syntax.TrimStatement(sql)
	body := strings.TrimSpace(trimmed[len("DELETE FROM "):])
	whereIdx := syntax.FindTopLevelKeyword(body, " WHERE ")
	parts := deleteParts{
		Table: stripIdentifier(strings.TrimSpace(body)),
	}
	if whereIdx >= 0 {
		parts.Table = stripIdentifier(strings.TrimSpace(body[:whereIdx]))
		parts.Where = strings.TrimSpace(body[whereIdx+len(" WHERE "):])
	}
	return parts, nil
}

func parseAssignments(input string) ([]ir.Assignment, error) {
	parts := syntax.SplitTopLevelCSV(input)
	assignments := make([]ir.Assignment, 0, len(parts))
	for _, part := range parts {
		pieces := strings.SplitN(part, "=", 2)
		if len(pieces) != 2 {
			return nil, fmt.Errorf("invalid assignment %q", part)
		}
		assignments = append(assignments, ir.Assignment{
			Column: stripIdentifier(strings.TrimSpace(pieces[0])),
			Value:  strings.TrimSpace(pieces[1]),
		})
	}
	return assignments, nil
}

func normalizeIdentifiers(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, stripIdentifier(value))
	}
	return out
}

func earliestPositive(values ...int) int {
	best := -1
	for _, value := range values {
		if value < 0 {
			continue
		}
		if best < 0 || value < best {
			best = value
		}
	}
	return best
}
