-- name: CreateParty :one
INSERT INTO parties DEFAULT VALUES RETURNING id;

-- name: CreatePerson :exec
INSERT INTO persons (party_id) VALUES ($1);
