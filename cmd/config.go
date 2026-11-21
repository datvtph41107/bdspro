package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

// Config preflight does not inherit Web/Migrate startup or logger Before hooks.
func newConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Inspect application configuration",
		Commands: []*cli.Command{
			{
				Name:   "check",
				Usage:  "Validate Web settings without opening resources",
				Before: rejectArguments("config check"),
				Action: runConfigCheck,
			},
		},
	}
}

func runConfigCheck(_ context.Context, cmd *cli.Command) error {
	if err := setting.CheckWebConfig(configPathOptions(cmd), os.Environ()); err != nil {
		return fmt.Errorf("config check: %w", err)
	}
	writer := cmd.Root().Writer
	if writer == nil {
		writer = os.Stdout
	}
	_, err := fmt.Fprintln(writer, "Configuration valid")
	return err
}
