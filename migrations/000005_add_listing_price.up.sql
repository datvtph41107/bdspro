ALTER TABLE listings
ADD COLUMN price BIGINT;

ALTER TABLE listings
ADD CONSTRAINT listings_price_non_negative
CHECK (price >=0);