package cmd

import (
    "fmt"

    "github.com/datvtph41107/bdspro/modules/log"
    "github.com/urfave/cli/v3"
)

type AppVersion struct { Version, Extra string }

func NewMainApp(appVer AppVersion) *cli.Command {
    app := &cli.Command{}
    app.Name = "bdspro"
    app.Usage = "BDSPro backend services"
    app.Description = "HTTP API and background jobs"
    app.Version = appVer.Version + appVer.Extra
    app.EnableShellCompletion = true
    app.Flags = []cli.Flag{
        &cli.StringFlag{Name: "config", Aliases: []string{"c"}, Value: "docker/root/etc/templates/app.ini", Usage: "Application app.ini path"},
    }
    webCmd := newWebCommand()
    subCmdWithConfig := []*cli.Command{webCmd, newWorkerCommand()}
    app.DefaultCommand = webCmd.Name
    app.Before = PrepareConsoleLoggerLevel(log.INFO)
    app.Commands = append(app.Commands, subCmdWithConfig...)
    return app
}

func RunMainApp(app *cli.Command, args ...string) error {
    ctx, cancel := installSignals()
    defer cancel()
    err := app.Run(ctx, args)
    if err != nil {
        _, _ = fmt.Fprintf(app.ErrWriter, "Command error: %v\n", err)
        cli.OsExiter(1)
    }
    return err
}
