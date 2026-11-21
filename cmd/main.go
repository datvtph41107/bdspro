package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

// AppVersion được main.go truyền vào root CLI.
type AppVersion struct {
	Version string
	Extra   string
}

// NewMainApp chỉ đăng ký CLI; không mở network hay đọc app.ini.
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
			Name: "work-path", Aliases: []string{"w"},
			Usage: "Absolute instance working directory (default: current directory)",
		},
		&cli.StringFlag{
			Name: "config", Aliases: []string{"c"},
			Usage: "Application configuration file (default: ./custom/conf/app.ini)",
		},
	}

	webCmd := newWebCommand()
	app.DefaultCommand = webCmd.Name
	app.Commands = []*cli.Command{webCmd, newMigrateCommand(), newConfigCommand()}
	app.Before = PrepareConsoleLoggerLevel(log.INFO)
	return app
}

// usageErr đánh dấu usage error đã in, tránh in lại từ RunMainApp.
type usageErr struct{ err error }

func (e usageErr) Error() string { return e.err.Error() }
func (e usageErr) Unwrap() error { return e.err }

// cliOnUsageError chặn cli/v3 tự dump toàn bộ help khi sai cú pháp.
func cliOnUsageError(_ context.Context, cmd *cli.Command, err error, _ bool) error {
	if writer := cmd.Root().ErrWriter; writer != nil {
		_, _ = fmt.Fprintf(writer, "Incorrect Usage: %s\n", err)
	}
	return usageErr{err}
}

func setCLIOnUsageError(cmd *cli.Command) {
	_ = cmd.Walk(func(c *cli.Command) error {
		c.OnUsageError = cliOnUsageError
		return nil
	})
}

// RunMainApp cài signal, thực thi CLI và giữ một nơi in process errors.
func RunMainApp(app *cli.Command, args ...string) error {
	ctx, cancel := installSignals()
	defer cancel()
	setCLIOnUsageError(app)
	app.ConfigureShellCompletionCommand = setCLIOnUsageError

	err := app.Run(ctx, args)
	if err == nil {
		return nil
	}
	var alreadyReported usageErr
	if errors.As(err, &alreadyReported) {
		cli.OsExiter(1)
		return err
	}
	if app.ErrWriter != nil {
		_, _ = fmt.Fprintf(app.ErrWriter, "Command error: %v\n", err)
	}
	cli.OsExiter(1)
	return err
}

func configPathOptions(cmd *cli.Command) setting.PathOptions {
	root := cmd.Root()
	var args setting.PathOptions
	if root.IsSet("work-path") {
		args.WorkPath = root.String("work-path")
	}
	if root.IsSet("config") {
		args.Config = root.String("config")
	}
	return args
}

// installSignals belongs to the CLI process entrypoint. It must not be hidden
// in a generic "helper" package, or called by a resource owner.
func installSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// PrepareConsoleLoggerLevel is a CLI Before adapter; modules/log owns the sink.
func PrepareConsoleLoggerLevel(level log.Level) func(context.Context, *cli.Command) (context.Context, error) {
	return func(ctx context.Context, _ *cli.Command) (context.Context, error) {
		if err := log.SetupStderrLogger(level); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

// rejectArguments is shared by CLI subcommands, not a generic module helper.
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
