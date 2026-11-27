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
	MaxSizeMB, MaxBackups, MaxAgeDays                          int
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
	level, err := zerolog.ParseLevel(o.Level)
	if err != nil || level < zerolog.DebugLevel || level > zerolog.ErrorLevel {
		return errors.New("log level must be debug, info, warn or error")
	}
	if o.RootPath == "" || o.FileName == "" || o.FileName == "." || o.FileName == ".." || filepath.Base(o.FileName) != o.FileName {
		return errors.New("invalid log file destination")
	}
	if o.MaxSizeMB < 1 || o.MaxSizeMB > 4096 || o.MaxBackups < 1 || o.MaxBackups > 100 || o.MaxAgeDays < 1 || o.MaxAgeDays > 365 {
		return errors.New("log rotation requires MAX_SIZE_MB 1..4096, MAX_BACKUPS 1..100 and MAX_AGE_DAYS 1..365")
	}
	return nil
}

func (m *LoggerManager) ConfigureOutputs(o OutputOptions) error {
	if err := o.Validate(); err != nil {
		return err
	}
	var outputs []io.Writer
	var owned io.Closer
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
		outputs = append(outputs, out)
	}
	level, _ := zerolog.ParseLevel(o.Level) // validated above
	return m.setOutput(fanoutWriter(outputs), owned, level)
}

// A failing terminal must not prevent a file write (or vice versa).
type fanoutWriter []io.Writer

func (writers fanoutWriter) Write(p []byte) (int, error) {
	var errs []error
	for _, writer := range writers {
		n, err := writer.Write(p)
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
