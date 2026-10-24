package util

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config stores BDSPro settings from app.env or process environment.
type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	GoogleClientID       string        `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret   string        `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURI    string        `mapstructure:"GOOGLE_REDIRECT_URI"`
	AllowedOrigins       []string      `mapstructure:"ALLOWED_ORIGINS"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	MigrationURL         string        `mapstructure:"MIGRATION_URL"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword  string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
	VerifyEmailURL       string        `mapstructure:"VERIFY_EMAIL_URL"`
}

// LoadConfig reads app.env, allowing environment variables to override file values.
func LoadConfig(path string) (config Config, err error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("app")
	v.SetConfigType("env")
	v.AutomaticEnv()
	for _, name := range []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REDIRECT_URI"} {
		if err = v.BindEnv(name); err != nil { return config, err }
	}

	if err = v.ReadInConfig(); err != nil {
		return config, err
	}
	if err = v.Unmarshal(&config); err != nil {
		return config, err
	}

	if config.DBSource == "" || config.MigrationURL == "" || config.HTTPServerAddress == "" {
		return config, fmt.Errorf("DB_SOURCE, MIGRATION_URL and HTTP_SERVER_ADDRESS are required")
	}
	if len(config.TokenSymmetricKey) < 32 {
		return config, fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= 0 {
		return config, fmt.Errorf("token durations must be positive")
	}
	if config.GoogleClientID != "" || config.GoogleClientSecret != "" || config.GoogleRedirectURI != "" {
        if config.GoogleClientID == "" || config.GoogleClientSecret == "" || config.GoogleRedirectURI == "" {
            return config, fmt.Errorf("Google OAuth requires GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URI together")
        }
        u, e := url.Parse(config.GoogleRedirectURI)
        if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/users/google/callback" {
            return config, fmt.Errorf("GOOGLE_REDIRECT_URI must be the absolute /users/google/callback URL")
        }
        if u.Scheme != "https" && !(u.Scheme == "http" && (strings.EqualFold(u.Hostname(), "localhost") || u.Hostname() == "127.0.0.1")) {
            return config, fmt.Errorf("GOOGLE_REDIRECT_URI requires https outside localhost")
        }
    }
	return config, nil
}
