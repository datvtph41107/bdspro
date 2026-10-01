BEGIN;

ALTER TABLE listings
ADD COLUMN status text NOT NULL DEFAULT 'DRAFT';

ALTER TABLE listings
ADD CONSTRAINT listings_status_check
CHECK (status IN ('DRAFT', 'PUBLISHED'));

COMMIT;