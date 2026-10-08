//go:build integration

package listing

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newIntegrationStore(
	t *testing.T,
) (*pgxpool.Pool, dbsqlc.Store) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create database pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool, dbsqlc.NewStore(pool)
}

func TestPublishRollsBackWhenPublicationInsertFails(
	t *testing.T,
) {
	pool, store := newIntegrationStore(t)

	ctx := context.Background()

	created, err := store.CreateListing(
		ctx,
		"rollback transaction proof",
	)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}

	listingID := created.ID

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		_, _ = pool.Exec(
			cleanupCtx,
			`
				DELETE FROM listing_publications
				WHERE listing_id = $1
			`,
			listingID,
		)

		_, _ = pool.Exec(
			cleanupCtx,
			`
				DELETE FROM listings
				WHERE id = $1
			`,
			listingID,
		)
	})

	_, err = store.UpdateListingDescription(
		ctx,
		dbsqlc.UpdateListingDescriptionParams{
			ID:          listingID,
			Description: "description for rollback proof",
		},
	)
	if err != nil {
		t.Fatalf(
			"update listing description: %v",
			err,
		)
	}

	err = store.CreateListingPublication(
		ctx,
		listingID,
	)
	if err != nil {
		t.Fatalf(
			"prepare existing publication: %v",
			err,
		)
	}

	err = func() error {
		_, err := Publish(
			ctx,
			store,
			listingID,
		)
		return err
	}()

	if err == nil {
		t.Fatal(
			"expected publish transaction to fail",
		)
	}

	status, err := store.GetListingStatus(
		ctx,
		listingID,
	)
	if err != nil {
		t.Fatalf(
			"get listing status after rollback: %v",
			err,
		)
	}

	if status != "DRAFT" {
		t.Fatalf(
			"expected listing status DRAFT after rollback, got %q",
			status,
		)
	}

	var publicationCount int

	err = pool.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM listing_publications
			WHERE listing_id = $1
		`,
		listingID,
	).Scan(&publicationCount)
	if err != nil {
		t.Fatalf(
			"count listing publications: %v",
			err,
		)
	}

	if publicationCount != 1 {
		t.Fatalf(
			"expected exactly 1 pre-existing publication, got %d",
			publicationCount,
		)
	}
}

func TestPublishConcurrentSameListingHasOneWinner(
	t *testing.T,
) {
	pool, store := newIntegrationStore(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	created, err := store.CreateListing(
		ctx,
		"concurrent publish proof",
	)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}

	listingID := created.ID

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		_, _ = pool.Exec(
			cleanupCtx,
			`
				DELETE FROM listing_publications
				WHERE listing_id = $1
			`,
			listingID,
		)

		_, _ = pool.Exec(
			cleanupCtx,
			`
				DELETE FROM listings
				WHERE id = $1
			`,
			listingID,
		)
	})

	_, err = store.UpdateListingDescription(
		ctx,
		dbsqlc.UpdateListingDescriptionParams{
			ID:          listingID,
			Description: "ready for concurrent publication",
		},
	)
	if err != nil {
		t.Fatalf(
			"update listing description: %v",
			err,
		)
	}

	start := make(chan struct{})
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		go func() {
			<-start

			_, err := Publish(
				ctx,
				store,
				listingID,
			)

			results <- err
		}()
	}

	close(start)

	successCount := 0
	alreadyPublishedCount := 0

	for i := 0; i < 2; i++ {
		err := <-results

		switch {
		case err == nil:
			successCount++

		case errors.Is(
			err,
			ErrAlreadyPublished,
		):
			alreadyPublishedCount++

		default:
			t.Fatalf(
				"unexpected publish result: %v",
				err,
			)
		}
	}

	if successCount != 1 {
		t.Fatalf(
			"expected exactly 1 successful publish, got %d",
			successCount,
		)
	}

	if alreadyPublishedCount != 1 {
		t.Fatalf(
			"expected exactly 1 already-published result, got %d",
			alreadyPublishedCount,
		)
	}

	status, err := store.GetListingStatus(
		ctx,
		listingID,
	)
	if err != nil {
		t.Fatalf(
			"get listing status: %v",
			err,
		)
	}

	if status != "PUBLISHED" {
		t.Fatalf(
			"expected listing status PUBLISHED, got %q",
			status,
		)
	}

	var publicationCount int

	err = pool.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM listing_publications
			WHERE listing_id = $1
		`,
		listingID,
	).Scan(&publicationCount)
	if err != nil {
		t.Fatalf(
			"count listing publications: %v",
			err,
		)
	}

	if publicationCount != 1 {
		t.Fatalf(
			"expected exactly 1 publication row, got %d",
			publicationCount,
		)
	}
}
