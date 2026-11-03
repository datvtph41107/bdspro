// Package log provides BDSPro logging with a Gitea-style public API.
package log

import "github.com/rs/zerolog"

type Level = zerolog.Level

const (
	TRACE   = zerolog.TraceLevel
	DEBUG   = zerolog.DebugLevel
	INFO    = zerolog.InfoLevel
	WARN    = zerolog.WarnLevel
	ERROR   = zerolog.ErrorLevel
	FATAL   = zerolog.FatalLevel
	DEFAULT = "default"
)

type Logger interface {
	Trace(format string, args ...any)
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}
