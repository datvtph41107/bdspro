package setting

import (
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

var (
	AppVer       string
	AppBuiltWith string
	AppStartTime time.Time
	AppWorkPath  string
	CustomPath   string
	CustomConf   string
	CfgProvider  ConfigProvider
	Common       CommonSettings
)

type CommonSettings struct {
	RunMode string
	Log     LogGlobalConfig
}

// Config is the immutable Web subsystem input consumed by existing services.
// Domain setting variables below remain the source of truth during startup.
type Config struct {
	IPActivePolicies     map[ipactive.Action]ipactive.Policy
	Environment          string
	LogFormat            string
	LogLevel             string
	LogMode              string
	LogRootPath          string
	LogFileName          string
	OIDC                 OAuth2Provider
	PhoneDelivery        string
	SMSGatewayURL        string
	SMSGatewayToken      string
	ZaloAccessToken      string
	ZaloTemplateID       string
	ZaloCodeField        string
	AllowedOrigins       []string
	TrustedProxies       []string
	DBSource             string
	MigrationURL         string
	RedisAddress         string
	HTTPServerAddress    string
	BrowserPublicOrigin  string
	TokenSymmetricKey    string
	AuditKey             string
	AuditRetentionDays   int
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	EmailSenderName      string
	EmailSenderAddress   string
	EmailSenderPassword  string
	SMTPAddress          string
	SMTPAuth             bool
	VerifyEmailURL       string
}

// InitWorkPathAndCommonConfig prepares one provider before publishing it.
func InitWorkPathAndCommonConfig(options PathOptions, envs []string) error {
	paths, err := ResolvePaths(options)
	if err != nil {
		return err
	}
	provider, err := NewConfigProviderFromFile(paths.ConfigFile)
	if err != nil {
		return err
	}
	if err = applyEnvironmentOverrides(provider, envs); err != nil {
		return err
	}
	common, err := loadCommonSettingsFrom(provider, paths.WorkPath)
	if err != nil {
		return err
	}

	AppWorkPath, CustomPath, CustomConf = paths.WorkPath, paths.CustomPath, paths.ConfigFile
	CfgProvider, Common = provider, common
	return nil
}

func loadCommonSettingsFrom(cfg ConfigProvider, workPath string) (CommonSettings, error) {
	if cfg == nil {
		return CommonSettings{}, fmt.Errorf("configuration provider is not initialized")
	}
	runMode := stringOrDefault(cfg.Section("").Key("RUN_MODE"), "development")
	switch runMode {
	case "development", "production", "test":
	default:
		return CommonSettings{}, fmt.Errorf("invalid RUN_MODE %q", runMode)
	}
	logCfg, err := loadLogGlobalFrom(cfg, workPath)
	if err != nil {
		return CommonSettings{}, err
	}
	return CommonSettings{RunMode: runMode, Log: logCfg}, nil
}

// LoadSettings reads Web settings; connection ownership belongs to routers/init.go.
func LoadSettings() error {
	if CfgProvider == nil {
		return fmt.Errorf("configuration provider is not initialized")
	}
	loadDBSetting(CfgProvider)
	if err := loadServerFrom(CfgProvider); err != nil {
		return err
	}
	loadRedisFrom(CfgProvider)
	if err := loadSecurityFrom(CfgProvider); err != nil {
		return err
	}
	if err := loadAuditFrom(CfgProvider); err != nil {
		return err
	}
	if err := loadOAuth2From(CfgProvider); err != nil {
		return err
	}
	loadPhoneFrom(CfgProvider)
	if err := loadMailerFrom(CfgProvider); err != nil {
		return err
	}
	if err := loadIPActiveFrom(CfgProvider); err != nil {
		return err
	}
	return nil
}

// WebConfig projects loaded domain settings for existing HTTP services.
// It does not reload config or own mutable configuration state.
func WebConfig() Config {
	return Config{
		IPActivePolicies: IPActive.Policies,
		Environment:      Common.RunMode,
		LogFormat:        Common.Log.Format, LogLevel: Common.Log.Level,
		LogMode: Common.Log.Mode, LogRootPath: Common.Log.RootPath,
		LogFileName:   Common.Log.FileName,
		OIDC:          OAuth2,
		PhoneDelivery: Phone.Delivery, SMSGatewayURL: Phone.SMSGatewayURL,
		SMSGatewayToken: Phone.SMSGatewayToken, ZaloAccessToken: Phone.ZaloAccessToken,
		ZaloTemplateID: Phone.ZaloTemplateID, ZaloCodeField: Phone.ZaloCodeField,
		AllowedOrigins: Server.AllowedOrigins, TrustedProxies: Server.TrustedProxies,
		DBSource: Database.DBSource, MigrationURL: Database.MigrationURL,
		RedisAddress: Redis.Address, HTTPServerAddress: Server.HTTPServerAddress,
		BrowserPublicOrigin: Server.BrowserPublicOrigin,
		TokenSymmetricKey:   Security.TokenSymmetricKey,
		AuditKey:            Audit.HMACKey, AuditRetentionDays: Audit.RetentionDays,
		AccessTokenDuration:  Security.AccessTokenDuration,
		RefreshTokenDuration: Security.RefreshTokenDuration,
		EmailSenderName:      Mailer.EmailSenderName, EmailSenderAddress: Mailer.EmailSenderAddress,
		EmailSenderPassword: Mailer.EmailSenderPassword, SMTPAddress: Mailer.SMTPAddress,
		SMTPAuth: Mailer.SMTPAuth, VerifyEmailURL: Mailer.VerifyEmailURL,
	}
}

func stringOrDefault(key ConfigKey, fallback string) string {
	if !key.IsSet() {
		return fallback
	}
	return key.String()
}
