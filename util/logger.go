package util

import (
	"fmt"
	"io"

	"github.com/rs/zerolog"
)

// NewLogger selects the output format without changing event fields or meaning.
func NewLogger(out io.Writer, format string) zerolog.Logger {
	if format == "console" {
		out = zerolog.ConsoleWriter{
			Out:        out,
			TimeFormat: "15:04:05",
			FormatLevel: func(value interface{}) string {
				level, err := zerolog.ParseLevel(fmt.Sprint(value))
				if err != nil {
					return "[???]"
				}

				tag := "[" + zerolog.FormattedLevels[level] + "]"

				color := zerolog.LevelColors[level]
				if color == 0 {
					return tag
				}

				return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, tag)
			},
		}
	}

	return zerolog.New(out).With().Timestamp().Logger()
}
