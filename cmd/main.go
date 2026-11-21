package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

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
	subCmdWithConfig := []*cli.Command{
		webCmd,
		newMigrateCommand(),
	}
	for _, cmd := range subCmdWithConfig {
		PrepareSubcommandWithGlobalFlags(cmd)
	}
	app.DefaultCommand = webCmd.Name
	app.Commands = append(app.Commands, subCmdWithConfig...)
	app.Commands = append(app.Commands, newConfigCommand())
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

// PrepareSubcommandWithGlobalFlags giữ Before gốc rồi nạp settings + logger.
func PrepareSubcommandWithGlobalFlags(originCmd *cli.Command) {
	originBefore := originCmd.Before
	originCmd.Before = func(ctxOrig context.Context, cmd *cli.Command) (ctx context.Context, err error) {
		ctx = ctxOrig
		if originBefore != nil {
			ctx, err = originBefore(ctx, cmd)
			if err != nil {
				return ctx, err
			}
		}
		if err = prepareConfig(cmd); err != nil {
			return ctx, err
		}
		if err = prepareCommandLoggers(); err != nil {
			return ctx, err
		}
		log.Info("Using configuration file: %s", setting.CustomConf)
		return ctx, nil
	}
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

func prepareConfig(cmd *cli.Command) error {
	return setting.InitWorkPathAndCommonConfig(configPathOptions(cmd), os.Environ())
}

func prepareCommandLoggers() error {
	return setting.InitLoggers(setting.Common.Log)
}
