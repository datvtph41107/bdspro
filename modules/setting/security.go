package setting

import (
    "fmt"
    "time"
)

var Security struct {
    TokenSymmetricKey string
    AccessTokenDuration time.Duration
    RefreshTokenDuration time.Duration
}

// loadSecurityFrom owns signing-key and session-lifetime configuration.
func loadSecurityFrom(rootCfg ConfigProvider) error {
    sec := rootCfg.Section("security")
    key := sec.Key("TOKEN_SYMMETRIC_KEY").String()
    if err := validateTokenSigningKey(Common.RunMode, key); err != nil { return err }

    accessTTL, err := time.ParseDuration(sec.Key("ACCESS_TOKEN_DURATION").String())
    if err != nil { return fmt.Errorf("ACCESS_TOKEN_DURATION: %w", err) }
    refreshTTL, err := time.ParseDuration(sec.Key("REFRESH_TOKEN_DURATION").String())
    if err != nil { return fmt.Errorf("REFRESH_TOKEN_DURATION: %w", err) }
    if accessTTL <= 0 || refreshTTL <= 0 { return fmt.Errorf("token durations must be positive") }

    Security.TokenSymmetricKey = key
    Security.AccessTokenDuration = accessTTL
    Security.RefreshTokenDuration = refreshTTL
    return nil
}

func validateTokenSigningKey(environment, key string) error {
    if environment != "development" && key == "01234567890123456789012345678901" {
        return fmt.Errorf("production signing key must differ from documented development example")
    }
    if len(key) < 32 { return fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes") }
    return nil
}
