package setting

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/datvtph41107/bdspro/modules/log"
)

type LogGlobalConfig struct {
	Mode     string
	Format   string
	Level    string
	RootPath string
	FileName string
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
	switch cfg.Mode {
	case "console", "file", "console,file", "file,console":
	default:
		return errors.New("log.MODE must be console, file or console,file")
	}
	switch cfg.Format {
	case "console", "json":
	default:
		return errors.New("log.FORMAT must be console or json")
	}
	switch strings.ToLower(cfg.Level) {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("log.LEVEL must be debug, info, warn or error")
	}
	if cfg.RootPath == "" || !filepath.IsAbs(cfg.RootPath) {
		return errors.New("log.ROOT_PATH must be an absolute path")
	}
	if cfg.FileName == "" || cfg.FileName == "." || cfg.FileName == ".." || filepath.Base(cfg.FileName) != cfg.FileName {
		return errors.New("log.file.FILE_NAME must be a filename")
	}
	return nil
}

func InitLoggers() error {
	cfg := Common.Log
	if err := log.GetManager().ConfigureOutputs(
		cfg.Mode, cfg.Format, strings.ToLower(cfg.Level), cfg.RootPath, cfg.FileName,
	); err != nil {
		return fmt.Errorf("configure logging outputs: %w", err)
	}
	return nil
}
