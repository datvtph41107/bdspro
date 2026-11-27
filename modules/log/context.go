package log

import (
	"context"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

// For preserves request correlation while identifying the owner of an event.
// Process work without an attached request logger uses the configured default.
func For(ctx context.Context, component string) zerolog.Logger {
	logger := zerologlog.Logger
	if ctx != nil {
		if attached := zerolog.Ctx(ctx); attached.GetLevel() != zerolog.Disabled {
			logger = *attached
		}
	}
	return logger.With().Str("component", component).Logger()
}
