package util

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config stores BDSPro settings from app.env or process environment.
type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
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

	if err = v.ReadInConfig(); err != nil {
		return config, err
	}
	if err = v.Unmarshal(&config); err != nil {
		return config, err
	}

	if config.DBSource == "" || config.MigrationURL == "" || config.HTTPServerAddress == "" || config.GRPCServerAddress == "" {
		return config, fmt.Errorf("DB_SOURCE, MIGRATION_URL, HTTP_SERVER_ADDRESS and GRPC_SERVER_ADDRESS are required")
	}
	if len(config.TokenSymmetricKey) < 32 {
		return config, fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= 0 {
		return config, fmt.Errorf("token durations must be positive")
	}
	return config, nil
}
