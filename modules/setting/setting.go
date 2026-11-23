package setting

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Process metadata is supplied by main, not by config loading.
var (
	AppVer       string
	AppBuiltWith string
	AppStartTime time.Time

	RunMode string
)

// startupSettings is the only private staging value. Runtime consumers access
// canonical setting package globals, never this temporary value.
type startupSettings struct {
	paths    Paths
	runMode  string
	log      LogGlobalConfig
	graceful GracefulSettings

	server   ServerSettings
	api      APISettings
	security SecuritySettings
	ipactive IPActiveSettings
	database DatabaseSettings
	redis    RedisSettings
}

// loadConfigAndPaths owns the common input path for web, migrate and config check.
func loadConfigAndPaths(args PathOptions, envs []string) (ConfigProvider, Paths, error) {
	paths, err := ResolvePaths(args)
	if err != nil {
		return nil, Paths{}, fmt.Errorf("resolve application paths: %w", err)
	}
	cfg, err := NewConfigProviderFromFile(paths.ConfigFile)
	if err != nil {
		return nil, Paths{}, err
	}
	if err := applyEnvironmentOverrides(cfg, envs); err != nil {
		return nil, Paths{}, fmt.Errorf("apply environment overrides: %w", err)
	}
	if err := validateKnownKeys(cfg); err != nil {
		return nil, Paths{}, fmt.Errorf("validate configuration keys: %w", err)
	}
	paths, err = resolveConfiguredWorkPath(paths, args, cfg)
	if err != nil {
		return nil, Paths{}, err
	}
	return cfg, paths, nil
}

func loadCommonSettingsFrom(cfg ConfigProvider, state *startupSettings) error {
	var err error
	state.runMode, err = loadRunModeFrom(cfg)
	if err != nil {
		return err
	}
	state.log, err = loadLogGlobalFrom(cfg, state.paths.WorkPath)
	if err != nil {
		return err
	}
	state.graceful, err = loadGracefulSettingsFrom(cfg)
	return err
}

// Same-package subsystem loaders are called in dependency order, as in Gitea.
// They only modify this private startup value until commitSettings succeeds.
func loadWebSettingsFrom(cfg ConfigProvider, state *startupSettings) error {
	var err error
	state.server, err = loadServerFrom(cfg, state.runMode)
	if err != nil {
		return fmt.Errorf("load server settings: %w", err)
	}
	state.api, err = loadAPIFrom(cfg, state.runMode, state.server)
	if err != nil {
		return fmt.Errorf("load API settings: %w", err)
	}
	state.security, err = loadSecurityFrom(cfg, state.runMode)
	if err != nil {
		return fmt.Errorf("load security settings: %w", err)
	}
	state.ipactive, err = loadIPActiveFrom(cfg)
	if err != nil {
		return fmt.Errorf("load IP admission settings: %w", err)
	}
	state.database, err = loadDBSetting(cfg, state.paths.WorkPath)
	if err != nil {
		return fmt.Errorf("load database settings: %w", err)
	}
	state.redis, err = loadRedisFrom(cfg)
	if err != nil {
		return fmt.Errorf("load Redis settings: %w", err)
	}
	return nil
}

func buildWebSettings(args PathOptions, envs []string) (startupSettings, error) {
	cfg, paths, err := loadConfigAndPaths(args, envs)
	if err != nil {
		return startupSettings{}, err
	}
	state := startupSettings{paths: paths}
	if err := loadCommonSettingsFrom(cfg, &state); err != nil {
		return startupSettings{}, fmt.Errorf("load common settings: %w", err)
	}
	if err := loadWebSettingsFrom(cfg, &state); err != nil {
		return startupSettings{}, err
	}
	return state, nil
}

func buildMigrateSettings(args PathOptions, envs []string) (startupSettings, error) {
	cfg, paths, err := loadConfigAndPaths(args, envs)
	if err != nil {
		return startupSettings{}, err
	}
	state := startupSettings{paths: paths}
	if err := loadCommonSettingsFrom(cfg, &state); err != nil {
		return startupSettings{}, fmt.Errorf("load common settings: %w", err)
	}
	state.database, err = loadDBSetting(cfg, paths.WorkPath)
	if err != nil {
		return startupSettings{}, fmt.Errorf("load database settings: %w", err)
	}
	return state, nil
}

var settingsInit struct {
	sync.Mutex
	done bool
}

// commitSettings is executed before the process starts concurrent consumers.
// All validation has completed; there is no partial state on loader failure.
func commitSettings(state startupSettings, web bool) error {
	settingsInit.Lock()
	defer settingsInit.Unlock()
	if settingsInit.done {
		return errors.New("setting: process configuration already initialized")
	}

	AppWorkPath = state.paths.WorkPath
	CustomPath = state.paths.CustomPath
	CustomConf = state.paths.ConfigFile
	RunMode = state.runMode
	Log = state.log
	Graceful = state.graceful
	Database = state.database

	if web {
		Server = state.server
		Server.TrustedProxies = append([]string(nil), state.server.TrustedProxies...)
		Server.AllowedOrigins = append([]string(nil), state.server.AllowedOrigins...)
		API = state.api
		Security = state.security
		IPActive = state.ipactive.clone()
		Redis = state.redis
	}
	settingsInit.done = true
	return nil
}

// LoadWebSettings is the sole publication entrypoint for the Web command.
func LoadWebSettings(args PathOptions, envs []string) error {
	state, err := buildWebSettings(args, envs)
	if err != nil {
		return err
	}
	return commitSettings(state, true)
}

// LoadMigrateSettings publishes Common and Database only, never Web resources.
func LoadMigrateSettings(args PathOptions, envs []string) error {
	state, err := buildMigrateSettings(args, envs)
	if err != nil {
		return err
	}
	return commitSettings(state, false)
}

// CheckWebConfig exercises the exact same loaders without any publication.
func CheckWebConfig(args PathOptions, envs []string) error {
	_, err := buildWebSettings(args, envs)
	return err
}

// stringOrDefault preserves the distinction between missing and explicit empty.
func stringOrDefault(key ConfigKey, fallback string) string {
	if key == nil || !key.IsSet() {
		return fallback
	}
	return key.String()
}

func loadRunModeFrom(cfg ConfigProvider) (string, error) {
	mode := stringOrDefault(cfg.Section("").Key("RUN_MODE"), "development")
	switch mode {
	case "development", "production", "test":
		return mode, nil
	default:
		return "", fmt.Errorf("invalid RUN_MODE %q", mode)
	}
}
