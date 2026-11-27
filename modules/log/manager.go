package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

type LoggerManager struct {
	mu            sync.Mutex
	defaultLogger atomic.Pointer[LoggerImpl]
	writer        io.Writer
	owned         io.Closer
	closed        bool
}

func NewManager() *LoggerManager {
	m := &LoggerManager{writer: os.Stderr}
	l := zerolog.New(zerolog.SyncWriter(os.Stderr)).With().Timestamp().Logger()
	l = l.Level(zerolog.InfoLevel)
	m.defaultLogger.Store(&LoggerImpl{inner: l})
	return m
}

var manager = NewManager()

func GetManager() *LoggerManager { return manager }

func (m *LoggerManager) GetLogger(name string) *LoggerImpl {
	current := m.defaultLogger.Load()
	if name == "" || name == DEFAULT {
		return current
	}
	return &LoggerImpl{inner: current.inner.With().Str("component", name).Logger()}
}

func newConsoleWriter(out io.Writer, noColor, compact bool) io.Writer {
	w := zerolog.ConsoleWriter{
		Out: out, TimeFormat: "2006-01-02 15:04:05Z07:00", NoColor: noColor,
		FormatLevel: func(value any) string {
			level, err := zerolog.ParseLevel(fmt.Sprint(value))
			if err != nil {
				return "[???]"
			}
			name := zerolog.FormattedLevels[level]
			if compact && len(name) > 0 {
				name = name[:1]
			}
			tag := "[" + name + "]"
			if c := zerolog.LevelColors[level]; !noColor && c != 0 {
				return fmt.Sprintf("\x1b[%dm%s\x1b[0m", c, tag)
			}
			return tag
		},
	}
	if compact {
		w.TimeFormat = "15:04:05"
		w.FormatPrepare = prepareConsoleEvent
		w.FieldsOrder = []string{"code", "error", "reason", "sqlstate", "rows", "action", "scope", "attempts", "limit", "retry_after", "provider", "provider_status", "provider_code", "rid"}
	}
	return w
}
func (m *LoggerManager) setOutput(out io.Writer, owned io.Closer, level zerolog.Level) error {
	next := zerolog.New(zerolog.SyncWriter(out)).With().Timestamp().Logger().Level(level)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		if owned != nil {
			_ = owned.Close()
		}
		return errors.New("logger manager is closed")
	}
	if m.owned != nil {
		if err := m.owned.Close(); err != nil {
			if owned != nil {
				_ = owned.Close()
			}
			return err
		}
	}
	m.owned, m.writer = owned, out
	m.defaultLogger.Store(&LoggerImpl{inner: next})

	zerologlog.Logger = next
	return nil
}

func (m *LoggerManager) Configure(out io.Writer, format string, level Level) error {
	if out == nil {
		return errors.New("log writer is nil")
	}
	if format != "json" && format != "console" {
		return fmt.Errorf("invalid log format %q", format)
	}
	if format == "console" {
		out = newConsoleWriter(out, false, true)
	}
	return m.setOutput(out, nil, level)
}
func (m *LoggerManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.closed = true
	if m.owned != nil {
		if err := m.owned.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "log close:", err)
		}
		m.owned = nil
	}
}
func SetupStderrLogger(level Level) error {
	return GetManager().Configure(os.Stderr, "console", level)
}
