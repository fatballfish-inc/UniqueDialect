package normalize

import (
	"strings"

	"github.com/fatballfish-inc/UniqueDialect/internal/ir"
	tidbast "github.com/fatballfish-inc/UniqueDialect/internal/parser/tidb/ast"
)

func normalizeTiDBCreateTable(stmt *tidbast.CreateTableStmt) (ir.Statement, error) {
	if stmt == nil || stmt.Table == nil {
		return ir.RawStatement{}, nil
	}

	columns := make([]ir.CreateTableColumn, 0, len(stmt.Cols))
	for _, column := range stmt.Cols {
		if column == nil || column.Name == nil {
			continue
		}
		normalized := normalizeTiDBColumnDef(column)
		columns = append(columns, normalized)
	}

	constraints := make([]string, 0, len(stmt.Constraints))
	for _, constraint := range stmt.Constraints {
		if constraint == nil {
			continue
		}
		constraints = append(constraints, restoreTiDBNode(constraint))
	}

	return ir.CreateTableStatement{
		Name:        strings.TrimSpace(stmt.Table.Name.O),
		IfNotExists: stmt.IfNotExists,
		Columns:     columns,
		Constraints: constraints,
	}, nil
}

func normalizeTiDBAlterTable(sql string, stmt *tidbast.AlterTableStmt) (ir.Statement, error) {
	if stmt == nil || stmt.Table == nil {
		return normalizeTiDBRawDDL(sql)
	}
	tableName := strings.TrimSpace(stmt.Table.Name.O)
	specs := make([]ir.AlterTableSpec, 0, len(stmt.Specs))
	statements := make([]ir.Statement, 0, len(stmt.Specs))
	flushAlterTable := func() {
		if len(specs) == 0 {
			return
		}
		statements = append(statements, ir.AlterTableStatement{
			Name:  tableName,
			Specs: specs,
		})
		specs = nil
	}
	for _, spec := range stmt.Specs {
		if spec == nil {
			continue
		}

		switch spec.Tp {
		case tidbast.AlterTableAddColumns:
			if len(spec.NewColumns) != 1 || len(spec.NewConstraints) != 0 || spec.Position == nil || spec.Position.Tp != tidbast.ColumnPositionNone {
				return normalizeTiDBRawDDL(sql)
			}
			specs = append(specs, ir.AlterTableSpec{
				Kind:   ir.AlterTableSpecAddColumn,
				Column: normalizeTiDBColumnDef(spec.NewColumns[0]),
			})
		case tidbast.AlterTableDropColumn:
			if spec.OldColumnName == nil {
				return normalizeTiDBRawDDL(sql)
			}
			specs = append(specs, ir.AlterTableSpec{
				Kind: ir.AlterTableSpecDropColumn,
				Name: strings.TrimSpace(spec.OldColumnName.Name.O),
			})
		case tidbast.AlterTableAlterColumn:
			if len(spec.NewColumns) != 1 || spec.NewColumns[0] == nil || spec.NewColumns[0].Name == nil {
				return normalizeTiDBRawDDL(sql)
			}
			columnName := strings.TrimSpace(spec.NewColumns[0].Name.Name.O)
			if len(spec.NewColumns[0].Options) == 1 {
				specs = append(specs, ir.AlterTableSpec{
					Kind:    ir.AlterTableSpecSetDefault,
					Name:    columnName,
					Default: normalizeWhitespace(tiDBNodeText(spec.NewColumns[0].Options[0].Expr)),
				})
				continue
			}
			specs = append(specs, ir.AlterTableSpec{
				Kind: ir.AlterTableSpecDropDefault,
				Name: columnName,
			})
		case tidbast.AlterTableModifyColumn:
			if len(spec.NewColumns) != 1 || spec.NewColumns[0] == nil || spec.Position == nil || spec.Position.Tp != tidbast.ColumnPositionNone {
				return normalizeTiDBRawDDL(sql)
			}
			column := normalizeTiDBColumnDef(spec.NewColumns[0])
			specs = append(specs, buildAlterColumnMutationSpecs(column, column.Name)...)
		case tidbast.AlterTableChangeColumn:
			if len(spec.NewColumns) != 1 || spec.NewColumns[0] == nil || spec.OldColumnName == nil || spec.Position == nil || spec.Position.Tp != tidbast.ColumnPositionNone {
				return normalizeTiDBRawDDL(sql)
			}
			column := normalizeTiDBColumnDef(spec.NewColumns[0])
			oldName := strings.TrimSpace(spec.OldColumnName.Name.O)
			if oldName == "" || column.Name == "" {
				return normalizeTiDBRawDDL(sql)
			}
			if oldName != column.Name {
				specs = append(specs, ir.AlterTableSpec{
					Kind:    ir.AlterTableSpecRenameColumn,
					Name:    oldName,
					NewName: column.Name,
				})
			}
			specs = append(specs, buildAlterColumnMutationSpecs(column, column.Name)...)
		case tidbast.AlterTableAddConstraint:
			if spec.Constraint == nil {
				return normalizeTiDBRawDDL(sql)
			}
			switch spec.Constraint.Tp {
			case tidbast.ConstraintPrimaryKey:
				specs = append(specs, ir.AlterTableSpec{
					Kind:  ir.AlterTableSpecAddPrimaryKey,
					Parts: normalizeTiDBIndexParts(spec.Constraint.Keys),
				})
			case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
				flushAlterTable()
				statements = append(statements, ir.AlterTableStatement{
					Name: tableName,
					Specs: []ir.AlterTableSpec{
						{
							Kind:           ir.AlterTableSpecAddUniqueKey,
							ConstraintName: strings.TrimSpace(spec.Constraint.Name),
							Parts:          normalizeTiDBIndexParts(spec.Constraint.Keys),
							IfNotExists:    spec.Constraint.IfNotExists,
						},
					},
				})
			case tidbast.ConstraintForeignKey:
				reference, ok := normalizeTiDBForeignKeyReference(spec.Constraint.Refer)
				if !ok {
					return normalizeTiDBRawDDL(sql)
				}
				parts := normalizeTiDBIndexParts(spec.Constraint.Keys)
				if shouldSplitAlterTableBeforeAddForeignKey(specs, parts) {
					flushAlterTable()
				}
				specs = append(specs, ir.AlterTableSpec{
					Kind:           ir.AlterTableSpecAddForeignKey,
					ConstraintName: strings.TrimSpace(spec.Constraint.Name),
					Parts:          parts,
					IfNotExists:    spec.Constraint.IfNotExists,
					Reference:      reference,
				})
			case tidbast.ConstraintKey, tidbast.ConstraintIndex:
				indexStatement, ok := normalizeTiDBAlterTableAddIndex(tableName, spec.Constraint)
				if !ok {
					return normalizeTiDBRawDDL(sql)
				}
				flushAlterTable()
				statements = append(statements, indexStatement)
			default:
				return normalizeTiDBRawDDL(sql)
			}
		case tidbast.AlterTableDropPrimaryKey:
			specs = append(specs, ir.AlterTableSpec{
				Kind:     ir.AlterTableSpecDropPrimaryKey,
				IfExists: spec.IfExists,
			})
		case tidbast.AlterTableDropIndex:
			flushAlterTable()
			statements = append(statements, ir.DropIndexStatement{
				Name:     strings.TrimSpace(spec.Name),
				IfExists: spec.IfExists,
			})
		case tidbast.AlterTableDropForeignKey:
			specs = append(specs, ir.AlterTableSpec{
				Kind:     ir.AlterTableSpecDropForeignKey,
				Name:     strings.TrimSpace(spec.Name),
				IfExists: spec.IfExists,
			})
		default:
			return normalizeTiDBRawDDL(sql)
		}
	}

	flushAlterTable()
	if len(statements) == 1 {
		return statements[0], nil
	}
	return ir.BatchStatement{Statements: statements}, nil
}

func normalizeTiDBDropIndex(stmt *tidbast.DropIndexStmt) (ir.Statement, error) {
	if stmt == nil {
		return ir.RawStatement{}, nil
	}

	tableName := ""
	if stmt.Table != nil {
		tableName = strings.TrimSpace(stmt.Table.Name.O)
	}

	return ir.DropIndexStatement{
		Name:     strings.TrimSpace(stmt.IndexName),
		IfExists: stmt.IfExists,
		Table:    tableName,
	}, nil
}

func normalizeTiDBCreateIndex(stmt *tidbast.CreateIndexStmt) (ir.Statement, error) {
	if stmt == nil {
		return ir.RawStatement{}, nil
	}

	tableName := ""
	if stmt.Table != nil {
		tableName = strings.TrimSpace(stmt.Table.Name.O)
	}

	using, parserName, where, comment, visibility := normalizeTiDBIndexOption(stmt.IndexOption)
	algorithm, lock := normalizeTiDBIndexLockAndAlgorithm(stmt.LockAlg)

	return ir.CreateIndexStatement{
		Name:        strings.TrimSpace(stmt.IndexName),
		Table:       tableName,
		Unique:      stmt.KeyType == tidbast.IndexKeyTypeUnique,
		IfNotExists: stmt.IfNotExists,
		Parts:       normalizeTiDBIndexParts(stmt.IndexPartSpecifications),
		Using:       using,
		ParserName:  parserName,
		Where:       where,
		Comment:     comment,
		Visibility:  visibility,
		Algorithm:   algorithm,
		Lock:        lock,
	}, nil
}

func normalizeTiDBColumnDef(column *tidbast.ColumnDef) ir.CreateTableColumn {
	normalized := ir.CreateTableColumn{}
	if column == nil || column.Name == nil {
		return normalized
	}

	normalized.Name = strings.TrimSpace(column.Name.Name.O)
	if column.Tp != nil {
		normalized.Type = strings.ToLower(strings.TrimSpace(column.Tp.String()))
	}

	for _, option := range column.Options {
		if option == nil {
			continue
		}
		switch option.Tp {
		case tidbast.ColumnOptionPrimaryKey:
			normalized.PrimaryKey = true
		case tidbast.ColumnOptionNotNull:
			normalized.NotNull = true
		case tidbast.ColumnOptionNull:
			normalized.NotNull = false
		case tidbast.ColumnOptionUniqKey:
			normalized.Unique = true
		case tidbast.ColumnOptionAutoIncrement:
			normalized.AutoIncrement = true
		case tidbast.ColumnOptionDefaultValue:
			normalized.Default = normalizeWhitespace(tiDBNodeText(option.Expr))
		case tidbast.ColumnOptionOnUpdate:
			continue
		}
	}

	return normalized
}

func buildAlterColumnMutationSpecs(column ir.CreateTableColumn, targetName string) []ir.AlterTableSpec {
	specs := []ir.AlterTableSpec{
		{
			Kind:   ir.AlterTableSpecAlterType,
			Name:   targetName,
			Column: column,
		},
	}

	if column.NotNull {
		specs = append(specs, ir.AlterTableSpec{
			Kind: ir.AlterTableSpecSetNotNull,
			Name: targetName,
		})
	} else {
		specs = append(specs, ir.AlterTableSpec{
			Kind: ir.AlterTableSpecDropNotNull,
			Name: targetName,
		})
	}

	return specs
}

func normalizeTiDBIndexParts(parts []*tidbast.IndexPartSpecification) []ir.IndexPart {
	normalizedParts := make([]ir.IndexPart, 0, len(parts))
	for _, part := range parts {
		if part == nil {
			continue
		}
		normalized := ir.IndexPart{
			Length: part.Length,
			Desc:   part.Desc,
		}
		if part.Column != nil {
			normalized.Column = strings.TrimSpace(part.Column.Name.O)
		}
		if part.Expr != nil {
			normalized.Expr = normalizeWhitespace(tiDBNodeText(part.Expr))
		}
		normalizedParts = append(normalizedParts, normalized)
	}
	return normalizedParts
}

func normalizeTiDBAlterTableAddIndex(tableName string, constraint *tidbast.Constraint) (ir.CreateIndexStatement, bool) {
	if constraint == nil {
		return ir.CreateIndexStatement{}, false
	}
	name := strings.TrimSpace(constraint.Name)
	if name == "" {
		return ir.CreateIndexStatement{}, false
	}
	using, parserName, where, comment, visibility := normalizeTiDBIndexOption(constraint.Option)
	return ir.CreateIndexStatement{
		Name:        name,
		Table:       tableName,
		IfNotExists: constraint.IfNotExists,
		Parts:       normalizeTiDBIndexParts(constraint.Keys),
		Using:       using,
		ParserName:  parserName,
		Where:       where,
		Comment:     comment,
		Visibility:  visibility,
	}, true
}

func normalizeTiDBIndexOption(option *tidbast.IndexOption) (using, parserName, where, comment, visibility string) {
	if option == nil {
		return "", "", "", "", ""
	}
	if option.Tp != tidbast.IndexTypeInvalid {
		using = strings.TrimSpace(option.Tp.String())
	}
	parserName = strings.TrimSpace(option.ParserName.O)
	if option.Condition != nil {
		where = normalizeWhitespace(tiDBNodeText(option.Condition))
	}
	comment = strings.TrimSpace(option.Comment)
	switch option.Visibility {
	case tidbast.IndexVisibilityVisible:
		visibility = "VISIBLE"
	case tidbast.IndexVisibilityInvisible:
		visibility = "INVISIBLE"
	}
	return using, parserName, where, comment, visibility
}

func normalizeTiDBIndexLockAndAlgorithm(lockAlg *tidbast.IndexLockAndAlgorithm) (algorithm, lock string) {
	if lockAlg == nil {
		return "", ""
	}
	if lockAlg.AlgorithmTp != tidbast.AlgorithmTypeDefault {
		algorithm = strings.TrimSpace(lockAlg.AlgorithmTp.String())
	}
	if lockAlg.LockTp != tidbast.LockTypeDefault {
		lock = strings.TrimSpace(lockAlg.LockTp.String())
	}
	return algorithm, lock
}

func shouldSplitAlterTableBeforeAddForeignKey(specs []ir.AlterTableSpec, parts []ir.IndexPart) bool {
	addedColumns := make(map[string]struct{})
	for _, spec := range specs {
		if spec.Kind != ir.AlterTableSpecAddColumn {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(spec.Column.Name))
		if name == "" {
			continue
		}
		addedColumns[name] = struct{}{}
	}
	if len(addedColumns) == 0 {
		return false
	}

	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part.Column))
		if name == "" {
			return true
		}
		if _, ok := addedColumns[name]; !ok {
			return true
		}
	}
	return false
}

func normalizeTiDBForeignKeyReference(reference *tidbast.ReferenceDef) (ir.ForeignKeyReference, bool) {
	if reference == nil || reference.Table == nil {
		return ir.ForeignKeyReference{}, false
	}

	normalized := ir.ForeignKeyReference{
		Table: strings.TrimSpace(normalizeTiDBTableName(reference.Table)),
		Parts: normalizeTiDBIndexParts(reference.IndexPartSpecifications),
		Match: normalizeTiDBMatchType(reference.Match),
	}
	if normalized.Table == "" || len(normalized.Parts) == 0 {
		return ir.ForeignKeyReference{}, false
	}
	if reference.OnDelete != nil {
		normalized.OnDelete = strings.TrimSpace(reference.OnDelete.ReferOpt.String())
	}
	if reference.OnUpdate != nil {
		normalized.OnUpdate = strings.TrimSpace(reference.OnUpdate.ReferOpt.String())
	}
	return normalized, true
}

func normalizeTiDBRenameTable(sql string, stmt *tidbast.RenameTableStmt) (ir.Statement, error) {
	if stmt == nil || len(stmt.TableToTables) == 0 {
		return normalizeTiDBRawDDL(sql)
	}

	statements := make([]ir.Statement, 0, len(stmt.TableToTables))
	for _, table := range stmt.TableToTables {
		if table == nil || table.OldTable == nil || table.NewTable == nil {
			return normalizeTiDBRawDDL(sql)
		}
		oldName := normalizeTiDBTableName(table.OldTable)
		newName := normalizeTiDBTableName(table.NewTable)
		if oldName == "" || newName == "" {
			return normalizeTiDBRawDDL(sql)
		}
		if !renameWithinSameSchema(oldName, newName) {
			return normalizeTiDBRawDDL(sql)
		}

		statements = append(statements, ir.RenameTableStatement{
			OldName: oldName,
			NewName: newName,
		})
	}

	if len(statements) == 1 {
		return statements[0], nil
	}
	return ir.BatchStatement{Statements: statements}, nil
}

func normalizeTiDBTableName(table *tidbast.TableName) string {
	if table == nil {
		return ""
	}
	if schema := strings.TrimSpace(table.Schema.O); schema != "" {
		return schema + "." + strings.TrimSpace(table.Name.O)
	}
	return strings.TrimSpace(table.Name.O)
}

func normalizeTiDBMatchType(match tidbast.MatchType) string {
	switch match {
	case tidbast.MatchFull:
		return "FULL"
	case tidbast.MatchPartial:
		return "PARTIAL"
	case tidbast.MatchSimple:
		return "SIMPLE"
	default:
		return ""
	}
}

func renameWithinSameSchema(oldName, newName string) bool {
	return extractSchema(oldName) == extractSchema(newName)
}

func extractSchema(identifier string) string {
	if idx := strings.Index(identifier, "."); idx >= 0 {
		return identifier[:idx]
	}
	return ""
}
