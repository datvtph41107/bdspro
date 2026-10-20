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

// ErrPhoneRegistrationDelivery must not be translated into a different
// client-visible response from an unknown or already-registered phone.
var ErrPhoneRegistrationDelivery = errors.New("phone registration delivery failed")

// StartPhoneRegistration issues a challenge without creating Party, Person,
// Account, or Session. Existing verified phones return fake challenge handles,
// preserving the same public status shape without sending an SMS.
func (s *Service) StartPhoneRegistration(ctx context.Context, rawPhone string) (PhoneLoginStart, error) {
	phone := strings.TrimSpace(rawPhone)
	if !e164Phone.MatchString(phone) {
		return PhoneLoginStart{}, ErrPhoneInvalid
	}
	if s.SMS == nil || len(s.OTPKey) < 32 {
		return PhoneLoginStart{}, ErrSMSUnavailable
	}
	handle, hash, err := randomToken()
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
		// Register has no Account to lock. Serialize with both authenticated
		// enrollment and anonymous login on the same phone advisory lock.
		if err := q.LockOTPPhoneDestination(ctx, phone); err != nil {
			return err
		}
		owners, err := q.PhoneAlreadyRegistered(ctx, phone)
		if err != nil {
			return err
		}
		if owners > 0 {
			return nil
		}
		allowed, err := q.CanIssuePhoneRegistration(ctx, phone)
		if err != nil {
			return err
		}
		if !allowed.Valid || !allowed.Bool {
			return nil
		}
		if err := q.InvalidatePhoneRegistrationChallenges(ctx, phone); err != nil {
			return err
		}
		if err := q.CreatePhoneRegistrationChallenge(ctx, dbsqlc.CreatePhoneRegistrationChallengeParams{
			Phone: phone, ChallengeHash: hash, Nonce: nonce,
			OtpMac: phoneMAC(s.OTPKey, 0, "register|"+phone+"|"+nonce, code),
		}); err != nil {
			return err
		}
		issued = true
		return nil
	})
	if err != nil {
		return PhoneLoginStart{}, fmt.Errorf("phone registration issue: %w", err)
	}
	if issued {
		if err := s.SMS.SendOTP(ctx, phone, code); err != nil {
			// No quota refund on provider failure: avoid SMS send abuse.
			return result, ErrPhoneRegistrationDelivery
		}
	}
	return result, nil
}

// VerifyPhoneRegistration is the ONLY conversion from anonymous proof to a
// new Account. A consumed or wrong challenge never creates a Party or session.
// The consumed proof, entire identity graph and session commit atomically.
func (s *Service) VerifyPhoneRegistration(ctx context.Context, handle, code string) (LoginResult, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(handle)
	if err != nil || len(decoded) != 32 || !sixDigits.MatchString(code) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if len(s.OTPKey) < 32 {
		return LoginResult{}, ErrSMSUnavailable
	}
	hash := digestToken(handle)
	phone, err := dbsqlc.New(s.Pool).GetPhoneRegistrationCandidate(ctx, hash)
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
		confirmed bool
		taken     bool
		accountID int64
		expiresAt time.Time
	)
	err = dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		if err := q.LockOTPPhoneDestination(ctx, phone); err != nil {
			return err
		}
		challenge, err := q.LockActivePhoneRegistrationChallenge(ctx, dbsqlc.LockActivePhoneRegistrationChallengeParams{
			ChallengeHash: hash, Phone: phone,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		expected := phoneMAC(s.OTPKey, 0, "register|"+phone+"|"+challenge.Nonce, code)
		// A failed attempt must be committed rather than rolled back.
		if err := q.CountPhoneRegistrationAttempt(ctx, challenge.ID); err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(challenge.OtpMac)) != 1 {
			return nil
		}
		if err := q.ConsumePhoneRegistrationChallenge(ctx, challenge.ID); err != nil {
			return err
		}
		owners, err := q.PhoneAlreadyRegistered(ctx, phone)
		if err != nil {
			return err
		}
		if owners > 0 {
			taken = true
			return nil // Persist the consumed challenge.
		}
		personID, err := q.CreatePersonParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreatePersonSubtype(ctx, personID); err != nil {
			return err
		}
		accountID, err = q.CreateAccount(ctx, personID)
		if err != nil {
			return err
		}
		if err := q.CreatePrimaryVerifiedAccountPhone(ctx, dbsqlc.CreatePrimaryVerifiedAccountPhoneParams{
			AccountID: accountID, Phone: phone,
		}); err != nil {
			// A concurrent authenticated phone-verification route could win
			// its unique verified-phone index without holding this advisory
			// lock; in that case rollback ALL identity writes.
			if pgErr := dbsqlc.PostgresError(err); pgErr != nil &&
				pgErr.Code == "23505" && pgErr.ConstraintName == "account_phones_verified_number_uq" {
				return ErrPhoneInUse
			}
			return err
		}
		expiresAt, err = q.CreateLoginSession(ctx, dbsqlc.CreateLoginSessionParams{
			AccountID: accountID, TokenHash: tokenHash,
		})
		if err != nil {
			return err
		}
		confirmed = true
		return q.PruneExcessSessions(ctx, accountID)
	})
	if err != nil {
		return LoginResult{}, err
	}
	if taken {
		return LoginResult{}, ErrPhoneInUse
	}
	if !confirmed {
		return LoginResult{}, ErrInvalidCredentials
	}
	return LoginResult{AccountID: accountID, Token: token, ExpiresAt: expiresAt}, nil
}
