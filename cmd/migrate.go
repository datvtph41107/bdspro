package cmd

import (
    "context"
    "errors"
    "fmt"

    "github.com/datvtph41107/bdspro/models/db"
    "github.com/datvtph41107/bdspro/modules/log"
    "github.com/datvtph41107/bdspro/modules/setting"
    "github.com/urfave/cli/v3"
)

func newMigrateCommand() *cli.Command {
    return &cli.Command{
        Name:"migrate",
        Usage:"Apply versioned SQL migrations using the same database config as web",
        Flags:[]cli.Flag{
            &cli.IntFlag{Name:"steps",Usage:"Number of migration versions (0 means all)"},
            &cli.BoolFlag{Name:"down",Usage:"Roll back schema (development only)"},
        },
        Action:runMigrate,
    }
}

func runMigrate(ctx context.Context, command *cli.Command) error {
    if command.Args().Len()!=0 {return fmt.Errorf("unexpected migrate argument: %s",command.Args().First())}
    filename,err:=resolveConfigFile(command)
    if err!=nil {return err}
    log.Info("Using configuration file: %s",filename)
    cfg,err:=setting.LoadDatabaseSettings(filename)
    if err!=nil {return err}
    steps:=command.Int("steps")
    if steps<0 {return errors.New("--steps cannot be negative")}
    if command.Bool("down") {
        if cfg.Environment!="development" {return errors.New("schema rollback is restricted to development")}
        changed,err:=db.MigrateDown(cfg.MigrationURL,cfg.DBSource,steps)
        if err!=nil {return err}
        if changed {log.Info("Migration status: rolled_back")} else {log.Info("Migration status: no_change")}
        return nil
    }
    changed,err:=db.MigrateUp(cfg.MigrationURL,cfg.DBSource,steps)
    if err!=nil {return err}
    if changed {log.Info("Migration status: applied")} else {log.Info("Migration status: no_change")}
    return nil
}
