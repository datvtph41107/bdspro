package util

import (
	"fmt"
	"net"

	"github.com/spf13/viper"
)

// Config stores application settings read from app.env or environment variables.
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

// LoadConfig follows SimpleBank's Viper file-and-environment workflow.
// A new Viper instance keeps independent test calls isolated.
func LoadConfig(path string) (config Config, err error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("app")
	v.SetConfigType("env")
	v.AutomaticEnv()

	if err = v.ReadInConfig(); err != nil {
		return
	}
	if err = v.Unmarshal(&config); err != nil {
		return
	}
	err = config.validate()
	return
}

// validate keeps deployment rules at the configuration boundary, not in main.
func (config Config) validate() error {
	if config.DBDriver != "postgres" {
		return fmt.Errorf("DB_DRIVER must be postgres")
	}
	if config.DBSource == "" {
		return fmt.Errorf("DB_SOURCE required")
	}
	host, _, err := net.SplitHostPort(config.ServerAddress)
	if err != nil {
		return fmt.Errorf("invalid SERVER_ADDRESS: %w", err)
	}
	if config.Environment == "development" {
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return fmt.Errorf("development verification tokens require a loopback listener")
		}
		return nil
	}
	if config.SMTPAddr == "" {
		return fmt.Errorf("SMTP_ADDR required outside development")
	}
	if config.AllowedOrigin == "" {
		return fmt.Errorf("ALLOWED_ORIGIN required outside development")
	}
	return nil
}
