CREATE TABLE account_settings (
    account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    locale text NOT NULL DEFAULT 'vi' CHECK (locale IN ('vi', 'en')),
    timezone text NOT NULL DEFAULT 'Asia/Ho_Chi_Minh' CHECK (timezone IN ('Asia/Ho_Chi_Minh', 'UTC')),
    currency text NOT NULL DEFAULT 'VND' CHECK (currency IN ('VND', 'USD')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
