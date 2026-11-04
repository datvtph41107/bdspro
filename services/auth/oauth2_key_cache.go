package auth

import (
	"context"
	"crypto/rsa"
	"net/http"
	"sync"
	"time"
)

// OIDCKeyCache is a bounded, issuer-endpoint-scoped cache of public JWKS keys.
// It caches only public verification keys, never ID/access/refresh tokens.
// Unknown KIDs trigger a fresh fetch for key rotation. Cached keys expire.
type OIDCKeyCache struct {
	mu      sync.RWMutex
	entries map[string]oidcCachedKey
}

type oidcCachedKey struct {
	publicKey *rsa.PublicKey
	expires   time.Time
}

const (
	oidcKeyCacheTTL  = 2 * time.Minute
	oidcKeyCacheSize = 32
)

func (cache *OIDCKeyCache) get(ctx context.Context, client *http.Client, uri, kid string) (*rsa.PublicKey, error) {
	if cache == nil {
		return fetchOIDCRSAKey(ctx, client, uri, kid)
	}
	keyID := uri + "\x00" + kid
	cache.mu.RLock()
	entry, ok := cache.entries[keyID]
	cache.mu.RUnlock()
	if ok && time.Now().Before(entry.expires) {
		return entry.publicKey, nil
	}
	// Do not hold a lock during provider I/O. Failed refreshes fail closed.
	publicKey, err := fetchOIDCRSAKey(ctx, client, uri, kid)
	if err != nil {
		return nil, err
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.entries == nil || len(cache.entries) >= oidcKeyCacheSize {
		cache.entries = make(map[string]oidcCachedKey)
	}
	cache.entries[keyID] = oidcCachedKey{
		publicKey: publicKey, expires: time.Now().Add(oidcKeyCacheTTL),
	}
	return publicKey, nil
}
