package setting

import "time"

// Authentication exposes the token settings consumed by services/auth.
type Authentication struct {
    JWTSecret string
    AccessTokenTTL time.Duration
    RefreshTokenTTL time.Duration
}

func (c Config) Authentication() Authentication {
    return Authentication{
        JWTSecret: c.TokenSymmetricKey,
        AccessTokenTTL: c.AccessTokenDuration,
        RefreshTokenTTL: c.RefreshTokenDuration,
    }
}
