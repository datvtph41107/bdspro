//go:build !windows

package graceful

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Subprocesses avoid delivering SIGTERM/SIGINT to the test runner itself.
func TestRealUnixSignals(t *testing.T) {
	if os.Getenv("GRACEFUL_SIGNAL_CHILD") == "1" {
		cliCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		g := newGracefulManager(cliCtx, time.Second)
		if err := g.RegisterServer(); err != nil {
			panic(err)
		}
		fmt.Println("READY")
		<-g.IsShutdown()
		fmt.Println("SHUTDOWN")
		g.ServerDone()
		<-g.Done()
		fmt.Println("DONE")
		return
	}
	for _, tc := range []struct {
		name string
		sig  syscall.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestRealUnixSignals$")
			cmd.Env = append(os.Environ(), "GRACEFUL_SIGNAL_CHILD=1")
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			lines := make(chan string, 10)
			go func() {
				sc := bufio.NewScanner(stdout)
				for sc.Scan() {
					lines <- sc.Text()
				}
				close(lines)
			}()
			expect := func(want string) {
				t.Helper()
				select {
				case got, ok := <-lines:
					if !ok || strings.TrimSpace(got) != want {
						t.Fatalf("got %q, want %q", got, want)
					}
				case <-time.After(5 * time.Second):
					t.Fatalf("timeout waiting for %s", want)
				}
			}
			expect("READY")
			if err := cmd.Process.Signal(tc.sig); err != nil {
				t.Fatal(err)
			}
			expect("SHUTDOWN")
			expect("DONE")
			if err := cmd.Wait(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
