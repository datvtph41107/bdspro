package log

import "github.com/rs/zerolog"

// LoggerImpl implements the formatted logger API. Structured events use Event
// or a logger stored in a request context, never formatted fields in messages.
type LoggerImpl struct {
    inner zerolog.Logger
}
func (l *LoggerImpl) Trace(f string, v ...any) { l.inner.Trace().Msgf(f, v...) }
func (l *LoggerImpl) Debug(f string, v ...any) { l.inner.Debug().Msgf(f, v...) }
func (l *LoggerImpl) Info(f string, v ...any)  { l.inner.Info().Msgf(f, v...) }
func (l *LoggerImpl) Warn(f string, v ...any)  { l.inner.Warn().Msgf(f, v...) }
func (l *LoggerImpl) Error(f string, v ...any) { l.inner.Error().Msgf(f, v...) }
func (l *LoggerImpl) GetLevel() Level          { return l.inner.GetLevel() }
