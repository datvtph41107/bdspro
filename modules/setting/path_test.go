package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	cwd := filepath.Join(string(filepath.Separator), "tmp", "bdspro-test-cwd")
	instance := filepath.Join(string(filepath.Separator), "srv", "bdspro")
	cases := []struct {
		name string
		args PathOptions
		work string
		conf string
	}{
		{"default", PathOptions{}, cwd, filepath.Join(cwd, "custom", "conf", "app.ini")},
		{"work-override", PathOptions{WorkPath: instance}, instance, filepath.Join(instance, "custom", "conf", "app.ini")},
		{"config-relative", PathOptions{Config: "conf/prod.ini"}, cwd, filepath.Join(cwd, "conf", "prod.ini")},
		{"both", PathOptions{WorkPath: instance, Config: "conf/prod.ini"}, instance, filepath.Join(cwd, "conf", "prod.ini")},
		{"config-absolute", PathOptions{Config: "/etc/bdspro/app.ini"}, cwd, "/etc/bdspro/app.ini"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := resolvePaths(cwd, tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if paths.WorkPath != tt.work || paths.ConfigFile != tt.conf || paths.CustomPath != filepath.Join(tt.work, "custom") {
				t.Fatalf("paths=%+v, want work=%q conf=%q", paths, tt.work, tt.conf)
			}
		})
	}
}

func TestResolvePathsRejectsRelativeWorkPath(t *testing.T) {
	_, err := resolvePaths("/tmp", PathOptions{WorkPath: "./relative"})
	if err == nil || !strings.Contains(err.Error(), "--work-path") {
		t.Fatalf("want --work-path error, got %v", err)
	}
}

func TestResolvePathsRejectsRelativeCWD(t *testing.T) {
	if _, err := resolvePaths("relative", PathOptions{}); err == nil {
		t.Fatal("relative cwd must be rejected")
	}
}

func TestSamePathSymlink(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "instance")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	if !samePath(target, link) {
		t.Fatal("symlink pointing to the same directory must not conflict")
	}
}

func TestConfiguredWorkPathMissingKeepsResolvedDefaults(t *testing.T) {
	cfg, err := NewConfigProviderFromData("RUN_MODE = development\n")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	paths := Paths{
		WorkPath:   work,
		CustomPath: filepath.Join(work, "custom"),
		ConfigFile: filepath.Join(work, "custom", "conf", "app.ini"),
	}
	got, err := resolveConfiguredWorkPath(paths, PathOptions{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != paths {
		t.Fatalf("missing WORK_PATH changed resolved defaults: got %+v, want %+v", got, paths)
	}
}

func TestConfigCheckRejectsExplicitEmptyWorkPath(t *testing.T) {
	saveGlobals(t)
	AppWorkPath = "before-config-check"
	work := t.TempDir()
	for _, tc := range []struct {
		name   string
		prefix string
		envs   []string
		cli    string
	}{
		{name: "ini", prefix: "WORK_PATH =\n"},
		{name: "env", envs: []string{"BDS____WORK_PATH="}},
		{name: "env-overrides-nonempty-ini", prefix: "WORK_PATH = " + work + "\n", envs: []string{"BDS____WORK_PATH="}},
		{name: "cli-does-not-hide-empty-ini", prefix: "WORK_PATH =\n", cli: work},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := writeConfig(t, tc.prefix+validWebFixture)
			err := CheckWebConfig(PathOptions{Config: file, WorkPath: tc.cli}, tc.envs)
			if err == nil || !strings.Contains(err.Error(), "WORK_PATH") {
				t.Fatalf("explicit empty WORK_PATH must fail preflight: %v", err)
			}
			if AppWorkPath != "before-config-check" {
				t.Fatal("preflight published process work path on failure")
			}
		})
	}
}

func TestMigrateRejectsExplicitEmptyWorkPathWithoutPublication(t *testing.T) {
	saveGlobals(t)
	AppWorkPath = "before-migrate"
	file := writeConfig(t, "RUN_MODE = development\nWORK_PATH =\n[database]\nDB_SOURCE = postgres://localhost/db\n")
	err := LoadMigrateSettings(PathOptions{Config: file}, nil)
	if err == nil || !strings.Contains(err.Error(), "WORK_PATH") {
		t.Fatalf("migrate must reject empty WORK_PATH before publication: %v", err)
	}
	if AppWorkPath != "before-migrate" {
		t.Fatal("migrate published process work path on failure")
	}
}
