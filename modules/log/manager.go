package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

// LoggerManager owns the output writer and closes only resources it opened.
// Unlike Gitea's queued writers, these outputs are synchronous.
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
	// Only the default logger is configured in R00. New named loggers must
	// have a documented writer policy before they are registered.
	return m.defaultLogger.Load()
}

func newConsoleWriter(out io.Writer) io.Writer {
	return zerolog.ConsoleWriter{
		Out: out, TimeFormat: "15:04:05",
		FormatLevel: func(value any) string {
			level, err := zerolog.ParseLevel(fmt.Sprint(value))
			if err != nil {
				return "[???]"
			}
			tag := "[" + zerolog.FormattedLevels[level] + "]"
			if c := zerolog.LevelColors[level]; c != 0 {
				return fmt.Sprintf("\x1b[%dm%s\x1b[0m", c, tag)
			}
			return tag
		},
	}
}
func (m *LoggerManager) setOutput(out io.Writer, owned io.Closer, format string, level zerolog.Level) error {
	if format == "console" {
		out = newConsoleWriter(out)
	}
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
	// Keep context loggers and existing Zerolog integrations on the same sink.
	zerologlog.Logger = next
	return nil
}

// Configure is mainly for tests and early stderr setup.
func (m *LoggerManager) Configure(out io.Writer, format string, level Level) error {
	if out == nil {
		return errors.New("log writer is nil")
	}
	if format != "json" && format != "console" {
		return fmt.Errorf("invalid log format %q", format)
	}
	return m.setOutput(out, nil, format, level)
}
func (m *LoggerManager) ConfigureOutputs(mode, format, level, rootPath, fileName string) error {
	if format != "json" && format != "console" {
		return fmt.Errorf("invalid log format %q", format)
	}
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	var outputs []io.Writer
	var owned io.Closer
	seen := map[string]bool{}
	for _, raw := range strings.Split(mode, ",") {
		dest := strings.TrimSpace(raw)
		if seen[dest] {
			if owned != nil {
				_ = owned.Close()
			}
			return fmt.Errorf("duplicate log output %q", dest)
		}
		seen[dest] = true
		switch dest {
		case "console":
			outputs = append(outputs, os.Stderr)
		case "file":
			if fileName == "" || filepath.Base(fileName) != fileName || rootPath == "" {
				return errors.New("invalid log file destination")
			}
			st, statErr := os.Stat(rootPath)
			if statErr != nil || !st.IsDir() {
				return fmt.Errorf("log root directory is unavailable: %s", rootPath)
			}
			f, openErr := os.OpenFile(filepath.Join(rootPath, fileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if openErr != nil {
				return openErr
			}
			owned = f
			outputs = append(outputs, f)
		default:
			if owned != nil {
				_ = owned.Close()
			}
			return fmt.Errorf("unsupported log output %q", dest)
		}
	}
	if len(outputs) == 0 {
		return errors.New("no log outputs configured")
	}
	var out io.Writer = outputs[0]
	if len(outputs) > 1 {
		out = io.MultiWriter(outputs...)
	}
	return m.setOutput(out, owned, format, parsed)
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
