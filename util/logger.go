package util

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// NewLogger uses compact colored level tags only in development.
// All other text remains uncolored. Production emits machine-readable JSON.
func NewLogger(out io.Writer, environment string) zerolog.Logger {
	useColor := false
	if terminal, ok := out.(*os.File); ok {
		if info, err := terminal.Stat(); err == nil {
			useColor = info.Mode()&os.ModeCharDevice != 0 &&
				os.Getenv("NO_COLOR") == "" &&
				os.Getenv("TERM") != "dumb"
		}
	}
	return newLogger(out, environment, useColor)
}

// newLogger makes the ANSI decision testable without requiring a real terminal.
func newLogger(out io.Writer, environment string, useColor bool) zerolog.Logger {
	if environment != "development" {
		return zerolog.New(out).With().Timestamp().Logger()
	}
	writer := zerolog.ConsoleWriter{
		Out:        out,
		TimeFormat: "15:04:05",
		NoColor:    true,
		FormatLevel: func(value interface{}) string {
			level := strings.ToLower(strings.TrimSpace(value.(string)))
			tag, color := "", ""
			switch level {
			case "trace":
				tag, color = "TRC", "36"
			case "debug":
				tag, color = "DBG", "36"
			case "info":
				tag, color = "INF", "32"
			case "warn":
				tag, color = "WRN", "33"
			case "error":
				tag, color = "ERR", "31"
			case "fatal":
				tag, color = "FTL", "31"
			case "panic":
				tag, color = "PNC", "31"
			default:
				return strings.ToUpper(level)
			}
			if !useColor {
				return tag
			}
			return "\x1b[" + color + "m" + tag + "\x1b[0m"
		},
	}
	return zerolog.New(writer).With().Timestamp().Logger()
}
