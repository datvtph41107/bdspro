package setting

import (
	"errors"
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
	var result SecuritySettings
	if err := mapSettings(cfg, "security", &result); err != nil {
		return SecuritySettings{}, err
	}
	if err := validateTokenSigningKey(runMode, result.TokenSymmetricKey); err != nil {
		return SecuritySettings{}, err
	}
	if result.AccessTokenDuration <= 0 || result.RefreshTokenDuration <= 0 {
		return SecuritySettings{}, errors.New("security token durations must be positive")
	}
	return result, nil
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
