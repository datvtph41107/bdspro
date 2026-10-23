package util

import (
	"fmt"
	"io"

	"github.com/rs/zerolog"
)

func NewLogger(out io.Writer, environment string) zerolog.Logger {
	if environment == "development" {
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
