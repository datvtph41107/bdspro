package source

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) Capture(ctx context.Context, accountID, organizationID int64, key string, input CaptureInput) (CaptureResult, error) {
	if accountID <= 0 || organizationID <= 0 {
		return CaptureResult{}, ErrForbidden
	}
	if err := validateKey(key); err != nil {
		return CaptureResult{}, err
	}
	input, err := prepareCapture(input)
	if err != nil {
		return CaptureResult{}, err
	}
	digest, err := hashRequest(input)
	if err != nil {
		return CaptureResult{}, err
	}
	var result CaptureResult
	err = dbsqlc.NewStore(s.pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		// Shared organization lock protects authorization and the resulting
		// write from a concurrent member removal on the same organization.
		if _, err := q.LockOrganizationForSourceWrite(ctx, organizationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrForbidden
			}
			return err
		}
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
	input, err := preparePrice(input)
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
		if _, err := q.LockOrganizationForSourceWrite(ctx, organizationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrForbidden
			}
			return err
		}
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
