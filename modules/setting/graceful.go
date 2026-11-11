package setting

import (
	"fmt"
	"time"
)

// GracefulSettings defines process shutdown policy, not HTTP routing policy.
type GracefulSettings struct {
	HTTPShutdownTimeout time.Duration
	HammerTimeout       time.Duration
}

func loadGracefulSettingsFrom(cfg ConfigProvider) (GracefulSettings, error) {
	sec := cfg.Section("server")
	result := GracefulSettings{
		HTTPShutdownTimeout: 10 * time.Second,
		HammerTimeout:       15 * time.Second,
	}
	if key := sec.Key("HTTP_SHUTDOWN_TIMEOUT"); key.IsSet() {
		value, err := time.ParseDuration(key.String())
		if err != nil {
			return GracefulSettings{}, fmt.Errorf("server.HTTP_SHUTDOWN_TIMEOUT: %w", err)
		}
		result.HTTPShutdownTimeout = value
	}
	if key := sec.Key("GRACEFUL_HAMMER_TIMEOUT"); key.IsSet() {
		value, err := time.ParseDuration(key.String())
		if err != nil {
			return GracefulSettings{}, fmt.Errorf("server.GRACEFUL_HAMMER_TIMEOUT: %w", err)
		}
		result.HammerTimeout = value
	}
	if result.HTTPShutdownTimeout <= 0 || result.HammerTimeout <= result.HTTPShutdownTimeout {
		return GracefulSettings{}, fmt.Errorf("graceful hammer timeout must exceed positive HTTP shutdown timeout")
	}
	return result, nil
}
