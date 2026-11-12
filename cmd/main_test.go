package cmd

import "testing"

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
