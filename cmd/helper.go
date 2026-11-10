package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

func installSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func PrepareConsoleLoggerLevel(level log.Level) func(context.Context, *cli.Command) (context.Context, error) {
	return func(ctx context.Context, command *cli.Command) (context.Context, error) {
		if err := log.SetupStderrLogger(level); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

func loadSettings(command *cli.Command) (setting.Config, error) {
	paths, err := resolveConfigPaths(command)
	filename := paths.ConfigFile
	if err != nil {
		return setting.Config{}, err
	}
	log.Info("Using configuration file: %s", filename)
	cfg, err := setting.LoadConfig(filename)
	if err != nil {
		return cfg, fmt.Errorf("load config %q: %w", filename, err)
	}
	if err := configureCommandLoggers(paths, setting.LogGlobalConfig{
		Mode: cfg.LogMode, Format: cfg.LogFormat, Level: cfg.LogLevel,
		RootPath: cfg.LogRootPath, FileName: cfg.LogFileName,
	}); err != nil {
		return cfg, err
	}
	return cfg, nil
}
func isValidDefaultSubCommand(command *cli.Command) (string, bool) {
	if command.Args().Len() != 0 {
		return command.Args().First(), false
	}
	return "", true
}

// resolveConfigPaths is the canonical source for both the selected app.ini
// and the application work path used by relative runtime resources.
func resolveConfigPaths(command *cli.Command) (setting.Paths, error) {
	paths, err := setting.ResolvePaths(setting.PathOptions{
		Config:   command.Root().String("config"),
		WorkPath: command.Root().String("work-path"),
	})
	if err != nil {
		return setting.Paths{}, fmt.Errorf("resolve config path: %w", err)
	}
	return paths, nil
}

// configureCommandLoggers is the single command-level logger initialization
// path. It is called only after the selected command's settings validate.
func configureCommandLoggers(paths setting.Paths, logger setting.LogGlobalConfig) error {
	// Gitea resolves relative runtime paths against its application work
	// path. Do not redirect an absolute ROOT_PATH or the app.ini location.
	if !filepath.IsAbs(logger.RootPath) {
		logger.RootPath = filepath.Join(paths.WorkPath, logger.RootPath)
	}
	if err := setting.InitLoggers(logger); err != nil {
		return fmt.Errorf("initialize loggers: %w", err)
	}
	setting.CustomConf = paths.ConfigFile
	return nil
}

// Database-only commands must not require HTTP, Redis, JWT or OAuth settings.
func loadDatabaseCommandSettings(command *cli.Command) (setting.DatabaseCommandSettings, error) {
	paths, err := resolveConfigPaths(command)
	filename := paths.ConfigFile
	if err != nil {
		return setting.DatabaseCommandSettings{}, err
	}
	cfg, err := setting.LoadDatabaseCommandSettings(filename)
	if err != nil {
		return cfg, fmt.Errorf("load database command settings %q: %w", filename, err)
	}
	if err := configureCommandLoggers(paths, cfg.Log); err != nil {
		return cfg, err
	}
	log.Info("Using configuration file: %s", filename)
	return cfg, nil
}

func loadCleanupSettings(command *cli.Command) (setting.CleanupSettings, error) {
	paths, err := resolveConfigPaths(command)
	filename := paths.ConfigFile
	if err != nil {
		return setting.CleanupSettings{}, err
	}
	cfg, err := setting.LoadCleanupSettings(filename)
	if err != nil {
		return cfg, fmt.Errorf("load cleanup settings %q: %w", filename, err)
	}
	if err := configureCommandLoggers(paths, cfg.Log); err != nil {
		return cfg, err
	}
	log.Info("Using configuration file: %s", filename)
	return cfg, nil
}

func loadWorkerSettings(command *cli.Command) (setting.WorkerSettings, error) {
	paths, err := resolveConfigPaths(command)
	filename := paths.ConfigFile
	if err != nil {
		return setting.WorkerSettings{}, err
	}
	cfg, err := setting.LoadWorkerSettings(filename)
	if err != nil {
		return cfg, fmt.Errorf("load worker settings %q: %w", filename, err)
	}
	if err := configureCommandLoggers(paths, cfg.Log); err != nil {
		return cfg, err
	}
	log.Info("Using configuration file: %s", filename)
	return cfg, nil
}

// loadMigrationSettings shares the database and logger initialization policy
// with other database-only commands, without requiring HTTP or Redis settings.
func loadMigrationSettings(command *cli.Command) (setting.DatabaseSettings, error) {
	paths, err := resolveConfigPaths(command)
	filename := paths.ConfigFile
	if err != nil {
		return setting.DatabaseSettings{}, err
	}
	cfg, err := setting.LoadDatabaseSettings(filename)
	if err != nil {
		return cfg, fmt.Errorf("load migration settings %q: %w", filename, err)
	}
	if err := configureCommandLoggers(paths, cfg.Log); err != nil {
		return cfg, err
	}
	log.Info("Using configuration file: %s", filename)
	return cfg, nil
}
