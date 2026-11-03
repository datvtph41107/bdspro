// Package architecture enforces verifiable source ownership constraints.
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

func TestSourcePackageOwnership(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, owner := range []string{"modules", "models", "services", "routers"} {
		t.Run(owner, func(t *testing.T) {
			err := filepath.WalkDir(filepath.Join(root, owner), func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
					return nil
				}
				file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
				if err != nil {
					return err
				}
				relative, err := filepath.Rel(root, path)
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
						t.Errorf("%s imports legacy internal path %q", relative, dep)
					}
					if owner == "modules" && (isRouter || isService) {
						t.Errorf("%s: module imports delivery/business package %q", relative, dep)
					}
					if owner == "models" && (isRouter || isService) {
						t.Errorf("%s: persistence imports delivery/business package %q", relative, dep)
					}
					if owner == "services" && (isRouter || dep == "github.com/gin-gonic/gin") {
						t.Errorf("%s: service imports HTTP delivery %q", relative, dep)
					}
					if strings.HasPrefix(filepath.ToSlash(relative), "routers/common/") && (isModel || isService) {
						t.Errorf("%s: common HTTP protocol imports persistence/business %q", relative, dep)
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
