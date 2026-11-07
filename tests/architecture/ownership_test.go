// Package architecture enforces a small set of testable BDSPro package
// ownership contracts, not a copy of Gitea's complete dependency graph.
package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/datvtph41107/bdspro"

// TestSourcePackageOwnership checks production imports only. Integration
// tests may cross package boundaries to verify real end-to-end behavior.
func TestSourcePackageOwnership(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, owner := range []string{"modules", "models", "services", "routers"} {
		t.Run(owner, func(t *testing.T) {
			err := filepath.WalkDir(filepath.Join(root, owner), func(path string, e fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
					return nil
				}
				file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				for _, imp := range file.Imports {
					dep, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						return err
					}
					isRouter := strings.HasPrefix(dep, modulePath+"/routers/")
					isService := strings.HasPrefix(dep, modulePath+"/services/")
					isModel := strings.HasPrefix(dep, modulePath+"/models/")
					if strings.HasPrefix(dep, modulePath+"/internal/") {
						t.Errorf("%s imports legacy internal package %q", rel, dep)
					}
					if owner == "modules" && (isRouter || isService) {
						t.Errorf("%s: technical module imports transport/business %q", rel, dep)
					}
					if owner == "models" && (isRouter || isService || dep == "net/http" || dep == "github.com/gin-gonic/gin") {
						t.Errorf("%s: persistence imports HTTP/business %q", rel, dep)
					}
					if strings.HasPrefix(filepath.ToSlash(rel), "routers/api/response/") && isModel {
						t.Errorf("%s: generic HTTP responses must not register domain model errors %q", rel, dep)
					}
					if owner == "services" && dep == "github.com/jackc/pgx/v5/pgconn" {
						t.Errorf("%s: PostgreSQL constraints must be classified by owning models, not services", rel)
					}
					// Gin is allowed only in the HTTP-specific context adapter, not business services.
					httpAdapter := strings.HasPrefix(filepath.ToSlash(rel), "services/context/")
					if owner == "services" && (isRouter || (dep == "github.com/gin-gonic/gin" && !httpAdapter)) {
						t.Errorf("%s: business service depends on HTTP delivery %q", rel, dep)
					}
					if strings.HasPrefix(filepath.ToSlash(rel), "routers/common/") && (isModel || isService) {
						t.Errorf("%s: common HTTP protocol imports business/persistence %q", rel, dep)
					}
					if dep == "gopkg.in/ini.v1" && !strings.HasPrefix(filepath.ToSlash(rel), "modules/setting/") {
						t.Errorf("%s: INI decoding must be owned by modules/setting, got %q", rel, dep)
					}
					if dep == "github.com/rs/zerolog/log" && !strings.HasPrefix(filepath.ToSlash(rel), "modules/log/") {
						t.Errorf("%s: application logging should use modules/log, got %q", rel, dep)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
