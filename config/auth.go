package config

import "time"

// Authentication selects the one account JWT configuration used by the API.
// Laravel guards resolve a driver and provider by name. BDSPro currently has
// only one account identity and JWT bearer authentication; no runtime guard
// registry or unused web/admin providers are necessary.
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
