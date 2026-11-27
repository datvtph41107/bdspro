package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type GoogleStartResult struct {
	URL     string
	Browser string
}

type GoogleResult struct {
	Login *LoginResult
	Proof RegistrationVerification
}

func StartGoogle(ctx context.Context, remember bool) (GoogleStartResult, error) {
	if !setting.Google.Enabled() {
		return GoogleStartResult{}, token.ErrGoogleUnavailable
	}
	browser, browserDigest, err := token.NewRecoveryProof()
	if err != nil {
		return GoogleStartResult{}, err
	}
	state, stateDigest, err := token.NewRecoveryProof()
	if err != nil {
		return GoogleStartResult{}, err
	}
	nonce, _, err := token.NewRecoveryProof()
	if err != nil {
		return GoogleStartResult{}, err
	}
	verifier, _, err := token.NewRecoveryProof()
	if err != nil {
		return GoogleStartResult{}, err
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return GoogleStartResult{}, err
	}
	if _, err := q.DeleteExpiredGoogleFlows(ctx, 100); err != nil {
		return GoogleStartResult{}, err
	}
	if err := q.CreateGoogleFlow(ctx, db.CreateGoogleFlowParams{
		BrowserDigest: browserDigest, StateDigest: stateDigest,
		Nonce: nonce, Verifier: verifier, Remember: remember,
	}); err != nil {
		return GoogleStartResult{}, err
	}
	return GoogleStartResult{URL: token.GoogleAuthorizationURL(setting.Google, state, nonce, verifier), Browser: browser}, nil
}

func GoogleCallback(ctx context.Context, browser []byte, state, code string) error {
	if !setting.Google.Enabled() {
		return token.ErrGoogleUnavailable
	}
	digest, err := token.RecoveryDigest(state)
	if err != nil {
		return ErrRegistrationProof
	}
	pending, err := db.WithTx2(ctx, func(txCtx context.Context) (db.LockPendingGoogleFlowRow, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return db.LockPendingGoogleFlowRow{}, err
		}
		row, err := q.LockPendingGoogleFlow(txCtx, db.LockPendingGoogleFlowParams{BrowserDigest: browser, StateDigest: digest})
		if errors.Is(err, pgx.ErrNoRows) {
			return db.LockPendingGoogleFlowRow{}, ErrRegistrationProof
		}
		if err != nil {
			return db.LockPendingGoogleFlowRow{}, err
		}
		n, err := q.ClaimGoogleFlow(txCtx, browser)
		if err != nil {
			return db.LockPendingGoogleFlowRow{}, err
		}
		if n != 1 {
			return db.LockPendingGoogleFlowRow{}, ErrRegistrationProof
		}
		return row, nil
	})
	if err != nil {
		return err
	}

	identity, err := token.ExchangeGoogle(ctx, setting.Google, code, pending.Verifier, pending.Nonce)
	if err != nil {
		return err
	}
	if pending.RequiresRecentAuth && !token.GoogleAuthTimeRecent(identity.AuthTime, time.Now()) {
		return ErrReauthRequired
	}
	if identity.Email != "" {
		normalized, err := NormalizeEmail(identity.Email)
		if err != nil {
			identity.Email, identity.AuthoritativeEmail = "", false
		} else {
			identity.Email = normalized
		}
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	n, err := q.FinishGoogleFlow(ctx, db.FinishGoogleFlowParams{
		BrowserDigest: browser, Subject: identity.Subject, Email: identity.Email,
		AuthoritativeEmail: identity.AuthoritativeEmail, AuthTime: max(0, identity.AuthTime),
	})
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrRegistrationProof
	}
	return nil
}

func googleOwner(ctx context.Context, q *db.Queries, subject string, lock bool) (int64, error) {
	var owner int64
	for _, issuer := range []string{token.GoogleIssuer, "accounts.google.com"} {
		var id int64
		var err error
		if lock {
			id, err = q.LockGoogleIdentityOwner(ctx, db.LockGoogleIdentityOwnerParams{Issuer: issuer, Subject: subject})
		} else {
			id, err = q.GetOAuth2AccountID(ctx, db.GetOAuth2AccountIDParams{Issuer: issuer, Subject: subject})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if owner != 0 && owner != id {
			return 0, ErrCredentialTaken
		}
		owner = id
	}
	return owner, nil
}

func ConsumeGoogleResult(ctx context.Context, browser []byte, ip, agent string) (GoogleResult, error) {
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return GoogleResult{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (GoogleResult, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return GoogleResult{}, err
		}
		row, err := q.LockGoogleResult(txCtx, browser)
		if errors.Is(err, pgx.ErrNoRows) {
			return GoogleResult{}, ErrRegistrationProof
		}
		if err != nil {
			return GoogleResult{}, err
		}
		owner, err := googleOwner(txCtx, q, row.Subject, false)
		if err != nil {
			return GoogleResult{}, err
		}
		if owner != 0 {
			if _, err = q.LockSessionRevocationAccount(txCtx, owner); errors.Is(err, pgx.ErrNoRows) {
				return GoogleResult{}, util.ErrUnauthenticated
			} else if err != nil {
				return GoogleResult{}, err
			}
			if err := requireActiveAccount(txCtx, q, owner); err != nil {
				return GoogleResult{}, err
			}
			current, err := googleOwner(txCtx, q, row.Subject, true)
			if err != nil {
				return GoogleResult{}, err
			}
			if current != owner {
				return GoogleResult{}, ErrRegistrationProof
			}
			if err := q.DeleteGoogleFlow(txCtx, browser); err != nil {
				return GoogleResult{}, err
			}
			login, err := issueSession(txCtx, q, owner, ip, agent, row.Remember)
			if err != nil {
				return GoogleResult{}, err
			}
			return GoogleResult{Login: &login}, nil
		}

		if row.AuthoritativeEmail && row.Email != "" {
			_, err := q.ReadAccountByEmail(txCtx, row.Email)
			if err == nil {
				if err := q.DeleteGoogleFlow(txCtx, browser); err != nil {
					return GoogleResult{}, err
				}
				return GoogleResult{Proof: RegistrationVerification{Status: "sign_in_required"}}, nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return GoogleResult{}, err
			}
		}
		n, err := q.IssueGoogleRegistrationProof(txCtx, db.IssueGoogleRegistrationProofParams{BrowserDigest: browser, ProofDigest: digest, TtlMinutes: registrationProofTTLMinutes})
		if err != nil {
			return GoogleResult{}, err
		}
		if n != 1 {
			return GoogleResult{}, ErrRegistrationProof
		}
		return GoogleResult{Proof: RegistrationVerification{
			Status: "registration_required", RegistrationToken: raw, ExpiresIn: registrationProofTTLMinutes * 60,
			RequiredActions: []string{"choose_username", "accept_terms"},
		}}, nil
	})
}

func CompleteGoogle(ctx context.Context, raw string, browser []byte, username, terms string, marketing bool, ip, agent string) (LoginResult, error) {
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
		row, err := q.LockGoogleRegistration(txCtx, db.LockGoogleRegistrationParams{BrowserDigest: browser, ProofDigest: digest})
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrRegistrationProof
		}
		if err != nil {
			return LoginResult{}, err
		}
		owner, err := googleOwner(txCtx, q, row.Subject, false)
		if err != nil {
			return LoginResult{}, err
		}
		if owner != 0 {
			return LoginResult{}, ErrCredentialTaken
		}
		if row.AuthoritativeEmail && row.Email != "" {
			if _, err := q.ReadAccountByEmail(txCtx, row.Email); err == nil {
				return LoginResult{}, ErrCredentialTaken
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return LoginResult{}, err
			}
		}
		person, err := q.CreateParty(txCtx)
		if err != nil {
			return LoginResult{}, err
		}
		if err := q.CreatePerson(txCtx, person); err != nil {
			return LoginResult{}, err
		}
		var accountID int64
		if row.AuthoritativeEmail && row.Email != "" {
			a, err := q.CreateAccount(txCtx, db.CreateAccountParams{PersonID: person, Email: row.Email})
			if err != nil {
				return LoginResult{}, err
			}
			accountID = a.ID
			if _, err := q.MarkEmailVerified(txCtx, db.MarkEmailVerifiedParams{ID: accountID, Email: row.Email}); err != nil {
				return LoginResult{}, err
			}
		} else {

			a, err := q.CreateAccountWithoutEmail(txCtx, person)
			if err != nil {
				return LoginResult{}, err
			}
			accountID = a.ID
		}
		linked, err := q.LinkOAuth2Credential(txCtx, db.LinkOAuth2CredentialParams{
			Issuer: token.GoogleIssuer, Subject: row.Subject, AccountID: accountID,
		})
		if errors.Is(err, pgx.ErrNoRows) || err == nil && linked != accountID {
			return LoginResult{}, ErrCredentialTaken
		}
		if err != nil {
			return LoginResult{}, err
		}
		n, err := q.SetRegistrationIdentity(txCtx, db.SetRegistrationIdentityParams{
			AccountID: accountID, Username: name, TermsVersion: terms, MarketingOptIn: marketing,
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
		if err := q.CreateAccountSettings(txCtx, accountID); err != nil {
			return LoginResult{}, err
		}
		n, err = q.UpdatePersonProfile(txCtx, db.UpdatePersonProfileParams{AccountID: accountID, DisplayName: pgtype.Text{String: name, Valid: true}})
		if err != nil {
			return LoginResult{}, err
		}
		if n != 1 {
			return LoginResult{}, ErrRegistrationProof
		}
		id := strconv.FormatInt(accountID, 10)
		if err := q.InsertAuditAction(txCtx, db.InsertAuditActionParams{
			Action: "auth.account.registered", ActorAccountID: accountID,
			ScopeType: "account", ScopeRef: id, TargetType: "account", TargetRef: id,
		}); err != nil {
			return LoginResult{}, err
		}
		if err := q.DeleteGoogleFlow(txCtx, browser); err != nil {
			return LoginResult{}, err
		}
		return issueSession(txCtx, q, accountID, ip, agent, row.Remember)
	})
	return result, identityConflict(err)
}
