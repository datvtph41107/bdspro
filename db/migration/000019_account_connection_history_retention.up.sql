-- The account-scoped lookup index cannot efficiently support global expiry.
CREATE INDEX account_connection_histories_retention_idx
    ON account_connection_histories(created_at, id);
