package setting

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config holds validated settings loaded from app.ini and process overrides.
type Config struct {
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
	DBSource             string
	MigrationURL         string
	RedisAddress         string
	HTTPServerAddress    string
	TokenSymmetricKey    string
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
	config.RedisAddress = get("redis", "ADDR")
	config.TokenSymmetricKey = get("security", "TOKEN_SYMMETRIC_KEY")
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
	if config.LogMode != "console" && config.LogMode != "file" &&
		config.LogMode != "console,file" && config.LogMode != "file,console" {
		return config, fmt.Errorf("LOG_MODE must be console, file or console,file")
	}
	if config.LogFormat != "json" && config.LogFormat != "console" {
		return config, fmt.Errorf("LOG_FORMAT must be console or json")
	}
	switch config.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return config, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
	}
	if config.DBSource == "" || config.MigrationURL == "" || config.HTTPServerAddress == "" {
		return config, fmt.Errorf("DB_SOURCE, MIGRATION_URL and HTTP_SERVER_ADDRESS are required")
	}
	if config.Environment != "development" && config.TokenSymmetricKey == "01234567890123456789012345678901" {
		return config, fmt.Errorf("production signing key must differ from documented development example")
	}
	if len(config.TokenSymmetricKey) < 32 {
		return config, fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= 0 {
		return config, fmt.Errorf("token durations must be positive")
	}
	return config, nil
}
