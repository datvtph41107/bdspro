package util

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// Values are read by Viper from app.env or environment variables.
type Config struct {
	Environment     string `mapstructure:"ENVIRONMENT"`
	DBDriver        string `mapstructure:"DB_DRIVER"`
	DBSource        string `mapstructure:"DB_SOURCE"`
	ServerAddress   string `mapstructure:"SERVER_ADDRESS"`
	AllowedOrigin   string `mapstructure:"ALLOWED_ORIGIN"`
	SMTPAddr        string `mapstructure:"SMTP_ADDR"`
	SMTPUser        string `mapstructure:"SMTP_USER"`
	SMTPPassword    string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom        string `mapstructure:"SMTP_FROM"`
	VerificationURL string `mapstructure:"VERIFICATION_URL"`
}

// LoadConfig reads configuration from app.env and environment variables.
func LoadConfig(path string) (config Config, err error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("app")
	v.SetConfigType("env")
	v.AutomaticEnv()

	// Environment-only values must also be known to Unmarshal.
	for _, key := range []string{
		"ENVIRONMENT", "DB_DRIVER", "DB_SOURCE", "SERVER_ADDRESS",
		"ALLOWED_ORIGIN", "SMTP_ADDR", "SMTP_USER", "SMTP_PASSWORD",
		"SMTP_FROM", "VERIFICATION_URL",
	} {
		if err = v.BindEnv(key); err != nil {
			return config, fmt.Errorf("bind %s: %w", key, err)
		}
	}

	if err = v.ReadInConfig(); err != nil {
		return config, err
	}
	err = v.Unmarshal(&config)
	return
}
