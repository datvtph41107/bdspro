package architecture

import (
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"
)

var markdownLocalLinkPattern = regexp.MustCompile("\\[[^\\]]+\\]\\(([^)]+)\\)")

// missingLocalLinks reports broken local Markdown links without fetching external URLs.
// It deliberately checks concrete navigation targets, not semantic correctness.
func missingLocalLinks(repoRoot, source string, markdown []byte) []string {
    var missing []string
    for _, match := range markdownLocalLinkPattern.FindAllStringSubmatch(string(markdown), -1) {
        target := strings.TrimSpace(match[1])
        if target == "" || strings.HasPrefix(target, "#") ||
            strings.HasPrefix(target, "https://") ||
            strings.HasPrefix(target, "http://") ||
            strings.HasPrefix(target, "mailto:") {
            continue
        }
        pathPart := strings.SplitN(target, "#", 2)[0]
        if filepath.IsAbs(pathPart) {
            missing = append(missing, target)
            continue
        }
        resolved := filepath.Clean(filepath.Join(repoRoot, filepath.Dir(source), filepath.FromSlash(pathPart)))
        if _, err := os.Stat(resolved); err != nil {
            missing = append(missing, target)
        }
    }
    return missing
}

func TestAgentGuidanceLinks(t *testing.T) {
    repoRoot := filepath.Clean(filepath.Join("..", ".."))
    paths := []string{
        "AGENTS.md",
        "CLAUDE.md",
        ".github/PULL_REQUEST_TEMPLATE.md",
        "docs/README.md",
        "docs/engineering/README.md",
        "docs/engineering/task-contract.md",
        "docs/engineering/capability-map.md",
        "docs/architecture/source-ownership.md",
    }
    for _, relative := range paths {
        t.Run(relative, func(t *testing.T) {
            body, err := os.ReadFile(filepath.Join(repoRoot, relative))
            if err != nil {
                t.Fatal(err)
            }
            if missing := missingLocalLinks(repoRoot, relative, body); len(missing) != 0 {
                t.Fatalf("unresolved local Markdown targets: %v", missing)
            }
        })
    }

    claude, err := os.ReadFile(filepath.Join(repoRoot, "CLAUDE.md"))
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(string(claude), "@AGENTS.md") {
        t.Fatal("CLAUDE.md must import the canonical AGENTS.md instructions")
    }
}

func TestAgentGuidanceLinkCheckerDetectsDrift(t *testing.T) {
    repoRoot := t.TempDir()
    source := "docs/workflow.md"
    link := []byte("[contract](missing-contract.md)")
    if got := missingLocalLinks(repoRoot, source, link); len(got) != 1 {
        t.Fatalf("expected broken link, got %v", got)
    }
    if err := os.MkdirAll(filepath.Join(repoRoot, "docs"), 0o755); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(repoRoot, "docs", "missing-contract.md"), []byte("# Contract\n"), 0o644); err != nil {
        t.Fatal(err)
    }
    if got := missingLocalLinks(repoRoot, source, link); len(got) != 0 {
        t.Fatalf("expected repaired link to pass, got %v", got)
    }
}
