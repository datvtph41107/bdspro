package auth

import (
	"context"
	"errors"
	"time"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/jackc/pgx/v5"
)

func googleReauthAction(action string) bool {
	switch action {
	case "password.set", "email.link", "phone.link", "email.unlink", "phone.unlink", "google.unlink", "password.unlink", "sessions.revoke_all":
		return true
	default:
		return false
	}
}

func StartAuthenticatedGoogle(ctx context.Context, actor authmodel.Actor, purpose, action, proof string) (GoogleStartResult, error) {
	if !setting.Google.Enabled() {
		return GoogleStartResult{}, token.ErrGoogleUnavailable
	}
	if purpose == "link" && action != "google.link" || purpose == "reauth" && !googleReauthAction(action) ||
		purpose != "link" && purpose != "reauth" {
		return GoogleStartResult{}, ErrReauthAction
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (GoogleStartResult, error) {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return GoogleStartResult{}, err
		}
		u, err := GetCurrentUser(txCtx, actor.AccountID)
		if err != nil {
			return GoogleStartResult{}, err
		}
		if purpose == "link" {
			if err := ConsumeRecentAuthProof(txCtx, q, actor, proof, action); err != nil {
				return GoogleStartResult{}, err
			}
			if u.Credentials.GoogleLinked {
				return GoogleStartResult{}, ErrCredentialTaken
			}
		} else if !u.Credentials.GoogleLinked {
			return GoogleStartResult{}, ErrReauthRequired
		}
		result, err := StartGoogle(txCtx, false)
		if err != nil {
			return GoogleStartResult{}, err
		}
		digest, err := token.RecoveryDigest(result.Browser)
		if err != nil {
			return GoogleStartResult{}, err
		}
		if err := q.CreateGoogleAuthenticatedIntent(txCtx, db.CreateGoogleAuthenticatedIntentParams{
			BrowserDigest: digest, AccountID: actor.AccountID, SessionID: actor.SessionID,
			Purpose: purpose, Action: action,
		}); err != nil {
			return GoogleStartResult{}, err
		}
		if purpose == "reauth" {
			result.URL, err = token.GoogleStepUpURL(result.URL)
			if err != nil {
				return GoogleStartResult{}, err
			}
		}
		return result, nil
	})
}

func CompleteGoogleLink(ctx context.Context, actor authmodel.Actor, browser []byte, ip, agent string) (LoginResult, error) {
	var oldEmail string
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return LoginResult{}, err
		}
		row, err := q.LockGoogleAuthenticatedResult(txCtx, db.LockGoogleAuthenticatedResultParams{
			BrowserDigest: browser, AccountID: actor.AccountID, SessionID: actor.SessionID, Purpose: "link",
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrRegistrationProof
		}
		if err != nil {
			return LoginResult{}, err
		}
		u, err := GetCurrentUser(txCtx, actor.AccountID)
		if err != nil {
			return LoginResult{}, err
		}
		if u.Credentials.GoogleLinked {
			return LoginResult{}, ErrCredentialTaken
		}

		owner, err := googleOwner(txCtx, q, row.Subject, false)
		if err != nil {
			return LoginResult{}, err
		}
		if owner != 0 {
			return LoginResult{}, ErrCredentialTaken
		}
		linked, err := q.LinkOAuth2Credential(txCtx, db.LinkOAuth2CredentialParams{
			Issuer: token.GoogleIssuer, Subject: row.Subject, AccountID: actor.AccountID,
		})
		if errors.Is(err, pgx.ErrNoRows) || err == nil && linked != actor.AccountID {
			return LoginResult{}, ErrCredentialTaken
		}
		if err != nil {
			return LoginResult{}, err
		}
		if u.Email != nil && u.Credentials.EmailVerified {
			oldEmail = *u.Email
		}
		if err := invalidateCredentialAuthority(txCtx, q, actor.AccountID); err != nil {
			return LoginResult{}, err
		}
		if err := credentialAudit(txCtx, q, actor.AccountID, "auth.credential.linked", "google"); err != nil {
			return LoginResult{}, err
		}
		return issueSession(txCtx, q, actor.AccountID, ip, agent, false)
	})
	if err == nil {
		notifyCredentialChange(ctx, oldEmail)
	}
	return result, identityConflict(err)
}

func CompleteGoogleReauth(ctx context.Context, actor authmodel.Actor, browser []byte) (RecentAuthResult, error) {
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return RecentAuthResult{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (RecentAuthResult, error) {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return RecentAuthResult{}, err
		}
		row, err := q.LockGoogleAuthenticatedResult(txCtx, db.LockGoogleAuthenticatedResultParams{
			BrowserDigest: browser, AccountID: actor.AccountID, SessionID: actor.SessionID, Purpose: "reauth",
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return RecentAuthResult{}, ErrReauthRequired
		}
		if err != nil {
			return RecentAuthResult{}, err
		}
		if !googleReauthAction(row.Action) || !token.GoogleAuthTimeRecent(row.AuthTime, time.Now()) {
			return RecentAuthResult{}, ErrReauthRequired
		}
		owner, err := googleOwner(txCtx, q, row.Subject, true)
		if err != nil {
			return RecentAuthResult{}, err
		}
		if owner != actor.AccountID {
			return RecentAuthResult{}, ErrReauthRequired
		}
		n, err := q.CreateGoogleRecentAuthProof(txCtx, db.CreateGoogleRecentAuthProofParams{
			Digest: digest, AccountID: actor.AccountID, SessionID: actor.SessionID,
			Action: row.Action, GoogleSubject: row.Subject, AuthTime: row.AuthTime,
		})
		if err != nil {
			return RecentAuthResult{}, err
		}
		if n != 1 {
			return RecentAuthResult{}, ErrReauthRequired
		}
		if err := q.DeleteGoogleFlow(txCtx, browser); err != nil {
			return RecentAuthResult{}, err
		}
		expires := min(600, int(time.Until(time.Unix(row.AuthTime, 0).Add(10*time.Minute)).Seconds()))
		return RecentAuthResult{Proof: raw, Action: row.Action, ExpiresIn: expires}, nil
	})
}
