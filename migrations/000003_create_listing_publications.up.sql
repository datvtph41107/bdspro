BEGIN;

CREATE TABLE listing_publications (
    listing_id bigint PRIMARY KEY
        REFERENCES listings(id)
);

INSERT INTO listing_publications (listing_id)
SELECT id
FROM listings
WHERE status = 'PUBLISHED';

COMMIT;