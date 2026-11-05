package account

import (
	"errors"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidProfile       = util.WithPublic(util.ErrInvalidArgument, "invalid profile", "account.invalid_profile", "invalid profile")
	ErrInvalidSettings      = util.WithPublic(util.ErrInvalidArgument, "invalid account settings", "account.invalid_settings", "invalid account settings")
	ErrAccountUnavailable   = util.WithPublic(util.ErrUnauthenticated, "account unavailable", "auth.account_unavailable", "account unavailable")
	ErrEmailRegistered      = util.WithPublic(util.ErrAlreadyExist, "email already registered", "auth.email_registered", "email already registered")
	ErrEmailAlreadyVerified = errors.New("email already verified")
	ErrEmailLinkUnavailable = errors.New("account cannot attach an email")
	ErrEmailLinkTaken       = errors.New("email address belongs to another account")
	ErrEmailLinkCooldown    = errors.New("email verification temporarily unavailable")
	ErrEmailLinkInvalid     = errors.New("invalid or expired email link")
)

// emailLoginTaken identifies the exact persisted login email uniqueness index.
// Other 23505 errors are not account-email conflicts.
func emailLoginTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "accounts_email_login_uq"
}
