package setting

import (
	"path/filepath"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	cwd := filepath.Join(
		string(filepath.Separator),
		"project",
		"bdspro",
	)
	envWork := filepath.Join(
		string(filepath.Separator),
		"srv",
		"bdspro",
	)

	getenv := func(key string) string {
		if key == "BDS_WORK_DIR" {
			return envWork
		}
		return ""
	}

	t.Run("environment work path", func(t *testing.T) {
		paths, err := resolvePaths(
			cwd, getenv, PathOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}

		want := filepath.Join(
			envWork, "custom", "conf", "app.ini",
		)
		if paths.ConfigFile != want {
			t.Fatalf(
				"ConfigFile = %q, want %q",
				paths.ConfigFile, want,
			)
		}
	})

	t.Run("CLI overrides environment", func(t *testing.T) {
		paths, err := resolvePaths(
			cwd,
			getenv,
			PathOptions{WorkPath: cwd},
		)
		if err != nil {
			t.Fatal(err)
		}

		if paths.WorkPath != cwd {
			t.Fatalf(
				"WorkPath = %q, want %q",
				paths.WorkPath, cwd,
			)
		}
	})

	t.Run("relative config follows cwd", func(t *testing.T) {
		paths, err := resolvePaths(
			cwd,
			getenv,
			PathOptions{Config: "test/app.ini"},
		)
		if err != nil {
			t.Fatal(err)
		}

		want := filepath.Join(cwd, "test", "app.ini")
		if paths.ConfigFile != want {
			t.Fatalf(
				"ConfigFile = %q, want %q",
				paths.ConfigFile, want,
			)
		}
	})

	t.Run("custom path follows work path", func(t *testing.T) {
		paths, err := resolvePaths(
			cwd,
			getenv,
			PathOptions{CustomPath: "instance"},
		)
		if err != nil {
			t.Fatal(err)
		}

		want := filepath.Join(
			envWork, "instance", "conf", "app.ini",
		)
		if paths.ConfigFile != want {
			t.Fatalf(
				"ConfigFile = %q, want %q",
				paths.ConfigFile, want,
			)
		}
	})
}
