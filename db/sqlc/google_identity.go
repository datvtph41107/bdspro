package db

import "context"

// LinkGoogleIdentityParams identifies the local account and verified subject.
type LinkGoogleIdentityParams struct {
    AccountID int64
    GoogleSub string
}

// LinkGoogleIdentity keeps the old public Store operation, with its SQL
// defined only in db/query/google_identity.sql.
func (s *SQLStore) LinkGoogleIdentity(ctx context.Context, arg LinkGoogleIdentityParams) error {
    _, err := s.Queries.LinkGoogleIdentityQuery(ctx, LinkGoogleIdentityQueryParams{
        AccountID: arg.AccountID, GoogleSub: arg.GoogleSub,
    })
    return err
}

func (s *SQLStore) GetGoogleAccountID(ctx context.Context, googleSub string) (int64, error) {
    return s.Queries.GetGoogleAccountIDQuery(ctx, googleSub)
}
