package uniquedialect

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/fatballfish/uniquedialect/internal/params"
	glesqlite "github.com/glebarez/go-sqlite"
	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func init() {
	sql.Register(DefaultSQLDriverName, &virtualDriver{})
}

// Open opens a database/sql DB from a UniqueDialect DSN.
func Open(dsn string) (*sql.DB, error) {
	opts, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	return OpenWithOptions(opts)
}

// OpenWithOptions opens a database/sql DB from canonical options.
func OpenWithOptions(opts Options) (*sql.DB, error) {
	connector, err := newConnector(opts.normalized())
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}

type virtualDriver struct{}

func (d *virtualDriver) Open(dsn string) (driver.Conn, error) {
	connector, err := newConnectorFromDSN(dsn)
	if err != nil {
		return nil, err
	}
	return connector.Connect(context.Background())
}

type connector struct {
	opts       Options
	translator *Translator
	logger     *sqlFailureLogger
	base       driver.Connector
}

func newConnectorFromDSN(dsn string) (*connector, error) {
	opts, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	return newConnector(opts)
}

func newConnector(opts Options) (*connector, error) {
	opts = opts.normalized()
	translator, err := NewTranslator(opts.Translator)
	if err != nil {
		return nil, err
	}

	base, err := buildBaseConnector(opts)
	if err != nil {
		return nil, err
	}
	return &connector{
		opts:       opts,
		translator: translator,
		logger:     newSQLFailureLogger(opts.Logging),
		base:       base,
	}, nil
}

func buildBaseConnector(opts Options) (driver.Connector, error) {
	dsn, err := opts.BuildDriverDSN()
	if err != nil {
		return nil, err
	}

	switch opts.Driver {
	case DriverSQLite:
		return staticConnector{driver: &glesqlite.Driver{}, dsn: dsn}, nil
	case DriverMySQL:
		return staticConnector{driver: &mysqldrv.MySQLDriver{}, dsn: dsn}, nil
	case DriverPostgres:
		config, err := pgx.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("parse postgres config: %w", err)
		}
		return stdlib.GetConnector(*config), nil
	default:
		return nil, fmt.Errorf("driver %q is not executable in MVP", opts.Driver)
	}
}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	baseConn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &virtualConn{
		Conn:       baseConn,
		translator: c.translator,
		logger:     c.logger,
	}, nil
}

func (c *connector) Driver() driver.Driver {
	return &virtualDriver{}
}

type staticConnector struct {
	driver driver.Driver
	dsn    string
}

func (c staticConnector) Connect(context.Context) (driver.Conn, error) {
	return c.driver.Open(c.dsn)
}

func (c staticConnector) Driver() driver.Driver {
	return c.driver
}

type virtualConn struct {
	driver.Conn
	translator *Translator
	logger     *sqlFailureLogger
}

func (c *virtualConn) Ping(ctx context.Context) error {
	pinger, ok := c.Conn.(driver.Pinger)
	if !ok {
		return nil
	}
	return pinger.Ping(ctx)
}

func (c *virtualConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *virtualConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	translated, err := c.translator.Translate(ctx, query)
	if err != nil {
		c.logger.logTranslationFailure("prepare", query, err)
		return nil, err
	}
	if preparer, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := preparer.PrepareContext(ctx, translated.SQL)
		if err != nil {
			c.logger.logExecutionFailure("prepare", query, translated.SQL, err)
			return nil, err
		}
		return &virtualStmt{Stmt: stmt, logger: c.logger, originalSQL: query, translatedSQL: translated.SQL}, nil
	}
	stmt, err := c.Conn.Prepare(translated.SQL)
	if err != nil {
		c.logger.logExecutionFailure("prepare", query, translated.SQL, err)
		return nil, err
	}
	return &virtualStmt{Stmt: stmt, logger: c.logger, originalSQL: query, translatedSQL: translated.SQL}, nil
}

func (c *virtualConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *virtualConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	beginner, ok := c.Conn.(driver.ConnBeginTx)
	if ok {
		return beginner.BeginTx(ctx, opts)
	}
	return c.Conn.Begin()
}

func (c *virtualConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	translated, err := c.translator.Translate(ctx, query, namedValuesToArgs(args)...)
	if err != nil {
		c.logger.logTranslationFailure("exec", query, err)
		return nil, err
	}
	result, err := execer.ExecContext(ctx, translated.SQL, argsFromSlice(translated.Args))
	if err != nil {
		c.logger.logExecutionFailure("exec", query, translated.SQL, err)
		return nil, err
	}
	return result, nil
}

func (c *virtualConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	translated, err := c.translator.Translate(ctx, query, namedValuesToArgs(args)...)
	if err != nil {
		c.logger.logTranslationFailure("query", query, err)
		return nil, err
	}
	rows, err := queryer.QueryContext(ctx, translated.SQL, argsFromSlice(translated.Args))
	if err != nil {
		c.logger.logExecutionFailure("query", query, translated.SQL, err)
		return nil, err
	}
	return rows, nil
}

func (c *virtualConn) CheckNamedValue(value *driver.NamedValue) error {
	checker, ok := c.Conn.(driver.NamedValueChecker)
	if !ok {
		return nil
	}
	return checker.CheckNamedValue(value)
}

func (c *virtualConn) ResetSession(ctx context.Context) error {
	resetter, ok := c.Conn.(driver.SessionResetter)
	if !ok {
		return nil
	}
	return resetter.ResetSession(ctx)
}

func (c *virtualConn) IsValid() bool {
	validator, ok := c.Conn.(driver.Validator)
	if !ok {
		return true
	}
	return validator.IsValid()
}

type virtualStmt struct {
	driver.Stmt
	logger        *sqlFailureLogger
	originalSQL   string
	translatedSQL string
}

func (s *virtualStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if execer, ok := s.Stmt.(driver.StmtExecContext); ok {
		result, err := execer.ExecContext(ctx, args)
		if err != nil {
			s.logger.logExecutionFailure("stmt_exec", s.originalSQL, s.translatedSQL, err)
			return nil, err
		}
		return result, nil
	}
	values, err := namedValuesToValues(args)
	if err != nil {
		return nil, err
	}
	result, err := s.Stmt.Exec(values)
	if err != nil {
		s.logger.logExecutionFailure("stmt_exec", s.originalSQL, s.translatedSQL, err)
		return nil, err
	}
	return result, nil
}

func (s *virtualStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if queryer, ok := s.Stmt.(driver.StmtQueryContext); ok {
		rows, err := queryer.QueryContext(ctx, args)
		if err != nil {
			s.logger.logExecutionFailure("stmt_query", s.originalSQL, s.translatedSQL, err)
			return nil, err
		}
		return rows, nil
	}
	values, err := namedValuesToValues(args)
	if err != nil {
		return nil, err
	}
	rows, err := s.Stmt.Query(values)
	if err != nil {
		s.logger.logExecutionFailure("stmt_query", s.originalSQL, s.translatedSQL, err)
		return nil, err
	}
	return rows, nil
}

func (s *virtualStmt) CheckNamedValue(value *driver.NamedValue) error {
	checker, ok := s.Stmt.(driver.NamedValueChecker)
	if !ok {
		return nil
	}
	return checker.CheckNamedValue(value)
}

func namedValuesToArgs(values []driver.NamedValue) []any {
	out := make([]any, len(values))
	for index := range values {
		out[index] = values[index].Value
	}
	return out
}

func argsFromSlice(values []any) []driver.NamedValue {
	out := make([]driver.NamedValue, len(values))
	for index := range values {
		converted, err := params.Normalize(values[index])
		if err != nil {
			converted = values[index]
		}
		out[index] = driver.NamedValue{
			Ordinal: index + 1,
			Value:   converted,
		}
	}
	return out
}

func namedValuesToValues(values []driver.NamedValue) ([]driver.Value, error) {
	out := make([]driver.Value, len(values))
	for index := range values {
		converted, err := params.Normalize(values[index].Value)
		if err != nil {
			return nil, err
		}
		out[index] = converted
	}
	return out, nil
}

var _ driver.Driver = (*virtualDriver)(nil)
var _ driver.Connector = (*connector)(nil)
var _ driver.Conn = (*virtualConn)(nil)
var _ driver.ExecerContext = (*virtualConn)(nil)
var _ driver.QueryerContext = (*virtualConn)(nil)
var _ driver.ConnPrepareContext = (*virtualConn)(nil)
var _ driver.ConnBeginTx = (*virtualConn)(nil)
var _ driver.Pinger = (*virtualConn)(nil)
var _ driver.StmtExecContext = (*virtualStmt)(nil)
var _ driver.StmtQueryContext = (*virtualStmt)(nil)
