package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"regexp"

	"github.com/datvtph41107/bdspro/internal/config"
	"github.com/datvtph41107/bdspro/internal/notification"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var phoneE164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var phoneCode = regexp.MustCompile(`^[0-9]{6}$`)

// PhoneOTP owns phone verification regardless of delivery channel. Plaintext
// codes are saved only in disposable development PostgreSQL, never in logs.
type PhoneOTP struct {
	Store      db.Store
	Config     config.Config
	Sender     notification.PhoneCodeSender
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
	if err := s.Store.StartChallenge(ctx, phone, purpose, accountID, s.mac(phone, purpose, accountID, code), devCode); err != nil {
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
		return apperrors.InvalidPhone
	}
	hash, err := s.Store.GetPasswordHash(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.InvalidCredentials
	}
	if err != nil {
		return err
	}
	if util.CheckPassword(password, hash) != nil {
		return apperrors.InvalidCredentials
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
		return apperrors.InvalidPhone
	}
	linkedID, err := s.Store.GetGoogleAccountID(ctx, googleSubject)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && linkedID != accountID) {
		return apperrors.InvalidCredentials
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
	existing, err := s.Store.AccountIDForPhone(ctx, phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	// Bind is for the first verified number only. Do not reuse it for a
	// phone-change workflow, even when the number already belongs to us.
	if err == nil || existing != 0 {
		return apperrors.PhoneUnavailable
	}
	err = s.start(ctx, phone, "bind", accountID)
	if errors.Is(err, db.ErrOTPCooldown) {
		return apperrors.OTPCooldown
	}
	return err
}
func (s PhoneOTP) VerifyBind(ctx context.Context, accountID int64, phone, code string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) || !phoneCode.MatchString(code) {
		return apperrors.OTPInvalid
	}
	err := s.Store.ConsumeChallenge(ctx, phone, "bind", accountID, s.mac(phone, "bind", accountID, code))
	if errors.Is(err, db.ErrOTPInvalid) {
		return apperrors.OTPInvalid
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.PhoneUnavailable
	}
	return err
}

