package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Verify the application shape rather than forbidding domain packages.
// BDSPro is rebuilding models/<domain> and services/<domain>; a legacy
// deletion test must not make that legitimate work fail.
func TestRequiredRuntimeStructure(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, name := range []string{
		"db/migration", "db/query", "models/db/sqlc",
		"models/party", "modules/web", "modules/graceful",
		"services/context", "services/jobs", "services/worker",
		"routers/common", "routers/init.go", "routers/api/v1/api.go",
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("required runtime source %s: %v", name, err)
		}
	}
}

// Framework libraries are implementation details of the HTTP boundary.
// These checks intentionally allow services/context and modules/web to
// depend on a router during migration, but not business services/models.
// The check covers any new domain packages without maintaining a list.
func TestDomainDoesNotImportHTTPFramework(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, top := range []string{"models", "services", "routers"} {
		base := filepath.Join(root, top)
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if top == "services" && path == filepath.Join(base, "context") {
					return filepath.SkipDir // HTTP application boundary
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Errorf("parse %s: %v", path, err)
				return nil
			}
			for _, spec := range file.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Errorf("invalid import in %s: %v", path, err)
					continue
				}
				// Router packages compose HTTP actions but must not
				// import framework-specific types. Only modules/web
				// owns the matcher and its transport configuration.
				if top == "routers" {
					if importPath == "github.com/gin-gonic/gin" ||
						importPath == "github.com/go-chi/chi/v5" {
						t.Errorf("%s imports route matcher %s", path, importPath)
					}
					continue
				}
				switch {
				case importPath == "github.com/gin-gonic/gin",
					importPath == "github.com/go-chi/chi/v5",
					strings.HasPrefix(importPath, "github.com/datvtph41107/bdspro/routers/"),
					importPath == "github.com/datvtph41107/bdspro/services/context":
					t.Errorf("%s imports HTTP layer %s", path, importPath)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
