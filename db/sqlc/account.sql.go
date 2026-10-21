package db

import "context"

type CreateAccountParams struct {
    PersonID     int64  `json:"person_id"`
    Email        string `json:"email"`
    PasswordHash string `json:"password_hash"`
}

const createAccount = `INSERT INTO accounts (person_id, email, password_hash)
VALUES ($1,$2,$3)
RETURNING id,person_id,email,password_hash,created_at`

func (q *Queries) CreateAccount(ctx context.Context, arg CreateAccountParams) (Account, error) {
    var a Account
    err := q.db.QueryRow(ctx, createAccount, arg.PersonID, arg.Email, arg.PasswordHash).Scan(
        &a.ID, &a.PersonID, &a.Email, &a.PasswordHash, &a.CreatedAt)
    return a, err
}

const getAccountByEmail = `SELECT id,person_id,email,password_hash,created_at
FROM accounts WHERE lower(btrim(email)) = lower(btrim($1)) LIMIT 1`

func (q *Queries) GetAccountByEmail(ctx context.Context, email string) (Account, error) {
    var a Account
    err := q.db.QueryRow(ctx, getAccountByEmail, email).Scan(
        &a.ID, &a.PersonID, &a.Email, &a.PasswordHash, &a.CreatedAt)
    return a, err
}
