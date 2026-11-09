package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// importBoundaryViolation enforces the *direction* of cross-package imports.
// It deliberately does not compare exact Gitea imports: Gitea's Chi/XORM
// dependencies are not BDSPro's Gin/SQLC dependencies.
func importBoundaryViolation(relativePath, imported string) string {
	rel := filepath.ToSlash(relativePath)
	dep := strings.TrimPrefix(imported, modulePath+"/")
	if dep == imported {
		// Check third-party framework imports only in persistence/service areas.
		if strings.HasPrefix(rel, "models/organization/") && imported == "github.com/gin-gonic/gin" {
			return "persistence imports Gin"
		}
		if strings.HasPrefix(rel, "services/organization/") && imported == "github.com/gin-gonic/gin" {
			return "business operation imports Gin"
		}
		return ""
	}
	isDomain := strings.HasPrefix(dep, "models/") || strings.HasPrefix(dep, "services/") || strings.HasPrefix(dep, "routers/")
	if strings.HasPrefix(rel, "modules/web/") || strings.HasPrefix(rel, "modules/setting/") {
		if isDomain {
			return "technical HTTP/settings module imports domain or delivery"
		}
	}
	if strings.HasPrefix(rel, "services/context/") {
		if strings.HasPrefix(dep, "routers/") ||
			strings.HasPrefix(dep, "services/organization/") || dep == "services/organization" ||
			strings.HasPrefix(dep, "services/auth/") || dep == "services/auth" ||
			strings.HasPrefix(dep, "services/oauth/") || dep == "services/oauth" {
			return "HTTP request context imports a business operation or router"
		}
	}
	if strings.HasPrefix(rel, "services/organization/") {
		if strings.HasPrefix(dep, "routers/") || strings.HasPrefix(dep, "services/context/") || dep == "services/context" ||
			strings.HasPrefix(dep, "modules/web/") || dep == "modules/web" {
			return "Organization service imports HTTP delivery"
		}
	}
	if strings.HasPrefix(rel, "models/organization/") {
		if strings.HasPrefix(dep, "services/") || strings.HasPrefix(dep, "routers/") ||
			strings.HasPrefix(dep, "modules/web/") || dep == "modules/web" {
			return "Organization model imports orchestration or delivery"
		}
	}
	return ""
}

func checkProductionImports(root, relativeDir string) (int, error) {
	found := 0
	err := filepath.WalkDir(filepath.Join(root, relativeDir), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found++
		for _, imp := range parsed.Imports {
			dep, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if violation := importBoundaryViolation(rel, dep); violation != "" {
				return fmt.Errorf("%s: %s: %s", filepath.ToSlash(rel), violation, dep)
			}
		}
		return nil
	})
	return found, err
}

// Explicit package-level import boundaries prevent a new facade or agent from
// introducing an inverted dependency in the name of Gitea-style symmetry.
func TestProgrammingModelImportBoundaries(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, dir := range []string{
		"modules/web",
		"modules/setting",
		"services/context",
		"services/organization",
		"models/organization",
	} {
		t.Run(dir, func(t *testing.T) {
			count, err := checkProductionImports(root, dir)
			if err != nil {
				t.Fatal(err)
			}
			if count == 0 {
				t.Fatalf("%s had no production sources; import policy was not exercised", dir)
			}
			t.Logf("%s: %d production files inspected", dir, count)
		})
	}
}

func TestProgrammingModelImportBoundaryDetectsInversion(t *testing.T) {
	tests := []struct {
		file string
		dep  string
		bad  bool
	}{
		{"modules/web/router.go", modulePath + "/models/organization", true},
		{"modules/setting/config.go", modulePath + "/services/auth", true},
		{"services/context/api.go", modulePath + "/services/organization", true},
		{"services/organization/team.go", modulePath + "/services/context", true},
		{"models/organization/team.go", modulePath + "/modules/web", true},
		{"models/organization/team.go", "github.com/gin-gonic/gin", true},
		{"modules/web/form.go", modulePath + "/modules/reqctx", false},
		{"services/context/api.go", modulePath + "/models/auth", false},
		{"services/organization/team.go", modulePath + "/models/db", false},
		{"models/organization/team.go", modulePath + "/models/db", false},
	}
	for _, tc := range tests {
		got := importBoundaryViolation(tc.file, tc.dep)
		if (got != "") != tc.bad {
			t.Errorf("%s imports %s: violation=%q, expected forbidden=%v", tc.file, tc.dep, got, tc.bad)
		}
	}
}
