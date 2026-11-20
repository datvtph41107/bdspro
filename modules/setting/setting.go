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

// InitWorkPathAndCommonConfig là một bootstrap boundary.
// Không thay globals nếu resolve, parse, env hoặc common validation lỗi.
func InitWorkPathAndCommonConfig(args PathOptions, envs []string) error {
	paths, err := ResolvePaths(args)
	if err != nil {
		return fmt.Errorf("resolve application paths: %w", err)
	}
	provider, err := NewConfigProviderFromFile(paths.ConfigFile)
	if err != nil {
		return err
	}
	if err := applyEnvironmentOverrides(provider, envs); err != nil {
		return fmt.Errorf("apply environment overrides: %w", err)
	}

	// Gitea: WORK_PATH từ INI có thể đổi effective work path.
	workPath := paths.WorkPath
	configWorkPath := ConfigSectionKeyString(provider.Section(""), "WORK_PATH")
	if configWorkPath != "" {
		if !filepath.IsAbs(configWorkPath) {
			return fmt.Errorf("WORK_PATH in %q must be absolute path: %q", paths.ConfigFile, configWorkPath)
		}
		configWorkPath = filepath.Clean(configWorkPath)
		if args.WorkPath != "" && !samePath(paths.WorkPath, configWorkPath) {
			return fmt.Errorf("WORK_PATH %q conflicts with --work-path %q", configWorkPath, paths.WorkPath)
		}
		workPath = configWorkPath
	}
	customPath := filepath.Join(workPath, "custom")

	common, err := loadCommonSettingsFrom(provider, workPath)
	if err != nil {
		return fmt.Errorf("load common settings: %w", err)
	}

	// Publish sau cùng; không sửa process CWD và không ghi app.ini.
	AppWorkPath = workPath
	CustomPath = customPath
	CustomConf = paths.ConfigFile
	CfgProvider = provider
	Common = common
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

// LoadSettings đọc cấu hình Web; không mở DB, Redis hay listener.
// Mọi domain global chỉ publish nếu toàn bộ loaders trả về thành công.
func LoadSettings() error {
	if CfgProvider == nil {
		return errors.New("configuration provider is not initialized")
	}
	server, err := loadServerFrom(CfgProvider, Common.RunMode)
	if err != nil {
		return fmt.Errorf("load server settings: %w", err)
	}
	api, err := loadAPIFrom(CfgProvider, Common.RunMode, server)
	if err != nil {
		return fmt.Errorf("load API settings: %w", err)
	}
	security, err := loadSecurityFrom(CfgProvider, Common.RunMode)
	if err != nil {
		return fmt.Errorf("load security settings: %w", err)
	}
	ipPolicies, err := loadIPActiveFrom(CfgProvider)
	if err != nil {
		return fmt.Errorf("load IP admission settings: %w", err)
	}
	database, err := loadDBSetting(CfgProvider, AppWorkPath)
	if err != nil {
		return fmt.Errorf("load database settings: %w", err)
	}
	redis := loadRedisFrom(CfgProvider)

	Server = server
	API = api
	Security = security
	IPActive = ipPolicies
	Database = database
	Redis = redis
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
