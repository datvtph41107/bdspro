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
	apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/datvtph41107/bdspro/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var phoneE164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var phoneCode = regexp.MustCompile(`^[0-9]{6}$`)

// PhoneOTP is a development-only delivery mode. Codes are saved only in a
// disposable dev PostgreSQL database, never in responses or application logs.
type PhoneOTP struct {
	Store      db.Store
	Challenges db.PhoneOTPRepository
	Maker      token.Maker
	Config     config.Config
}

func (s PhoneOTP) allowed() error {
	if s.Config.Environment != "development" {
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
	return s.Challenges.StartChallenge(ctx, phone, purpose, accountID, s.mac(phone, purpose, accountID, code), code)
}

// StartBind requires an already-authenticated BDSPro account plus its current
// password before adding a new login method.
func (s PhoneOTP) StartBind(ctx context.Context, accountID int64, password, phone string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return apperrors.InvalidPhone
	}
	account, err := s.Store.GetAccountByID(ctx, accountID)
	if err != nil {
		return err
	}
	if util.CheckPassword(password, account.PasswordHash) != nil {
		return apperrors.InvalidCredentials
	}
	existing, err := s.Challenges.AccountIDForPhone(ctx, phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil && existing != accountID {
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
	err := s.Challenges.ConsumeChallenge(ctx, phone, "bind", accountID, s.mac(phone, "bind", accountID, code))
	if errors.Is(err, db.ErrOTPInvalid) {
		return apperrors.OTPInvalid
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.PhoneUnavailable
	}
	return err
}

// Unknown phone and resend cooldown return the same success response.
func (s PhoneOTP) StartLogin(ctx context.Context, phone string) error {
	if err := s.allowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return apperrors.InvalidPhone
	}
	accountID, err := s.Challenges.AccountIDForPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	err = s.start(ctx, phone, "login", accountID)
	if errors.Is(err, db.ErrOTPCooldown) {
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
		return empty, apperrors.OTPInvalid
	}
	accountID, err := s.Challenges.AccountIDForPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, apperrors.OTPInvalid
	}
	if err != nil {
		return empty, err
	}
	if err = s.Challenges.ConsumeChallenge(ctx, phone, "login", accountID, s.mac(phone, "login", accountID, code)); err != nil {
		if errors.Is(err, db.ErrOTPInvalid) {
			return empty, apperrors.OTPInvalid
		}
		return empty, err
	}
	account, err := s.Store.GetAccountByID(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, apperrors.OTPInvalid
	}
	if err != nil {
		return empty, err
	}
	return IssueSession(ctx, s.Store, s.Maker, s.Config, account, userAgent, clientIP)
}
