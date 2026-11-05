package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAuthFileOwnership prevents a return to one-file-per-authentication-action
// handlers or multiple independent phone/oidc authentication owners.
func TestAuthFileOwnership(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	removed := []string{
		"routers/api/v1/user/phone.go",
		"routers/api/v1/user/email.go",
		"routers/api/v1/user/password.go",
		"routers/api/v1/user/verify_email.go",
		"routers/api/v1/user/token.go",
		"services/auth/phone_otp.go",
		"services/auth/oauth2_auth.go",
		"models/account/verify_email.go",
		"models/account/verification_issue.go",
		"models/organization/team_list.go",
	}
	for _, name := range removed {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("fragmented authentication ownership returned: %s, stat error: %v", name, err)
		}
	}
	for _, required := range []string{
		"routers/api/v1/user/auth.go",
		"routers/api/v1/user/oauth2.go",
		"services/auth/auth.go",
		"services/auth/oauth2.go",
		"models/account/email.go",
		"models/auth/auth.go",
		"models/auth/session.go",
	} {
		if _, err := os.Stat(filepath.Join(root, required)); err != nil {
			t.Errorf("canonical owner missing: %s: %v", required, err)
		}
	}
}

// TestCanonicalAuthOwner rejects duplicate service owners and duplicate
// issuance functions, even when a new fragment is introduced under a
// previously unseen file name. Each method uses the same AuthService.
func TestCanonicalAuthOwner(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	files, err := filepath.Glob(filepath.Join(root, "services", "auth", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	types := make(map[string]int)
	methods := make(map[string]int)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range source.Decls {
			switch item := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range item.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						types[ts.Name.Name]++
					}
				}
			case *ast.FuncDecl:
				methods[item.Name.Name]++
			}
		}
	}
	if types["AuthService"] != 1 || types["PhoneOTP"] != 0 || types["OAuth2AuthService"] != 0 {
		t.Fatalf("duplicate or fragmented authentication owners: %#v", types)
	}
	if methods["issueSession"] != 1 || methods["IssueSession"] != 0 {
		t.Fatalf("session issuance authority not canonical: %#v", methods)
	}
}
