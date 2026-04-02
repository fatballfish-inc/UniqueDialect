package uniquedialect

import (
	"context"
	"fmt"
	"sync"

	"github.com/fatballfish/uniquedialect/internal/normalize"
	internalparser "github.com/fatballfish/uniquedialect/internal/parser"
	"github.com/fatballfish/uniquedialect/internal/render"
	"github.com/fatballfish/uniquedialect/internal/syntax"
	mysqlsyntax "github.com/fatballfish/uniquedialect/internal/syntax/mysql"
	postgressyntax "github.com/fatballfish/uniquedialect/internal/syntax/postgres"
)

// Translation contains a translated SQL statement and its mapped arguments.
type Translation struct {
	SQL  string
	Args []any
}

// TranslatorStats exposes in-memory cache hit statistics.
type TranslatorStats struct {
	Hits   int
	Misses int
}

// Translator translates SQL between dialects and caches results.
type Translator struct {
	opts  TranslatorOptions
	mu    sync.RWMutex
	cache map[string]string
	stats TranslatorStats
}

// NewTranslator constructs a translator for a dialect pair.
func NewTranslator(opts TranslatorOptions) (*Translator, error) {
	if opts.InputDialect == "" {
		return nil, fmt.Errorf("input dialect is required")
	}
	if opts.TargetDialect == "" {
		return nil, fmt.Errorf("target dialect is required")
	}
	return &Translator{
		opts:  opts,
		cache: map[string]string{},
	}, nil
}

// Stats returns a snapshot of cache statistics.
func (t *Translator) Stats() TranslatorStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stats
}

// Translate rewrites SQL text and returns mapped arguments.
func (t *Translator) Translate(_ context.Context, sql string, args ...any) (Translation, error) {
	key := string(t.opts.InputDialect) + "->" + string(t.opts.TargetDialect) + ":" + sql

	t.mu.RLock()
	cached, ok := t.cache[key]
	t.mu.RUnlock()
	if ok {
		t.mu.Lock()
		t.stats.Hits++
		t.mu.Unlock()
		return Translation{SQL: cached, Args: cloneArgs(args)}, nil
	}

	translated, err := t.translateSQL(sql)
	if err != nil {
		return Translation{}, err
	}

	t.mu.Lock()
	t.cache[key] = translated
	t.stats.Misses++
	t.mu.Unlock()

	return Translation{SQL: translated, Args: cloneArgs(args)}, nil
}

func (t *Translator) translateSQL(sql string) (string, error) {
	if t.opts.InputDialect == t.opts.TargetDialect {
		return sql, nil
	}

	parsed, err := internalparser.ParseOne(sql, t.opts.InputDialect)
	if err == nil {
		normalized, normalizeErr := normalize.ParsedStatement(parsed)
		if normalizeErr == nil {
			return render.Statement(normalized, string(t.opts.InputDialect), string(t.opts.TargetDialect))
		}
		if parsed.Kind == internalparser.StatementKindShow && parsed.Status == internalparser.SupportStatusSupported {
			return "", normalizeErr
		}
		if shouldBlockRecognizedUnadapted(parsed.Kind, parsed.Status) {
			return "", &ParserAdaptationError{
				SourceDialect:  string(t.opts.InputDialect),
				StatementKind:  string(parsed.Kind),
				NativeNodeType: parsed.NativeNodeType,
				Status:         string(parsed.Status),
			}
		}
	}

	stmt, err := t.parseLegacy(sql)
	if err != nil {
		return "", err
	}

	normalized := normalize.Statement(stmt)
	return render.Statement(normalized, string(t.opts.InputDialect), string(t.opts.TargetDialect))
}

func shouldBlockRecognizedUnadapted(kind internalparser.StatementKind, status internalparser.SupportStatus) bool {
	if status != internalparser.SupportStatusRecognizedUnadapted {
		return false
	}
	switch kind {
	case internalparser.StatementKindWith, internalparser.StatementKindSetOp, internalparser.StatementKindShow:
		return true
	default:
		return false
	}
}

func cloneArgs(args []any) []any {
	out := make([]any, len(args))
	copy(out, args)
	return out
}

func (t *Translator) parseLegacy(sql string) (syntax.Statement, error) {
	switch t.opts.InputDialect {
	case DialectMySQL:
		return mysqlsyntax.Parse(sql)
	case DialectPostgres:
		return postgressyntax.Parse(sql)
	case DialectSQLite, DialectOracle:
		return mysqlsyntax.Parse(sql)
	default:
		return nil, fmt.Errorf("unsupported input dialect %s", t.opts.InputDialect)
	}
}
