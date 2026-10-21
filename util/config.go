package util

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config stores the application settings loaded from app.env.
type Config struct {
	AllowedOrigins       []string      `mapstructure:"ALLOWED_ORIGINS"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	VerifyEmailURL       string        `mapstructure:"VERIFY_EMAIL_URL"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword  string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
}

// LoadConfig reads app.env and lets process environment override file keys.
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

	if config.DBSource == "" || config.ServerAddress == "" {
		return config, fmt.Errorf("DB_SOURCE and SERVER_ADDRESS are required")
	}
	if len(config.TokenSymmetricKey) < 32 {
		return config, fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= 0 {
		return config, fmt.Errorf("token durations must be positive")
	}
	return config, nil
}
