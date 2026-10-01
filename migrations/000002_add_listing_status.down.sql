BEGIN;

ALTER TABLE listings
DROP CONSTRAINT listings_status_check;

ALTER TABLE listings
DROP COLUMN status;

COMMIT;