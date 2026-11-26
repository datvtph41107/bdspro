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
	"github.com/datvtph41107/bdspro/modules/sms"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrPhoneInvalid = util.WithPublic(util.ErrInvalidArgument, "invalid phone syntax", "validation_failed", "invalid phone number")

type PhoneVerification struct {
	Login *LoginResult
	Proof RegistrationVerification
}

func StartPhone(ctx context.Context, phone string, browser []byte) (RegistrationChallenge, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return RegistrationChallenge{}, err
	}
	code, err := token.NewPhoneCode()
	if err != nil {
		return RegistrationChallenge{}, err
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return RegistrationChallenge{}, err
	}
	if _, err := q.DeleteExpiredPhoneContinuations(ctx, 100); err != nil {
		return RegistrationChallenge{}, err
	}
	if err := q.CreatePhoneContinuation(ctx, db.CreatePhoneContinuationParams{
		ID: id, PhoneE164: phone, BrowserDigest: browser,
		CodeMac: token.PhoneCodeMAC(setting.Security.TokenSymmetricKey, id, phone, browser, code),
	}); err != nil {
		return RegistrationChallenge{}, err
	}
	if err := sms.SendCode(ctx, phone, code); err != nil {
		return RegistrationChallenge{}, err
	}
	return challengeResult(id), nil
}

func PhoneDestination(ctx context.Context, id uuid.UUID, browser []byte) (string, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return "", err
	}
	phone, err := q.ReadPhoneContinuationDestination(ctx, db.ReadPhoneContinuationDestinationParams{ID: id, BrowserDigest: browser})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRegistrationProof
	}
	return phone, err
}

func ResendPhone(ctx context.Context, id uuid.UUID, browser []byte) (RegistrationChallenge, error) {
	code, err := token.NewPhoneCode()
	if err != nil {
		return RegistrationChallenge{}, err
	}
	phone, err := db.WithTx2(ctx, func(txCtx context.Context) (string, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return "", err
		}
		row, err := q.LockPhoneContinuation(txCtx, db.LockPhoneContinuationParams{ID: id, BrowserDigest: browser})
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRegistrationProof
		}
		if err != nil {
			return "", err
		}
		n, err := q.RotatePhoneContinuation(txCtx, db.RotatePhoneContinuationParams{
			ID: id, CodeMac: token.PhoneCodeMAC(setting.Security.TokenSymmetricKey, id, row.PhoneE164, browser, code),
		})
		if err != nil {
			return "", err
		}
		if n != 1 {
			return "", ErrRegistrationWait
		}
		return row.PhoneE164, nil
	})
	if err != nil {
		return RegistrationChallenge{}, err
	}
	if err := sms.SendCode(ctx, phone, code); err != nil {
		return RegistrationChallenge{}, err
	}
	return challengeResult(id), nil
}

func VerifyPhone(ctx context.Context, id uuid.UUID, browser []byte, code string, remember bool, ip, agent string) (PhoneVerification, error) {
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return PhoneVerification{}, err
	}
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (PhoneVerification, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return PhoneVerification{}, err
		}
		row, err := q.LockPhoneContinuation(txCtx, db.LockPhoneContinuationParams{ID: id, BrowserDigest: browser})
		if errors.Is(err, pgx.ErrNoRows) {
			return PhoneVerification{}, nil
		}
		if err != nil {
			return PhoneVerification{}, err
		}
		if row.FailedAttempts >= 5 || !row.ExpiresAt.After(time.Now()) {
			return PhoneVerification{}, nil
		}
		expected := token.PhoneCodeMAC(setting.Security.TokenSymmetricKey, id, row.PhoneE164, browser, code)
		if !token.ValidPhoneCode(code) || !hmac.Equal(expected, row.CodeMac) {
			return PhoneVerification{}, q.FailPhoneContinuation(txCtx, id)
		}
		owner, err := q.GetPhoneAccountIDQuery(txCtx, row.PhoneE164)
		if errors.Is(err, pgx.ErrNoRows) {
			n, err := q.VerifyPhoneContinuation(txCtx, db.VerifyPhoneContinuationParams{ID: id, ProofDigest: digest})
			if err != nil || n != 1 {
				return PhoneVerification{}, err
			}
			return PhoneVerification{Proof: RegistrationVerification{
				Status: "registration_required", RegistrationToken: raw, ExpiresIn: 900,
				RequiredActions: []string{"choose_username", "accept_terms"},
			}}, nil
		}
		if err != nil {
			return PhoneVerification{}, err
		}
		if _, err = q.LockSessionRevocationAccount(txCtx, owner); errors.Is(err, pgx.ErrNoRows) {
			return PhoneVerification{}, util.ErrUnauthenticated
		} else if err != nil {
			return PhoneVerification{}, err
		}
		if err := requireActiveAccount(txCtx, q, owner); err != nil {
			return PhoneVerification{}, err
		}
		current, err := q.LockPhoneCredentialQuery(txCtx, row.PhoneE164)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && current != owner {
			return PhoneVerification{}, ErrRegistrationCode
		}
		if err != nil {
			return PhoneVerification{}, err
		}

		if !row.ExpiresAt.After(time.Now()) {
			return PhoneVerification{}, ErrRegistrationCode
		}
		if err := q.DeletePhoneContinuation(txCtx, id); err != nil {
			return PhoneVerification{}, err
		}
		login, err := issueSession(txCtx, q, owner, ip, agent, remember)
		if err != nil {
			return PhoneVerification{}, err
		}
		return PhoneVerification{Login: &login}, nil
	})
	if err == nil && result.Login == nil && result.Proof.Status == "" {
		err = ErrRegistrationCode
	}
	return result, err
}

func CompletePhone(ctx context.Context, raw string, browser []byte, username, terms string, marketing, remember bool, ip, agent string) (LoginResult, error) {
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
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return LoginResult{}, err
		}
		proof, err := q.LockPhoneContinuationProof(txCtx, db.LockPhoneContinuationProofParams{ProofDigest: digest, BrowserDigest: browser})
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrRegistrationProof
		}
		if err != nil {
			return LoginResult{}, err
		}
		if _, err = q.GetPhoneAccountIDQuery(txCtx, proof.PhoneE164); err == nil {
			return LoginResult{}, ErrCredentialTaken
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, err
		}
		person, err := q.CreateParty(txCtx)
		if err != nil {
			return LoginResult{}, err
		}
		if err := q.CreatePerson(txCtx, person); err != nil {
			return LoginResult{}, err
		}
		account, err := q.CreatePhoneOnlyAccount(txCtx, person)
		if err != nil {
			return LoginResult{}, err
		}
		if err := q.BindPhoneCredentialQuery(txCtx, db.BindPhoneCredentialQueryParams{AccountID: account.ID, PhoneE164: proof.PhoneE164}); err != nil {
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
		if err := q.DeletePhoneContinuation(txCtx, proof.ID); err != nil {
			return LoginResult{}, err
		}
		return issueSession(txCtx, q, account.ID, ip, agent, remember)
	})
	return result, identityConflict(err)
}
