package db

import "context"

// CreateGoogleAccountTx atomically creates a passwordless account and its
// verified Google identity. It never links an existing account by email.
func (s *SQLStore) CreateGoogleAccountTx(ctx context.Context, email, googleSub string) (Account, error) {
	var account Account
	err := s.ExecTx(ctx, func(q *Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreatePerson(ctx, partyID); err != nil {
			return err
		}
		if err := q.CreatePersonProfile(ctx, partyID); err != nil {
			return err
		}
		account, err = q.CreateVerifiedGoogleAccount(ctx, CreateVerifiedGoogleAccountParams{
			PersonID: partyID, Email: email,
		})
		if err != nil {
			return err
		}
		if err := q.CreateAccountSettings(ctx, account.ID); err != nil {
			return err
		}
		_, err = q.LinkGoogleIdentityQuery(ctx, LinkGoogleIdentityQueryParams{
			AccountID: account.ID, GoogleSub: googleSub,
		})
		return err
	})
	return account, err
}
