package db

import "context"

const createPersonParty = `INSERT INTO parties DEFAULT VALUES RETURNING id`
func (q *Queries) CreatePersonParty(ctx context.Context) (int64, error) {
    var id int64
    err := q.db.QueryRow(ctx, createPersonParty).Scan(&id)
    return id, err
}

const createPerson = `INSERT INTO persons (party_id) VALUES ($1)`
func (q *Queries) CreatePerson(ctx context.Context, partyID int64) error {
    _, err := q.db.Exec(ctx, createPerson, partyID)
    return err
}
