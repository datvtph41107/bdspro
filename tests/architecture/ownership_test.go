// Package architecture enforces a few verifiable BDSPro ownership contracts.
// It does not attempt to clone Gitea's complete import topology.
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

// This checks production imports only; package-local integration tests are
// allowed to cross responsibilities when verifying real application flows.
func TestSourcePackageOwnership(t *testing.T) {
    root := filepath.Clean(filepath.Join("..", ".."))
    for _, owner := range []string{"modules", "models", "services", "routers"} {
        t.Run(owner,func(t *testing.T) {
            err := filepath.WalkDir(filepath.Join(root,owner),func(path string, e fs.DirEntry, walkErr error) error {
                if walkErr != nil { return walkErr }
                if e.IsDir() || !strings.HasSuffix(e.Name(),".go") ||
                    strings.HasSuffix(e.Name(),"_test.go") { return nil }
                f, err := parser.ParseFile(token.NewFileSet(),path,nil,parser.ImportsOnly)
                if err != nil { return err }
                rel, err := filepath.Rel(root,path)
                if err != nil { return err }
                for _, imp := range f.Imports {
                    dep, err := strconv.Unquote(imp.Path.Value)
                    if err != nil {return err}
                    isRouter := strings.HasPrefix(dep,modulePath+"/routers/")
                    isService := strings.HasPrefix(dep,modulePath+"/services/")
                    isModel := strings.HasPrefix(dep,modulePath+"/models/")
                    if strings.HasPrefix(dep,modulePath+"/internal/") {
                        t.Errorf("%s imports legacy internal path %q",rel,dep)
                    }
                    if owner=="modules" && (isRouter || isService) {
                        t.Errorf("%s: technical module imports higher-level package %q",rel,dep)
                    }
                    if owner=="models" && (isRouter || isService) {
                        t.Errorf("%s: persistence imports higher-level package %q",rel,dep)
                    }
                    if owner=="services" && (isRouter || dep=="github.com/gin-gonic/gin") {
                        t.Errorf("%s: service depends on HTTP delivery %q",rel,dep)
                    }
                    if strings.HasPrefix(filepath.ToSlash(rel),"routers/common/") &&
                        (isModel || isService) {
                        t.Errorf("%s: common HTTP transport imports business/persistence %q",rel,dep)
                    }
                }
                return nil
            })
            if err != nil {t.Fatal(err)}
        })
    }
}
