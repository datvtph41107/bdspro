package auth

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A compiling Go program may still have an undesirable outward dependency.
// Registration must not import the queue runtime; the worker adapter owns it.
func TestAuthSourceDoesNotDependOnWorkerRuntime(t *testing.T) {
	files, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		source, err := parser.ParseFile(token.NewFileSet(), filepath.Join(".", f.Name()), nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, spec := range source.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			require.NoError(t, err)
			require.NotEqual(t, "github.com/datvtph41107/bdspro/worker", path, f.Name())
			require.NotEqual(t, "github.com/hibiken/asynq", path, f.Name())
		}
	}
}
