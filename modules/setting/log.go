package setting

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/datvtph41107/bdspro/modules/log"
)

// LogGlobalConfig describes the application's
// process-wide logging configuration.
//
// The configuration is loaded by modules/setting.
// Log writers are owned by modules/log.
type LogGlobalConfig struct {
	Mode     string
	Format   string
	Level    string
	RootPath string
	FileName string
}

// loadLogGlobalFrom reads logging configuration
// from the prepared configuration provider.
//
// It does not initialize or replace any log writer.
func loadLogGlobalFrom(
	rootCfg ConfigProvider,
	workPath string,
) (LogGlobalConfig, error) {
	if rootCfg == nil {
		return LogGlobalConfig{},
			fmt.Errorf("log config provider is nil")
	}

	sec := rootCfg.Section("log")
	fileSec := rootCfg.Section("log.file")

	cfg := LogGlobalConfig{
		Mode: stringOrDefault(
			sec.Key("MODE"),
			"console",
		),
		Format: stringOrDefault(
			sec.Key("FORMAT"),
			"json",
		),
		Level: stringOrDefault(
			sec.Key("LEVEL"),
			"info",
		),
		RootPath: stringOrDefault(
			sec.Key("ROOT_PATH"),
			"data/log",
		),
		FileName: stringOrDefault(
			fileSec.Key("FILE_NAME"),
			"bdspro.log",
		),
	}

	if workPath == "" || !filepath.IsAbs(workPath) {
		return LogGlobalConfig{},
			fmt.Errorf(
				"application work path must be absolute",
			)
	}

	// Runtime resources are resolved from
	// AppWorkPath, not from the shell's cwd.
	if !filepath.IsAbs(cfg.RootPath) {
		cfg.RootPath = filepath.Join(
			workPath,
			cfg.RootPath,
		)
	}

	cfg.RootPath = filepath.Clean(cfg.RootPath)

	if err := cfg.Validate(); err != nil {
		return LogGlobalConfig{},
			fmt.Errorf(
				"invalid log configuration: %w",
				err,
			)
	}

	return cfg, nil
}

// Validate checks the logging configuration
// before a runtime logger is initialized.
func (cfg LogGlobalConfig) Validate() error {
	switch cfg.Mode {
	case "console", "file",
		"console,file", "file,console":
	default:
		return fmt.Errorf(
			"log.MODE must be console, file or console,file",
		)
	}

	switch cfg.Format {
	case "console", "json":
	default:
		return fmt.Errorf(
			"log.FORMAT must be console or json",
		)
	}

	switch strings.ToLower(cfg.Level) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf(
			"log.LEVEL must be debug, info, warn or error",
		)
	}

	if cfg.RootPath == "" ||
		!filepath.IsAbs(cfg.RootPath) {
		return fmt.Errorf(
			"log.ROOT_PATH must be an absolute path",
		)
	}

	if cfg.FileName == "" ||
		cfg.FileName == "." ||
		cfg.FileName == ".." ||
		filepath.Base(cfg.FileName) != cfg.FileName {
		return fmt.Errorf(
			"log.file.FILE_NAME must be a filename",
		)
	}

	return nil
}

// InitLoggers activates the validated logging
// configuration.
//
// It does not read app.ini and must not be
// called independently by every subsystem.
func InitLoggers(cfg LogGlobalConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := log.GetManager().ConfigureOutputs(
		cfg.Mode,
		cfg.Format,
		strings.ToLower(cfg.Level),
		cfg.RootPath,
		cfg.FileName,
	); err != nil {
		return fmt.Errorf(
			"initialize logging outputs: %w",
			err,
		)
	}

	return nil
}
