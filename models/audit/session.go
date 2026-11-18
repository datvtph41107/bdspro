package audit

import (
	"context"
	"errors"
	"strconv"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// RecordSessionRevoked records a security-critical mutation while the
// caller's PostgreSQL transaction is still open. Callers MUST propagate
// txCtx, and MUST fail the transaction if the audit INSERT fails.
func RecordSessionRevoked(
	ctx context.Context,
	actorAccountID int64,
	targetSessionID uuid.UUID,
) error {
	if ctx == nil || !db.InTransaction(ctx) {
		return errors.New("audit: session revoke requires transaction")
	}
	if actorAccountID <= 0 || targetSessionID == uuid.Nil {
		return errors.New("audit: invalid session revoke")
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	return q.InsertAuditEvent(ctx, db.InsertAuditEventParams{
		Action:          "auth.session.revoked",
		ActorID:         pgtype.Int8{Int64: actorAccountID, Valid: true},
		ActorName:       "",
		ActorCredential: "session",
		ScopeType:       "account",
		ScopeRef:        strconv.FormatInt(actorAccountID, 10),
		TargetType:      "session",
		TargetRef:       targetSessionID.String(),
		Origin:          "api",
		Message:         "session revoked",
		Metadata:        []byte("{}"),
	})
}
