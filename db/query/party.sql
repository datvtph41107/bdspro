-- Durable party identity; subtype writes occur within the service transaction.
-- name: CreatePersonParty :one
INSERT INTO parties (kind) VALUES ('PERSON') RETURNING id;

-- name: CreatePersonSubtype :exec
INSERT INTO persons (party_id) VALUES ($1);

-- name: CreateOrganizationParty :one
INSERT INTO parties (kind) VALUES ('ORGANIZATION') RETURNING id;

-- name: CreateOrganizationSubtype :exec
INSERT INTO organizations (party_id, name) VALUES ($1, $2);
