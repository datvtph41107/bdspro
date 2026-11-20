package setting

import (
	"errors"
	"fmt"
	"time"
)

// SecuritySettings là JWT policy, không giữ runtime token maker.
type SecuritySettings struct {
	TokenSymmetricKey    string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

var Security SecuritySettings

func loadSecurityFrom(cfg ConfigProvider, runMode string) (SecuritySettings, error) {
	sec := cfg.Section("security")
	key := sec.Key("TOKEN_SYMMETRIC_KEY").String()
	if err := validateTokenSigningKey(runMode, key); err != nil {
		return SecuritySettings{}, err
	}
	accessTTL, err := time.ParseDuration(sec.Key("ACCESS_TOKEN_DURATION").String())
	if err != nil {
		return SecuritySettings{}, fmt.Errorf("security.ACCESS_TOKEN_DURATION: %w", err)
	}
	refreshTTL, err := time.ParseDuration(sec.Key("REFRESH_TOKEN_DURATION").String())
	if err != nil {
		return SecuritySettings{}, fmt.Errorf("security.REFRESH_TOKEN_DURATION: %w", err)
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return SecuritySettings{}, errors.New("security token durations must be positive")
	}
	return SecuritySettings{
		TokenSymmetricKey: key, AccessTokenDuration: accessTTL, RefreshTokenDuration: refreshTTL,
	}, nil
}

func validateTokenSigningKey(runMode, key string) error {
	if runMode != "development" && key == "01234567890123456789012345678901" {
		return errors.New("production signing key must differ from development example")
	}
	if len(key) < 32 {
		return errors.New("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	return nil
}
