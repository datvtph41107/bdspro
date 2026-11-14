package setting

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathOptions contains explicit CLI overrides.
// An empty Config means that no override was supplied.
type PathOptions struct {
	Config string
}

// Paths contains the resolved application paths.
type Paths struct {
	WorkPath   string
	ConfigFile string
}

// ResolvePaths resolves application paths from the current
// working directory and an optional config file override.
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

	workPath := filepath.Clean(cwd)
	configFile := filepath.Join(workPath, "custom", "conf", "app.ini")

	if options.Config != "" {
		configFile = options.Config
		// Explicit relative --config follows the shell working directory.
		if !filepath.IsAbs(configFile) {
			configFile = filepath.Join(workPath, configFile)
		}
	}

	return Paths{
		WorkPath:   workPath,
		ConfigFile: filepath.Clean(configFile),
	}, nil
}
