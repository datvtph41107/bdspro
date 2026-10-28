package db

import "context"

// LinkGoogleIdentityParams identifies the local account and verified subject.
type LinkGoogleIdentityParams struct {
	AccountID int64
	GoogleSub string
}

// LinkGoogleIdentity links a verified Google subject to an account.
// The SQL is owned by db/query/google_identity.sql.
func (s *SQLStore) LinkGoogleIdentity(ctx context.Context, arg LinkGoogleIdentityParams) error {
	_, err := s.Queries.LinkGoogleIdentityQuery(ctx, LinkGoogleIdentityQueryParams{
		AccountID: arg.AccountID, GoogleSub: arg.GoogleSub,
	})
	return err
}

func (s *SQLStore) GetGoogleAccountID(ctx context.Context, googleSub string) (int64, error) {
	return s.Queries.GetGoogleAccountIDQuery(ctx, googleSub)
}
