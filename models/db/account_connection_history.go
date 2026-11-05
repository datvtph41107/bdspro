package db

import "context"

// CreateSessionWithConnection persists a refresh session and its immutable login
// event in one PostgreSQL transaction. No successful session can lack an audit
// event, and failed history writes cannot strand a committed credential.
func (s *SQLStore) CreateSessionWithConnection(ctx context.Context, params CreateSessionParams) (Session, error) {
	return WithTxResult(ctx, s, func(q *Queries) (Session, error) {
		session, err := q.CreateSession(ctx, params)
		if err != nil {
			return Session{}, err
		}
		err = q.CreateAccountConnectionHistory(ctx, CreateAccountConnectionHistoryParams{
			AccountID: params.AccountID,
			SessionID: session.ID,
			IpAddress: params.ClientIp,
			UserAgent: params.UserAgent,
		})
		if err != nil {
			return Session{}, err
		}
		return session, nil
	})
}
