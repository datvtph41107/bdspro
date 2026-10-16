package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalid            = errors.New("invalid registration input")
	ErrEmailUsed          = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnverified         = errors.New("email not verified")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrMailerUnavailable  = errors.New("verification delivery unavailable")
)

type Mailer interface {
	SendVerification(context.Context, string, string) error
}
type Service struct {
	Pool   *pgxpool.Pool
	Mailer Mailer
}

func New(pool *pgxpool.Pool, mailer Mailer) *Service { return &Service{Pool: pool, Mailer: mailer} }

func normalizedEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, "@") || len(email) > 254 || len(email) < 3 {
		return "", ErrInvalid
	}
	return email, nil
}
func randomToken() (string, string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(data[:])
	digest := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(digest[:]), nil
}
func digestToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
func hashPassword(password string) (string, error) {
	var salt [16]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt[:], 2, 19*1024, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=19456,t=2,p=1$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt[:]), base64.RawStdEncoding.EncodeToString(key)), nil
}
func checkPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=19456,t=2,p=1" {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(parts[4])
	key, e2 := base64.RawStdEncoding.DecodeString(parts[5])
	if e1 != nil || e2 != nil || len(salt) != 16 || len(key) != 32 {
		return false
	}
	comparison := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)
	return subtle.ConstantTimeCompare(comparison, key) == 1
}

var dummyHash = func() string {
	salt := make([]byte, 16)
	key := argon2.IDKey([]byte("nonexistent-account"), salt, 2, 19*1024, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=19456,t=2,p=1$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}()

type Registration struct {
	AccountID int64
	// Debug only, never return this token in publicly deployed HTTP responses.
	VerificationToken string
}

func (s *Service) Register(ctx context.Context, email, password string) (Registration, error) {
	email, err := normalizedEmail(email)
	if err != nil || len(password) < 12 || len(password) > 128 {
		return Registration{}, ErrInvalid
	}
	hash, err := hashPassword(password)
	if err != nil {
		return Registration{}, err
	}
	token, tokenHash, err := randomToken()
	if err != nil {
		return Registration{}, err
	}
	var accountID int64
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := dbsqlc.New(tx)
		partyID, err := q.CreatePersonParty(ctx)
		if err != nil { return err }
		if err := q.CreatePersonSubtype(ctx, partyID); err != nil { return err }
		accountID, err = q.CreateAccount(ctx, partyID)
		if err != nil { return err }
		emailID, err := q.CreatePrimaryAccountEmail(ctx, dbsqlc.CreatePrimaryAccountEmailParams{
			AccountID: accountID, Email: email,
		})
		if err != nil { return err }
		if err := q.CreatePasswordCredential(ctx, dbsqlc.CreatePasswordCredentialParams{
			AccountID: accountID, PasswordHash: hash,
		}); err != nil { return err }
		return q.CreateEmailVerificationToken(ctx, dbsqlc.CreateEmailVerificationTokenParams{
			AccountEmailID: emailID, TokenHash: tokenHash,
		})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Registration{}, ErrEmailUsed
		}
		return Registration{}, fmt.Errorf("registration transaction: %w", err)
	}
	if s.Mailer != nil {
		if err := s.Mailer.SendVerification(ctx, email, token); err != nil {
			return Registration{AccountID: accountID}, fmt.Errorf("%w: %v", ErrMailerUnavailable, err)
		}
	}
	return Registration{AccountID: accountID, VerificationToken: token}, nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if len(token) < 30 || len(token) > 256 { return ErrInvalidToken }
	hash := digestToken(token)
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := dbsqlc.New(tx)
		emailID, err := q.ConsumeEmailVerificationToken(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) { return ErrInvalidToken }
		if err != nil { return err }
		return q.MarkAccountEmailVerified(ctx, emailID)
	})
}

type LoginResult struct {
	Token string
	ExpiresAt time.Time
	AccountID int64
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email, err := normalizedEmail(email)
	if err != nil {
		_ = checkPassword(password, dummyHash)
		return LoginResult{}, ErrInvalidCredentials
	}
	q := dbsqlc.New(s.Pool)
	credentials, err := q.GetLoginCredentials(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = checkPassword(password, dummyHash)
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil { return LoginResult{}, err }
	if !checkPassword(password, credentials.PasswordHash) { return LoginResult{}, ErrInvalidCredentials }
	if !credentials.EmailVerified { return LoginResult{}, ErrUnverified }
	token, hash, err := randomToken()
	if err != nil { return LoginResult{}, err }
	expiry, err := q.CreateLoginSession(ctx, dbsqlc.CreateLoginSessionParams{
		AccountID: credentials.AccountID, TokenHash: hash,
	})
	if err != nil { return LoginResult{}, err }
	return LoginResult{Token: token, ExpiresAt: expiry, AccountID: credentials.AccountID}, nil
}

func (s *Service) AccountFromToken(ctx context.Context, token string) (int64, error) {
	if token == "" { return 0, ErrUnauthorized }
	id, err := dbsqlc.New(s.Pool).GetSessionAccount(ctx, digestToken(token))
	if errors.Is(err, pgx.ErrNoRows) { return 0, ErrUnauthorized }
	return id, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" { return nil }
	return dbsqlc.New(s.Pool).RevokeSession(ctx, digestToken(token))
}

// Stable API boundary validates explicitly supplied integer route IDs.
func PositiveID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalid
	}
	return id, nil
}
