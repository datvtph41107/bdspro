ALTER TABLE listings
DROP CONSTRAINT listings_price_non_negative;

ALTER TABLE listings
DROP COLUMN price;