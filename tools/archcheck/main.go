// Command archcheck verifies selected source-level dependency boundaries.
// New rules require evidence and an explicit architectural decision.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const module = "github.com/datvtph41107/bdspro/"

type goPackage struct {
	ImportPath string   `json:"ImportPath"`
	Imports    []string `json:"Imports"`
}

type violation struct {
	Rule   string
	Source string
	Import string
}

func forbidden(source, dependency string) string {
	if !strings.HasPrefix(source, module) || !strings.HasPrefix(dependency, module) {
		return ""
	}
	src := strings.TrimPrefix(source, module)
	dep := strings.TrimPrefix(dependency, module)

	// Resource owners may not depend on process configuration or lifecycle.
	if src == "models/db" || src == "modules/conn" {
		if dep == "modules/setting" || dep == "modules/graceful" {
			return "ARCH-RESOURCE-001"
		}
	}
	// Business services may not depend on HTTP delivery or process commands.
	if strings.HasPrefix(src, "services/") {
		if dep == "cmd" || strings.HasPrefix(dep, "cmd/") ||
			dep == "routers" || strings.HasPrefix(dep, "routers/") {
			return "ARCH-SERVICE-001"
		}
	}
	return ""
}

func check(r io.Reader) ([]violation, error) {
	decoder := json.NewDecoder(r)
	var found []violation
	for {
		var pkg goPackage
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		for _, dep := range pkg.Imports {
			if rule := forbidden(pkg.ImportPath, dep); rule != "" {
				found = append(found, violation{rule, pkg.ImportPath, dep})
			}
		}
	}
	sort.Slice(found, func(i, j int) bool {
		a, b := found[i], found[j]
		return a.Source+a.Import < b.Source+b.Import
	})
	return found, nil
}

func main() {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck: go list:", err)
		os.Exit(2)
	}
	issues, err := check(strings.NewReader(string(output)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(2)
	}
	for _, v := range issues {
		fmt.Printf("%s: %s imports %s\n", v.Rule, v.Source, v.Import)
	}
	if len(issues) != 0 {
		os.Exit(1)
	}
	fmt.Println("archcheck: selected boundaries OK")
}
