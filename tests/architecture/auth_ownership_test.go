package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestAuthDependencyDirection guards responsibilities rather than filenames:
// future cohesive source splits must not require changing architecture tests.
func TestAuthDependencyDirection(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, pkg := range []string{"services/auth", "models/account", "models/auth", "models/audit"} {
		paths, err := filepath.Glob(filepath.Join(root, pkg, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) == 0 {
			t.Fatalf("missing auth package: %s", pkg)
		}
		for _, path := range paths {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range file.Imports {
				name, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				if name == "github.com/gin-gonic/gin" ||
					strings.Contains(name, "/routers/") ||
					strings.HasSuffix(name, "/routers") {
					t.Errorf("%s imports HTTP routing dependency %s", path, name)
				}
				if strings.HasPrefix(pkg, "models/") &&
					(strings.Contains(name, "/services/") || strings.HasSuffix(name, "/services")) {
					t.Errorf("%s imports higher-level service %s", path, name)
				}
			}
		}
	}
}

// TestAuthServiceOwner counts the concrete application owner without tying
// the architecture to filenames or incidental method names. Real credential
// issuance and rollback behavior are verified in services/auth and models/auth.
func TestAuthServiceOwner(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	files, err := filepath.Glob(filepath.Join(root, "services", "auth", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	owners := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			specs, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range specs.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok && typ.Name.Name == "AuthService" {
					owners++
				}
			}
		}
	}
	if owners != 1 {
		t.Fatalf("expected exactly one AuthService owner, got %d", owners)
	}
}
