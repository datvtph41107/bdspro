package db

import "context"

// The database permits accounts without email. The SQLC projected rows
// coalesce that nullable column at the query boundary; these two methods
// restore the stable Account contract used by Service and background jobs.
func (s *SQLStore) GetAccountByID(ctx context.Context, id int64) (Account, error) {
 row,err:=s.Queries.ReadAccountByID(ctx,id)
 if err!=nil {return Account{},err}
 return Account{
  ID:row.ID,PersonID:row.PersonID,Email:row.Email,
  CreatedAt:row.CreatedAt,IsEmailVerified:row.IsEmailVerified,
 },nil
}
func (s *SQLStore) GetAccountByEmail(ctx context.Context, email string) (Account,error) {
 row,err:=s.Queries.ReadAccountByEmail(ctx,email)
 if err!=nil {return Account{},err}
 return Account{
  ID:row.ID,PersonID:row.PersonID,Email:row.Email,
  CreatedAt:row.CreatedAt,IsEmailVerified:row.IsEmailVerified,
 },nil
}
