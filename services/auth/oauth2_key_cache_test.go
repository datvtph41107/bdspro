package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestOIDCKeyCacheCachesPublicKeysAndRefreshesUnknownKid(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var requested atomic.Int32
	var activeKid atomic.Value
	activeKid.Store("first-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requested.Add(1)
		encode := base64.RawURLEncoding.EncodeToString
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA", "kid": activeKid.Load().(string),
				"alg": "RS256", "use": "sig",
				"n": encode(key.PublicKey.N.Bytes()),
				"e": encode(big.NewInt(int64(key.PublicKey.E)).Bytes()),
			}},
		})
	}))
	defer server.Close()
	cache := &OIDCKeyCache{}
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := cache.get(ctx, server.Client(), server.URL, "first-key"); err != nil {
			t.Fatal(err)
		}
	}
	if n := requested.Load(); n != 1 {
		t.Fatalf("cached key fetched %d times, expected once", n)
	}
	activeKid.Store("rotated-key")
	if _, err := cache.get(ctx, server.Client(), server.URL, "rotated-key"); err != nil {
		t.Fatalf("new key ID not refreshed: %v", err)
	}
	if n := requested.Load(); n != 2 {
		t.Fatalf("rotated key fetched %d times, expected twice", n)
	}
	if _, err := cache.get(ctx, server.Client(), server.URL, "first-key"); err != nil {
		t.Fatalf("previously trusted cached key not usable before TTL: %v", err)
	}
	if n := requested.Load(); n != 2 {
		t.Fatalf("cache miss for a valid unexpired key: %d", n)
	}
}
