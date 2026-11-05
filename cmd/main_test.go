package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMainAppExposesWebAndWorker(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	if app.DefaultCommand != "web" {
		t.Fatalf("default command=%q", app.DefaultCommand)
	}
	names := map[string]bool{}
	for _, command := range app.Commands {
		names[command.Name] = true
	}
	if !names["web"] || !names["worker"] || !names["migrate"] || !names["cleanup"] {
		t.Fatalf("commands=%v", names)
	}
	var output bytes.Buffer
	app.Writer = &output
	if err := app.Run(context.Background(), []string{"bdspro", "--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "worker") || !strings.Contains(output.String(), "cleanup") {
		t.Fatalf("missing worker help: %s", output.String())
	}
}
func TestUnknownCommandNeverStartsRuntime(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	var output bytes.Buffer
	app.ErrWriter = &output
	if err := app.Run(context.Background(), []string{"bdspro", "bad-command"}); err == nil {
		t.Fatal("unknown command entered runtime")
	}
}

func TestCleanupRejectsArgumentsBeforeOpeningDatabase(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	if err := app.Run(context.Background(), []string{"bdspro", "cleanup", "unexpected"}); err == nil ||
		!strings.Contains(err.Error(), "unexpected cleanup argument") {
		t.Fatalf("cleanup accepted extra arguments or reached dependencies: %v", err)
	}
}
