package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

func TestWebConfigBootstrapBeforeAction(t *testing.T) {
	workPath := t.TempDir()
	conf := filepath.Join(workPath, "custom", "conf", "app.ini")
	if err := os.MkdirAll(filepath.Dir(conf), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conf, []byte("RUN_MODE = development\n[log]\nMODE = console\n"), 0600); err != nil {
		t.Fatal(err)
	}

	app := NewMainApp(AppVersion{Version: "test"})
	ran := false
	for _, command := range app.Commands {
		if command.Name == "web" {
			command.Action = func(_ context.Context, _ *cli.Command) error {
				ran = true
				if setting.CfgProvider == nil || setting.CustomConf != conf {
					t.Fatal("CLI did not publish canonical configuration before web action")
				}
				return nil
			}
		}
	}
	if err := app.Run(context.Background(), []string{"bdspro", "--work-path", workPath, "web"}); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("web action not invoked")
	}
}
