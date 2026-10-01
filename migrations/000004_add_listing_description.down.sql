BEGIN;

ALTER TABLE listings
DROP CONSTRAINT listings_description_nonblank;

ALTER TABLE listings
DROP COLUMN description;

COMMIT;