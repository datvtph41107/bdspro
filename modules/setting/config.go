package setting

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

// Config holds validated settings loaded from app.ini and process overrides.
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

// LoadConfig resolves the selected INI file and materializes typed settings.
// Only one config file is read. Environment overrides stay in-memory.
func LoadConfig(filename string) (Config, error) {
	provider, err := InitCfgProvider(filename)
	if err != nil {
		return Config{}, err
	}
	return LoadSettings(provider)
}

type CommonSettings struct {
	RunMode, LogFormat, LogLevel, LogMode, LogRootPath, LogFileName string
}

// LoadCommonSettings reads settings needed before runtime dependencies start.
func LoadCommonSettings(p *ConfigProvider) CommonSettings {
	return CommonSettings{
		RunMode:     p.StringDefault("", "RUN_MODE", "development"),
		LogFormat:   p.StringDefault("log", "FORMAT", "json"),
		LogLevel:    p.StringDefault("log", "LEVEL", "info"),
		LogMode:     p.StringDefault("log", "MODE", "console"),
		LogRootPath: p.StringDefault("log", "ROOT_PATH", "data/log"),
		LogFileName: p.StringDefault("log.file", "FILE_NAME", "bdspro.log"),
	}
}

// LoadSettings validates the complete typed config; it does not open DB/Redis.
func LoadSettings(p *ConfigProvider) (config Config, err error) {
	common := LoadCommonSettings(p)
	config.Environment = common.RunMode
	config.LogFormat = common.LogFormat
	config.LogLevel = common.LogLevel
	config.LogMode = common.LogMode
	config.LogRootPath = common.LogRootPath
	config.LogFileName = common.LogFileName
	get := p.String
	config.DBSource = get("database", "DB_SOURCE")
	config.MigrationURL = get("database", "MIGRATION_URL")
	config.HTTPServerAddress = get("server", "HTTP_SERVER_ADDRESS")
	config.BrowserPublicOrigin = get("server", "PUBLIC_ORIGIN")
	if err := ValidateBrowserPublicOrigin(config.BrowserPublicOrigin, config.Environment); err != nil {
		return config, err
	}
	config.TrustedProxies, err = ParseTrustedProxies(get("server", "TRUSTED_PROXIES"))
	if err != nil {
		return config, err
	}
	config.RedisAddress = get("redis", "ADDR")
	config.TokenSymmetricKey = get("security", "TOKEN_SYMMETRIC_KEY")
	config.AuditKey = get("audit", "HMAC_KEY")
	config.AuditRetentionDays, err = LoadAuditRetentionDays(p)
	if err != nil {
		return config, err
	}
	config.AccessTokenDuration, err = time.ParseDuration(get("security", "ACCESS_TOKEN_DURATION"))
	if err != nil {
		return config, fmt.Errorf("ACCESS_TOKEN_DURATION: %w", err)
	}
	config.RefreshTokenDuration, err = time.ParseDuration(get("security", "REFRESH_TOKEN_DURATION"))
	if err != nil {
		return config, fmt.Errorf("REFRESH_TOKEN_DURATION: %w", err)
	}
	config.OIDC = OAuth2Provider{
		Issuer:           get("oauth2", "ISSUER"),
		AuthorizationURL: get("oauth2", "AUTHORIZATION_URL"),
		TokenURL:         get("oauth2", "TOKEN_URL"),
		JWKSURL:          get("oauth2", "JWKS_URL"),
		ClientID:         get("oauth2", "CLIENT_ID"),
		ClientSecret:     get("oauth2", "CLIENT_SECRET"),
		RedirectURI:      get("oauth2", "REDIRECT_URI"),
	}
	if config.OIDC.Enabled() {
		if err := config.OIDC.Validate(); err != nil {
			return config, err
		}
	}
	if get("oauth", "GOOGLE_CLIENT_ID") != "" || get("oauth", "GOOGLE_CLIENT_SECRET") != "" || get("oauth", "GOOGLE_REDIRECT_URI") != "" {
		return config, fmt.Errorf("legacy OAuth provider settings are unsupported; configure [oauth2]")
	}
	config.PhoneDelivery = get("phone", "PHONE_DELIVERY")
	config.SMSGatewayURL = get("phone", "SMS_GATEWAY_URL")
	config.SMSGatewayToken = get("phone", "SMS_GATEWAY_TOKEN")
	config.ZaloAccessToken = get("phone", "ZALO_ACCESS_TOKEN")
	config.ZaloTemplateID = get("phone", "ZALO_TEMPLATE_ID")
	config.ZaloCodeField = p.StringDefault("phone", "ZALO_CODE_FIELD", "otp")
	for _, origin := range strings.Split(get("cors", "ALLOWED_ORIGINS"), ",") {
		if s := strings.TrimSpace(origin); s != "" {
			config.AllowedOrigins = append(config.AllowedOrigins, s)
		}
	}
	config.EmailSenderName = get("mailer", "EMAIL_SENDER_NAME")
	config.EmailSenderAddress = get("mailer", "EMAIL_SENDER_ADDRESS")
	config.EmailSenderPassword = get("mailer", "EMAIL_SENDER_PASSWORD")
	config.SMTPAddress = get("mailer", "SMTP_ADDRESS")
	config.SMTPAuth, err = strconv.ParseBool(p.StringDefault("mailer", "SMTP_AUTH", "false"))
	if err != nil {
		return config, fmt.Errorf("SMTP_AUTH: %w", err)
	}
	config.VerifyEmailURL = get("mailer", "VERIFY_EMAIL_URL")
	config.IPActivePolicies, err = LoadIPActivePolicies(p)
	if err != nil {
		return config, err
	}
	if err := (LogGlobalConfig{
		Mode: config.LogMode, Format: config.LogFormat, Level: config.LogLevel,
		RootPath: config.LogRootPath, FileName: config.LogFileName,
	}).Validate(); err != nil {
		return config, err
	}
	if config.DBSource == "" || config.MigrationURL == "" || config.HTTPServerAddress == "" {
		return config, fmt.Errorf("DB_SOURCE, MIGRATION_URL and HTTP_SERVER_ADDRESS are required")
	}
	if err := validateTokenSigningKey(config.Environment, config.TokenSymmetricKey); err != nil {
		return config, err
	}
	if config.AuditKey == "" && config.Environment == "development" {
		// Development-only compatibility for in-memory tests and old local INI.
		// Production requires a separately managed audit key.
		config.AuditKey = config.TokenSymmetricKey
	}
	if len(config.AuditKey) < 32 {
		return config, fmt.Errorf("[audit] HMAC_KEY must contain at least 32 bytes")
	}
	if config.Environment != "development" && config.AuditKey == "abcdef0123456789abcdef0123456789" {
		return config, fmt.Errorf("production audit key must differ from development example")
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= 0 {
		return config, fmt.Errorf("token durations must be positive")
	}
	return config, nil
}
