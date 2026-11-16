package cmd

import (
	"testing"

	"github.com/urfave/cli/v3"
)

func TestInfrastructureCommands(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	want := map[string]bool{"web": false, "migrate": false}
	for _, command := range app.Commands {
		if _, ok := want[command.Name]; !ok {
			t.Errorf("unexpected command after domain reset: %s", command.Name)
			continue
		}
		want[command.Name] = true
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing infrastructure command: %s", name)
		}
	}
}

func TestGlobalPathFlags(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	found := map[string]bool{"work-path": false, "config": false}
	for _, flag := range app.Flags {
		switch f := flag.(type) {
		case *cli.StringFlag:
			if _, ok := found[f.Name]; ok {
				found[f.Name] = true
			} else {
				t.Errorf("unexpected string flag: %s", f.Name)
			}
		}
	}
	for name, present := range found {
		if !present {
			t.Errorf("missing global path flag: %s", name)
		}
	}
}
