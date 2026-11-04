package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"math/big"
	"regexp"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/datvtph41107/bdspro/services/notification"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var phoneE164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var phoneCode = regexp.MustCompile(`^[0-9]{6}$`)

// PhoneOTP owns phone verification regardless of delivery channel. Plaintext
// codes are saved only in disposable development PostgreSQL, never in logs.
type PhoneOTP struct {
	Store  db.Store
	Config setting.Config
	Sender notification.PhoneCodeSender
	Maker  token.Maker
}

func (s PhoneOTP) allowed() error {
	if s.Config.Environment != "development" && s.Sender == nil {
		return errors.New("phone OTP delivery unavailable")
	}
	return nil
}
func (s PhoneOTP) generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
func (s PhoneOTP) mac(phone, purpose string, accountID int64, code string) []byte {
	h := hmac.New(sha256.New, []byte(s.Config.TokenSymmetricKey))
	_, _ = fmt.Fprintf(h, "bdspro:phone-otp:v1|%s|%s|%d|%s", purpose, phone, accountID, code)
	return h.Sum(nil)
}
func (s PhoneOTP) start(ctx context.Context, phone, purpose string, accountID int64) error {
	code, err := s.generateCode()
	if err != nil {
		return err
	}
	devCode := ""
	if s.Config.Environment == "development" {
		devCode = code
	}
	if err := auth_model.StartChallenge(ctx, s.Store, phone, purpose, accountID, s.mac(phone, purpose, accountID, code), devCode); err != nil {
		return err
	}
	if s.Sender == nil {
		return nil // Only a disposable development database permits this.
	}
	return s.Sender.SendCode(ctx, phone, code)
}

// StartBind authenticates an account with its existing BDSPro password.
// The challenge can only be created after fresh proof of account control.
func (s PhoneOTP) StartBind(ctx context.Context, accountID int64, password, phone string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	hash, err := s.Store.GetPasswordHash(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if authcrypto.CheckPassword(password, hash) != nil {
		return auth_model.ErrInvalidCredentials
	}
	return s.startBind(ctx, accountID, phone)
}

// StartOAuth2Bind requires a fresh OIDC assertion whose issuer/subject is
// already linked to this very account. Matching email never authorizes binding.
func (s PhoneOTP) StartOAuth2Bind(ctx context.Context, accountID int64, identity auth_model.OIDCClaims, phone string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	key, err := identity.Key()
	if err != nil || !s.Config.OIDC.Enabled() || key.Issuer != s.Config.OIDC.Issuer {
		return auth_model.ErrInvalidCredentials
	}
	linkedID, err := auth_model.GetOAuth2AccountID(ctx, s.Store, key)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && linkedID != accountID) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	return s.startBind(ctx, accountID, phone)
}

// StartGoogleBind is called only after an OAuth callback verified Google's ID
// token, state, nonce and PKCE. The linked subject must own this account.
func (s PhoneOTP) StartGoogleBind(ctx context.Context, accountID int64, googleSubject, phone string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	linkedID, err := s.Store.GetGoogleAccountID(ctx, googleSubject)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && linkedID != accountID) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	return s.startBind(ctx, accountID, phone)
}

func (s PhoneOTP) startBind(ctx context.Context, accountID int64, phone string) error {
	if _, err := s.Store.GetAccountByID(ctx, accountID); err != nil {
		return err
	}
	// A previously verified account cannot start binding another number.
	// The unique account_id constraint remains the concurrency safeguard.
	if _, lookupErr := s.Store.GetPhoneForAccount(ctx, accountID); lookupErr == nil {
		return auth_model.ErrPhoneUnavailable
	} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return lookupErr
	}
	existing, err := auth_model.AccountIDForPhone(ctx, s.Store, phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	// Bind is for the first verified number only. Do not reuse it for a
	// phone-change workflow, even when the number already belongs to us.
	if err == nil || existing != 0 {
		return auth_model.ErrPhoneUnavailable
	}
	err = s.start(ctx, phone, "bind", accountID)
	return err
}
func (s PhoneOTP) VerifyBind(ctx context.Context, accountID int64, phone, code string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) || !phoneCode.MatchString(code) {
		return auth_model.ErrOTPInvalid
	}
	err := auth_model.ConsumeChallenge(ctx, s.Store, phone, "bind", accountID, s.mac(phone, "bind", accountID, code))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return auth_model.ErrPhoneUnavailable
	}
	return err
}

// StartRegister sends an OTP before any account is created. A registered
// phone returns a distinct conflict, while signup remains unique in PostgreSQL.
func (s PhoneOTP) StartRegister(ctx context.Context, phone string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	_, err := auth_model.AccountIDForPhone(ctx, s.Store, phone)
	if err == nil {
		return auth_model.ErrPhoneRegistered
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	code, err := s.generateCode()
	if err != nil {
		return err
	}
	devCode := ""
	if s.Config.Environment == "development" {
		devCode = code
	}
	if err = auth_model.StartPhoneSignup(ctx, s.Store, phone, s.mac(phone, "register", 0, code), devCode); err != nil {
		if errors.Is(err, auth_model.ErrOTPCooldown) {
			return auth_model.ErrOTPCooldown
		}
		return err
	}
	if s.Sender == nil {
		return nil
	}
	return s.Sender.SendCode(ctx, phone, code)
}

// VerifyRegister creates a phone-only Account and issues the same session
// contract used by password and Google sign-in. A failed session issuance
// leaves the verified account accessible through OTP login.
func (s PhoneOTP) VerifyRegister(ctx context.Context, phone, code, userAgent, clientIP string) (LoginResult, error) {
	var empty LoginResult
	if err := s.allowed(); err != nil {
		return empty, err
	}
	if !phoneE164.MatchString(phone) || !phoneCode.MatchString(code) {
		return empty, auth_model.ErrOTPInvalid
	}
	a, err := auth_model.CompletePhoneSignup(ctx, s.Store, phone, s.mac(phone, "register", 0, code))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return empty, auth_model.ErrPhoneRegistered
	}
	if err != nil {
		return empty, err
	}
	return IssueSession(ctx, s.Store, s.Maker, s.Config, a, userAgent, clientIP)
}

// StartLogin does not reveal whether a number exists. Unknown numbers
// receive the same accepted response, but no OTP is delivered.
func (s PhoneOTP) StartLogin(ctx context.Context, phone string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	accountID, err := auth_model.AccountIDForPhone(ctx, s.Store, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	err = s.start(ctx, phone, "login", accountID)
	if errors.Is(err, auth_model.ErrOTPCooldown) {
		// An existing number and an unknown number must not be distinguishable.
		return nil
	}
	return err
}
func (s PhoneOTP) VerifyLogin(ctx context.Context, phone, code, userAgent, clientIP string) (LoginResult, error) {
	var empty LoginResult
	if err := s.allowed(); err != nil {
		return empty, err
	}
	if !phoneE164.MatchString(phone) || !phoneCode.MatchString(code) {
		return empty, auth_model.ErrOTPInvalid
	}
	accountID, err := auth_model.AccountIDForPhone(ctx, s.Store, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, auth_model.ErrOTPInvalid
	}
	if err != nil {
		return empty, err
	}
	if err := auth_model.ConsumeChallenge(ctx, s.Store, phone, "login", accountID, s.mac(phone, "login", accountID, code)); err != nil {
		if errors.Is(err, auth_model.ErrOTPInvalid) {
			return empty, auth_model.ErrOTPInvalid
		}
		return empty, err
	}
	account, err := s.Store.GetAccountByID(ctx, accountID)
	if err != nil {
		return empty, err
	}
	return IssueSession(ctx, s.Store, s.Maker, s.Config, account, userAgent, clientIP)
}

// AuthorizeGoogleLink proves fresh possession of the *already bound* phone.
// The login OTP is single-use, so it cannot be replayed for OAuth linking.
// A linked Google identity is not inferred from a matching email address.
func (s PhoneOTP) AuthorizePhoneProof(ctx context.Context, accountID int64, code string) error {
	if !phoneCode.MatchString(code) {
		return auth_model.ErrOTPInvalid
	}
	phone, err := s.Store.GetPhoneForAccount(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	err = auth_model.ConsumeChallenge(ctx, s.Store, phone, "login", accountID, s.mac(phone, "login", accountID, code))
	if errors.Is(err, auth_model.ErrOTPInvalid) {
		return auth_model.ErrOTPInvalid
	}
	return err
}
