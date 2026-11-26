package setting

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

var (
	AppVer       string
	AppBuiltWith string
	AppStartTime time.Time

	Common CommonSettings
)

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
	mailer   MailerSettings
	sms      SMSSettings
	google   GoogleSettings
}

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
	mailer, err := loadMailerFrom(cfg)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load mailer settings: %w", err)
	}
	sms, err := loadSMSFrom(cfg)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load SMS settings: %w", err)
	}
	google, err := loadGoogleFrom(cfg, server)
	if err != nil {
		return loadedWebSettings{}, fmt.Errorf("load Google settings: %w", err)
	}
	return loadedWebSettings{
		server: server, api: api, security: security, ip: ipPolicies,
		database: database, redis: loadRedisFrom(cfg), mailer: mailer, sms: sms, google: google,
	}, nil
}

type WebConfig struct {
	Paths    Paths
	Common   CommonSettings
	Server   ServerSettings
	API      APISettings
	Security SecuritySettings
	IPActive IPActiveSettings
	Database DatabaseSettings
	Redis    RedisSettings
	Mailer   MailerSettings
	SMS      SMSSettings
	Google   GoogleSettings
}

func LoadWebConfig(args PathOptions, envs []string) (WebConfig, error) {
	prepared, err := prepareCommonConfig(args, envs)
	if err != nil {
		return WebConfig{}, err
	}
	loaded, err := loadWebSettingsFrom(prepared.provider, prepared.common, prepared.paths.WorkPath)
	if err != nil {
		return WebConfig{}, err
	}
	return WebConfig{
		Paths: prepared.paths, Common: prepared.common,
		Server: loaded.server, API: loaded.api, Security: loaded.security,
		IPActive: loaded.ip, Database: loaded.database, Redis: loaded.redis, Mailer: loaded.mailer, SMS: loaded.sms, Google: loaded.google,
	}, nil
}

func LoadWebSettings(args PathOptions, envs []string) error {
	cfg, err := LoadWebConfig(args, envs)
	if err != nil {
		return err
	}
	return publishWebConfig(cfg)
}

func publishWebConfig(cfg WebConfig) error {
	return publishSettings(func() {
		publishCommonSettings(cfg.Paths, cfg.Common)
		Server = cfg.Server
		Server.TrustedProxies = append([]string(nil), cfg.Server.TrustedProxies...)
		Server.AllowedOrigins = append([]string(nil), cfg.Server.AllowedOrigins...)
		API = cfg.API
		Security = cfg.Security
		IPActive = cfg.IPActive.clone()
		Database = cfg.Database
		Redis = cfg.Redis
		Mailer = cfg.Mailer
		SMS = cfg.SMS
		Google = cfg.Google
	})
}

var publication struct {
	sync.Mutex
	done bool
}

func publishSettings(set func()) error {
	publication.Lock()
	defer publication.Unlock()
	if publication.done {
		return errors.New("setting: process configuration already initialized")
	}
	set()
	publication.done = true
	return nil
}

func publishCommonSettings(paths Paths, common CommonSettings) {
	AppWorkPath = paths.WorkPath
	CustomPath = paths.CustomPath
	CustomConf = paths.ConfigFile
	Common = common
}

func CheckWebConfig(args PathOptions, envs []string) error {
	_, err := LoadWebConfig(args, envs)
	return err
}

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
