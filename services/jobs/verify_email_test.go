package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type verificationStoreFake struct {
	account    db.Account
	record     db.VerifyEmail
	accountErr error
	tokenErr   error
	tokens     int
}

func (s *verificationStoreFake) GetAccountByID(_ context.Context, _ int64) (db.Account, error) {
	if s.accountErr != nil {
		return db.Account{}, s.accountErr
	}
	return s.account, nil
}

// ExecTx runs the real SQLC-generated query methods against a small DBTX fake.
// SQL logic and transaction semantics are separately tested with PostgreSQL.
func (s *verificationStoreFake) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	s.tokens++
	if s.tokenErr != nil {
		return s.tokenErr
	}
	return fn(db.New(verificationDBTX{store: s}))
}

type verificationDBTX struct {
	store *verificationStoreFake
}

func (v verificationDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected SQLC Exec")
}

func (v verificationDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected SQLC Query")
}

func (v verificationDBTX) QueryRow(_ context.Context, statement string, args ...any) pgx.Row {
	return verificationRow{store: v.store, statement: statement, args: args}
}

type verificationRow struct {
	store     *verificationStoreFake
	statement string
	args      []any
}

func (r verificationRow) Scan(dest ...any) error {
	switch {
	case strings.Contains(r.statement, "SELECT is_email_verified FROM accounts"):
		*dest[0].(*bool) = r.store.account.IsEmailVerified
		return nil
	case strings.Contains(r.statement, "FROM verify_emails"):
		if r.store.record.ID == 0 {
			return pgx.ErrNoRows
		}
	case strings.Contains(r.statement, "INSERT INTO verify_emails"):
		if r.store.record.ID == 0 {
			r.store.record = db.VerifyEmail{
				ID: 20, AccountID: r.args[0].(int64),
				Email: r.args[1].(string), SecretCode: r.args[2].(string),
			}
		}
	default:
		return errors.New("unexpected SQLC statement")
	}
	record := r.store.record
	*dest[0].(*int64) = record.ID
	*dest[1].(*int64) = record.AccountID
	*dest[2].(*string) = record.Email
	*dest[3].(*string) = record.SecretCode
	*dest[4].(*bool) = record.IsUsed
	*dest[5].(*time.Time) = record.CreatedAt
	*dest[6].(*time.Time) = record.ExpiredAt
	return nil
}

type fakeMailSender struct {
	fail    error
	content []string
}

func (m *fakeMailSender) SendEmail(_, content string, _, _, _, _ []string) error {
	if m.fail != nil {
		return m.fail
	}
	m.content = append(m.content, content)
	return nil
}

func TestVerificationJobSkipsAlreadyVerifiedAccount(t *testing.T) {
	store := &verificationStoreFake{account: db.Account{ID: 8, IsEmailVerified: true}}
	sender := &fakeMailSender{}
	if err := SendVerificationEmail(context.Background(), store, sender, "https://example.test/verify", 8); err != nil {
		t.Fatal(err)
	}
	if store.tokens != 0 || len(sender.content) != 0 {
		t.Fatal("verified account must not request a token or email")
	}
}

func TestVerificationJobRetriesWithReusableCode(t *testing.T) {
	store := &verificationStoreFake{
		account: db.Account{ID: 8, Email: "example@example.test"},
		record:  db.VerifyEmail{ID: 20, AccountID: 8, SecretCode: "reused-secret"},
	}
	sender := &fakeMailSender{fail: errors.New("SMTP unavailable")}
	ctx := context.Background()
	url := "https://example.test/users/verify_email"
	if err := SendVerificationEmail(ctx, store, sender, url, 8); err == nil {
		t.Fatal("SMTP failure must remain retryable")
	}
	sender.fail = nil
	if err := SendVerificationEmail(ctx, store, sender, url, 8); err != nil {
		t.Fatal(err)
	}
	if store.tokens != 2 || len(sender.content) != 1 ||
		!strings.Contains(sender.content[0], "reused-secret") {
		t.Fatal("retry did not reuse persisted verification token")
	}
	store.accountErr = pgx.ErrNoRows
	if err := SendVerificationEmail(ctx, store, sender, url, 8); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("missing account must remain retryable: %v", err)
	}
}
