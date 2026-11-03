package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
	filename, err := resolveConfigFile(command)
	if err != nil {
		return setting.Config{}, err
	}
	log.Info("Using configuration file: %s", filename)
	cfg, err := setting.LoadConfig(filename)
	if err != nil {
		return cfg, fmt.Errorf("load config %q: %w", filename, err)
	}
	if err := setting.InitLoggers(setting.LogGlobalConfig{
		Mode: cfg.LogMode, Format: cfg.LogFormat, Level: cfg.LogLevel,
		RootPath: cfg.LogRootPath, FileName: cfg.LogFileName,
	}); err != nil {
		return cfg, fmt.Errorf("initialize loggers: %w", err)
	}
	setting.CustomConf = filename
	return cfg, nil
}
func isValidDefaultSubCommand(command *cli.Command) (string, bool) {
	if command.Args().Len() != 0 {
		return command.Args().First(), false
	}
	return "", true
}

func resolveConfigFile(command *cli.Command) (string, error) {
	paths, err := setting.ResolvePaths(setting.PathOptions{
		Config:   command.Root().String("config"),
		WorkPath: command.Root().String("work-path"),
	})
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	return paths.ConfigFile, nil
}
