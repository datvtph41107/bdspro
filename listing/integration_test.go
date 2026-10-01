//go:build integration

package listing

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func openIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}

	db, err := pgxpool.New(
		context.Background(),
		databaseURL,
	)
	if err != nil {
		t.Fatalf("create test database pool: %v", err)
	}

	t.Cleanup(db.Close)

	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	return db
}

func resetIntegrationDB(
	t *testing.T,
	db *pgxpool.Pool,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`
			TRUNCATE
				listing_publications,
				listings
			RESTART IDENTITY
			CASCADE
		`,
	)
	if err != nil {
		t.Fatalf("reset test database: %v", err)
	}
}

func TestPublishIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)

	ctx := context.Background()

	created, err := Create(
		ctx,
		db,
		"Integration publish proof",
	)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}

	_, err = UpdateDescription(
		ctx,
		db,
		created.ID,
		"Complete description",
	)
	if err != nil {
		t.Fatalf("update description: %v", err)
	}

	published, err := Publish(
		ctx,
		db,
		created.ID,
	)
	if err != nil {
		t.Fatalf("publish listing: %v", err)
	}

	if published.Status != "PUBLISHED" {
		t.Fatalf(
			"status = %q, want PUBLISHED",
			published.Status,
		)
	}

	var publicationCount int

	err = db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM listing_publications
			WHERE listing_id = $1
		`,
		created.ID,
	).Scan(&publicationCount)
	if err != nil {
		t.Fatalf("count publications: %v", err)
	}

	if publicationCount != 1 {
		t.Fatalf(
			"publication count = %d, want 1",
			publicationCount,
		)
	}

	_, err = Publish(
		ctx,
		db,
		created.ID,
	)

	if !errors.Is(err, ErrAlreadyPublished) {
		t.Fatalf(
			"second publish error = %v, want ErrAlreadyPublished",
			err,
		)
	}
}

func TestPublishRollsBackWhenPublicationInsertFails(
	t *testing.T,
) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)

	ctx := context.Background()

	created, err := Create(
		ctx,
		db,
		"Rollback proof",
	)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}

	_, err = UpdateDescription(
		ctx,
		db,
		created.ID,
		"Complete description",
	)
	if err != nil {
		t.Fatalf("update description: %v", err)
	}

	_, err = db.Exec(
		ctx,
		`
			ALTER TABLE listing_publications
			ADD CONSTRAINT integration_transaction_probe
			CHECK (false)
		`,
	)
	if err != nil {
		t.Fatalf("add failure constraint: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			context.Background(),
			`
				ALTER TABLE listing_publications
				DROP CONSTRAINT IF EXISTS
				integration_transaction_probe
			`,
		)
	})

	_, err = Publish(
		ctx,
		db,
		created.ID,
	)
	if err == nil {
		t.Fatal("Publish succeeded, want failure")
	}

	var status string

	err = db.QueryRow(
		ctx,
		`
			SELECT status
			FROM listings
			WHERE id = $1
		`,
		created.ID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("read listing status: %v", err)
	}

	if status != "DRAFT" {
		t.Fatalf(
			"status = %q, want DRAFT after rollback",
			status,
		)
	}

	var publicationCount int

	err = db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM listing_publications
			WHERE listing_id = $1
		`,
		created.ID,
	).Scan(&publicationCount)
	if err != nil {
		t.Fatalf("count publications: %v", err)
	}

	if publicationCount != 0 {
		t.Fatalf(
			"publication count = %d, want 0",
			publicationCount,
		)
	}
}

func TestConcurrentPublishHasExactlyOneWinner(
	t *testing.T,
) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)

	ctx := context.Background()

	created, err := Create(
		ctx,
		db,
		"Concurrent publish proof",
	)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}

	_, err = UpdateDescription(
		ctx,
		db,
		created.ID,
		"Complete description",
	)
	if err != nil {
		t.Fatalf("update description: %v", err)
	}

	results := make(chan error, 2)

	for range 2 {
		go func() {
			_, err := Publish(
				context.Background(),
				db,
				created.ID,
			)

			results <- err
		}()
	}

	err1 := <-results
	err2 := <-results

	successes := 0
	conflicts := 0

	for _, err := range []error{err1, err2} {
		switch {
		case err == nil:
			successes++

		case errors.Is(err, ErrAlreadyPublished):
			conflicts++

		default:
			t.Fatalf(
				"unexpected concurrent publish error: %v",
				err,
			)
		}
	}

	if successes != 1 {
		t.Fatalf(
			"successful publishes = %d, want 1",
			successes,
		)
	}

	if conflicts != 1 {
		t.Fatalf(
			"publish conflicts = %d, want 1",
			conflicts,
		)
	}

	var publicationCount int

	err = db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM listing_publications
			WHERE listing_id = $1
		`,
		created.ID,
	).Scan(&publicationCount)
	if err != nil {
		t.Fatalf("count publications: %v", err)
	}

	if publicationCount != 1 {
		t.Fatalf(
			"publication count = %d, want 1",
			publicationCount,
		)
	}
}
