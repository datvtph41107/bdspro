package auth

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
	FailurePassword        = "password"
	FailurePhoneOTP        = "phone_otp"
	FailureOIDC            = "oidc"
	FailureUnknownAccount  = "unknown_account"
	FailurePasswordUnset   = "password_unset"
	FailureBadPassword     = "invalid_password"
	FailureBadOTP          = "invalid_otp"
	FailureBadIdentity     = "invalid_identity"
	FailureEmailUnverified = "email_unverified"
	FailureLinkRequired    = "link_required"
)

var errFailureAuditKey = errors.New("authentication event key unavailable")

// RejectedProof records a rejected authentication proof, never a successful
// login. Credential is transient and must not be stored, returned or logged.
type RejectedProof struct {
	AccountID  int64
	Method     string
	Reason     string
	Credential string
	ClientIP   string
}

// RecordAuthenticationFailure saves only a keyed, domain-separated credential
// digest. It has no dependency on Gin, HTTP responses, or raw credential logs.
// Its caller owns the policy for handling audit-write failures.
func RecordAuthenticationFailure(ctx context.Context, store interface {
	InsertAuthenticationFailure(context.Context, db.InsertAuthenticationFailureParams) error
}, key string, proof RejectedProof) error {
	if len(key) < 32 {
		return errFailureAuditKey
	}
	credential := strings.ToLower(strings.TrimSpace(proof.Credential))
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte("auth-rejected-proof:v1\x00"))
	_, _ = mac.Write([]byte(proof.Method))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(credential))

	var accountID pgtype.Int8
	if proof.AccountID > 0 {
		accountID = pgtype.Int8{Int64: proof.AccountID, Valid: true}
	}
	ipAddress := ""
	if ip := net.ParseIP(strings.TrimSpace(proof.ClientIP)); ip != nil {
		ipAddress = ip.String()
	}
	return store.InsertAuthenticationFailure(ctx, db.InsertAuthenticationFailureParams{
		AccountID: accountID,
		Method: proof.Method,
		Reason: proof.Reason,
		CredentialMac: mac.Sum(nil),
		IpAddress: ipAddress,
	})
}
