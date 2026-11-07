package oauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	consentProofTTL       = 5 * time.Minute
	consentProofMaxLength = 8192
	consentProofPrefix    = "v1"
)

var ErrConsentProofInvalid = errors.New("invalid or expired OAuth consent proof")

// ConsentProofFields are the exact request attributes shown during consent
// preview. They are bound to the currently authenticated account, not to a
// client-provided account_id. This is NOT a browser cookie session or CSRF
// protection: those must be established separately for HTML-based consent.
type ConsentProofFields struct {
	AccountID           int64  `json:"account_id"`
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	ResponseType        string `json:"response_type"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	State               string `json:"state"`
	ConsentVersion      int64  `json:"consent_version"`
}

type consentProofPayload struct {
	ConsentProofFields
	IssuedAt  int64 `json:"issued_at"`
	ExpiresAt int64 `json:"expires_at"`
}

// ConsentProofs prevents a caller from changing the approved request between
// GET preview and POST. The key is domain separated from JWT signing.
type ConsentProofs struct{ key [sha256.Size]byte }

func NewConsentProofs(jwtSecret string) (*ConsentProofs, error) {
	if len(jwtSecret) < 32 {
		return nil, errors.New("OAuth consent proof requires a 32-byte secret")
	}
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	_, _ = mac.Write([]byte("oauth.consent-proof.v1"))
	var p ConsentProofs
	copy(p.key[:], mac.Sum(nil))
	return &p, nil
}

func (p *ConsentProofs) Issue(fields ConsentProofFields, now time.Time) (string, error) {
	if p == nil || fields.AccountID <= 0 || fields.ClientID == "" || fields.ConsentVersion < 0 {
		return "", ErrConsentProofInvalid
	}
	body, err := json.Marshal(consentProofPayload{
		ConsentProofFields: fields,
		IssuedAt:           now.Unix(),
		ExpiresAt:          now.Add(consentProofTTL).Unix(),
	})
	if err != nil {
		return "", err
	}
	prefix := consentProofPrefix + "." + base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, p.key[:])
	_, _ = mac.Write([]byte(prefix))
	token := prefix + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if len(token) > consentProofMaxLength {
		return "", ErrConsentProofInvalid
	}
	return token, nil
}

func (p *ConsentProofs) Verify(token string, expected ConsentProofFields, now time.Time) error {
	if p == nil || len(token) == 0 || len(token) > consentProofMaxLength {
		return ErrConsentProofInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != consentProofPrefix {
		return ErrConsentProofInvalid
	}
	providedMAC, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(providedMAC) != sha256.Size {
		return ErrConsentProofInvalid
	}
	signed := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, p.key[:])
	_, _ = mac.Write([]byte(signed))
	if !hmac.Equal(mac.Sum(nil), providedMAC) {
		return ErrConsentProofInvalid
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrConsentProofInvalid
	}
	var payload consentProofPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrConsentProofInvalid
	}
	if payload.ConsentProofFields != expected || expected.AccountID <= 0 || expected.ConsentVersion < 0 ||
		payload.IssuedAt > now.Add(30*time.Second).Unix() ||
		payload.ExpiresAt <= now.Unix() ||
		payload.ExpiresAt-payload.IssuedAt != int64(consentProofTTL/time.Second) {
		return ErrConsentProofInvalid
	}
	return nil
}
