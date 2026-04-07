package uniquedialect

import (
	glsql "github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// OpenGORM opens a GORM dialector from a UniqueDialect DSN.
func OpenGORM(dsn string, cfg GORMConfig) (gorm.Dialector, error) {
	opts, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	return OpenGORMWithOptions(opts, cfg)
}

// OpenGORMWithOptions opens a GORM dialector from canonical options.
func OpenGORMWithOptions(opts Options, cfg GORMConfig) (gorm.Dialector, error) {
	opts = opts.normalized()
	virtualDSN := opts.FormatDSN()

	var base gorm.Dialector
	switch opts.TargetDialect {
	case DialectSQLite:
		base = glsql.Dialector{DriverName: DefaultSQLDriverName, DSN: virtualDSN}
	case DialectPostgres:
		base = postgres.New(postgres.Config{
			DriverName:           DefaultSQLDriverName,
			DSN:                  virtualDSN,
			PreferSimpleProtocol: cfg.PreferSimpleProtocol,
		})
	case DialectMySQL:
		base = mysql.New(mysql.Config{
			DriverName:                DefaultSQLDriverName,
			DSN:                       virtualDSN,
			SkipInitializeWithVersion: cfg.SkipInitializeWithVersion,
		})
	default:
		return nil, ErrUnsupportedDialector(opts.TargetDialect)
	}

	var encryptor Encryptor
	if opts.Encryption.Enabled {
		var err error
		if opts.Encryption.Provider != nil {
			encryptor = opts.Encryption.Provider
		} else {
			encryptor, err = NewDefaultEncryptor(opts.Encryption.Key)
			if err != nil {
				return nil, err
			}
		}
	}

	return &gormDialector{
		base:      base,
		encryptor: encryptor,
	}, nil
}

// ErrUnsupportedDialector reports an unsupported target for GORM execution.
func ErrUnsupportedDialector(dialect Dialect) error {
	return &unsupportedDialectorError{dialect: dialect}
}

type unsupportedDialectorError struct {
	dialect Dialect
}

func (e *unsupportedDialectorError) Error() string {
	return "unsupported gorm dialector target: " + string(e.dialect)
}

type gormDialector struct {
	base      gorm.Dialector
	encryptor Encryptor
}

func (d *gormDialector) Name() string {
	return d.base.Name()
}

func (d *gormDialector) Initialize(db *gorm.DB) error {
	if err := d.base.Initialize(db); err != nil {
		return err
	}
	if d.encryptor != nil {
		return db.Use(newEncryptionPlugin(d.encryptor))
	}
	return nil
}

func (d *gormDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return d.base.Migrator(db)
}

func (d *gormDialector) DataTypeOf(field *schema.Field) string {
	return d.base.DataTypeOf(field)
}

func (d *gormDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return d.base.DefaultValueOf(field)
}

func (d *gormDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, value interface{}) {
	d.base.BindVarTo(writer, stmt, value)
}

func (d *gormDialector) QuoteTo(writer clause.Writer, value string) {
	d.base.QuoteTo(writer, value)
}

func (d *gormDialector) Explain(sql string, vars ...interface{}) string {
	return d.base.Explain(sql, vars...)
}
