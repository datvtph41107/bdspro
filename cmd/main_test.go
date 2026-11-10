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
	if !names["web"] || !names["worker"] || !names["migrate"] || !names["cleanup"] || !names["doctor"] {
		t.Fatalf("commands=%v", names)
	}
	var output bytes.Buffer
	app.Writer = &output
	if err := app.Run(context.Background(), []string{"bdspro", "--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "worker") || !strings.Contains(output.String(), "cleanup") || !strings.Contains(output.String(), "doctor") {
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

func TestDoctorRejectsArgumentsBeforeOpeningDatabase(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	if err := app.Run(context.Background(), []string{"bdspro", "doctor", "unexpected"}); err == nil ||
		!strings.Contains(err.Error(), "unexpected doctor argument") {
		t.Fatalf("doctor accepted extra arguments or reached database: %v", err)
	}
}

func TestDoctorRejectsNegativeCursorBeforeOpeningDatabase(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	if err := app.Run(context.Background(), []string{"bdspro", "doctor", "--after-id", "-1"}); err == nil ||
		!strings.Contains(err.Error(), "--after-id cannot be negative") {
		t.Fatalf("doctor accepted negative cursor or reached config/database: %v", err)
	}
}

func TestServeAliasRejectsArgumentsBeforeLoadingConfig(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	err := app.Run(context.Background(), []string{"bdspro", "serve", "unexpected"})
	if err == nil || !strings.Contains(err.Error(), "unknown command: unexpected") {
		t.Fatalf("serve alias failed to dispatch to web command validation: %v", err)
	}
}

func TestWorkerRejectsArgumentsBeforeLoadingConfig(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	err := app.Run(context.Background(), []string{"bdspro", "worker", "unexpected"})
	if err == nil || !strings.Contains(err.Error(), "unexpected worker argument") {
		t.Fatalf("worker parsed config before rejecting argument: %v", err)
	}
}

func TestHelpCommandRequiresNoConfigurationOrDependencies(t *testing.T) {
	app := NewMainApp(AppVersion{Version: "test"})
	var output bytes.Buffer
	app.Writer = &output
	if err := app.Run(context.Background(), []string{"bdspro", "help"}); err != nil {
		t.Fatalf("built-in help must work without app.ini: %v", err)
	}
	for _, name := range []string{"web", "migrate", "worker"} {
		if !strings.Contains(output.String(), name) {
			t.Fatalf("help output omitted %s: %s", name, output.String())
		}
	}
}
