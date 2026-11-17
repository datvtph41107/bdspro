package cmd

import (
	"context"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/urfave/cli/v3"
)

func newWebCommand() *cli.Command {
	return &cli.Command{
		Name:    "web",
		Aliases: []string{"serve"},
		Usage:   "Start the BDSPro HTTP API",
		Before:  rejectArguments("web"),
		Action:  runWeb,
	}
}

func runWeb(ctx context.Context, _ *cli.Command) error {
	managerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	graceful.InitManager(managerCtx)

	return serveInstalled(managerCtx)
}
