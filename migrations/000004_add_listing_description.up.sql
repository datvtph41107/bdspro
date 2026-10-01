BEGIN;

ALTER TABLE listings
ADD COLUMN description text;

ALTER TABLE listings
ADD CONSTRAINT listings_description_nonblank
CHECK (
    description IS NULL
    OR btrim(description) <> ''
);

COMMIT;