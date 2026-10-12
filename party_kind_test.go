package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestPartyKindCannotBeReclassified(t *testing.T) {
	for _, tt := range []struct {
		original string
		target   string
	}{
		{original: "PERSON", target: "ORGANIZATION"},
		{original: "ORGANIZATION", target: "PERSON"},
	} {
		t.Run(tt.original+"To"+tt.target, func(t *testing.T) {
			db := coreDB(t)
			ctx := context.Background()
			tx, err := db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)

			var id int64
			if err := tx.QueryRow(ctx, "INSERT INTO parties(kind) VALUES($1) RETURNING id", tt.original).Scan(&id); err != nil {
				t.Fatal(err)
			}
			if tt.original == "PERSON" {
				_, err = tx.Exec(ctx, "INSERT INTO persons(party_id) VALUES($1)", id)
			} else {
				_, err = tx.Exec(ctx, "INSERT INTO organizations(party_id,name) VALUES($1,'Original organization')", id)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("valid Party + subtype failed: %v", err)
			}

			// A no-op kind update must remain valid.
			if _, err := db.Exec(ctx, "UPDATE parties SET kind=kind WHERE id=$1", id); err != nil {
				t.Fatalf("unchanged kind was rejected: %v", err)
			}

			// Deleting the original subtype first used to allow reclassification:
			// the old C01 trigger checked only the state at COMMIT.
			tx2, err := db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx2.Rollback(ctx)

			if tt.original == "PERSON" {
				_, err = tx2.Exec(ctx, "DELETE FROM persons WHERE party_id=$1", id)
			} else {
				_, err = tx2.Exec(ctx, "DELETE FROM organizations WHERE party_id=$1", id)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = tx2.Exec(ctx, "UPDATE parties SET kind=$2 WHERE id=$1", id, tt.target)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
				t.Fatalf("reclassification should fail with 23514, got %v", err)
			}
			if err := tx2.Rollback(ctx); err != nil {
				t.Fatal(err)
			}

			var kind string
			var personCount, organizationCount int
			if err := db.QueryRow(ctx, `SELECT p.kind,
				(SELECT count(*) FROM persons WHERE party_id=p.id),
				(SELECT count(*) FROM organizations WHERE party_id=p.id)
				FROM parties p WHERE p.id=$1`, id).
				Scan(&kind, &personCount, &organizationCount); err != nil {
				t.Fatal(err)
			}
			if kind != tt.original || personCount+organizationCount != 1 {
				t.Fatalf("rollback did not preserve identity: kind=%s persons=%d organizations=%d", kind, personCount, organizationCount)
			}
			if tt.original == "PERSON" && personCount != 1 || tt.original == "ORGANIZATION" && organizationCount != 1 {
				t.Fatalf("original subtype lost: persons=%d organizations=%d", personCount, organizationCount)
			}
		})
	}
}

func TestPartyKindGuardWithCompetingUpdates(t *testing.T) {
	db := coreDB(t)
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var partyID int64
	if err := tx.QueryRow(ctx, "INSERT INTO parties(kind) VALUES('PERSON') RETURNING id").Scan(&partyID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO persons(party_id) VALUES($1)", partyID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	holder, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	if _, err := holder.Exec(ctx, "UPDATE parties SET kind=kind WHERE id=$1", partyID); err != nil {
		t.Fatal(err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		_, updateErr := db.Exec(waitCtx, "UPDATE parties SET kind='ORGANIZATION' WHERE id=$1", partyID)
		result <- updateErr
	}()
	<-started
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var pgErr *pgconn.PgError
	if err := <-result; !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("competing reclassification should fail with 23514, got %v", err)
	}
	var kind string
	if err := db.QueryRow(ctx, "SELECT kind FROM parties WHERE id=$1", partyID).Scan(&kind); err != nil {
		t.Fatal(err)
	}
	if kind != "PERSON" {
		t.Fatalf("identity kind changed under contention: %s", kind)
	}
}
