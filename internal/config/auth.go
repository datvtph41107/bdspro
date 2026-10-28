package config

import "time"

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
