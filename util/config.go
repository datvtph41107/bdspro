package util

import (
    "fmt"
    "time"

    "github.com/spf13/viper"
)

type Config struct {
    Environment          string        `mapstructure:"ENVIRONMENT"`
    DBSource             string        `mapstructure:"DB_SOURCE"`
    ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
    TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
    AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
    RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
}

func LoadConfig(path string) (config Config, err error) {
    v := viper.New()
    v.AddConfigPath(path)
    v.SetConfigName("app")
    v.SetConfigType("env")
    v.AutomaticEnv()
    if err = v.ReadInConfig(); err != nil { return }
    if err = v.Unmarshal(&config); err != nil { return }
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
