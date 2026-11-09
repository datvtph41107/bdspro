package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These ceilings are documented transitional debt, NOT an architectural target.
// Every new v1 endpoint should use typed route registration; migrations must
// decrease these values rather than perpetuating both styles.
const (
	legacyV1GinRegistrations = 60
	legacyV1ContextWraps     = 19
	minimumV1TypedActions    = 5
	legacyOrgReceiverMethods = 20
)

type routeStyleCounts struct {
	gin      int
	typed    int
	wrappers int
}

func countRouteStyles(filename string, src any) (routeStyleCounts, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, src, 0)
	if err != nil {
		return routeStyleCounts{}, err
	}
	contextAlias := ""
	for _, imp := range parsed.Imports {
		dep, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return routeStyleCounts{}, err
		}
		if dep == modulePath+"/services/context" {
			contextAlias = "context"
			if imp.Name != nil {
				contextAlias = imp.Name.Name
			}
		}
	}
	var counts routeStyleCounts
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if contextAlias != "" && recv.Name == contextAlias && sel.Sel.Name == "Wrap" {
			counts.wrappers++
		}
		switch sel.Sel.Name {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "Any":
			counts.gin++
		case "Get", "Post", "Put", "Patch", "Delete", "Head", "Options":
			// Only a web.Router[T] convention named m is currently used in
			// the pinned BDSPro v1 registration function. Changing this API
			// requires updating the classifier and its negative test first.
			if recv.Name == "m" {
				counts.typed++
			}
		}
		return true
	})
	return counts, nil
}

func routeDebtWithinBudget(actual routeStyleCounts, maxGin, maxWrap, minTyped int) error {
	if actual.gin > maxGin || actual.wrappers > maxWrap || actual.typed < minTyped {
		return fmt.Errorf("HTTP registration debt grew: direct Gin=%d (max %d), context.Wrap=%d (max %d), typed=%d (min %d)",
			actual.gin, maxGin, actual.wrappers, maxWrap, actual.typed, minTyped)
	}
	return nil
}

// TestProgrammingModelV1MigrationBudget catches newly introduced legacy
// HTTP registration patterns in the canonical v1 routing source. It does not
// replace route inventory, middleware-order, HTTP or authorization tests.
func TestProgrammingModelV1MigrationBudget(t *testing.T) {
	path := filepath.Join("..", "..", "routers", "api", "v1", "api.go")
	counts, err := countRouteStyles(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("v1 migration baseline: direct Gin=%d; context.Wrap=%d; typed actions=%d", counts.gin, counts.wrappers, counts.typed)
	if err := routeDebtWithinBudget(counts, legacyV1GinRegistrations, legacyV1ContextWraps, minimumV1TypedActions); err != nil {
		t.Fatal(err)
	}
}

// New v1 route declarations must be mounted from api.go; resource subpackages
// contain action functions. Otherwise a migration could hide legacy Gin calls
// in another file and still stay below the debt ceiling.
func TestProgrammingModelV1SingleRouteOwner(t *testing.T) {
	root := filepath.Join("..", "..", "routers", "api", "v1")
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			return nil
		}
		if filepath.Clean(path) == filepath.Join(root, "api.go") {
			return nil
		}
		counts, err := countRouteStyles(path, nil)
		if err != nil {
			return err
		}
		if counts.gin != 0 || counts.typed != 0 {
			t.Errorf("%s registers routes outside routers/api/v1/api.go: Gin=%d typed=%d", path, counts.gin, counts.typed)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func countLegacyOrgMethods(root string) (int, error) {
	total := 0
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			typ := fn.Recv.List[0].Type
			if ptr, ok := typ.(*ast.StarExpr); ok {
				typ = ptr.X
			}
			name, ok := typ.(*ast.Ident)
			if ok && (name.Name == "TeamHandler" || name.Name == "Handler") {
				total++
			}
		}
		return nil
	})
	return total, err
}

// The 20 receiver methods in org are existing migration debt. An agent should
// not add another competing action interface instead of migrating a consumer.
func TestProgrammingModelOrganizationReceiverBudget(t *testing.T) {
	total, err := countLegacyOrgMethods(filepath.Join("..", "..", "routers", "api", "v1", "org"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Organization legacy receiver methods: %d (max %d)", total, legacyOrgReceiverMethods)
	if total > legacyOrgReceiverMethods {
		t.Fatalf("Organization HTTP receiver debt grew from %d to %d", legacyOrgReceiverMethods, total)
	}
}

func TestProgrammingModelAuditDetectsBacksliding(t *testing.T) {
	baseline := strings.Join([]string{
		"package apiv1",
		"import appctx \"github.com/datvtph41107/bdspro/services/context\"",
		"func Routes(root *Router, route *Group, m *Typed) {",
		"    root.GET(\"/health\", ok)",
		"    route.POST(\"/account\", appctx.Wrap(old))",
		"    m.Get(\"/teams\", action)",
		"}",
	}, "\n")
	counts, err := countRouteStyles("baseline.go", baseline)
	if err != nil {
		t.Fatal(err)
	}
	if counts != (routeStyleCounts{gin: 2, typed: 1, wrappers: 1}) {
		t.Fatalf("unexpected AST classification: %+v", counts)
	}
	if err := routeDebtWithinBudget(counts, 2, 1, 1); err != nil {
		t.Fatal(err)
	}
	addedLegacy := strings.Replace(baseline, "m.Get(\"/teams\", action)", "m.Get(\"/teams\", action)\n    route.DELETE(\"/account\", appctx.Wrap(old))", 1)
	after, err := countRouteStyles("added.go", addedLegacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := routeDebtWithinBudget(after, 2, 1, 1); err == nil {
		t.Fatalf("legacy registration regression passed debt limit: %+v", after)
	}
	// Comments do not count as source-level HTTP registrations.
	commented := baseline + "\n// route.DELETE(\"/account\", appctx.Wrap(old))\n"
	still, err := countRouteStyles("comment.go", commented)
	if err != nil || still != counts {
		t.Fatalf("comment changed AST inventory: %+v, %v", still, err)
	}
}
