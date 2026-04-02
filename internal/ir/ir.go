package ir

// Statement is the normalized IR statement.
type Statement interface {
	statementNode()
}

// BatchStatement preserves a sequence of statements that must be rendered in order.
type BatchStatement struct {
	Statements []Statement
}

func (BatchStatement) statementNode() {}

// RawStatement preserves raw SQL for pass-through rendering.
type RawStatement struct {
	SQL string
}

func (RawStatement) statementNode() {}

// SelectStatement is the normalized SELECT IR.
type SelectStatement struct {
	Columns []string
	From    string
	Joins   []Join
	Where   string
	OrderBy []string
	Limit   *LimitClause
}

func (SelectStatement) statementNode() {}

// InsertStatement is the normalized INSERT IR.
type InsertStatement struct {
	Table    string
	Columns  []string
	Values   []string
	Conflict *ConflictClause
}

func (InsertStatement) statementNode() {}

// UpdateStatement is the normalized UPDATE IR.
type UpdateStatement struct {
	Table       string
	Assignments []Assignment
	Where       string
}

func (UpdateStatement) statementNode() {}

// DeleteStatement is the normalized DELETE IR.
type DeleteStatement struct {
	Table string
	Where string
}

func (DeleteStatement) statementNode() {}

// UseStatement is the normalized USE/current-schema statement.
type UseStatement struct {
	Database string
}

func (UseStatement) statementNode() {}

// ShowTablesStatement is the normalized SHOW TABLES statement.
type ShowTablesStatement struct {
	Database string
}

func (ShowTablesStatement) statementNode() {}

// ShowColumnsStatement is the normalized SHOW COLUMNS statement.
type ShowColumnsStatement struct {
	Table    string
	Database string
}

func (ShowColumnsStatement) statementNode() {}

// ShowIndexStatement is the normalized SHOW INDEX statement.
type ShowIndexStatement struct {
	Table    string
	Database string
}

func (ShowIndexStatement) statementNode() {}

// ShowTableStatusStatement is the normalized SHOW TABLE STATUS statement.
type ShowTableStatusStatement struct {
	Database string
}

func (ShowTableStatusStatement) statementNode() {}

// ShowDatabasesStatement is the normalized SHOW DATABASES statement.
type ShowDatabasesStatement struct{}

func (ShowDatabasesStatement) statementNode() {}

// ShowCreateDatabaseStatement is the normalized SHOW CREATE DATABASE statement.
type ShowCreateDatabaseStatement struct {
	Name        string
	IfNotExists bool
}

func (ShowCreateDatabaseStatement) statementNode() {}

// ShowCreateTableStatement is the normalized SHOW CREATE TABLE statement.
type ShowCreateTableStatement struct {
	Schema string
	Name   string
}

func (ShowCreateTableStatement) statementNode() {}

// ShowCreateViewStatement is the normalized SHOW CREATE VIEW statement.
type ShowCreateViewStatement struct {
	Name string
}

func (ShowCreateViewStatement) statementNode() {}

// CreateTableStatement is the normalized CREATE TABLE IR.
type CreateTableStatement struct {
	Name        string
	IfNotExists bool
	Columns     []CreateTableColumn
	Constraints []string
}

func (CreateTableStatement) statementNode() {}

// CreateTableColumn is a normalized CREATE TABLE column definition.
type CreateTableColumn struct {
	Name          string
	Type          string
	NotNull       bool
	PrimaryKey    bool
	Unique        bool
	AutoIncrement bool
	Default       string
}

// AlterTableStatement is the normalized ALTER TABLE IR.
type AlterTableStatement struct {
	Name  string
	Specs []AlterTableSpec
}

func (AlterTableStatement) statementNode() {}

// AlterTableSpecKind identifies the normalized ALTER TABLE operation kind.
type AlterTableSpecKind string

const (
	AlterTableSpecAddColumn      AlterTableSpecKind = "add_column"
	AlterTableSpecDropColumn     AlterTableSpecKind = "drop_column"
	AlterTableSpecSetDefault     AlterTableSpecKind = "set_default"
	AlterTableSpecDropDefault    AlterTableSpecKind = "drop_default"
	AlterTableSpecAlterType      AlterTableSpecKind = "alter_type"
	AlterTableSpecSetNotNull     AlterTableSpecKind = "set_not_null"
	AlterTableSpecDropNotNull    AlterTableSpecKind = "drop_not_null"
	AlterTableSpecRenameColumn   AlterTableSpecKind = "rename_column"
	AlterTableSpecAddPrimaryKey  AlterTableSpecKind = "add_primary_key"
	AlterTableSpecAddUniqueKey   AlterTableSpecKind = "add_unique_key"
	AlterTableSpecAddForeignKey  AlterTableSpecKind = "add_foreign_key"
	AlterTableSpecDropPrimaryKey AlterTableSpecKind = "drop_primary_key"
	AlterTableSpecDropForeignKey AlterTableSpecKind = "drop_foreign_key"
)

// AlterTableSpec is a normalized ALTER TABLE operation.
type AlterTableSpec struct {
	Kind           AlterTableSpecKind
	Column         CreateTableColumn
	Name           string
	NewName        string
	Default        string
	ConstraintName string
	Parts          []IndexPart
	IfExists       bool
	IfNotExists    bool
	Reference      ForeignKeyReference
}

// ForeignKeyReference is a normalized foreign-key reference definition.
type ForeignKeyReference struct {
	Table    string
	Parts    []IndexPart
	Match    string
	OnDelete string
	OnUpdate string
}

// DropIndexStatement is the normalized DROP INDEX IR.
type DropIndexStatement struct {
	Name     string
	IfExists bool
	Table    string
}

func (DropIndexStatement) statementNode() {}

// CreateIndexStatement is the normalized CREATE INDEX IR.
type CreateIndexStatement struct {
	Name        string
	Table       string
	Unique      bool
	IfNotExists bool
	Parts       []IndexPart
	Using       string
	ParserName  string
	Where       string
	Comment     string
	Visibility  string
	Algorithm   string
	Lock        string
}

func (CreateIndexStatement) statementNode() {}

// RenameTableStatement is the normalized RENAME TABLE IR.
type RenameTableStatement struct {
	OldName string
	NewName string
}

func (RenameTableStatement) statementNode() {}

// IndexPart is a normalized index key part.
type IndexPart struct {
	Column string
	Expr   string
	Length int
	Desc   bool
}

// Join is a normalized join node.
type Join struct {
	Kind  string
	Table string
	On    string
}

// LimitClause is the normalized limit node.
type LimitClause struct {
	Offset *int
	Count  int
}

// Assignment is the normalized assignment node.
type Assignment struct {
	Column string
	Value  string
}

// ConflictStyle identifies the source upsert form.
type ConflictStyle string

const (
	// ConflictStyleMySQL represents mysql upsert semantics.
	ConflictStyleMySQL ConflictStyle = "mysql"
	// ConflictStylePostgres represents postgres upsert semantics.
	ConflictStylePostgres ConflictStyle = "postgres"
)

// ConflictClause is the normalized upsert node.
type ConflictClause struct {
	Style         ConflictStyle
	TargetColumns []string
	Assignments   []Assignment
}
