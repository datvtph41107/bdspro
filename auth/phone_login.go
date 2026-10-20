package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
)

// ErrPhoneLoginDelivery is internal to the adapter. Returning it as a distinct
// public HTTP status would reveal whether a phone belongs to an account.
var ErrPhoneLoginDelivery = errors.New("phone login SMS delivery failed")

type PhoneLoginStart struct {
	// A random, non-authenticating handle. Unknown numbers get a fake handle
	// with exactly the same shape as eligible numbers.
	ChallengeToken string
}

// StartPhoneLogin is anonymous. It sends SMS only to an ALREADY VERIFIED phone
// on an open Account. Every well-formed request gets a random challenge handle;
// an unknown/over-quota phone receives an unpersisted handle.
func (s *Service) StartPhoneLogin(ctx context.Context, rawPhone string) (PhoneLoginStart, error) {
	phone := strings.TrimSpace(rawPhone)
	if !e164Phone.MatchString(phone) {
		return PhoneLoginStart{}, ErrPhoneInvalid
	}
	if s.SMS == nil || len(s.OTPKey) < 32 {
		return PhoneLoginStart{}, ErrSMSUnavailable
	}
	handle, challengeHash, err := randomToken()
	if err != nil {
		return PhoneLoginStart{}, err
	}
	result := PhoneLoginStart{ChallengeToken: handle}
	code, nonce, err := newPhoneCode()
	if err != nil {
		return PhoneLoginStart{}, err
	}
	var issued bool
	err = dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		target, err := q.GetVerifiedPhoneLoginTarget(ctx, phone)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		// Lock order is Account first, then advisory phone lock, matching
		// existing enrollment and account-wide session revocation.
		if _, err := q.LockSessionAccount(ctx, target.AccountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		if err := q.LockOTPPhoneDestination(ctx, phone); err != nil {
			return err
		}
		// A concurrent unlink/close may have changed eligibility while
		// acquiring locks. Check again inside the transaction.
		fresh, err := q.GetVerifiedPhoneLoginTarget(ctx, phone)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if fresh.AccountID != target.AccountID || fresh.AccountPhoneID != target.AccountPhoneID {
			return nil
		}
		allowed, err := q.CanIssuePhoneLogin(ctx, dbsqlc.CanIssuePhoneLoginParams{
			Phone: phone, AccountID: target.AccountID,
		})
		if err != nil {
			return err
		}
		if !allowed.Valid || !allowed.Bool {
			return nil
		}
		if err := q.InvalidatePriorPhoneLoginChallenges(ctx, target.AccountPhoneID); err != nil {
			return err
		}
		if err := q.CreatePhoneLoginChallenge(ctx, dbsqlc.CreatePhoneLoginChallengeParams{
			AccountPhoneID: target.AccountPhoneID,
			ChallengeHash: challengeHash,
			Nonce: nonce,
			OtpMac: phoneMAC(s.OTPKey, target.AccountPhoneID, "login|"+nonce, code),
		}); err != nil {
			return err
		}
		issued = true
		return nil
	})
	if err != nil {
		return PhoneLoginStart{}, fmt.Errorf("phone login issue transaction: %w", err)
	}
	if issued {
		if err := s.SMS.SendOTP(ctx, phone, code); err != nil {
			// Keep the public response indistinguishable from an unknown
			// number. Quota is intentionally not refunded on provider errors.
			return result, ErrPhoneLoginDelivery
		}
	}
	return result, nil
}

// VerifyPhoneLogin promotes a valid, one-time, account-eligible challenge to
// a real PostgreSQL session. Invalid attempts COMMIT their attempt counter.
// A challenge is never itself a session and never authorizes business routes.
func (s *Service) VerifyPhoneLogin(ctx context.Context, challenge, code string) (LoginResult, error) {
	data, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil || len(data) != 32 || !sixDigits.MatchString(code) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if len(s.OTPKey) < 32 {
		return LoginResult{}, ErrSMSUnavailable
	}
	hash := digestToken(challenge)
	q := dbsqlc.New(s.Pool)
	accountID, err := q.GetPhoneLoginChallengeAccount(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}
	token, tokenHash, err := randomToken()
	if err != nil {
		return LoginResult{}, err
	}
	var (
		valid  bool
		expiry time.Time
	)
	err = dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(tx *dbsqlc.Queries) error {
		if _, err := tx.LockSessionAccount(ctx, accountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		c, err := tx.LockActivePhoneLoginChallenge(ctx, dbsqlc.LockActivePhoneLoginChallengeParams{
			ChallengeHash: hash, AccountID: accountID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		expected := phoneMAC(s.OTPKey, c.AccountPhoneID, "login|"+c.Nonce, code)
		if err := tx.CountPhoneLoginAttempt(ctx, c.ID); err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(c.OtpMac)) != 1 {
			// Return nil so that a failed attempt does not roll back.
			return nil
		}
		if err := tx.ConsumePhoneLoginChallenge(ctx, c.ID); err != nil {
			return err
		}
		expiry, err = tx.CreateLoginSession(ctx, dbsqlc.CreateLoginSessionParams{
			AccountID: accountID, TokenHash: tokenHash,
		})
		if err != nil {
			return err
		}
		if err := tx.PruneExcessSessions(ctx, accountID); err != nil {
			return err
		}
		valid = true
		return nil
	})
	if err != nil {
		return LoginResult{}, err
	}
	if !valid {
		return LoginResult{}, ErrInvalidCredentials
	}
	return LoginResult{AccountID: accountID, Token: token, ExpiresAt: expiry}, nil
}
