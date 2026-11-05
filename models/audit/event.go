package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"net"
	"strings"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	MethodPassword = "password"
	MethodPhoneOTP = "phone_otp"
	MethodOIDC     = "oidc"

	ReasonUnknownAccount  = "unknown_account"
	ReasonPasswordUnset   = "password_unset"
	ReasonInvalidPassword = "invalid_password"
	ReasonInvalidOTP      = "invalid_otp"
	ReasonInvalidIdentity = "invalid_identity"
	ReasonEmailUnverified = "email_unverified"
	ReasonLinkRequired    = "link_required"
)

var ErrMissingKey = errors.New("audit event key must contain at least 32 bytes")

// LoginRejected captures a rejected identity proof, not an HTTP status or a
// successfully issued login session. Identifier is transient and never stored.
type LoginRejected struct {
	AccountID  int64
	Method     string
	Reason     string
	Identifier string
	IP         string
}

// RecordLoginRejected owns the audit persistence invariant. Callers decide
// whether an audit write failure should fail their business operation.
func RecordLoginRejected(ctx context.Context, store interface {
	InsertAuditEvent(context.Context, db.InsertAuditEventParams) error
}, key string, event LoginRejected) error {
	if len(key) < 32 {
		return ErrMissingKey
	}
	identifier := event.Identifier
	switch event.Method {
	case MethodPassword:
		identifier = strings.ToLower(strings.TrimSpace(identifier))
	case MethodPhoneOTP:
		identifier = strings.TrimSpace(identifier)
	case MethodOIDC:
		// OIDC issuer and subject remain case-sensitive.
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte("audit:login-rejected:v1\x00"))
	_, _ = mac.Write([]byte(event.Method))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(identifier))

	var accountID pgtype.Int8
	if event.AccountID > 0 {
		accountID = pgtype.Int8{Int64: event.AccountID, Valid: true}
	}
	ip := ""
	if parsed := net.ParseIP(strings.TrimSpace(event.IP)); parsed != nil {
		ip = parsed.String()
	}
	return store.InsertAuditEvent(ctx, db.InsertAuditEventParams{
		AccountID:  accountID,
		Method:     pgtype.Text{String: event.Method, Valid: true},
		Reason:     pgtype.Text{String: event.Reason, Valid: true},
		SubjectMac: mac.Sum(nil),
		IpAddress:  ip,
	})
}
