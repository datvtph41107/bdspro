-- name: CreatePersonProfile :exec
INSERT INTO person_profiles (person_id) VALUES ($1);

-- name: CreateAccountSettings :exec
INSERT INTO account_settings (account_id) VALUES ($1);

-- name: GetAccountProfile :one
SELECT COALESCE(p.display_name, '')::text AS display_name,
       COALESCE(p.first_name, '')::text AS first_name,
       COALESCE(p.last_name, '')::text AS last_name,
       COALESCE(s.locale, 'vi')::text AS locale,
       COALESCE(s.timezone, 'Asia/Ho_Chi_Minh')::text AS timezone,
       COALESCE(s.currency, 'VND')::text AS currency
FROM accounts a
LEFT JOIN person_profiles p ON p.person_id = a.person_id
LEFT JOIN account_settings s ON s.account_id = a.id
WHERE a.id = $1;

-- name: EnsurePersonProfileForAccount :exec
INSERT INTO person_profiles (person_id)
SELECT person_id FROM accounts WHERE id = $1
ON CONFLICT (person_id) DO NOTHING;

-- name: EnsureAccountSettings :exec
INSERT INTO account_settings (account_id) VALUES ($1)
ON CONFLICT (account_id) DO NOTHING;

-- name: UpdatePersonProfile :execrows
UPDATE person_profiles
SET display_name = COALESCE(sqlc.narg(display_name)::text, display_name),
    first_name = COALESCE(sqlc.narg(first_name)::text, first_name),
    last_name = COALESCE(sqlc.narg(last_name)::text, last_name),
    updated_at = now()
WHERE person_id = (SELECT person_id FROM accounts WHERE id = sqlc.arg(account_id));

-- name: UpdateAccountSettings :execrows
UPDATE account_settings
SET locale = COALESCE(sqlc.narg(locale)::text, locale),
    timezone = COALESCE(sqlc.narg(timezone)::text, timezone),
    currency = COALESCE(sqlc.narg(currency)::text, currency),
    updated_at = now()
WHERE account_id = sqlc.arg(account_id);
