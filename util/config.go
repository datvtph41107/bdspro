package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config stores the Lecture 37 authentication service settings.
type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
}

// LoadConfig uses .env for local overrides, app.env as the Lecture 37
// checked-in fallback, and process environment variables with top priority.
func LoadConfig(path string) (config Config, err error) {
	v := viper.New()
	v.SetConfigType("env")

	keys := []string{
		"ENVIRONMENT",
		"DB_SOURCE",
		"SERVER_ADDRESS",
		"TOKEN_SYMMETRIC_KEY",
		"ACCESS_TOKEN_DURATION",
		"REFRESH_TOKEN_DURATION",
	}
	for _, key := range keys {
		if err = v.BindEnv(key); err != nil {
			return config, err
		}
	}

	v.SetConfigFile(filepath.Join(path, ".env"))
	if err = v.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return config, err
		}
		v.SetConfigFile(filepath.Join(path, "app.env"))
		if err = v.ReadInConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return config, err
		}
	}
	if err = v.Unmarshal(&config); err != nil {
		return config, err
	}
	if config.DBSource == "" || config.ServerAddress == "" {
		return config, fmt.Errorf("DB_SOURCE and SERVER_ADDRESS are required")
	}
	if len(config.TokenSymmetricKey) < 32 {
		return config, fmt.Errorf("TOKEN_SYMMETRIC_KEY must be at least 32 bytes")
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= 0 {
		return config, fmt.Errorf("token durations must be positive")
	}
	return config, nil
}
