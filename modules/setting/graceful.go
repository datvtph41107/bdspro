package setting

import (
	"fmt"
	"time"
)

// GracefulSettings là deadline của process shutdown.
type GracefulSettings struct {
	HTTPShutdownTimeout time.Duration `ini:"HTTP_SHUTDOWN_TIMEOUT"`
	HammerTimeout       time.Duration `ini:"GRACEFUL_HAMMER_TIMEOUT"`
}

var Graceful GracefulSettings

const defaultHTTPShutdownTimeout = 10 * time.Second

// HTTPShutdownTimeout trả về deadline HTTP drain, kể cả khi chạy test riêng.
func HTTPShutdownTimeout() time.Duration {
	if Graceful.HTTPShutdownTimeout > 0 {
		return Graceful.HTTPShutdownTimeout
	}
	return defaultHTTPShutdownTimeout
}

func loadGracefulSettingsFrom(cfg ConfigProvider) (GracefulSettings, error) {
	result := GracefulSettings{
		HTTPShutdownTimeout: defaultHTTPShutdownTimeout,
		HammerTimeout:       15 * time.Second,
	}
	if err := mapSettings(cfg, "server", &result); err != nil {
		return GracefulSettings{}, err
	}
	if result.HTTPShutdownTimeout <= 0 || result.HammerTimeout <= result.HTTPShutdownTimeout {
		return GracefulSettings{}, fmt.Errorf("graceful hammer timeout must exceed positive HTTP shutdown timeout")
	}
	return result, nil
}
