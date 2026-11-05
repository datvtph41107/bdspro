-- Destructive: back up account access history before rolling back this migration.
DROP TABLE IF EXISTS account_connection_histories;
