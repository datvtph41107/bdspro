-- name: CreateListing :one
INSERT INTO listings (
    title,
    description,
    price
) VALUES (
    $1,
    $2,
    $3
) RETURNING id, title, status, description, price;