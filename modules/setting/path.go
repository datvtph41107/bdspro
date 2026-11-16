package setting

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathOptions contains explicit CLI overrides.
// Empty fields mean no override was supplied.
type PathOptions struct {
	WorkPath string
	Config   string
}

// Paths contains the resolved application paths.
type Paths struct {
	WorkPath   string
	ConfigFile string
}

// ResolvePaths uses the current directory as the default instance root.
// CLI --work-path may override that root without changing process CWD.
func ResolvePaths(options PathOptions) (Paths, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Paths{}, fmt.Errorf("get current working directory: %w", err)
	}

	return resolvePaths(cwd, options)
}

// resolvePaths is pure so path resolution can be tested
// without mutating the process working directory.
func resolvePaths(cwd string, options PathOptions) (Paths, error) {
	if !filepath.IsAbs(cwd) {
		return Paths{}, fmt.Errorf("current working directory must be absolute")
	}

	workPath := cwd
	if options.WorkPath != "" {
		workPath = options.WorkPath
	}
	if !filepath.IsAbs(workPath) {
		return Paths{}, fmt.Errorf("application work path must be absolute: %q", workPath)
	}
	workPath = filepath.Clean(workPath)

	configFile := filepath.Join(workPath, "custom", "conf", "app.ini")

	if options.Config != "" {
		configFile = options.Config
		// Explicit relative --config follows the shell CWD, not --work-path.
		if !filepath.IsAbs(configFile) {
			configFile = filepath.Join(cwd, configFile)
		}
	}

	return Paths{
		WorkPath:   workPath,
		ConfigFile: filepath.Clean(configFile),
	}, nil
}
