-- name: CreateListing :one
INSERT INTO listings (
    title
) VALUES (
    sqlc.arg(title)
)
RETURNING
    id,
    title,
    status;


-- name: UpdateListingDescription :one
UPDATE listings
SET description = sqlc.arg(description)::text
WHERE id = sqlc.arg(id)
  AND status = 'DRAFT'
RETURNING
    id,
    title,
    description,
    status;


-- name: GetListingStatus :one
SELECT status
FROM listings
WHERE id = sqlc.arg(id);


-- name: MarkListingPublished :one
UPDATE listings
SET status = 'PUBLISHED'
WHERE id = sqlc.arg(id)
  AND status = 'DRAFT'
  AND description IS NOT NULL
RETURNING
    id,
    title,
    status;


-- name: CreateListingPublication :exec
INSERT INTO listing_publications (
    listing_id
) VALUES (
    sqlc.arg(listing_id)
);


-- name: GetListingForPublicationReadiness :one
SELECT
    id,
    title,
    description,
    status
FROM listings
WHERE id = sqlc.arg(id);