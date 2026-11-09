package setting

import (
	"fmt"

	"github.com/datvtph41107/bdspro/modules/log"
)

// LogGlobalConfig configures the process logger, not an HTTP-only component.
type LogGlobalConfig struct {
	Mode, Format, Level, RootPath, FileName string
}

func (c LogGlobalConfig) Validate() error {
	if c.Mode != "console" && c.Mode != "file" &&
		c.Mode != "console,file" && c.Mode != "file,console" {
		return fmt.Errorf("LOG_MODE must be console, file or console,file")
	}
	if c.Format != "json" && c.Format != "console" {
		return fmt.Errorf("LOG_FORMAT must be console or json")
	}
	switch c.Level {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
	}
}

// LoadLogSettings reuses the same defaults for web, worker and maintenance.
func LoadLogSettings(provider *ConfigProvider) LogGlobalConfig {
	common := LoadCommonSettings(provider)
	return LogGlobalConfig{
		Mode: common.LogMode, Format: common.LogFormat, Level: common.LogLevel,
		RootPath: common.LogRootPath, FileName: common.LogFileName,
	}
}

func InitLoggers(c LogGlobalConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	return log.GetManager().ConfigureOutputs(c.Mode, c.Format, c.Level, c.RootPath, c.FileName)
}
