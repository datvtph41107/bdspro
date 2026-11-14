package setting

import (
	"path/filepath"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	cwd := filepath.Join(string(filepath.Separator), "project", "bdspro")

	t.Run("default config follows working directory", func(t *testing.T) {
		paths, err := resolvePaths(cwd, PathOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if paths.WorkPath != cwd {
			t.Fatalf("WorkPath = %q, want %q", paths.WorkPath, cwd)
		}
		want := filepath.Join(cwd, "custom", "conf", "app.ini")
		if paths.ConfigFile != want {
			t.Fatalf("ConfigFile = %q, want %q", paths.ConfigFile, want)
		}
	})

	t.Run("relative config follows working directory", func(t *testing.T) {
		paths, err := resolvePaths(cwd, PathOptions{Config: "test/app.ini"})
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(cwd, "test", "app.ini")
		if paths.ConfigFile != want {
			t.Fatalf("ConfigFile = %q, want %q", paths.ConfigFile, want)
		}
	})

	t.Run("absolute config is independent from working directory", func(t *testing.T) {
		configFile := filepath.Join(string(filepath.Separator), "etc", "bdspro", "app.ini")
		paths, err := resolvePaths(cwd, PathOptions{Config: configFile})
		if err != nil {
			t.Fatal(err)
		}
		if paths.ConfigFile != configFile {
			t.Fatalf("ConfigFile = %q, want %q", paths.ConfigFile, configFile)
		}
	})

	t.Run("reject relative working directory", func(t *testing.T) {
		if _, err := resolvePaths("relative/path", PathOptions{}); err == nil {
			t.Fatal("expected error for relative working directory")
		}
	})
}
