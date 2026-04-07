package uniquedialect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type sqlFailureLogger struct {
	enabled          bool
	translateLogPath string
	executionLogPath string
	mu               sync.Mutex
}

type sqlFailureLogEntry struct {
	Timestamp     string `json:"timestamp"`
	Kind          string `json:"kind"`
	Stage         string `json:"stage"`
	InputSQL      string `json:"input_sql"`
	TranslatedSQL string `json:"translated_sql,omitempty"`
	Error         string `json:"error"`
}

func newSQLFailureLogger(opts LoggingOptions) *sqlFailureLogger {
	if !opts.LogFailedSQL {
		return nil
	}
	return &sqlFailureLogger{
		enabled:          true,
		translateLogPath: opts.TranslateFailureLogPath,
		executionLogPath: opts.ExecutionFailureLogPath,
	}
}

func (l *sqlFailureLogger) logTranslationFailure(stage string, inputSQL string, err error) {
	if l == nil || !l.enabled || err == nil {
		return
	}
	l.write(l.translateLogPath, sqlFailureLogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Kind:      "translation_failure",
		Stage:     stage,
		InputSQL:  inputSQL,
		Error:     err.Error(),
	})
}

func (l *sqlFailureLogger) logExecutionFailure(stage string, inputSQL string, translatedSQL string, err error) {
	if l == nil || !l.enabled || err == nil {
		return
	}
	l.write(l.executionLogPath, sqlFailureLogEntry{
		Timestamp:     time.Now().Format(time.RFC3339Nano),
		Kind:          "execution_failure",
		Stage:         stage,
		InputSQL:      inputSQL,
		TranslatedSQL: translatedSQL,
		Error:         err.Error(),
	})
}

func (l *sqlFailureLogger) write(path string, entry sqlFailureLogEntry) {
	if path == "" {
		return
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}
	payload = append(payload, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	_, _ = file.Write(payload)
}
