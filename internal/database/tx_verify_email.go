package db

import "context"

// VerifyEmailTx consumes one code and updates the owning account in one transaction.
func (s *SQLStore) VerifyEmailTx(ctx context.Context, arg ConsumeVerifyEmailParams) (Account, error) {
	var account Account
	err := s.ExecTx(ctx, func(q *Queries) error {
		verification, err := q.ConsumeVerifyEmail(ctx, arg)
		if err != nil {
			return err
		}
		account, err = q.MarkEmailVerified(ctx, MarkEmailVerifiedParams{ID: verification.AccountID, Email: verification.Email})
		return err
	})
	return account, err
}
