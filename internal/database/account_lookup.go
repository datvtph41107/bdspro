package db

import "context"

// Account is the stable application representation even when email is NULL.
// SQLC query rows are projected through COALESCE to avoid fabricated emails.
func (s *SQLStore) GetAccountByID(ctx context.Context, id int64) (Account, error) {
	row, err := s.Queries.ReadAccountByID(ctx, id)
	if err != nil {
		return Account{}, err
	}
	return Account{
		ID: row.ID, PersonID: row.PersonID, Email: row.Email,
		CreatedAt: row.CreatedAt, IsEmailVerified: row.IsEmailVerified,
	}, nil
}

func (s *SQLStore) GetAccountByEmail(ctx context.Context, email string) (Account, error) {
	row, err := s.Queries.ReadAccountByEmail(ctx, email)
	if err != nil {
		return Account{}, err
	}
	return Account{
		ID: row.ID, PersonID: row.PersonID, Email: row.Email,
		CreatedAt: row.CreatedAt, IsEmailVerified: row.IsEmailVerified,
	}, nil
}
