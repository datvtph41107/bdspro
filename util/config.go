package util

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

// Config stores application settings read from app.env or environment variables.
type Config struct {
	Environment           string `mapstructure:"ENVIRONMENT"`
	AllowContainerDevBind bool   `mapstructure:"ALLOW_CONTAINER_DEV_BIND"`
	DBDriver              string `mapstructure:"DB_DRIVER"`
	DBSource              string `mapstructure:"DB_SOURCE"`
	ServerAddress         string `mapstructure:"SERVER_ADDRESS"`
	GRPCServerAddress     string `mapstructure:"GRPC_SERVER_ADDRESS"`
	AllowedOrigin         string `mapstructure:"ALLOWED_ORIGIN"`
	TokenSymmetricKey     string `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	PhoneOTPKey           string `mapstructure:"PHONE_OTP_KEY"`
	DevSMSOutbox          bool   `mapstructure:"DEV_SMS_OUTBOX"`
	TwilioAccountSID      string `mapstructure:"TWILIO_ACCOUNT_SID"`
	TwilioAuthToken       string `mapstructure:"TWILIO_AUTH_TOKEN"`
	TwilioFrom            string `mapstructure:"TWILIO_FROM"`
	SMTPAddr              string `mapstructure:"SMTP_ADDR"`
	SMTPUser              string `mapstructure:"SMTP_USER"`
	SMTPPassword          string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom              string `mapstructure:"SMTP_FROM"`
	VerificationURL       string `mapstructure:"VERIFICATION_URL"`
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
	if config.GRPCServerAddress != "" {
		grpcHost, grpcPort, err := net.SplitHostPort(config.GRPCServerAddress)
		if err != nil || grpcPort == "" || (grpcHost != "127.0.0.1" && grpcHost != "::1" && grpcHost != "localhost") {
			return fmt.Errorf("GRPC_SERVER_ADDRESS must be a loopback host:port (TLS is not configured)")
		}
		if len(config.TokenSymmetricKey) < 32 {
			return fmt.Errorf("gRPC requires TOKEN_SYMMETRIC_KEY >=32 bytes")
		}
	}
	if config.AllowedOrigin != "" {
		origin, err := url.Parse(config.AllowedOrigin)
		if err != nil || origin.Hostname() == "" || origin.User != nil || origin.Opaque != "" ||
			(origin.Scheme != "https" && origin.Scheme != "http") ||
			origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" ||
			strings.ContainsAny(config.AllowedOrigin, " \t\r\n") ||
			strings.Contains(config.AllowedOrigin, "*") ||
			origin.String() != config.AllowedOrigin {
			return fmt.Errorf("ALLOWED_ORIGIN must be an exact HTTP(S) origin without path, query or wildcard")
		}
		if config.Environment != "development" && origin.Scheme != "https" {
			return fmt.Errorf("ALLOWED_ORIGIN must use HTTPS outside development")
		}
	}
	if config.TokenSymmetricKey != "" && len(config.TokenSymmetricKey) < 32 {
		return fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	if config.PhoneOTPKey != "" && len(config.PhoneOTPKey) < 32 {
		return fmt.Errorf("PHONE_OTP_KEY must contain at least 32 bytes")
	}
	if config.DevSMSOutbox && config.Environment != "development" {
		return fmt.Errorf("DEV_SMS_OUTBOX is allowed only in development")
	}
	if config.DevSMSOutbox && (config.TwilioAccountSID != "" || config.TwilioAuthToken != "" || config.TwilioFrom != "") {
		return fmt.Errorf("DEV_SMS_OUTBOX and Twilio delivery are mutually exclusive")
	}
	if config.DevSMSOutbox && len(config.PhoneOTPKey) < 32 {
		return fmt.Errorf("DEV_SMS_OUTBOX requires PHONE_OTP_KEY")
	}
	if config.TwilioAccountSID != "" || config.TwilioAuthToken != "" || config.TwilioFrom != "" {
		if len(config.TwilioAccountSID) != 34 || !strings.HasPrefix(config.TwilioAccountSID, "AC") ||
			config.TwilioAuthToken == "" || config.PhoneOTPKey == "" ||
			len(config.TwilioFrom) < 9 || config.TwilioFrom[0] != '+' {
			return fmt.Errorf("Twilio SMS requires valid TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM and PHONE_OTP_KEY")
		}
	}
	if config.Environment == "development" {
		if host != "localhost" && host != "127.0.0.1" && host != "::1" && !config.AllowContainerDevBind {
			return fmt.Errorf("development verification tokens require a loopback listener")
		}
		return nil
	}
	if config.AllowContainerDevBind {
		return fmt.Errorf("ALLOW_CONTAINER_DEV_BIND only allowed in development")
	}
	if len(config.TokenSymmetricKey) < 32 || config.TokenSymmetricKey == "dev-only-local-key-change-before-deployment-2026" {
		return fmt.Errorf("TOKEN_SYMMETRIC_KEY must be a non-development secret with at least 32 bytes")
	}
	if config.SMTPAddr == "" {
		return fmt.Errorf("SMTP_ADDR required outside development")
	}
	from, err := mail.ParseAddress(config.SMTPFrom)
	if err != nil || from.Address != config.SMTPFrom {
		return fmt.Errorf("SMTP_FROM must be a valid sender address outside development")
	}
	verificationURL, err := url.Parse(config.VerificationURL)
	if err != nil || verificationURL.Scheme != "https" || verificationURL.Host == "" {
		return fmt.Errorf("VERIFICATION_URL must be an absolute HTTPS URL outside development")
	}
	if config.AllowedOrigin == "" {
		return fmt.Errorf("ALLOWED_ORIGIN required outside development")
	}
	return nil
}
