package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

// AppVersion contains the application version and
// optional build information provided by main.go.
type AppVersion struct {
	Version string
	Extra   string
}

// NewMainApp constructs the BDSPro CLI application.
//
// This function only registers commands, flags and
// lifecycle hooks. It must not read configuration,
// initialize subsystems or open network connections.
func NewMainApp(appVer AppVersion) *cli.Command {
	app := &cli.Command{
		Name:                  "bdspro",
		Usage:                 "BDSPro backend services",
		Description:           "HTTP API and database maintenance commands",
		Version:               appVer.Version + appVer.Extra,
		EnableShellCompletion: true,
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "work-path",
			Aliases: []string{"w"},
			Usage:   "Absolute application working directory",
		},
		&cli.StringFlag{
			Name:  "custom-path",
			Usage: "Custom configuration and resource directory",
		},
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Application configuration file (default: {custom-path}/conf/app.ini)",
		},
	}

	// A minimal console logger is available before
	// the selected application's config is loaded.
	app.Before = PrepareConsoleLoggerLevel(log.INFO)

	webCmd := newWebCommand()

	commands := []*cli.Command{
		webCmd,
		newMigrateCommand(),
	}

	// Preserve each command's original Before hook,
	// then prepare the canonical configuration provider.
	//
	// This follows Gitea's
	// PrepareSubcommandWithGlobalFlags pattern.
	for _, command := range commands {
		PrepareSubcommandWithGlobalFlags(command)
	}

	app.DefaultCommand = webCmd.Name
	app.Commands = commands

	return app
}

// RunMainApp executes the CLI and manages process
// cancellation on SIGINT or SIGTERM.
//
// Application initialization belongs to command
// actions, not this function.
func RunMainApp(app *cli.Command, args ...string) error {
	ctx, cancel := installSignals()
	defer cancel()

	err := app.Run(ctx, args)
	if err != nil {
		if app.ErrWriter != nil {
			_, _ = fmt.Fprintf(
				app.ErrWriter,
				"Command error: %v\n",
				err,
			)
		}

		cli.OsExiter(1)
	}

	return err
}

// PrepareSubcommandWithGlobalFlags prepares a
// command to use the application's global flags.
//
// The command's existing Before hook is preserved.
// Configuration is initialized once before Action.
func PrepareSubcommandWithGlobalFlags(
	command *cli.Command,
) {
	previousBefore := command.Before

	command.Before = func(
		ctx context.Context,
		current *cli.Command,
	) (context.Context, error) {
		// Preserve command-specific preparation.
		if previousBefore != nil {
			var err error

			ctx, err = previousBefore(ctx, current)
			if err != nil {
				return ctx, err
			}
		}

		if err := prepareWorkPathAndCustomConf(
			current,
		); err != nil {
			return ctx, err
		}

		if err := prepareCommandLoggers(); err != nil {
			return ctx, err
		}

		log.Info(
			"Using configuration file: %s",
			setting.CustomConf,
		)

		return ctx, nil
	}
}

// prepareWorkPathAndCustomConf resolves the
// application's selected configuration.
//
// It delegates all path and provider ownership
// to modules/setting.
func prepareWorkPathAndCustomConf(
	command *cli.Command,
) error {
	root := command.Root()

	options := setting.PathOptions{
		WorkPath:   root.String("work-path"),
		CustomPath: root.String("custom-path"),
		Config:     root.String("config"),
	}

	return setting.InitWorkPathAndCommonConfig(options, os.Environ())
}

// prepareCommandLoggers activates the logger
// using already validated common settings.
//
// It must not read app.ini or environment keys.
func prepareCommandLoggers() error {
	return setting.InitLoggers(setting.Common.Log)
}
