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
	for _, pkg := range []string{"services/auth", "models/account", "models/auth"} {
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

// TestCanonicalAuthOwner rejects duplicate issuance authorities while allowing
// cohesive files to be split or renamed as the implementation evolves.
func TestCanonicalAuthOwner(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	files, err := filepath.Glob(filepath.Join(root, "services", "auth", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	types := make(map[string]int)
	methods := make(map[string]int)
	issuers := map[string]bool{
		"Login":               false,
		"VerifyPhoneRegister": false,
		"VerifyPhoneLogin":    false,
		"LoginOIDC":           false,
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			switch item := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range item.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						types[ts.Name.Name]++
					}
				}
			case *ast.FuncDecl:
				methods[item.Name.Name]++
				if _, required := issuers[item.Name.Name]; !required {
					continue
				}
				ast.Inspect(item.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if ok && sel.Sel.Name == "issueSession" {
						issuers[item.Name.Name] = true
					}
					return true
				})
			}
		}
	}
	if types["AuthService"] != 1 || types["PhoneOTP"] != 0 || types["OAuth2AuthService"] != 0 {
		t.Fatalf("duplicate authentication owners: %#v", types)
	}
	if methods["issueSession"] != 1 || methods["IssueSession"] != 0 {
		t.Fatalf("session issuance authority not canonical: %#v", methods)
	}
	for method, callsIssuer := range issuers {
		if !callsIssuer {
			t.Errorf("%s must issue sessions through AuthService.issueSession", method)
		}
	}
}
