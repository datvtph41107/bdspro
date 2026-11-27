package auth

import (
	"context"
	"strconv"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	delivery "github.com/datvtph41107/bdspro/modules/mail"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
)

var ErrLastCredential = util.WithPublic(util.ErrPermissionDenied, "last usable login method", "last_credential", "at least one usable sign-in method must remain")
var ErrEmailPasswordDependency = util.WithPublic(util.ErrPermissionDenied, "password requires verified email", "credential_dependency", "remove the password before removing its email")
var ErrCredentialAbsent = util.WithPublic(util.ErrNotExist, "credential not present", "resource_not_found", "credential not found")
var ErrPasswordAlreadySet = util.WithPublic(util.ErrAlreadyExist, "password already set", "credential_conflict", "password is already set")

func credentialRemovalAllowed(c authmodel.Credentials, kind string) error {
	switch kind {
	case "email":
		if c.PasswordSet {
			return ErrEmailPasswordDependency
		}
		c.EmailVerified = false
	case "phone":
		if !c.PhoneVerified {
			return ErrCredentialAbsent
		}
		c.PhoneVerified = false
	case "google":
		if !c.GoogleLinked {
			return ErrCredentialAbsent
		}
		c.GoogleLinked = false
	case "password":
		if !c.PasswordSet {
			return ErrCredentialAbsent
		}
		c.PasswordSet = false
	default:
		return ErrReauthAction
	}
	if !c.PhoneVerified && !c.GoogleLinked && !(c.EmailVerified && c.PasswordSet) {
		return ErrLastCredential
	}
	return nil
}

func SetPassword(ctx context.Context, actor authmodel.Actor, proof, password, ip, agent string) (LoginResult, error) {
	hash, err := newPasswordHash(password)
	if err != nil {
		return LoginResult{}, err
	}
	var oldEmail string
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return LoginResult{}, err
		}
		u, err := GetCurrentUser(txCtx, actor.AccountID)
		if err != nil {
			return LoginResult{}, err
		}
		if u.Credentials.PasswordSet {
			return LoginResult{}, ErrPasswordAlreadySet
		}
		if u.Email == nil || !u.Credentials.EmailVerified {
			return LoginResult{}, ErrReauthRequired
		}
		if err := ConsumeRecentAuthProof(txCtx, q, actor, proof, "password.set"); err != nil {
			return LoginResult{}, err
		}
		if err := q.CreatePasswordCredential(txCtx, db.CreatePasswordCredentialParams{AccountID: actor.AccountID, PasswordHash: hash}); err != nil {
			return LoginResult{}, err
		}
		oldEmail = *u.Email
		if err := invalidateCredentialAuthority(txCtx, q, actor.AccountID); err != nil {
			return LoginResult{}, err
		}
		if err := credentialAudit(txCtx, q, actor.AccountID, "auth.password.set", "password"); err != nil {
			return LoginResult{}, err
		}
		return issueSession(txCtx, q, actor.AccountID, ip, agent, false)
	})
	if err == nil {
		notifyCredentialChange(ctx, oldEmail)
	}
	return result, identityConflict(err)
}

func RemoveCredential(ctx context.Context, actor authmodel.Actor, kind, proof, ip, agent string) (LoginResult, error) {
	var oldEmail string
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return LoginResult{}, err
		}
		if err := ConsumeRecentAuthProof(txCtx, q, actor, proof, kind+".unlink"); err != nil {
			return LoginResult{}, err
		}
		u, err := GetCurrentUser(txCtx, actor.AccountID)
		if err != nil {
			return LoginResult{}, err
		}
		if kind == "email" && u.Email == nil {
			return LoginResult{}, ErrCredentialAbsent
		}
		if err := credentialRemovalAllowed(u.Credentials, kind); err != nil {
			return LoginResult{}, err
		}
		if u.Email != nil && u.Credentials.EmailVerified {
			oldEmail = *u.Email
		}
		var n int64
		switch kind {
		case "email":
			n, err = q.RemoveAccountEmail(txCtx, actor.AccountID)
		case "phone":
			n, err = q.RemoveAccountPhone(txCtx, actor.AccountID)
		case "google":
			n, err = q.RemoveAccountGoogle(txCtx, actor.AccountID)
		case "password":
			n, err = q.RemoveAccountPassword(txCtx, actor.AccountID)
		}
		if err != nil {
			return LoginResult{}, err
		}
		if n == 0 {
			return LoginResult{}, ErrCredentialAbsent
		}
		if err := invalidateCredentialAuthority(txCtx, q, actor.AccountID); err != nil {
			return LoginResult{}, err
		}
		if err := credentialAudit(txCtx, q, actor.AccountID, "auth.credential.removed", kind); err != nil {
			return LoginResult{}, err
		}
		return issueSession(txCtx, q, actor.AccountID, ip, agent, false)
	})
	if err == nil {
		notifyCredentialChange(ctx, oldEmail)
	}
	return result, err
}

func invalidateCredentialAuthority(ctx context.Context, q *db.Queries, accountID int64) error {
	if err := q.DeleteAccountGoogleFlows(ctx, accountID); err != nil {
		return err
	}
	if err := q.DeleteAccountCredentialLinks(ctx, accountID); err != nil {
		return err
	}
	if err := q.BumpPasswordResetVersion(ctx, accountID); err != nil {
		return err
	}
	if err := q.DeletePasswordResetTokensForAccount(ctx, accountID); err != nil {
		return err
	}
	if err := q.DeleteAccountRecentAuthProofs(ctx, accountID); err != nil {
		return err
	}
	_, err := q.RevokeAllAccountSessions(ctx, accountID)
	return err
}

func credentialAudit(ctx context.Context, q *db.Queries, accountID int64, action, target string) error {
	id := strconv.FormatInt(accountID, 10)
	return q.InsertAuditAction(ctx, db.InsertAuditActionParams{
		Action: action, ActorAccountID: accountID,
		ScopeType: "account", ScopeRef: id, TargetType: target, TargetRef: id,
	})
}

func RevokeAllSessions(ctx context.Context, actor authmodel.Actor, proof string) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return err
		}
		if err := ConsumeRecentAuthProof(txCtx, q, actor, proof, "sessions.revoke_all"); err != nil {
			return err
		}
		if err := q.DeleteAccountGoogleFlows(txCtx, actor.AccountID); err != nil {
			return err
		}
		if err := q.DeleteAccountRecentAuthProofs(txCtx, actor.AccountID); err != nil {
			return err
		}
		if _, err := q.RevokeAllAccountSessions(txCtx, actor.AccountID); err != nil {
			return err
		}
		return credentialAudit(txCtx, q, actor.AccountID, "auth.sessions.revoked_all", "account")
	})
}

func notifyCredentialChange(ctx context.Context, email string) {
	if email == "" || !setting.Mailer.Enabled() {
		return
	}

	if err := delivery.SendCredentialNotice(ctx, email); err != nil {
		log.Ctx(ctx).Warn().Msg("auth.credential.notice_not_delivered")
	}
}
