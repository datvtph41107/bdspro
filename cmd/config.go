package cmd

import (
	"context"
	"os"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

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
