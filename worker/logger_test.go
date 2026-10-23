package worker

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggerFatalExitsWithCodeOne(t *testing.T) {
	if os.Getenv("BDS_TEST_FATAL_CHILD") == "1" {
		NewLogger().Fatal("fatal contract test")
		return
	}

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestLoggerFatalExitsWithCodeOne$",
	)
	cmd.Env = append(os.Environ(), "BDS_TEST_FATAL_CHILD=1")

	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())
}
