package uniquedialect

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	internalparser "github.com/fatballfish/uniquedialect/internal/parser"
	tidbast "github.com/fatballfish/uniquedialect/internal/parser/tidb/ast"
	tidbformat "github.com/fatballfish/uniquedialect/internal/parser/tidb/format"
)

// BootstrapKind identifies the artifact type.
type BootstrapKind string

const (
	// BootstrapKindFunction is a database function shim.
	BootstrapKindFunction BootstrapKind = "function"
	// BootstrapKindTrigger is a database trigger shim.
	BootstrapKindTrigger BootstrapKind = "trigger"
)

// BootstrapArtifact is an executable compatibility artifact.
type BootstrapArtifact struct {
	Name       string
	Kind       BootstrapKind
	SQL        string
	Idempotent bool
}

// BootstrapOptions configures bootstrap planning.
type BootstrapOptions struct {
	InputDialect  Dialect
	TargetDialect Dialect
	Enabled       bool
}

// BootstrapPlanner plans compatibility artifacts for translated SQL and DDL.
type BootstrapPlanner struct {
	opts BootstrapOptions
}

var (
	createTablePattern       = regexp.MustCompile("(?i)create\\s+table\\s+[`\"]?([a-zA-Z0-9_]+)[`\"]?")
	onUpdateTimestampPattern = regexp.MustCompile("(?i)[`\"]?([a-zA-Z0-9_]+)[`\"]?\\s+timestamp(?:[^,]*)on\\s+update\\s+current_timestamp")
)

// NewBootstrapPlanner creates a compatibility artifact planner.
func NewBootstrapPlanner(opts BootstrapOptions) *BootstrapPlanner {
	return &BootstrapPlanner{opts: opts}
}

// Plan analyzes SQL/DDL and returns required bootstrap artifacts.
func (p *BootstrapPlanner) Plan(_ context.Context, sql string) ([]BootstrapArtifact, error) {
	if !p.opts.Enabled {
		return nil, nil
	}
	switch {
	case p.opts.InputDialect == DialectMySQL && p.opts.TargetDialect == DialectPostgres:
		return p.planMySQLToPostgres(sql)
	default:
		return nil, nil
	}
}

func (p *BootstrapPlanner) planMySQLToPostgres(sql string) ([]BootstrapArtifact, error) {
	artifacts, matched, err := p.planMySQLToPostgresFromAST(sql)
	if err != nil {
		return nil, err
	}
	if matched {
		return artifacts, nil
	}

	tableMatches := createTablePattern.FindStringSubmatch(sql)
	if len(tableMatches) != 2 {
		return nil, nil
	}
	columnMatches := onUpdateTimestampPattern.FindAllStringSubmatch(sql, -1)
	if len(columnMatches) == 0 {
		return nil, nil
	}

	tableName := strings.ToLower(tableMatches[1])
	columns := make([]string, 0, len(columnMatches))
	for _, match := range columnMatches {
		if len(match) != 2 {
			continue
		}
		columns = append(columns, strings.ToLower(match[1]))
	}
	return buildOnUpdateArtifacts(tableName, columns), nil
}

func (p *BootstrapPlanner) planMySQLToPostgresFromAST(sql string) ([]BootstrapArtifact, bool, error) {
	parsed, err := internalparser.ParseOne(sql, p.opts.InputDialect)
	if err != nil {
		return nil, false, nil
	}

	createStmt, ok := parsed.NativeAST.(*tidbast.CreateTableStmt)
	if !ok || createStmt == nil || createStmt.Table == nil {
		return nil, false, nil
	}

	tableName := strings.ToLower(createStmt.Table.Name.O)
	if tableName == "" {
		return nil, true, nil
	}

	var columns []string
	for _, column := range createStmt.Cols {
		if column == nil || column.Name == nil {
			continue
		}
		for _, option := range column.Options {
			if option == nil || option.Tp != tidbast.ColumnOptionOnUpdate {
				continue
			}
			if !isCurrentTimestampExpr(option.Expr) {
				continue
			}
			columns = append(columns, strings.ToLower(column.Name.Name.O))
			break
		}
	}

	if len(columns) == 0 {
		return nil, true, nil
	}
	return buildOnUpdateArtifacts(tableName, columns), true, nil
}

func isCurrentTimestampExpr(expr tidbast.ExprNode) bool {
	if expr == nil {
		return false
	}
	switch node := expr.(type) {
	case *tidbast.FuncCallExpr:
		return strings.EqualFold(node.FnName.O, "CURRENT_TIMESTAMP")
	default:
		rendered := strings.TrimSpace(restoreTiDBExpr(expr))
		return strings.EqualFold(rendered, "CURRENT_TIMESTAMP")
	}
}

func restoreTiDBExpr(expr tidbast.Node) string {
	var builder strings.Builder
	ctx := tidbformat.NewRestoreCtx(tidbformat.DefaultRestoreFlags, &builder)
	if err := expr.Restore(ctx); err != nil {
		return ""
	}
	return builder.String()
}

func buildOnUpdateArtifacts(tableName string, columns []string) []BootstrapArtifact {
	artifacts := make([]BootstrapArtifact, 0, len(columns)*2)
	for _, columnName := range columns {
		functionName := fmt.Sprintf("ud_bootstrap_%s_%s_on_update", tableName, columnName)
		triggerName := fmt.Sprintf("ud_bootstrap_%s_%s_on_update_trg", tableName, columnName)

		functionSQL := fmt.Sprintf(`CREATE OR REPLACE FUNCTION %s()
RETURNS TRIGGER AS $$
BEGIN
    NEW.%s = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql`, functionName, columnName)

		triggerSQL := fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s;
CREATE TRIGGER %s
BEFORE UPDATE ON %s
FOR EACH ROW
EXECUTE FUNCTION %s()`, triggerName, tableName, triggerName, tableName, functionName)

		artifacts = append(artifacts,
			BootstrapArtifact{
				Name:       functionName,
				Kind:       BootstrapKindFunction,
				SQL:        functionSQL,
				Idempotent: true,
			},
			BootstrapArtifact{
				Name:       triggerName,
				Kind:       BootstrapKindTrigger,
				SQL:        triggerSQL,
				Idempotent: true,
			},
		)
	}
	return artifacts
}
