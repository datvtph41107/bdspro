package log

import (
    "context"
    "fmt"
    "os"

    "github.com/rs/zerolog"
    zerologlog "github.com/rs/zerolog/log"
)

func GetLogger(name string) Logger { return GetManager().GetLogger(name) }
func GetLevel() Level { return GetManager().GetLogger(DEFAULT).GetLevel() }
func Trace(f string, v ...any) { GetLogger(DEFAULT).Trace(f, v...) }
func Debug(f string, v ...any) { GetLogger(DEFAULT).Debug(f, v...) }
func Info(f string, v ...any)  { GetLogger(DEFAULT).Info(f, v...) }
func Warn(f string, v ...any)  { GetLogger(DEFAULT).Warn(f, v...) }
func Error(f string, v ...any) { GetLogger(DEFAULT).Error(f, v...) }

// Event preserves typed fields for HTTP, security and queue diagnostics.
func Event(level Level) *zerolog.Event { return zerologlog.WithLevel(level) }
func With() zerolog.Context { return zerologlog.With() }
func Ctx(ctx context.Context) *zerolog.Logger { return zerologlog.Ctx(ctx) }
func WithLevel(level Level) *zerolog.Event { return zerologlog.WithLevel(level) }

func Fatal(f string, v ...any) {
    Error(f, v...)
    GetManager().Close()
    _, _ = fmt.Fprintln(os.Stderr, "fatal: process terminated")
    os.Exit(1)
}
