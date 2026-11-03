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

// Legacy route groups use a path prefix established at the caller.
// Keep this table specific to the pinned R01 snapshot, not future routers.
var legacyGroupPrefix = map[string]string{
	"registerRoutes":              "",
	"registerPublicRoutes":        "",
	"registerAuthenticatedRoutes": "",
	"registerOrganizationRoutes":  "/organizations",
	"registerPhoneBindRoutes":     "/users/me/phone",
	"registerSwagger":             "",
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
	for _, name := range []string{"routes.go", "swagger.go"} {
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

func execute(root, contractPath string, output io.Writer) error {
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
	if err := checkRoutes(found, expected); err != nil {
		return err
	}
	fmt.Fprintf(output, "PASS: %d legacy route method/path pairs match %s\n", len(found), expected.Commit)
	return nil
}

func main() {
	root := flag.String("root", ".", "R01 repository root")
	contractFile := flag.String("contract", "docs/architecture/r02/routes-r01.json", "pinned R01 route contract")
	flag.Parse()
	if err := execute(*root, *contractFile, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "R02 preflight failed:", err)
		os.Exit(1)
	}
}
