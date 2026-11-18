package auth

import (
    "context"
    "errors"
    "strings"

    "github.com/datvtph41107/bdspro/models/db"
    token "github.com/datvtph41107/bdspro/modules/auth"
    "github.com/jackc/pgx/v5"
)

var ErrInvalidPasswordProof = errors.New("auth: invalid credentials")

// This constant is a fixed, unused bcrypt hash. It is not an account password.
// Unknown usernames still perform a bcrypt comparison, reducing username
// enumeration through coarse timing differences. Rate limiting is still required.
const dummyPasswordHash = "$2y$10$okz.yzFWtvrBOT7Cb2elve7ihPHJMFnf5kn51J6879owXVPQ/S1bi"

type passwordReader interface {
    ReadAccountByEmail(context.Context, string) (db.ReadAccountByEmailRow, error)
    GetPasswordHash(context.Context, int64) (string, error)
}

// VerifyPasswordProof verifies a local password and returns the canonical
// account ID. It DOES NOT issue a session or access token; an authenticated
// login operation must separately persist its session and audit outcome.
func VerifyPasswordProof(ctx context.Context, email, password string) (int64, error) {
    if ctx == nil { return 0, errors.New("auth: nil context") }
    if err := ctx.Err(); err != nil { return 0, err }
    if len(email) == 0 || len(email) > 254 || len(password) == 0 || len(password) > 1024 {
        return 0, ErrInvalidPasswordProof
    }
    q, err := db.GetQueries(ctx)
    if err != nil { return 0, err }
    return verifyPasswordProof(ctx, q, email, password)
}

func verifyPasswordProof(ctx context.Context, q passwordReader, email, password string) (int64, error) {
    account, err := q.ReadAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
    if errors.Is(err, pgx.ErrNoRows) {
        _ = token.CheckPassword(password, dummyPasswordHash)
        return 0, ErrInvalidPasswordProof
    }
    if err != nil { return 0, err }
    hash, err := q.GetPasswordHash(ctx, account.ID)
    if errors.Is(err, pgx.ErrNoRows) {
        _ = token.CheckPassword(password, dummyPasswordHash)
        return 0, ErrInvalidPasswordProof
    }
    if err != nil { return 0, err }
    if err := token.CheckPassword(password, hash); err != nil {
        return 0, ErrInvalidPasswordProof
    }
    return account.ID, nil
}
