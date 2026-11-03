package setting

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathOptions configures where to find the instance configuration.
// The caller owns CLI parsing; this package owns path resolution.
type PathOptions struct {
	Config   string
	WorkPath string
}

type Paths struct {
	WorkPath   string
	CustomPath string
	ConfigFile string
}

// ResolvePaths uses the current directory for source development, unless
// BDS_WORK_DIR or --work-path explicitly selects an absolute directory.
// An explicit --config is resolved relative to the shell working directory.
// This differs intentionally from Gitea's installed-binary default because
// go run builds a temporary binary outside the source directory.
func ResolvePaths(options PathOptions) (Paths, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Paths{}, fmt.Errorf("current directory: %w", err)
	}
	work := cwd
	if env := os.Getenv("BDS_WORK_DIR"); env != "" {
		work = env
	}
	if options.WorkPath != "" {
		work = options.WorkPath
	}
	if !filepath.IsAbs(work) {
		return Paths{}, fmt.Errorf("work path must be absolute: %q", work)
	}
	work = filepath.Clean(work)
	custom := filepath.Join(work, "custom")
	file := filepath.Join(custom, "conf", "app.ini")
	if options.Config != "" {
		file, err = filepath.Abs(options.Config)
		if err != nil {
			return Paths{}, fmt.Errorf("config path: %w", err)
		}
		file = filepath.Clean(file)
	}
	return Paths{WorkPath: work, CustomPath: custom, ConfigFile: file}, nil
}
