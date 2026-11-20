package setting

import (
	"fmt"
	"time"
)

// GracefulSettings là deadline của process shutdown.
type GracefulSettings struct {
	HTTPShutdownTimeout time.Duration
	HammerTimeout       time.Duration
}

const defaultHTTPShutdownTimeout = 10 * time.Second

// HTTPShutdownTimeout trả về deadline HTTP drain, kể cả khi chạy test riêng.
func HTTPShutdownTimeout() time.Duration {
	if Common.Graceful.HTTPShutdownTimeout > 0 {
		return Common.Graceful.HTTPShutdownTimeout
	}
	return defaultHTTPShutdownTimeout
}

func loadGracefulSettingsFrom(cfg ConfigProvider) (GracefulSettings, error) {
	sec := cfg.Section("server")
	result := GracefulSettings{HTTPShutdownTimeout: defaultHTTPShutdownTimeout, HammerTimeout: 15 * time.Second}
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
