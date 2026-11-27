package setting

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/datvtph41107/bdspro/modules/log"
)

type LogGlobalConfig struct {
	Mode                              string
	Format                            string
	Level                             string
	RootPath                          string
	FileName                          string
	ConsoleFormat, FileFormat         string
	MaxSizeMB, MaxBackups, MaxAgeDays int
}

func loadLogGlobalFrom(rootCfg ConfigProvider, workPath string) (LogGlobalConfig, error) {
	if rootCfg == nil {
		return LogGlobalConfig{}, errors.New("log config provider is nil")
	}
	sec := rootCfg.Section("log")
	fileSec := rootCfg.Section("log.file")
	cfg := LogGlobalConfig{
		Mode:     stringOrDefault(sec.Key("MODE"), "console"),
		Format:   stringOrDefault(sec.Key("FORMAT"), "json"),
		Level:    stringOrDefault(sec.Key("LEVEL"), "info"),
		RootPath: stringOrDefault(sec.Key("ROOT_PATH"), "data/log"),
		FileName: stringOrDefault(fileSec.Key("FILE_NAME"), "bdspro.log"),
	}
	cfg.ConsoleFormat = stringOrDefault(rootCfg.Section("log.console").Key("FORMAT"), cfg.Format)
	cfg.FileFormat = stringOrDefault(fileSec.Key("FORMAT"), cfg.Format)
	for _, option := range []struct {
		key      string
		value    *int
		fallback int
	}{
		{"MAX_SIZE_MB", &cfg.MaxSizeMB, 20}, {"MAX_BACKUPS", &cfg.MaxBackups, 5}, {"MAX_AGE_DAYS", &cfg.MaxAgeDays, 7},
	} {
		value, err := strconv.Atoi(stringOrDefault(fileSec.Key(option.key), strconv.Itoa(option.fallback)))
		if err != nil {
			return LogGlobalConfig{}, fmt.Errorf("log.file.%s must be an integer", option.key)
		}
		*option.value = value
	}
	if workPath == "" || !filepath.IsAbs(workPath) {
		return LogGlobalConfig{}, errors.New("application work path must be absolute")
	}
	if cfg.RootPath == "" {
		return LogGlobalConfig{}, errors.New("log.ROOT_PATH must not be empty")
	}
	if !filepath.IsAbs(cfg.RootPath) {
		cfg.RootPath = filepath.Join(workPath, cfg.RootPath)
	}
	cfg.RootPath = filepath.Clean(cfg.RootPath)
	if err := cfg.Validate(); err != nil {
		return LogGlobalConfig{}, fmt.Errorf("invalid log configuration: %w", err)
	}
	return cfg, nil
}

func (cfg LogGlobalConfig) Validate() error {
	if cfg.Format != "json" && cfg.Format != "console" {
		return errors.New("log.FORMAT must be console or json")
	}
	if !filepath.IsAbs(cfg.RootPath) {
		return errors.New("log.ROOT_PATH must be absolute")
	}
	return cfg.OutputOptions().Validate()
}

// OutputOptions maps validated application settings to the writer package.
func (cfg LogGlobalConfig) OutputOptions() log.OutputOptions {
	return log.OutputOptions{
		Mode: cfg.Mode, ConsoleFormat: cfg.ConsoleFormat, FileFormat: cfg.FileFormat,
		Level: strings.ToLower(cfg.Level), RootPath: cfg.RootPath, FileName: cfg.FileName,
		MaxSizeMB: cfg.MaxSizeMB, MaxBackups: cfg.MaxBackups, MaxAgeDays: cfg.MaxAgeDays,
	}
}

func InitLoggers() error {
	if err := log.GetManager().ConfigureOutputs(Common.Log.OutputOptions()); err != nil {
		return fmt.Errorf("configure logging outputs: %w", err)
	}
	return nil
}
