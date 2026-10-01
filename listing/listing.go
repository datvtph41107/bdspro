package listing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
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
	db *pgxpool.Pool,
	title string,
) (Listing, error) {
	if strings.TrimSpace(title) == "" {
		return Listing{}, ErrTitleRequired
	}

	var result Listing

	err := db.QueryRow(
		ctx,
		`
			INSERT INTO listings (title)
			VALUES ($1)
			RETURNING id, title, description, status	
		`,
		title,
	).Scan(
		&result.ID,
		&result.Title,
		&result.Description,
		&result.Status,
	)

	if err != nil {
		return Listing{}, fmt.Errorf(
			"create listing: %w",
			err,
		)
	}

	return result, nil
}

func UpdateDescription(
	ctx context.Context,
	db *pgxpool.Pool,
	id int64,
	description string,
) (Listing, error) {
	if strings.TrimSpace(description) == "" {
		return Listing{}, ErrDescriptionRequired
	}

	var result Listing

	err := db.QueryRow(
		ctx,
		`
			UPDATE listings
			SET description = $2
			WHERE id = $1
			  AND status = 'DRAFT'
			RETURNING id, title, description, status
		`,
		id,
		description,
	).Scan(
		&result.ID,
		&result.Title,
		&result.Description,
		&result.Status,
	)

	if err == nil {
		return result, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return Listing{}, fmt.Errorf(
			"update listing description: %w",
			err,
		)
	}

	var status string

	err = db.QueryRow(
		ctx,
		`
			SELECT status
			FROM listings
			WHERE id = $1
		`,
		id,
	).Scan(&status)

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
	db *pgxpool.Pool,
	id int64,
) (Listing, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return Listing{}, fmt.Errorf(
			"begin publish listing transaction: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var result Listing

	err = tx.QueryRow(
		ctx,
		`
			UPDATE listings
			SET status = 'PUBLISHED'
			WHERE id = $1
			 	AND status = 'DRAFT'
				AND description IS NOT NULL
			RETURNING id, title, status
		`,
		id,
	).Scan(
		&result.ID,
		&result.Title,
		&result.Status,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		var status string

		err = tx.QueryRow(
			ctx,
			`
				SELECT status
				FROM listings
				WHERE id = $1
			`,
			id,
		).Scan(&status)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Listing{}, ErrNotFound

		case err != nil:
			return Listing{}, fmt.Errorf(
				"read listing status: %w",
				err,
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

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO listing_publications (listing_id)
			VALUES ($1)
		`,
		id,
	)
	if err != nil {
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

	return result, nil
}

func CheckPublicationReadiness(
	ctx context.Context,
	db *pgxpool.Pool,
	id int64,
) (PublicationReadiness, error) {
	var item Listing

	err := db.QueryRow(
		ctx,
		`
			SELECT id, title, description, status
			FROM listings
			WHERE id = $1
		`,
		id,
	).Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.Status,
	)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return PublicationReadiness{}, ErrNotFound

	case err != nil:
		return PublicationReadiness{}, fmt.Errorf(
			"read listing publication readiness: %w",
			err,
		)
	}

	return publicationReadiness(item), nil
}

func publicationReadiness(item Listing) PublicationReadiness {
	missing := make([]string, 0)

	if item.Description == nil || strings.TrimSpace(*item.Description) == "" {
		missing = append(missing, "description")
	}

	return PublicationReadiness{
		Ready:   item.Status == "DRAFT" && len(missing) == 0,
		Status:  item.Status,
		Missing: missing,
	}
}
