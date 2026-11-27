package phonedelivery

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Protect the specific boundary this feature relies on; this is not a whole-repo architecture audit.
func TestPhoneDeliveryDependencyBoundaries(t *testing.T) {
	const root = "github.com/datvtph41107/bdspro/"
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := entry.Name()
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			name, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(name, root+"routers") || strings.HasPrefix(name, root+"models") || name == root+"services/auth" {
				t.Errorf("%s: delivery orchestrator depends on auth/HTTP/storage: %s", path, name)
			}
		}
	}
}
