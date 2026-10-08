package listing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	store dbsqlc.Store,
	title string,
) (Listing, error) {
	if strings.TrimSpace(title) == "" {
		return Listing{}, ErrTitleRequired
	}

	row, err := store.CreateListing(ctx, title)
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
	store dbsqlc.Store,
	id int64,
	description string,
) (Listing, error) {
	if strings.TrimSpace(description) == "" {
		return Listing{}, ErrDescriptionRequired
	}

	row, err := store.UpdateListingDescription(
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

	status, err := store.GetListingStatus(ctx, id)

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
	store dbsqlc.Store,
	id int64,
) (Listing, error) {
	var published dbsqlc.MarkListingPublishedRow

	err := store.ExecTx(
		ctx,
		func(queries dbsqlc.Querier) error {
			row, err := queries.MarkListingPublished(
				ctx,
				id,
			)

			if errors.Is(err, pgx.ErrNoRows) {
				status, statusErr := queries.GetListingStatus(
					ctx,
					id,
				)

				switch {
				case errors.Is(statusErr, pgx.ErrNoRows):
					return ErrNotFound

				case statusErr != nil:
					return fmt.Errorf(
						"read listing status: %w",
						statusErr,
					)

				case status == "PUBLISHED":
					return ErrAlreadyPublished

				case status == "DRAFT":
					return ErrNotPublishable

				default:
					return fmt.Errorf(
						"listing %d cannot be published from status %q",
						id,
						status,
					)
				}
			}

			if err != nil {
				return fmt.Errorf(
					"mark listing published: %w",
					err,
				)
			}

			if err := queries.CreateListingPublication(
				ctx,
				id,
			); err != nil {
				return fmt.Errorf(
					"record listing publication: %w",
					err,
				)
			}

			published = row

			return nil
		},
	)
	if err != nil {
		return Listing{}, fmt.Errorf(
			"publish listing transaction: %w",
			err,
		)
	}

	return Listing{
		ID:     published.ID,
		Title:  published.Title,
		Status: published.Status,
	}, nil
}

func CheckPublicationReadiness(
	ctx context.Context,
	store dbsqlc.Store,
	id int64,
) (PublicationReadiness, error) {
	row, err := store.GetListingForPublicationReadiness(
		ctx,
		id,
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
		missing = append(
			missing,
			"description",
		)
	}

	return PublicationReadiness{
		Ready:   item.Status == "DRAFT" && len(missing) == 0,
		Status:  item.Status,
		Missing: missing,
	}
}
