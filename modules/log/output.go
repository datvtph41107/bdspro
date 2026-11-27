package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// OutputOptions describes sinks, not application event policy.
type OutputOptions struct {
	Mode, ConsoleFormat, FileFormat, Level, RootPath, FileName string
	ConsoleLevel, FileLevel, ConsoleComponents                 string
	MaxSizeMB, MaxBackups, MaxAgeDays                           int
}

func (o OutputOptions) Validate() error {
	switch o.Mode {
	case "console", "file", "console,file", "file,console":
	default:
		return errors.New("invalid log output mode")
	}
	for _, format := range []string{o.ConsoleFormat, o.FileFormat} {
		if format != "console" && format != "json" {
			return errors.New("log format must be console or json")
		}
	}
	for _, entry := range []struct{ key, value string }{
		{"log.LEVEL", o.Level},
		{"log.console.LEVEL", o.levelFor("console")},
		{"log.file.LEVEL", o.levelFor("file")},
	} {
		level, err := zerolog.ParseLevel(entry.value)
		if err != nil || level < zerolog.DebugLevel || level > zerolog.ErrorLevel {
			return fmt.Errorf("%s must be debug, info, warn or error", entry.key)
		}
	}
	if _, err := parseComponents(o.ConsoleComponents); err != nil {
		return err
	}
	if o.RootPath == "" || o.FileName == "" || o.FileName == "." || o.FileName == ".." || filepath.Base(o.FileName) != o.FileName {
		return errors.New("invalid log file destination")
	}
	if o.MaxSizeMB < 1 || o.MaxSizeMB > 4096 || o.MaxBackups < 1 || o.MaxBackups > 100 || o.MaxAgeDays < 1 || o.MaxAgeDays > 365 {
		return errors.New("log rotation requires MAX_SIZE_MB 1..4096, MAX_BACKUPS 1..100 and MAX_AGE_DAYS 1..365")
	}
	return nil
}

func (o OutputOptions) levelFor(destination string) string {
	switch destination {
	case "console":
		if o.ConsoleLevel != "" {
			return o.ConsoleLevel
		}
	case "file":
		if o.FileLevel != "" {
			return o.FileLevel
		}
	}
	return o.Level
}

// emissionLevel is the lowest enabled destination threshold. An inactive
// destination must never increase the amount of logging work.
func (o OutputOptions) emissionLevel() zerolog.Level {
	level := zerolog.ErrorLevel
	for _, destination := range strings.Split(o.Mode, ",") {
		next, _ := zerolog.ParseLevel(o.levelFor(destination))
		if next < level {
			level = next
		}
	}
	return level
}

func (m *LoggerManager) ConfigureOutputs(o OutputOptions) error {
	if err := o.Validate(); err != nil {
		return err
	}
	var outputs []io.Writer
	var owned io.Closer
	consoleComponents, _ := parseComponents(o.ConsoleComponents) // validated above
	for _, dest := range strings.Split(o.Mode, ",") {
		var out io.Writer
		format := o.ConsoleFormat
		noColor := os.Getenv("NO_COLOR") != "" || !isatty.IsTerminal(os.Stderr.Fd())
		switch dest {
		case "console":
			out = os.Stderr
		case "file":
			if err := os.MkdirAll(o.RootPath, 0700); err != nil {
				return fmt.Errorf("prepare log directory: %w", err)
			}
			name := filepath.Join(o.RootPath, o.FileName)
			// Preflight eagerly: bad paths must fail startup, not silently lose the first event.
			f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("open log file: %w", err)
			}
			if err := f.Close(); err != nil {
				return err
			}
			writer := &lumberjack.Logger{Filename: name, MaxSize: o.MaxSizeMB, MaxBackups: o.MaxBackups, MaxAge: o.MaxAgeDays}
			owned, out = writer, writer
			format, noColor = o.FileFormat, true
		}
		if format == "console" {
			out = newConsoleWriter(out, noColor, dest == "console")
		}
		level, _ := zerolog.ParseLevel(o.levelFor(dest)) // validated above
		var components map[string]struct{}
		if dest == "console" {
			components = consoleComponents
		}
		outputs = append(outputs, filteredSink{out: out, level: level, components: components})
	}
	return m.setOutput(fanoutWriter(outputs), owned, o.emissionLevel())
}

// filteredSink evaluates severity using Zerolog's native LevelWriter contract.
// Only selected low-severity console events require JSON component decoding.
type filteredSink struct {
	out        io.Writer
	level      zerolog.Level
	components map[string]struct{}
}

func (s filteredSink) Write(p []byte) (int, error) {
	return s.out.Write(p)
}

func (s filteredSink) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	if level < s.level {
		return len(p), nil
	}
	// Component selection must not hide warnings or errors that meet the sink level.
	if level < zerolog.WarnLevel && !componentAllowed(s.components, p) {
		return len(p), nil
	}
	return s.out.Write(p)
}

var _ zerolog.LevelWriter = filteredSink{}

// A failing terminal must not prevent a file write (or vice versa).
type fanoutWriter []io.Writer

func (writers fanoutWriter) Write(p []byte) (int, error) {
	return writers.write(p, func(w io.Writer, p []byte) (int, error) { return w.Write(p) })
}

func (writers fanoutWriter) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	return writers.write(p, func(w io.Writer, p []byte) (int, error) {
		if lw, ok := w.(zerolog.LevelWriter); ok {
			return lw.WriteLevel(level, p)
		}
		return w.Write(p)
	})
}

var _ zerolog.LevelWriter = fanoutWriter{}

func (writers fanoutWriter) write(p []byte, write func(io.Writer, []byte) (int, error)) (int, error) {
	var errs []error
	for _, writer := range writers {
		n, err := write(writer, p)
		if err == nil && n != len(p) {
			err = io.ErrShortWrite
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return 0, errors.Join(errs...)
	}
	return len(p), nil
}
