package setting

import (
	"fmt"
	"time"
)

// Authentication contains JWT settings for account access and refresh tokens.
type Authentication struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func (c Config) Authentication() Authentication {
	return Authentication{
		JWTSecret:       c.TokenSymmetricKey,
		AccessTokenTTL:  c.AccessTokenDuration,
		RefreshTokenTTL: c.RefreshTokenDuration,
	}
}

// validateTokenSigningKey is shared between HTTP token issuance and the
// worker that derives team-invitation tokens. No command can weaken it.
func validateTokenSigningKey(environment, key string) error {
	if environment != "development" && key == "01234567890123456789012345678901" {
		return fmt.Errorf("production signing key must differ from documented development example")
	}
	if len(key) < 32 {
		return fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	return nil
}
