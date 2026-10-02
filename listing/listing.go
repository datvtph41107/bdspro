package listing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("listing not found")
	ErrAlreadyPublished = errors.New("listing already published")
	ErrNotPublishable   = errors.New("listing is not publishable")

	ErrTitleRequired       = errors.New("listing title is required")
	ErrDescriptionRequired = errors.New("listing description is required")

	ErrNotEditable = errors.New("listing is not editable")
)

type Listing struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
}

type PublicationReadiness struct {
	Ready   bool     `json:"ready"`
	Status  string   `json:"status"`
	Missing []string `json:"missing"`
}

func Create(
	ctx context.Context,
	pool *pgxpool.Pool,
	title string,
) (Listing, error) {
	if strings.TrimSpace(title) == "" {
		return Listing{}, ErrTitleRequired
	}

	queries := dbsqlc.New(pool)

	row, err := queries.CreateListing(ctx, title)
	if err != nil {
		return Listing{}, fmt.Errorf(
			"create listing: %w",
			err,
		)
	}

	return Listing{
		ID:     row.ID,
		Title:  row.Title,
		Status: row.Status,
	}, nil
}

func UpdateDescription(
	ctx context.Context,
	pool *pgxpool.Pool,
	id int64,
	description string,
) (Listing, error) {
	if strings.TrimSpace(description) == "" {
		return Listing{}, ErrDescriptionRequired
	}

	queries := dbsqlc.New(pool)

	row, err := queries.UpdateListingDescription(
		ctx,
		dbsqlc.UpdateListingDescriptionParams{
			Description: description,
			ID:          id,
		},
	)

	if err == nil {
		return Listing{
			ID:          row.ID,
			Title:       row.Title,
			Description: textPtr(row.Description),
			Status:      row.Status,
		}, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return Listing{}, fmt.Errorf(
			"update listing description: %w",
			err,
		)
	}

	status, err := queries.GetListingStatus(ctx, id)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Listing{}, ErrNotFound

	case err != nil:
		return Listing{}, fmt.Errorf(
			"read listing status: %w",
			err,
		)

	case status == "PUBLISHED":
		return Listing{}, ErrNotEditable

	default:
		return Listing{}, fmt.Errorf(
			"listing %d cannot be edited from status %q",
			id,
			status,
		)
	}
}

func Publish(
	ctx context.Context,
	pool *pgxpool.Pool,
	id int64,
) (Listing, error) {
	queries := dbsqlc.New(pool)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Listing{}, fmt.Errorf(
			"begin publish listing transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := queries.WithTx(tx)

	row, err := txQueries.MarkListingPublished(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		status, statusErr := txQueries.GetListingStatus(ctx, id)

		switch {
		case errors.Is(statusErr, pgx.ErrNoRows):
			return Listing{}, ErrNotFound

		case statusErr != nil:
			return Listing{}, fmt.Errorf(
				"read listing status: %w",
				statusErr,
			)

		case status == "PUBLISHED":
			return Listing{}, ErrAlreadyPublished

		case status == "DRAFT":
			return Listing{}, ErrNotPublishable

		default:
			return Listing{}, fmt.Errorf(
				"listing %d cannot be published from status %q",
				id,
				status,
			)
		}
	}

	if err != nil {
		return Listing{}, fmt.Errorf(
			"publish listing: %w",
			err,
		)
	}

	if err := txQueries.CreateListingPublication(ctx, id); err != nil {
		return Listing{}, fmt.Errorf(
			"record listing publication: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return Listing{}, fmt.Errorf(
			"commit publish listing: %w",
			err,
		)
	}

	return Listing{
		ID:     row.ID,
		Title:  row.Title,
		Status: row.Status,
	}, nil
}

func CheckPublicationReadiness(
	ctx context.Context,
	pool *pgxpool.Pool,
	id int64,
) (PublicationReadiness, error) {
	queries := dbsqlc.New(pool)

	row, err := queries.GetListingForPublicationReadiness(ctx, id)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return PublicationReadiness{}, ErrNotFound

	case err != nil:
		return PublicationReadiness{}, fmt.Errorf(
			"read listing publication readiness: %w",
			err,
		)
	}

	item := Listing{
		ID:          row.ID,
		Title:       row.Title,
		Description: textPtr(row.Description),
		Status:      row.Status,
	}

	return publicationReadiness(item), nil
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	text := value.String

	return &text
}

func publicationReadiness(item Listing) PublicationReadiness {
	missing := make([]string, 0)

	if item.Description == nil ||
		strings.TrimSpace(*item.Description) == "" {
		missing = append(missing, "description")
	}

	return PublicationReadiness{
		Ready:   item.Status == "DRAFT" && len(missing) == 0,
		Status:  item.Status,
		Missing: missing,
	}
}
