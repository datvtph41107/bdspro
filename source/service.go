package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalid     = errors.New("invalid source input")
	ErrForbidden   = errors.New("source operation not permitted")
	ErrNotFound    = errors.New("source not found")
	ErrKeyConflict = errors.New("idempotency key reused with different input")
)
var (
	areaRe     = regexp.MustCompile("^[0-9]{1,14}(\\.[0-9]{1,4})?$")
	priceRe    = regexp.MustCompile("^[0-9]{1,15}(\\.[0-9]{1,4})?$")
	currencyRe = regexp.MustCompile("^[A-Z]{3}$")
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// The accountID parameter must originate from a verified session, not request JSON.
type PriceInput struct {
	Amount      string
	Currency    string
	StatementAt string
	PriceBasis  string
	SourceBasis string
	Note        string
}
type CaptureInput struct {
	Channel      string
	Label        string
	AreaM2       string
	AreaBasis    string
	ReceivedAt   string
	Summary      string
	InitialPrice *PriceInput
}
type CaptureResult struct {
	ID       int64
	Replayed bool
}
type PriceResult struct {
	ID       int64
	Replayed bool
}
type Source struct {
	ID                  int64
	OrganizationID      int64
	RecordedByAccountID int64
	Channel             string
	Label               string
	AreaM2              string
	AreaBasis           string
	Summary             string
	Version             int32
	RecordedAt          time.Time
	ReceivedAt          *time.Time
}
type PriceReport struct {
	ID                  int64
	IntakeID            int64
	RecordedByAccountID int64
	Amount              string
	Currency            string
	PriceBasis          string
	SourceBasis         string
	Note                string
	RecordedAt          time.Time
	StatementAt         *time.Time
}

func normalizeDecimal(v string, pattern *regexp.Regexp) (string, error) {
	v = strings.TrimSpace(v)
	if !pattern.MatchString(v) {
		return "", ErrInvalid
	}
	r := new(big.Rat)
	if _, ok := r.SetString(v); !ok || r.Sign() <= 0 {
		return "", ErrInvalid
	}
	parts := strings.SplitN(v, ".", 2)
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	if len(parts) < 2 {
		return whole, nil
	}
	frac := strings.TrimRight(parts[1], "0")
	if frac == "" {
		return whole, nil
	}
	return whole + "." + frac, nil
}
func normalizeTimestamp(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", ErrInvalid
	}
	return t.UTC().Format(time.RFC3339Nano), nil
}
func normalizePrice(p PriceInput) (PriceInput, error) {
	var err error
	p.Amount, err = normalizeDecimal(p.Amount, priceRe)
	if err != nil {
		return PriceInput{}, fmt.Errorf("%w: price amount", ErrInvalid)
	}
	p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
	if !currencyRe.MatchString(p.Currency) {
		return PriceInput{}, fmt.Errorf("%w: currency", ErrInvalid)
	}
	p.StatementAt, err = normalizeTimestamp(p.StatementAt)
	if err != nil {
		return PriceInput{}, fmt.Errorf("%w: statement_at", ErrInvalid)
	}
	p.PriceBasis = strings.TrimSpace(p.PriceBasis)
	p.SourceBasis = strings.TrimSpace(p.SourceBasis)
	p.Note = strings.TrimSpace(p.Note)
	if len(p.Note) > 4000 {
		return PriceInput{}, fmt.Errorf("%w: price note too long", ErrInvalid)
	}
	return p, nil
}
func normalizeCapture(in CaptureInput) (CaptureInput, error) {
	in.Channel = strings.ToLower(strings.TrimSpace(in.Channel))
	in.Label = strings.TrimSpace(in.Label)
	in.AreaM2 = strings.TrimSpace(in.AreaM2)
	in.AreaBasis = strings.TrimSpace(in.AreaBasis)
	in.Summary = strings.TrimSpace(in.Summary)
	var stampErr error
	in.ReceivedAt, stampErr = normalizeTimestamp(in.ReceivedAt)
	if stampErr != nil {
		return CaptureInput{}, fmt.Errorf("%w: received_at", ErrInvalid)
	}
	if in.Channel == "" || len(in.Channel) > 50 || len(in.Label) > 400 ||
		len(in.Summary) > 4000 || len(in.AreaBasis) > 80 {
		return CaptureInput{}, fmt.Errorf("%w: source metadata", ErrInvalid)
	}
	if in.AreaM2 != "" {
		var err error
		in.AreaM2, err = normalizeDecimal(in.AreaM2, areaRe)
		if err != nil {
			return CaptureInput{}, fmt.Errorf("%w: area", ErrInvalid)
		}
	}
	if in.InitialPrice != nil {
		p, err := normalizePrice(*in.InitialPrice)
		if err != nil {
			return CaptureInput{}, err
		}
		in.InitialPrice = &p
	}
	return in, nil
}
func validateKey(key string) error {
	if len(key) == 0 || len(key) > 100 || strings.TrimSpace(key) != key {
		return fmt.Errorf("%w: Idempotency-Key must be 1..100 bytes", ErrInvalid)
	}
	return nil
}
func hashRequest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func (s *Service) Capture(ctx context.Context, accountID, organizationID int64, key string, input CaptureInput) (CaptureResult, error) {
	if accountID <= 0 || organizationID <= 0 {
		return CaptureResult{}, ErrForbidden
	}
	if err := validateKey(key); err != nil {
		return CaptureResult{}, err
	}
	input, err := normalizeCapture(input)
	if err != nil {
		return CaptureResult{}, err
	}
	digest, err := hashRequest(input)
	if err != nil {
		return CaptureResult{}, err
	}
	var result CaptureResult
	err = dbsqlc.NewStore(s.pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		id, err := q.InsertSourceIntake(ctx, dbsqlc.InsertSourceIntakeParams{
			OrganizationID: organizationID, ActorAccountID: accountID,
			Channel: input.Channel, Label: input.Label, AreaM2: input.AreaM2,
			AreaBasis: input.AreaBasis, ReceivedAt: input.ReceivedAt, Summary: input.Summary,
			RequestKey: key, RequestHash: digest, HasInitialPrice: input.InitialPrice != nil,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			old, e := q.FindAuthorizedCaptureReplay(ctx, dbsqlc.FindAuthorizedCaptureReplayParams{
				OrganizationID: organizationID, ActorAccountID: accountID, RequestKey: key,
			})
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrForbidden
			}
			if e != nil {
				return e
			}
			if old.RequestHash != digest {
				return ErrKeyConflict
			}
			result = CaptureResult{ID: old.ID, Replayed: true}
			return nil
		}
		if err != nil {
			return fmt.Errorf("insert source: %w", err)
		}
		if input.InitialPrice != nil {
			p := *input.InitialPrice
			_, err = q.InsertPriceReport(ctx, dbsqlc.InsertPriceReportParams{
				OrganizationID: organizationID, IntakeID: id, ActorAccountID: accountID,
				Amount: p.Amount, Currency: p.Currency, PriceBasis: p.PriceBasis,
				SourceBasis: p.SourceBasis, StatementAt: p.StatementAt, Note: p.Note,
				PriceRequestKey: "c:" + key, PriceRequestHash: digest,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrForbidden
			}
			if err != nil {
				return fmt.Errorf("insert initial price: %w", err)
			}
		}
		result = CaptureResult{ID: id}
		return nil
	})
	if err != nil {
		return CaptureResult{}, err
	}
	return result, nil
}

func (s *Service) ReportPrice(ctx context.Context, accountID, organizationID, intakeID int64, key string, input PriceInput) (PriceResult, error) {
	if accountID <= 0 || organizationID <= 0 || intakeID <= 0 {
		return PriceResult{}, ErrForbidden
	}
	if err := validateKey(key); err != nil {
		return PriceResult{}, err
	}
	input, err := normalizePrice(input)
	if err != nil {
		return PriceResult{}, err
	}
	digest, err := hashRequest(input)
	if err != nil {
		return PriceResult{}, err
	}
	key = "p:" + key
	var result PriceResult
	err = dbsqlc.NewStore(s.pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		id, err := q.InsertPriceReport(ctx, dbsqlc.InsertPriceReportParams{
			OrganizationID: organizationID, IntakeID: intakeID, ActorAccountID: accountID,
			Amount: input.Amount, Currency: input.Currency, PriceBasis: input.PriceBasis,
			SourceBasis: input.SourceBasis, StatementAt: input.StatementAt, Note: input.Note,
			PriceRequestKey: key, PriceRequestHash: digest,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			old, e := q.FindAuthorizedPriceReplay(ctx, dbsqlc.FindAuthorizedPriceReplayParams{
				OrganizationID: organizationID, IntakeID: intakeID, ActorAccountID: accountID,
				PriceRequestKey: key,
			})
			if errors.Is(e, pgx.ErrNoRows) {
				return ErrForbidden
			}
			if e != nil {
				return e
			}
			if old.RequestHash != digest {
				return ErrKeyConflict
			}
			result = PriceResult{ID: old.ID, Replayed: true}
			return nil
		}
		if err != nil {
			return fmt.Errorf("insert price report: %w", err)
		}
		result = PriceResult{ID: id}
		return nil
	})
	if err != nil {
		return PriceResult{}, err
	}
	return result, nil
}

func (s *Service) Read(ctx context.Context, accountID, organizationID, intakeID int64) (Source, error) {
	if accountID <= 0 || organizationID <= 0 || intakeID <= 0 {
		return Source{}, ErrNotFound
	}
	q := dbsqlc.New(s.pool)
	r, err := q.GetAuthorizedSource(ctx, dbsqlc.GetAuthorizedSourceParams{
		OrganizationID: organizationID, IntakeID: intakeID, ActorAccountID: accountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	if err != nil {
		return Source{}, err
	}
	var receivedAt *time.Time
	if r.ReceivedAt.Valid {
		t := r.ReceivedAt.Time
		receivedAt = &t
	}
	return Source{
		ID: r.ID, OrganizationID: r.OrganizationID, RecordedByAccountID: r.RecordedByAccountID,
		Channel: r.SourceChannelCode, Label: r.SourceLabel, AreaM2: r.ReportedAreaM2,
		AreaBasis: r.AreaBasisCode, Summary: r.Summary, Version: r.Version, RecordedAt: r.RecordedAt, ReceivedAt: receivedAt,
	}, nil
}
func (s *Service) PriceHistory(ctx context.Context, accountID, organizationID, intakeID int64) ([]PriceReport, error) {
	if _, err := s.Read(ctx, accountID, organizationID, intakeID); err != nil {
		return nil, err
	}
	q := dbsqlc.New(s.pool)
	rows, err := q.ListAuthorizedPriceReports(ctx, dbsqlc.ListAuthorizedPriceReportsParams{
		OrganizationID: organizationID, IntakeID: intakeID, ActorAccountID: accountID,
	})
	if err != nil {
		return nil, err
	}
	history := make([]PriceReport, 0, len(rows))
	for _, r := range rows {
		var statementAt *time.Time
		if r.StatementAt.Valid {
			t := r.StatementAt.Time
			statementAt = &t
		}
		history = append(history, PriceReport{
			ID: r.ID, IntakeID: r.IntakeID, RecordedByAccountID: r.RecordedByAccountID,
			Amount: r.Amount, Currency: r.CurrencyCode, PriceBasis: r.PriceBasisCode,
			SourceBasis: r.SourceBasisCode, Note: r.Note, RecordedAt: r.RecordedAt, StatementAt: statementAt,
		})
	}
	return history, nil
}
