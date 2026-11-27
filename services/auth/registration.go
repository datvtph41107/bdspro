package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"strconv"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/services/mailer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Verification lifetimes are authentication policy, not transport defaults.
const (
	verificationCodeTTLMinutes         = 5
	VerificationResendCooldownSeconds = 60
	registrationProofTTLMinutes        = 15

	// The current schema independently enforces a storage ceiling of five.
	verificationMaxFailedAttempts     = 5
	registrationResendHorizonMinutes  = 30
	verificationCleanupRetentionMinutes = 30
	verificationCleanupBatchSize        = 100
)

var (
	ErrRegistrationProof = util.WithPublic(util.ErrInvalidArgument, "registration proof invalid", "invalid_proof", "invalid or expired registration proof")
	ErrRegistrationCode  = util.WithPublic(util.ErrInvalidArgument, "verification code invalid", "invalid_verification_code", "invalid or expired verification code")
	ErrRegistrationWait  = util.WithPublic(util.ErrRateLimited, "registration resend denied", "rate_limited", "verification resend unavailable")
)

type RegistrationChallenge struct {
	DeliveryChannel string    `json:"delivery_channel,omitempty"`
	Status          string    `json:"status"`
	ChallengeID     uuid.UUID `json:"challenge_id"`
	ExpiresIn       int       `json:"expires_in"`
	ResendAfter     int       `json:"resend_after"`
}

type RegistrationVerification struct {
	Status            string   `json:"status"`
	RegistrationToken string   `json:"registration_token,omitempty"`
	ExpiresIn         int      `json:"expires_in,omitempty"`
	RequiredActions   []string `json:"required_actions,omitempty"`
}

func challengeResult(id uuid.UUID) RegistrationChallenge {
	return RegistrationChallenge{Status: "verification_required", ChallengeID: id, ExpiresIn: verificationCodeTTLMinutes * 60, ResendAfter: VerificationResendCooldownSeconds}
}

func StartRegistration(ctx context.Context, email string, browser []byte) (RegistrationChallenge, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return RegistrationChallenge{}, err
	}
	code, err := token.NewRegistrationCode()
	if err != nil {
		return RegistrationChallenge{}, err
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return RegistrationChallenge{}, err
	}
	if _, err := q.DeleteExpiredEmailRegistrations(ctx, db.DeleteExpiredEmailRegistrationsParams{RetentionMinutes: verificationCleanupRetentionMinutes, Take: verificationCleanupBatchSize}); err != nil {
		return RegistrationChallenge{}, err
	}
	err = q.CreateEmailRegistration(ctx, db.CreateEmailRegistrationParams{
		TtlMinutes: verificationCodeTTLMinutes, CooldownSeconds: VerificationResendCooldownSeconds,
		ID: id, Email: email, BrowserDigest: browser,
		CodeMac: token.RegistrationCodeMAC(setting.Security.TokenSymmetricKey, id, email, browser, code),
	})
	if err != nil {
		return RegistrationChallenge{}, err
	}
	if err := mailer.SendRegistrationCode(ctx, email, code, verificationCodeTTLMinutes); err != nil {
		return RegistrationChallenge{}, err
	}
	return challengeResult(id), nil
}

func RegistrationDestination(ctx context.Context, id uuid.UUID, browser []byte) (string, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return "", err
	}
	email, err := q.ReadEmailRegistrationDestination(ctx, db.ReadEmailRegistrationDestinationParams{ID: id, BrowserDigest: browser})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRegistrationProof
	}
	return email, err
}

func ResendRegistration(ctx context.Context, id uuid.UUID, browser []byte) (RegistrationChallenge, error) {
	code, err := token.NewRegistrationCode()
	if err != nil {
		return RegistrationChallenge{}, err
	}
	email, err := db.WithTx2(ctx, func(txCtx context.Context) (string, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return "", err
		}
		row, err := q.LockEmailRegistration(txCtx, db.LockEmailRegistrationParams{ID: id, BrowserDigest: browser})
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRegistrationProof
		}
		if err != nil {
			return "", err
		}
		n, err := q.RotateEmailRegistration(txCtx, db.RotateEmailRegistrationParams{
			MaxFailedAttempts: verificationMaxFailedAttempts, ResendHorizonMinutes: registrationResendHorizonMinutes,
			TtlMinutes: verificationCodeTTLMinutes, CooldownSeconds: VerificationResendCooldownSeconds,
			ID: id, CodeMac: token.RegistrationCodeMAC(setting.Security.TokenSymmetricKey, id, row.Email, browser, code),
		})
		if err != nil {
			return "", err
		}
		if n != 1 {
			return "", ErrRegistrationWait
		}
		return row.Email, nil
	})
	if err != nil {
		return RegistrationChallenge{}, err
	}
	if err := mailer.SendRegistrationCode(ctx, email, code, verificationCodeTTLMinutes); err != nil {
		return RegistrationChallenge{}, err
	}
	return challengeResult(id), nil
}

func VerifyRegistration(ctx context.Context, id uuid.UUID, browser []byte, code string) (RegistrationVerification, error) {
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return RegistrationVerification{}, err
	}
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (RegistrationVerification, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return RegistrationVerification{}, err
		}
		row, err := q.LockEmailRegistration(txCtx, db.LockEmailRegistrationParams{ID: id, BrowserDigest: browser})
		if errors.Is(err, pgx.ErrNoRows) {
			return RegistrationVerification{}, nil
		}
		if err != nil {
			return RegistrationVerification{}, err
		}
		if row.FailedAttempts >= verificationMaxFailedAttempts || !row.ExpiresAt.After(time.Now()) {
			return RegistrationVerification{}, nil
		}
		expected := token.RegistrationCodeMAC(setting.Security.TokenSymmetricKey, id, row.Email, browser, code)
		if !token.ValidRegistrationCode(code) || !hmac.Equal(expected, row.CodeMac) {

			return RegistrationVerification{}, q.FailEmailRegistration(txCtx, db.FailEmailRegistrationParams{ID: id, MaxFailedAttempts: verificationMaxFailedAttempts})
		}
		_, err = q.ReadAccountByEmail(txCtx, row.Email)
		if err == nil {
			if err := q.DeleteEmailRegistration(txCtx, id); err != nil {
				return RegistrationVerification{}, err
			}
			return RegistrationVerification{Status: "sign_in_required"}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return RegistrationVerification{}, err
		}
		n, err := q.VerifyEmailRegistration(txCtx, db.VerifyEmailRegistrationParams{ID: id, ProofDigest: digest, TtlMinutes: registrationProofTTLMinutes, MaxFailedAttempts: verificationMaxFailedAttempts})
		if err != nil {
			return RegistrationVerification{}, err
		}
		if n != 1 {
			return RegistrationVerification{}, nil
		}
		return RegistrationVerification{
			Status: "registration_required", RegistrationToken: raw, ExpiresIn: registrationProofTTLMinutes * 60,
			RequiredActions: []string{"choose_username", "set_password", "accept_terms"},
		}, nil
	})
	if err == nil && result.Status == "" {
		err = ErrRegistrationCode
	}
	return result, err
}

func CompleteRegistration(ctx context.Context, raw string, browser []byte, username, password, terms string, marketing, remember bool, ip, agent string) (LoginResult, error) {
	name, err := NormalizeUsername(username)
	if err != nil {
		return LoginResult{}, err
	}
	if terms != CurrentTermsVersion {
		return LoginResult{}, ErrTermsInvalid
	}
	digest, err := token.RecoveryDigest(raw)
	if err != nil {
		return LoginResult{}, ErrRegistrationProof
	}
	hash, err := newPasswordHash(password)
	if err != nil {
		return LoginResult{}, err
	}
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return LoginResult{}, err
		}
		proof, err := q.LockEmailRegistrationProof(txCtx, db.LockEmailRegistrationProofParams{ProofDigest: digest, BrowserDigest: browser})
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrRegistrationProof
		}
		if err != nil {
			return LoginResult{}, err
		}

		_, err = q.ReadAccountByEmail(txCtx, proof.Email)
		if err == nil {
			return LoginResult{}, ErrCredentialTaken
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, err
		}
		person, err := q.CreateParty(txCtx)
		if err != nil {
			return LoginResult{}, err
		}
		if err := q.CreatePerson(txCtx, person); err != nil {
			return LoginResult{}, err
		}
		account, err := q.CreateAccount(txCtx, db.CreateAccountParams{PersonID: person, Email: proof.Email})
		if err != nil {
			return LoginResult{}, err
		}
		if _, err := q.MarkEmailVerified(txCtx, db.MarkEmailVerifiedParams{ID: account.ID, Email: proof.Email}); err != nil {
			return LoginResult{}, err
		}
		n, err := q.SetRegistrationIdentity(txCtx, db.SetRegistrationIdentityParams{
			AccountID: account.ID, Username: name, TermsVersion: terms, MarketingOptIn: marketing,
		})
		if err != nil {
			return LoginResult{}, err
		}
		if n != 1 {
			return LoginResult{}, ErrRegistrationProof
		}
		if err := q.CreatePasswordCredential(txCtx, db.CreatePasswordCredentialParams{AccountID: account.ID, PasswordHash: hash}); err != nil {
			return LoginResult{}, err
		}
		if err := q.CreatePersonProfile(txCtx, person); err != nil {
			return LoginResult{}, err
		}
		if err := q.CreateAccountSettings(txCtx, account.ID); err != nil {
			return LoginResult{}, err
		}
		n, err = q.UpdatePersonProfile(txCtx, db.UpdatePersonProfileParams{AccountID: account.ID, DisplayName: pgtype.Text{String: name, Valid: true}})
		if err != nil {
			return LoginResult{}, err
		}
		if n != 1 {
			return LoginResult{}, ErrRegistrationProof
		}
		id := strconv.FormatInt(account.ID, 10)
		if err := q.InsertAuditAction(txCtx, db.InsertAuditActionParams{
			Action: "auth.account.registered", ActorAccountID: account.ID,
			ScopeType: "account", ScopeRef: id, TargetType: "account", TargetRef: id,
		}); err != nil {
			return LoginResult{}, err
		}
		if err := q.DeleteEmailRegistration(txCtx, proof.ID); err != nil {
			return LoginResult{}, err
		}
		return issueSession(txCtx, q, account.ID, ip, agent, remember)
	})
	return result, identityConflict(err)
}
