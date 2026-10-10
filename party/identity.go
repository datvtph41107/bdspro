package party

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrOrganizationNameRequired is returned before a database transaction
// when the name used to create an unverified Organization is blank.
var ErrOrganizationNameRequired = errors.New("organization name is required")

// CreatePerson creates a Party and its Person subtype atomically.
// A Person does not need an Account. Identity proof/onboarding is a
// separate boundary: creating a row is not verifying a real-world identity.
func CreatePerson(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin create person: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var partyID int64
	err = tx.QueryRow(ctx,
		"INSERT INTO parties DEFAULT VALUES RETURNING id",
	).Scan(&partyID)
	if err != nil {
		return 0, fmt.Errorf("insert party: %w", err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO persons (party_id) VALUES ($1)", partyID,
	); err != nil {
		return 0, fmt.Errorf("insert person subtype: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit person identity: %w", err)
	}
	return partyID, nil
}

// CreateOrganization creates a Party and Organization subtype atomically.
// The supplied name is declarative and is NOT proof of legal existence,
// brand ownership or authority to represent the Organization.
func CreateOrganization(ctx context.Context, pool *pgxpool.Pool, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, ErrOrganizationNameRequired
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin create organization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var partyID int64
	err = tx.QueryRow(ctx,
		"INSERT INTO parties DEFAULT VALUES RETURNING id",
	).Scan(&partyID)
	if err != nil {
		return 0, fmt.Errorf("insert party: %w", err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO organizations (party_id, name) VALUES ($1, $2)",
		partyID, name,
	); err != nil {
		return 0, fmt.Errorf("insert organization subtype: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit organization identity: %w", err)
	}
	return partyID, nil
}
