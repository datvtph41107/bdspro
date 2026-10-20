package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPhoneInvalid   = errors.New("invalid E.164 phone")
	ErrPhoneThrottle  = errors.New("phone verification temporarily limited")
	ErrPhoneCode      = errors.New("invalid or expired phone verification code")
	ErrPhoneInUse     = errors.New("phone number unavailable")
	ErrSMSUnavailable = errors.New("SMS delivery unavailable")
)

var e164Phone = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

type SMSSender interface {
	SendOTP(context.Context, string, string) error
}

// ConfigurePhoneVerification binds OTP hashes to a distinct server-side secret
// and a real SMS sender. A missing sender disables new challenges, fail-closed.
func (s *Service) ConfigurePhoneVerification(secret string, sms SMSSender) error {
	if len(secret) < 32 {
		return fmt.Errorf("PHONE_OTP_KEY must contain at least 32 bytes")
	}
	s.OTPKey = []byte(secret)
	s.SMS = sms
	return nil
}

func newPhoneCode() (string, string, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", "", err
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%06d", number.Int64()), hex.EncodeToString(nonceBytes), nil
}

func phoneMAC(key []byte, phoneID int64, nonce, code string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(strconv.FormatInt(phoneID, 10)))
	mac.Write([]byte("|"))
	mac.Write([]byte(nonce))
	mac.Write([]byte("|"))
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

// StartPhoneVerification creates a DB-enforced, five-minute OTP challenge
// before sending SMS. Sender failure does NOT refund cooldown/quota; the
// failure is returned without exposing the code or its destination.
func (s *Service) StartPhoneVerification(ctx context.Context, accountID int64, rawPhone string) error {
	phone := strings.TrimSpace(rawPhone)
	if !e164Phone.MatchString(phone) {
		return ErrPhoneInvalid
	}
	if accountID <= 0 {
		return ErrUnauthorized
	}
	if s.SMS == nil || len(s.OTPKey) < 32 {
		return ErrSMSUnavailable
	}
	code, nonce, err := newPhoneCode()
	if err != nil {
		return err
	}
	var issued bool
	err = dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		if _, err := q.LockSessionAccount(ctx, accountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return err
		}
		// Serialize all accounts targeting this phone, then check global quota.
		if err := q.LockOTPPhoneDestination(ctx, phone); err != nil {
			return err
		}
		phoneID, err := q.UpsertAccountPhone(ctx, dbsqlc.UpsertAccountPhoneParams{
			AccountID: accountID, Phone: phone,
		})
		if err != nil {
			return err
		}
		allowed, err := q.CanIssuePhoneChallenge(ctx, phoneID)
		if err != nil {
			return err
		}
		if !allowed.Valid || !allowed.Bool {
			return ErrPhoneThrottle
		}
		// Both anonymous login and authenticated phone verification send SMS.
		// Apply the same destination cooldown and daily budget to both paths
		// while holding the shared advisory lock.
		shared, err := q.CanIssuePhoneLogin(ctx, dbsqlc.CanIssuePhoneLoginParams{
			Phone: phone, AccountID: accountID,
		})
		if err != nil {
			return err
		}
		if !shared.Valid || !shared.Bool {
			return ErrPhoneThrottle
		}
		sharedRegistration, err := q.CanIssuePhoneRegistration(ctx, phone)
		if err != nil {
			return err
		}
		if !sharedRegistration.Valid || !sharedRegistration.Bool {
			return ErrPhoneThrottle
		}
		// A newly issued code invalidates older unconsumed codes atomically.
		if err := q.InvalidatePriorPhoneChallenges(ctx, phoneID); err != nil {
			return err
		}
		if err := q.CreatePhoneChallenge(ctx, dbsqlc.CreatePhoneChallengeParams{
			AccountPhoneID: phoneID, Nonce: nonce, OtpMac: phoneMAC(s.OTPKey, phoneID, nonce, code),
		}); err != nil {
			return err
		}
		issued = true
		return nil
	})
	if err != nil {
		return err
	}
	if !issued {
		return ErrPhoneThrottle
	}
	if err := s.SMS.SendOTP(ctx, phone, code); err != nil {
		return fmt.Errorf("%w", ErrSMSUnavailable)
	}
	return nil
}

// VerifyPhoneCode serializes attempts on the owning account row and challenge.
// Every submitted guess consumes one of at most five attempts. Successful
// confirmation is single-use and marks the number verified atomically.
func (s *Service) VerifyPhoneCode(ctx context.Context, accountID int64, rawPhone, code string) error {
	phone := strings.TrimSpace(rawPhone)
	if !e164Phone.MatchString(phone) {
		return ErrPhoneInvalid
	}
	if !sixDigits.MatchString(code) {
		return ErrPhoneCode
	}
	if accountID <= 0 {
		return ErrUnauthorized
	}
	if len(s.OTPKey) < 32 {
		return ErrSMSUnavailable
	}
	var verified bool
	err := dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		if _, err := q.LockSessionAccount(ctx, accountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return err
		}
		phoneID, err := q.GetOwnedAccountPhone(ctx, dbsqlc.GetOwnedAccountPhoneParams{
			AccountID: accountID, Phone: phone,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPhoneCode
		}
		if err != nil {
			return err
		}
		challenge, err := q.LockLatestPhoneChallenge(ctx, phoneID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPhoneCode
		}
		if err != nil {
			return err
		}
		expected := phoneMAC(s.OTPKey, phoneID, challenge.Nonce, code)
		// A failed attempt must COMMIT (not roll back), so return the
		// business failure only after transaction completion.
		valid := subtle.ConstantTimeCompare([]byte(expected), []byte(challenge.OtpMac)) == 1
		if err := q.CountPhoneChallengeAttempt(ctx, challenge.ID); err != nil {
			return err
		}
		if !valid {
			return nil
		}
		if err := q.MarkAccountPhoneVerified(ctx, phoneID); err != nil {
			if pgErr := dbsqlc.PostgresError(err); pgErr != nil && pgErr.Code == "23505" &&
				pgErr.ConstraintName == "account_phones_verified_number_uq" {
				return ErrPhoneInUse
			}
			return err
		}
		if err := q.ConsumePhoneChallenge(ctx, challenge.ID); err != nil {
			return err
		}
		verified = true
		return nil
	})
	if err != nil {
		return err
	}
	if !verified {
		return ErrPhoneCode
	}
	return nil
}
