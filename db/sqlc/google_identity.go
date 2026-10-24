package db

import "context"

// LinkGoogleIdentityParams identifies a previously authenticated local account
// and a Google subject verified with Google's signed ID token.
type LinkGoogleIdentityParams struct {
    AccountID int64
    GoogleSub string
}

// LinkGoogleIdentity is idempotent for the same pair, never reassigns identities.
func (s *SQLStore) LinkGoogleIdentity(ctx context.Context, arg LinkGoogleIdentityParams) error {
    const query = `INSERT INTO google_identities (account_id, google_sub)
VALUES ($1,$2)
ON CONFLICT (google_sub) DO UPDATE SET google_sub = EXCLUDED.google_sub
WHERE google_identities.account_id = EXCLUDED.account_id
RETURNING account_id`
    var accountID int64
    return s.pool.QueryRow(ctx,query,arg.AccountID,arg.GoogleSub).Scan(&accountID)
}

func (s *SQLStore) GetGoogleAccountID(ctx context.Context, googleSub string) (int64,error) {
    const query = `SELECT account_id FROM google_identities WHERE google_sub = $1`
    var accountID int64
    err := s.pool.QueryRow(ctx,query,googleSub).Scan(&accountID)
    return accountID,err
}
