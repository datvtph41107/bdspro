package setting

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathOptions contains explicit CLI overrides.
// Empty fields mean that no override was supplied.
type PathOptions struct {
	WorkPath   string
	CustomPath string
	Config     string
}

// Paths contains the resolved application paths.
type Paths struct {
	WorkPath   string
	CustomPath string
	ConfigFile string
}

// ResolvePaths resolves application paths using
// the current working directory and environment.
func ResolvePaths(options PathOptions) (Paths, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Paths{}, fmt.Errorf(
			"get current working directory: %w", err,
		)
	}

	return resolvePaths(cwd, os.Getenv, options)
}

// resolvePaths owns the precedence policy.
//
// cwd and getenv are explicit so this policy
// can be tested without mutating process state.
func resolvePaths(
	cwd string,
	getenv func(string) string,
	options PathOptions,
) (Paths, error) {
	if !filepath.IsAbs(cwd) {
		return Paths{}, fmt.Errorf(
			"current working directory must be absolute",
		)
	}

	// Work path:
	// default < BDS_WORK_DIR < --work-path
	workPath := cwd

	if value := getenv("BDS_WORK_DIR"); value != "" {
		workPath = value
	}

	if options.WorkPath != "" {
		workPath = options.WorkPath
	}

	if !filepath.IsAbs(workPath) {
		return Paths{}, fmt.Errorf(
			"application work path must be absolute: %q",
			workPath,
		)
	}

	workPath = filepath.Clean(workPath)

	// Custom path:
	// default < BDS_CUSTOM < --custom-path
	customPath := "custom"

	if value := getenv("BDS_CUSTOM"); value != "" {
		customPath = value
	}

	if options.CustomPath != "" {
		customPath = options.CustomPath
	}

	if !filepath.IsAbs(customPath) {
		customPath = filepath.Join(workPath, customPath)
	}

	customPath = filepath.Clean(customPath)

	// Configuration file:
	// default to <custom>/conf/app.ini
	configFile := filepath.Join(
		customPath,
		"conf",
		"app.ini",
	)

	if options.Config != "" {
		configFile = options.Config

		// Explicit relative --config is resolved
		// against the shell working directory.
		if !filepath.IsAbs(configFile) {
			configFile = filepath.Join(cwd, configFile)
		}
	}

	configFile = filepath.Clean(configFile)

	return Paths{
		WorkPath:   workPath,
		CustomPath: customPath,
		ConfigFile: configFile,
	}, nil
}
