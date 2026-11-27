package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const CurrentTermsVersion = "v1"

var (
	ErrEmailInvalid    = util.WithPublic(util.ErrInvalidArgument, "invalid email", "validation_failed", "invalid email address")
	ErrUsernameInvalid = util.WithPublic(util.ErrInvalidArgument, "invalid username", "validation_failed", "invalid username")
	ErrUsernameTaken   = util.WithPublic(util.ErrAlreadyExist, "username uniqueness conflict", "username_unavailable", "username unavailable")
	ErrCredentialTaken = util.WithPublic(util.ErrAlreadyExist, "credential ownership conflict", "credential_conflict", "credential cannot be registered")
	ErrTermsInvalid    = util.WithPublic(util.ErrInvalidArgument, "terms version invalid", "validation_failed", "current terms acceptance is required")
)

// NormalizeEmail canonicalizes and validates an email used as an account identity.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) > 254 || strings.ContainsAny(email, "\r\n\t") {
		return "", ErrEmailInvalid
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", ErrEmailInvalid
	}
	return email, nil
}

func NormalizeUsername(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if len(name) < 3 || len(name) > 30 || name == "admin" || name == "system" || name == "support" {
		return "", ErrUsernameInvalid
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return "", ErrUsernameInvalid
		}
	}
	return name, nil
}

func identityConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "accounts_username_uq":
			return ErrUsernameTaken
		case "accounts_email_login_uq", "phone_credentials_phone_e164_key", "oauth2_credentials_pkey", "oauth2_credentials_account_issuer_uq":
			return ErrCredentialTaken
		}
	}
	return err
}

func requireActiveAccount(ctx context.Context, q *db.Queries, accountID int64) error {
	identity, err := q.GetAccountIdentity(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return util.ErrUnauthenticated
	}
	if err != nil {
		return err
	}
	if identity.Status != "active" {
		return util.ErrUnauthenticated
	}
	return nil
}

func UpdateUsername(ctx context.Context, actor authmodel.Actor, raw string) (authmodel.CurrentUser, error) {
	name, err := NormalizeUsername(raw)
	if err != nil {
		return authmodel.CurrentUser{}, err
	}
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (authmodel.CurrentUser, error) {
		q, err := LockActorForMutation(txCtx, actor)
		if err != nil {
			return authmodel.CurrentUser{}, err
		}
		n, err := q.SetAccountUsername(txCtx, db.SetAccountUsernameParams{AccountID: actor.AccountID, Username: name})
		if err != nil {
			return authmodel.CurrentUser{}, err
		}
		if n != 1 {
			return authmodel.CurrentUser{}, util.ErrUnauthenticated
		}
		return GetCurrentUser(txCtx, actor.AccountID)
	})
	return result, identityConflict(err)
}
