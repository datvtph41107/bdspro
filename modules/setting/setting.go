package setting

import (
	"fmt"
	"time"
)

var (
	AppVer       string
	AppBuiltWith string
	AppStartTime time.Time

	AppWorkPath string
	CustomPath  string
	CustomConf  string

	CfgProvider ConfigProvider
	Common      CommonSettings
)

// CommonSettings contains settings required
// before application subsystems initialize.
type CommonSettings struct {
	RunMode string
	Log     LogGlobalConfig
}

// InitWorkPathAndCommonConfig prepares the
// application configuration during CLI startup.
//
// It does not open PostgreSQL, Redis, HTTP,
// or queue connections.
func InitWorkPathAndCommonConfig(
	options PathOptions,
	envs []string,
) error {
	paths, err := ResolvePaths(options)
	if err != nil {
		return fmt.Errorf(
			"resolve application paths: %w",
			err,
		)
	}

	provider, err := NewConfigProviderFromFile(
		paths.ConfigFile,
	)
	if err != nil {
		return fmt.Errorf(
			"initialize config provider: %w",
			err,
		)
	}

	if err := applyEnvironmentOverrides(
		provider,
		envs,
	); err != nil {
		return fmt.Errorf(
			"apply environment overrides: %w",
			err,
		)
	}

	common, err := loadCommonSettingsFrom(
		provider,
	)
	if err != nil {
		return fmt.Errorf(
			"load common settings: %w",
			err,
		)
	}

	// Publish only after preparation succeeds.
	AppWorkPath = paths.WorkPath
	CustomPath = paths.CustomPath
	CustomConf = paths.ConfigFile

	CfgProvider = provider
	Common = common

	return nil
}

// LoadCommonSettings requires an initialized
// provider and validates common configuration.
func LoadCommonSettings() error {
	if CfgProvider == nil {
		return fmt.Errorf(
			"configuration provider is not initialized",
		)
	}

	common, err := loadCommonSettingsFrom(
		CfgProvider,
	)
	if err != nil {
		return err
	}

	Common = common
	return nil
}

// loadCommonSettingsFrom reads the minimum
// settings required for early initialization.

func loadCommonSettingsFrom(
	provider ConfigProvider,
	workPath string,
) (CommonSettings, error) {
	if provider == nil {
		return CommonSettings{},
			fmt.Errorf("configuration provider is nil")
	}

	root := provider.Section("")

	runMode := stringOrDefault(
		root.Key("RUN_MODE"),
		"development",
	)

	switch runMode {
	case "development", "production", "test":
	default:
		return CommonSettings{},
			fmt.Errorf(
				"invalid RUN_MODE %q",
				runMode,
			)
	}

	logConfig, err := loadLogGlobalFrom(
		provider,
		workPath,
	)
	if err != nil {
		return CommonSettings{}, err
	}

	return CommonSettings{
		RunMode: runMode,
		Log:     logConfig,
	}, nil
}

// stringOrDefault is consumed by the common
// loader and later by selected domain loaders.
//
// An explicitly empty key is preserved.
// Domain validation decides whether it is legal.
func stringOrDefault(
	key ConfigKey,
	fallback string,
) string {
	if !key.IsSet() {
		return fallback
	}

	return key.String()
}
