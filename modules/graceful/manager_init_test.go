package graceful

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The singleton is process-wide: test each initialization contract in a fresh
// subprocess instead of resetting package globals (which could race watchers).
func TestExplicitInitializationContract(t *testing.T) {
	if scenario := os.Getenv("GRACEFUL_INIT_SCENARIO"); scenario != "" {
		switch scenario {
		case "get-before-init":
			defer func() { fmt.Printf("PANIC:%v\n", recover()) }()
			GetManager()
		case "init-once":
			ctx, cancel := context.WithCancel(context.Background())
			InitManagerWithTimeout(ctx, -1)
			g := GetManager()
			if g == nil {
				panic("manager unexpectedly nil")
			}
			cancel()
			<-g.Done()
			fmt.Println("OK")
		case "duplicate-init":
			InitManagerWithTimeout(context.Background(), -1)
			defer func() { fmt.Printf("PANIC:%v\n", recover()) }()
			InitManagerWithTimeout(context.Background(), -1)
		}
		return
	}
	tests := []struct{ scenario, want string }{
		{"get-before-init", "PANIC:graceful: GetManager called before InitManager"},
		{"init-once", "OK"},
		{"duplicate-init", "PANIC:graceful: InitManager called more than once"},
	}
	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestExplicitInitializationContract$")
			cmd.Env = append(os.Environ(), "GRACEFUL_INIT_SCENARIO="+tt.scenario)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("subprocess failed: %v\n%s", err, output)
			}
			if !strings.Contains(string(output), tt.want) {
				t.Fatalf("got %q; want %q", output, tt.want)
			}
		})
	}
}
