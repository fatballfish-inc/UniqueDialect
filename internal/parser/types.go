package parser

// StatementKind identifies the recognized SQL statement family.
type StatementKind string

const (
	StatementKindSelect         StatementKind = "select"
	StatementKindInsert         StatementKind = "insert"
	StatementKindUpdate         StatementKind = "update"
	StatementKindDelete         StatementKind = "delete"
	StatementKindBegin          StatementKind = "begin"
	StatementKindCommit         StatementKind = "commit"
	StatementKindRollback       StatementKind = "rollback"
	StatementKindWith           StatementKind = "with"
	StatementKindSetOp          StatementKind = "set_op"
	StatementKindSet            StatementKind = "set"
	StatementKindCreateTable    StatementKind = "create_table"
	StatementKindCreateDatabase StatementKind = "create_database"
	StatementKindCreateView     StatementKind = "create_view"
	StatementKindExplain        StatementKind = "explain"
	StatementKindAlterTable     StatementKind = "alter_table"
	StatementKindDropDatabase   StatementKind = "drop_database"
	StatementKindDropTable      StatementKind = "drop_table"
	StatementKindDropView       StatementKind = "drop_view"
	StatementKindShow           StatementKind = "show"
	StatementKindUse            StatementKind = "use"
	StatementKindCreateIndex    StatementKind = "create_index"
	StatementKindDropIndex      StatementKind = "drop_index"
	StatementKindTruncate       StatementKind = "truncate_table"
	StatementKindRenameTable    StatementKind = "rename_table"
	StatementKindOtherDDL       StatementKind = "other_ddl"
	StatementKindOtherDML       StatementKind = "other_dml"
	StatementKindOther          StatementKind = "other"
)

// SupportStatus records whether a parsed statement is already adapted upstream.
type SupportStatus string

const (
	SupportStatusSupported           SupportStatus = "supported"
	SupportStatusRecognizedUnadapted SupportStatus = "recognized_unadapted"
	SupportStatusUnsupported         SupportStatus = "unsupported"
)

// ParsedStatement is the project-owned parse result returned by the parser facade.
type ParsedStatement struct {
	SQL            string
	SourceDialect  string
	Kind           StatementKind
	Status         SupportStatus
	NativeNodeType string
	NativeAST      any
}
