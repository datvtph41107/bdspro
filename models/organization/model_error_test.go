package organization

import (
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestModelWriteErrorsPreserveIdentityWithoutLeakingSQL(t *testing.T) {
	duplicate := &pgconn.PgError{
		Code: "23505", ConstraintName: "teams_org_name_uq", Message: "secret DSN detail",
	}
	err := teamWriteError(duplicate)
	if err.Error() != ErrTeamAlreadyExists.Error() ||
		!errors.Is(err, ErrTeamAlreadyExists) ||
		!errors.Is(err, util.ErrAlreadyExist) ||
		!errors.Is(err, duplicate) {
		t.Fatalf("error identity or safe message lost: %v", err)
	}
	if raw := teamWriteError(&pgconn.PgError{
		Code: "23505", ConstraintName: "unrelated_unique", Message: "private",
	}); errors.Is(raw, ErrTeamAlreadyExists) {
		t.Fatalf("unrecognized SQL constraint classified as domain conflict: %v", raw)
	}
}
