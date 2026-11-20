package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/urfave/cli/v3"
)

func installSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
}

func PrepareConsoleLoggerLevel(level log.Level) func(context.Context, *cli.Command) (context.Context, error) {
	return func(ctx context.Context, _ *cli.Command) (context.Context, error) {
		if err := log.SetupStderrLogger(level); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

func rejectArguments(name string) func(context.Context, *cli.Command) (context.Context, error) {
	return func(ctx context.Context, command *cli.Command) (context.Context, error) {
		if command.Args().Len() != 0 {
			if name == "web" {
				return ctx, fmt.Errorf("unknown command: %s", command.Args().First())
			}
			return ctx, fmt.Errorf("unexpected %s argument %q", name, command.Args().First())
		}
		return ctx, nil
	}
}
