package setting

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// Runtime metadata do main.go khởi tạo; config globals chỉ publish sau validate.
var (
	AppVer       string
	AppBuiltWith string
	AppStartTime time.Time

	CfgProvider ConfigProvider
	Common      CommonSettings
)

// CommonSettings là dữ liệu đã validate dùng bởi mọi command.
type CommonSettings struct {
	RunMode  string
	Log      LogGlobalConfig
	Graceful GracefulSettings
}

type preparedCommonConfig struct {
	paths    Paths
	provider ConfigProvider
	common   CommonSettings
}

// prepareCommonConfig validates without publishing process globals or opening resources.
func prepareCommonConfig(args PathOptions, envs []string) (preparedCommonConfig, error) {
	paths, err := ResolvePaths(args)
	if err != nil {
		return preparedCommonConfig{}, fmt.Errorf("resolve application paths: %w", err)
	}
	provider, err := NewConfigProviderFromFile(paths.ConfigFile)
	if err != nil {
		return preparedCommonConfig{}, err
	}
	if err := applyEnvironmentOverrides(provider, envs); err != nil {
		return preparedCommonConfig{}, fmt.Errorf("apply environment overrides: %w", err)
	}
	if err := validateKnownKeys(provider); err != nil {
		return preparedCommonConfig{}, fmt.Errorf("validate configuration keys: %w", err)
	}
	// Legacy WORK_PATH is resolved after finding the original INI file.
	workPath := paths.WorkPath
	configWorkPath := ConfigSectionKeyString(provider.Section(""), "WORK_PATH")
	if configWorkPath != "" {
		if !filepath.IsAbs(configWorkPath) {
			return preparedCommonConfig{}, fmt.Errorf("WORK_PATH in %q must be absolute path: %q", paths.ConfigFile, configWorkPath)
		}
		configWorkPath = filepath.Clean(configWorkPath)
		if args.WorkPath != "" && !samePath(paths.WorkPath, configWorkPath) {
			return preparedCommonConfig{}, fmt.Errorf("WORK_PATH %q conflicts with --work-path %q", configWorkPath, paths.WorkPath)
		}
		workPath = configWorkPath
	}
	paths.WorkPath = workPath
	paths.CustomPath = filepath.Join(workPath, "custom")
	common, err := loadCommonSettingsFrom(provider, workPath)
	if err != nil {
		return preparedCommonConfig{}, fmt.Errorf("load common settings: %w", err)
	}
	return preparedCommonConfig{paths: paths, provider: provider, common: common}, nil
}

// InitWorkPathAndCommonConfig preserves the existing process-wide publication contract.
func InitWorkPathAndCommonConfig(args PathOptions, envs []string) error {
	prepared, err := prepareCommonConfig(args, envs)
	if err != nil {
		return err
	}
	AppWorkPath = prepared.paths.WorkPath
	CustomPath = prepared.paths.CustomPath
	CustomConf = prepared.paths.ConfigFile
	CfgProvider = prepared.provider
	Common = prepared.common
	return nil
}

func loadCommonSettingsFrom(cfg ConfigProvider, workPath string) (CommonSettings, error) {
	if cfg == nil {
		return CommonSettings{}, errors.New("configuration provider is not initialized")
	}
	runMode, err := loadRunModeFrom(cfg)
	if err != nil {
		return CommonSettings{}, err
	}
	logCfg, err := loadLogGlobalFrom(cfg, workPath)
	if err != nil {
		return CommonSettings{}, err
	}
	gracefulCfg, err := loadGracefulSettingsFrom(cfg)
	if err != nil {
		return CommonSettings{}, err
	}
	return CommonSettings{RunMode: runMode, Log: logCfg, Graceful: gracefulCfg}, nil
}

type loadedWebSettings struct {
	server   ServerSettings
	api      APISettings
	security SecuritySettings
	ip       IPActiveSettings
	database DatabaseSettings
	redis    RedisSettings
}

// loadWebSettingsFrom runs exactly the same loaders for preflight and runtime.
func loadWebSettingsFrom(cfg ConfigProvider, common CommonSettings, workPath string) (loadedWebSettings, error) {
	server, err := loadServerFrom(cfg, common.RunMode)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load server settings: %w", err)
	}
	api, err := loadAPIFrom(cfg, common.RunMode, server)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load API settings: %w", err)
	}
	security, err := loadSecurityFrom(cfg, common.RunMode)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load security settings: %w", err)
	}
	ipPolicies, err := loadIPActiveFrom(cfg)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load IP admission settings: %w", err)
	}
	database, err := loadDBSetting(cfg, workPath)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load database settings: %w", err)
	}
	return loadedWebSettings{
		server: server, api: api, security: security, ip: ipPolicies,
		database: database, redis: loadRedisFrom(cfg),
	}, nil
}

// CheckWebConfig checks static Web settings without logging, opening DB/Redis,
// opening a listener, or changing any package-level setting.
func CheckWebConfig(args PathOptions, envs []string) error {
	prepared, err := prepareCommonConfig(args, envs)
	if err != nil {
		return err
	}
	_, err = loadWebSettingsFrom(prepared.provider, prepared.common, prepared.paths.WorkPath)
	return err
}

// LoadSettings is the legacy Web publication point; all loaders must succeed first.
func LoadSettings() error {
	if CfgProvider == nil {
		return errors.New("configuration provider is not initialized")
	}
	loaded, err := loadWebSettingsFrom(CfgProvider, Common, AppWorkPath)
	if err != nil {
		return err
	}
	Server = loaded.server
	API = loaded.api
	Security = loaded.security
	IPActive = loaded.ip
	Database = loaded.database
	Redis = loaded.redis
	return nil
}

func stringOrDefault(key ConfigKey, fallback string) string {
	if key == nil || !key.IsSet() {
		return fallback
	}
	return key.String()
}

// loadRunModeFrom kiểm tra runtime mode trước khi nạp subsystem khác.
func loadRunModeFrom(cfg ConfigProvider) (string, error) {
	mode := stringOrDefault(cfg.Section("").Key("RUN_MODE"), "development")
	switch mode {
	case "development", "production", "test":
		return mode, nil
	default:
		return "", fmt.Errorf("invalid RUN_MODE %q", mode)
	}
}
