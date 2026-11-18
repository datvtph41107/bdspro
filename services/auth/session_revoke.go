package auth

import (
	"context"
	"errors"
	"fmt"

	auditmodel "github.com/datvtph41107/bdspro/models/audit"
	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RevokeSession blocks an owned session with the verified acting session
// held under a database lock. The required audit record is part of the
// same transaction: audit failure rolls back the revocation.
func RevokeSession(
	ctx context.Context,
	doer *authmodel.Actor,
	targetSessionID uuid.UUID,
) (bool, error) {
	if ctx == nil {
		return false, errors.New("auth: nil context")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if doer == nil || doer.AccountID <= 0 || doer.SessionID == uuid.Nil {
		return false, util.ErrUnauthenticated
	}
	if targetSessionID == uuid.Nil {
		return false, errors.New("auth: invalid target session")
	}

	// Unlike issuing a new token, a revoke operation is security-critical:
	// account lock -> acting session lock -> owned target update -> audit.
	return db.WithTx2(ctx, func(txCtx context.Context) (bool, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return false, err
		}
		_, err = q.LockSessionRevocationAccount(txCtx, doer.AccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, util.ErrUnauthenticated
		}
		if err != nil {
			return false, fmt.Errorf("auth: lock revocation account: %w", err)
		}

		_, err = q.LockActingSession(txCtx, db.LockActingSessionParams{
			SessionID: doer.SessionID,
			AccountID: doer.AccountID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return false, util.ErrUnauthenticated
		}
		if err != nil {
			return false, fmt.Errorf("auth: validate acting session: %w", err)
		}

		changed, err := q.RevokeOwnedSession(txCtx, db.RevokeOwnedSessionParams{
			ID:        targetSessionID,
			AccountID: doer.AccountID,
		})
		if err != nil {
			return false, fmt.Errorf("auth: revoke session: %w", err)
		}
		if changed == 0 {
			return false, nil
		}
		if changed != 1 {
			return false, errors.New("auth: unexpected revoke row count")
		}

		if err := auditmodel.RecordSessionRevoked(txCtx, doer.AccountID, targetSessionID); err != nil {
			return false, fmt.Errorf("auth: audit session revocation: %w", err)
		}
		return true, nil
	})
}
