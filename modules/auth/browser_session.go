package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	BrowserSessionTTL    = 8 * time.Hour
	BrowserCSRFTokenTTL  = 5 * time.Minute
	browserCookieVersion = "v1"
	browserCSRFVersion   = "c1"
)

var ErrBrowserProofInvalid = errors.New("browser session or CSRF proof invalid")

// BrowserSessionCodec authenticates a reference to an existing DB session.
// Decoding the cookie alone MUST NEVER authorize a request: callers must
// look up the session and verify is_blocked=false and expires_at>now.
// The cookie never stores access tokens, refresh tokens, or account IDs.
// Keys are domain-separated from the existing JWT signing and consent keys.
type BrowserSessionCodec struct {
	cookieKey [sha256.Size]byte
	csrfKey   [sha256.Size]byte
}

func NewBrowserSessionCodec(master string) (*BrowserSessionCodec, error) {
	if len(master) < 32 {
		return nil, errors.New("browser sessions require at least a 32-byte secret")
	}
	c := &BrowserSessionCodec{}
	c.cookieKey = deriveBrowserKey(master, "bdspro.browser.session.v1")
	c.csrfKey = deriveBrowserKey(master, "bdspro.browser.csrf.v1")
	return c, nil
}

func deriveBrowserKey(secret, context string) [sha256.Size]byte {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(context))
	var key [sha256.Size]byte
	copy(key[:], m.Sum(nil))
	return key
}

// IssueCookie returns an HMAC-authenticated, short-lived reference to an
// EXISTING database session, never a refresh token. The caller must use
// HttpOnly, Secure, SameSite and Path=/ when setting the HTTP cookie.
func (c *BrowserSessionCodec) IssueCookie(sessionID string, now, sessionExpiry time.Time) (string, error) {
	if c == nil || !canonicalUUID(sessionID) || !sessionExpiry.After(now) {
		return "", ErrBrowserProofInvalid
	}
	exp := now.Add(BrowserSessionTTL)
	if sessionExpiry.Before(exp) {
		exp = sessionExpiry
	}
	if !exp.After(now) {
		return "", ErrBrowserProofInvalid
	}
	content := browserCookieVersion + "." + sessionID + "." + strconv.FormatInt(exp.Unix(), 10)
	return content + "." + signBrowser(c.cookieKey, content), nil
}

// ReadCookie verifies the signature and TTL. It does NOT verify that the
// referenced database session remains active (the middleware must do so).
func (c *BrowserSessionCodec) ReadCookie(token string, now time.Time) (string, error) {
	if c == nil || len(token) > 256 {
		return "", ErrBrowserProofInvalid
	}
	p := strings.Split(token, ".")
	if len(p) != 4 || p[0] != browserCookieVersion || !canonicalUUID(p[1]) {
		return "", ErrBrowserProofInvalid
	}
	exp, err := strconv.ParseInt(p[2], 10, 64)
	if err != nil || exp <= now.Unix() || exp > now.Add(BrowserSessionTTL+time.Second).Unix() {
		return "", ErrBrowserProofInvalid
	}
	body := strings.Join(p[:3], ".")
	if !validBrowserMAC(c.cookieKey, body, p[3]) {
		return "", ErrBrowserProofInvalid
	}
	return p[1], nil
}

// IssueCSRF generates a signed form token bound to a browser context and an
// explicit operation. For login, context is a random nonce stored in a
// same-origin host-only cookie; for consent, context is the DB session UUID.
// These tokens do NOT replace an Origin/Fetch-Metadata check.
func (c *BrowserSessionCodec) IssueCSRF(context, purpose string, now time.Time) (string, error) {
	if c == nil || !validBrowserPurpose(context, purpose) {
		return "", ErrBrowserProofInvalid
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	body := browserCSRFVersion + "." + base64.RawURLEncoding.EncodeToString(nonce[:]) + "." + strconv.FormatInt(now.Add(BrowserCSRFTokenTTL).Unix(), 10)
	// Never put context or purpose into the form; authenticate both via MAC.
	macInput := body + "|" + purpose + "|" + context
	return body + "." + signBrowser(c.csrfKey, macInput), nil
}

func (c *BrowserSessionCodec) VerifyCSRF(token, context, purpose string, now time.Time) error {
	if c == nil || len(token) > 256 || !validBrowserPurpose(context, purpose) {
		return ErrBrowserProofInvalid
	}
	p := strings.Split(token, ".")
	if len(p) != 4 || p[0] != browserCSRFVersion {
		return ErrBrowserProofInvalid
	}
	nonce, err := base64.RawURLEncoding.DecodeString(p[1])
	if err != nil || len(nonce) != 32 || base64.RawURLEncoding.EncodeToString(nonce) != p[1] {
		return ErrBrowserProofInvalid
	}
	exp, err := strconv.ParseInt(p[2], 10, 64)
	if err != nil || exp <= now.Unix() || exp > now.Add(BrowserCSRFTokenTTL+time.Second).Unix() {
		return ErrBrowserProofInvalid
	}
	body := strings.Join(p[:3], ".")
	if !validBrowserMAC(c.csrfKey, body+"|"+purpose+"|"+context, p[3]) {
		return ErrBrowserProofInvalid
	}
	return nil
}

func signBrowser(key [sha256.Size]byte, data string) string {
	mac := hmac.New(sha256.New, key[:])
	_, _ = mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func validBrowserMAC(key [sha256.Size]byte, data, encoded string) bool {
	got, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(got) != sha256.Size {
		return false
	}
	expected := signBrowser(key, data)
	raw, _ := base64.RawURLEncoding.DecodeString(expected)
	return hmac.Equal(got, raw)
}
func validBrowserPurpose(context, purpose string) bool {
	return len(context) > 0 && len(context) <= 128 && len(purpose) > 0 && len(purpose) <= 64
}
func canonicalUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, ch := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if ch != '-' {
				return false
			}
			continue
		}
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			return false
		}
	}
	return true
}
