// Command r02audit captures the pre-refactor HTTP route surface.
// It is deliberately limited to static Gin route declarations in BDSPro R01;
// runtime policy and handler behavior require separate integration tests.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Route groups use a path prefix established at the caller.
// R01 groups retain their pinned baseline; later groups require an explicit
// additive route contract rather than rewriting the historic snapshot.
var legacyGroupPrefix = map[string]string{
	"registerRoutes":              "",
	"Routes":                      "",
	"registerPublicRoutes":        "",
	"registerAuthenticatedRoutes": "",
	"registerOrganizationRoutes":  "/organizations",
	"registerOrgRoutes":           "/orgs",
	"registerTeamRoutes":          "/teams",
	"registerPhoneBindRoutes":     "/users/me/phone",
	"registerSwagger":             "",
	"RegisterSwagger":             "",
}

var legacyHTTPMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "HEAD": true, "OPTIONS": true,
}

type route struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler,omitempty"`
	Source  string `json:"source,omitempty"`
}

type contract struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Routes     []struct {
		Method    string `json:"method"`
		Path      string `json:"path"`
		Access    string `json:"access"`
		Condition string `json:"condition,omitempty"`
	} `json:"routes"`
}

func routeKey(method, path string) string { return method + " " + path }

func literalString(expr ast.Expr) (string, bool) {
	v, ok := expr.(*ast.BasicLit)
	if !ok || v.Kind != token.STRING {
		return "", false
	}
	decoded, err := strconv.Unquote(v.Value)
	return decoded, err == nil
}

func selectorText(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return selectorText(x.X) + "." + x.Sel.Name
	default:
		return "<expression>"
	}
}

func extractRoutes(filename string, src any) ([]route, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filename, src, 0)
	if err != nil {
		return nil, err
	}
	var found []route
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		prefix, known := legacyGroupPrefix[fn.Name.Name]
		if !known {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !legacyHTTPMethods[sel.Sel.Name] {
				return true
			}
			path, ok := literalString(call.Args[0])
			if !ok || !strings.HasPrefix(path, "/") && path != "" {
				return true
			}
			full := prefix + path
			if full == "" {
				full = "/"
			}
			found = append(found, route{
				Method:  sel.Sel.Name,
				Path:    full,
				Handler: selectorText(call.Args[len(call.Args)-1]),
				Source:  fn.Name.Name,
			})
			return true
		})
	}
	sort.Slice(found, func(i, j int) bool {
		return routeKey(found[i].Method, found[i].Path) < routeKey(found[j].Method, found[j].Path)
	})
	return found, nil
}

func collect(root string) ([]route, error) {
	var all []route
	for _, name := range []string{"v1/api.go", "swagger.go"} {
		path := filepath.Join(root, "routers", "api", name)
		found, err := extractRoutes(path, nil)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		all = append(all, found...)
	}
	sort.Slice(all, func(i, j int) bool {
		return routeKey(all[i].Method, all[i].Path) < routeKey(all[j].Method, all[j].Path)
	})
	return all, nil
}

func checkRoutes(actual []route, expected contract) error {
	if len(expected.Routes) == 0 {
		return errors.New("contract must contain at least one route")
	}
	seen := make(map[string]bool, len(expected.Routes))
	for _, r := range expected.Routes {
		k := routeKey(r.Method, r.Path)
		if seen[k] {
			return fmt.Errorf("duplicate contract route: %s", k)
		}
		if !legacyHTTPMethods[r.Method] || !strings.HasPrefix(r.Path, "/") || r.Access == "" {
			return fmt.Errorf("invalid contract route: %s", k)
		}
		seen[k] = true
	}
	actualSeen := make(map[string]bool, len(actual))
	for _, r := range actual {
		k := routeKey(r.Method, r.Path)
		if actualSeen[k] {
			return fmt.Errorf("duplicate source route: %s", k)
		}
		actualSeen[k] = true
		if !seen[k] {
			return fmt.Errorf("unexpected source route: %s", k)
		}
	}
	for k := range seen {
		if !actualSeen[k] {
			return fmt.Errorf("missing source route: %s", k)
		}
	}
	return nil
}

// applyRemovals requires every removed route to be present in the effective
// baseline+additions and refuses duplicate, invalid or unexplained removals.
func applyRemovals(expected, removals contract) (contract, error) {
	if len(removals.Routes) == 0 {
		return contract{}, errors.New("reviewed removals cannot be empty")
	}
	removed := make(map[string]bool, len(removals.Routes))
	for _, r := range removals.Routes {
		k := routeKey(r.Method, r.Path)
		if !legacyHTTPMethods[r.Method] || !strings.HasPrefix(r.Path, "/") || r.Access == "" {
			return contract{}, fmt.Errorf("invalid reviewed removal: %s", k)
		}
		if removed[k] {
			return contract{}, fmt.Errorf("duplicate reviewed removal: %s", k)
		}
		removed[k] = true
	}
	filtered := expected.Routes[:0:0]
	for _, r := range expected.Routes {
		k := routeKey(r.Method, r.Path)
		if removed[k] {
			delete(removed, k)
			continue
		}
		filtered = append(filtered, r)
	}
	if len(removed) != 0 {
		for k := range removed {
			return contract{}, fmt.Errorf("reviewed removal not present in baseline: %s", k)
		}
	}
	expected.Routes = filtered
	return expected, nil
}

func execute(root, contractPath string, output io.Writer) error {
	return executeWithAdditions(root, contractPath, "", output)
}

// executeWithAdditions verifies the immutable R01 snapshot plus a separately
// reviewed additive route manifest. It rejects missing, duplicate and unknown
// HTTP method/path pairs without rewriting the historical contract.
func executeWithAdditions(root, contractPath, additionsPath string, output io.Writer) error {
	return executeWithChanges(root, contractPath, additionsPath, "", output)
}

// executeWithChanges retains the pinned baseline and applies separately
// reviewed additions and removals. A removal must exist in the expected
// historical surface; unknown or repeated removals fail closed.
func executeWithChanges(root, contractPath, additionsPath, removalsPath string, output io.Writer) error {
	found, err := collect(root)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(contractPath)
	if err != nil {
		return err
	}
	var expected contract
	if err := json.Unmarshal(data, &expected); err != nil {
		return fmt.Errorf("decode route contract: %w", err)
	}
	if additionsPath != "" {
		for _, addition := range strings.Split(additionsPath, ",") {
			addition = strings.TrimSpace(addition)
			if addition == "" {
				return errors.New("reviewed additions path cannot be empty")
			}
			extensionsData, err := os.ReadFile(addition)
			if err != nil {
				return fmt.Errorf("read reviewed additions %s: %w", addition, err)
			}
			var extensions contract
			if err := json.Unmarshal(extensionsData, &extensions); err != nil {
				return fmt.Errorf("decode reviewed additions %s: %w", addition, err)
			}
			if extensions.Repository != expected.Repository {
				return fmt.Errorf("additions repository mismatch: %q", extensions.Repository)
			}
			expected.Routes = append(expected.Routes, extensions.Routes...)
		}
	}
	if removalsPath != "" {
		removedData, err := os.ReadFile(removalsPath)
		if err != nil {
			return fmt.Errorf("read reviewed removals %s: %w", removalsPath, err)
		}
		var removals contract
		if err := json.Unmarshal(removedData, &removals); err != nil {
			return fmt.Errorf("decode reviewed removals %s: %w", removalsPath, err)
		}
		if removals.Repository != expected.Repository {
			return fmt.Errorf("removals repository mismatch: %q", removals.Repository)
		}
		expected, err = applyRemovals(expected, removals)
		if err != nil {
			return err
		}
	}
	if err := checkRoutes(found, expected); err != nil {
		return err
	}
	fmt.Fprintf(output, "PASS: %d routes match pinned R01 baseline %s and reviewed changes\n", len(found), expected.Commit)
	return nil
}

func main() {
	root := flag.String("root", ".", "R01 repository root")
	contractFile := flag.String("contract", "docs/architecture/r02/routes-r01.json", "pinned R01 route contract")
	additionsFile := flag.String("additions", "", "separately reviewed route additions")
	removalsFile := flag.String("removals", "", "separately reviewed route removals")
	flag.Parse()
	if err := executeWithChanges(*root, *contractFile, *additionsFile, *removalsFile, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "R02 preflight failed:", err)
		os.Exit(1)
	}
}
