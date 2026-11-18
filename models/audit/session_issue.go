package audit

import (
	"context"
	"errors"
	"strconv"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// RecordSessionIssued is a required security record. It runs in the
// transaction that inserts the session and login-history row.
// No access token, refresh token, credential secret or password is stored.
func RecordSessionIssued(ctx context.Context, accountID int64, sessionID uuid.UUID) error {
	if ctx == nil || !db.InTransaction(ctx) {
		return errors.New("audit: session issuance requires transaction")
	}
	if accountID <= 0 || sessionID == uuid.Nil {
		return errors.New("audit: invalid session issuance")
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	return q.InsertAuditEvent(ctx, db.InsertAuditEventParams{
		Action:          "auth.session.issued",
		ActorID:         pgtype.Int8{Int64: accountID, Valid: true},
		ActorName:       "",
		ActorCredential: "verified",
		ScopeType:       "account",
		ScopeRef:        strconv.FormatInt(accountID, 10),
		TargetType:      "session",
		TargetRef:       sessionID.String(),
		Origin:          "api",
		Message:         "authenticated session issued",
		Metadata:        []byte("{}"),
	})
}
