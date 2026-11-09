package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestOrganizationRoleWriteOwnership prevents future production callers
// from bypassing the service's owner-count/session transaction by directly
// invoking SQLC's role mutations. This is a source-ownership contract,
// NOT a substitute for database constraints or runtime authorization.
func TestOrganizationRoleWriteOwnership(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))

	// These functions exist at the SQLC boundary. Only models/organization
	// may call them from production (generated sqlc itself is excluded).
	modelWrite := map[string]bool{
		"CreateOrganizationMember":              true,
		"PromoteOrganizationMember":             true,
		"DemoteOrganizationOwner":               true,
		"EndOrdinaryOrganizationMemberForOwner": true,
		"DeleteOrgMemberTeamLinks":              true,
	}
	// Transaction-bound model operations may be orchestrated only by the
	// organization service, not directly by Gin handlers or technical modules.
	serviceWrite := map[string]bool{
		"PromoteOwnerTx":                true,
		"DemoteOwnerTx":                 true,
		"RemoveOrdinaryOrgMemberTx":     true,
		"CountActiveOrganizationOwners": true,
		"LockRoleChangeOrganization":    true,
		"RequireRoleChangeOwner":        true,
	}
	seenModel, seenService := 0, 0
	for _, dir := range []string{"cmd", "modules", "models", "services", "routers"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
				strings.HasSuffix(entry.Name(), "_test.go") || strings.HasSuffix(entry.Name(), ".gen.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "models/db/sqlc/") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			// A Gin handler may call the session-aware service function with
			// the same name as a generated SQLC method. Distinguish
			// that import rather than condemning an authorized service call.
			serviceAlias := ""
			for _, imp := range file.Imports {
				importPath, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				if importPath == modulePath+"/services/org" {
					serviceAlias = "org"
					if imp.Name != nil {
						serviceAlias = imp.Name.Name
					}
				}
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				name := selector.Sel.Name
				if modelWrite[name] {
					// Service-owned lifecycle is allowed at HTTP callsites;
					// direct invocation through a db.Queries receiver is not.
					if name == "DemoteOrganizationOwner" && serviceAlias != "" {
						if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == serviceAlias {
							return true
						}
					}
					seenModel++
					if !strings.HasPrefix(rel, "models/organization/") {
						t.Errorf("%s calls low-level %s outside models/organization", rel, name)
					}
				}
				if serviceWrite[name] {
					seenService++
					if !strings.HasPrefix(rel, "services/org/") {
						t.Errorf("%s bypasses organization service for %s", rel, name)
					}
				}
				return true
			})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Prevent the test from silently passing if the protected functions are
	// renamed/removed and the policy no longer observes anything.
	if seenModel == 0 || seenService == 0 {
		t.Fatalf("role-mutation ownership guard not exercised: model=%d service=%d", seenModel, seenService)
	}
}
