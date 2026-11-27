package phonedelivery

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Protect the specific boundary this feature relies on; this is not a whole-repo architecture audit.
func TestPhoneDeliveryDependencyBoundaries(t *testing.T) {
	const root = "github.com/datvtph41107/bdspro/"
	for _, directory := range []string{"../../modules/phonedelivery", ".", "../auth"} {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range file.Imports {
				name, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				switch directory {
				case "../../modules/phonedelivery":
					if strings.HasPrefix(name, root) {
						t.Errorf("%s: reusable transport depends on application package %s", path, name)
					}
				case ".":
					if strings.HasPrefix(name, root+"routers") || strings.HasPrefix(name, root+"models") || name == root+"services/auth" {
						t.Errorf("%s: delivery orchestrator depends on auth/HTTP/storage: %s", path, name)
					}
				case "../auth":
					if name == root+"modules/phonedelivery" || name == root+"modules/sms" {
						t.Errorf("%s: authentication bypasses delivery policy: %s", path, name)
					}
				}
			}
		}
	}
}
