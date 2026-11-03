package queue

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/log"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Print(level log.Level, args ...any) {
	log.WithLevel(level).Msg(fmt.Sprint(args...))
}

func (l *Logger) Printf(ctx context.Context, format string, v ...any) {
	log.Ctx(ctx).Debug().Msgf(format, v...)
}

func (l *Logger) Debug(args ...any) {
	l.Print(log.DEBUG, args...)
}

func (l *Logger) Info(args ...any) {
	l.Print(log.INFO, args...)
}

func (l *Logger) Warn(args ...any) {
	l.Print(log.WARN, args...)
}

func (l *Logger) Error(args ...any) {
	l.Print(log.ERROR, args...)
}

func (l *Logger) Fatal(args ...any) {
	log.Fatal("%s", fmt.Sprint(args...))
}
