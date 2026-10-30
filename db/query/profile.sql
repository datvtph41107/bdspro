-- name: CreatePersonProfile :exec
INSERT INTO person_profiles (person_id) VALUES ($1);

-- name: CreateAccountSettings :exec
INSERT INTO account_settings (account_id) VALUES ($1);

-- name: GetAccountProfile :one
SELECT a.id AS account_id, a.person_id, COALESCE(a.email, '')::text AS email,
       a.is_email_verified, a.created_at,
       p.display_name, p.first_name, p.last_name,
       s.locale, s.timezone, s.currency,
       EXISTS (SELECT 1 FROM password_credentials pw WHERE pw.account_id = a.id)::boolean AS has_password,
       EXISTS (SELECT 1 FROM phone_credentials ph WHERE ph.account_id = a.id)::boolean AS has_phone,
       EXISTS (SELECT 1 FROM google_identities gg WHERE gg.account_id = a.id)::boolean AS has_google
FROM accounts a
JOIN person_profiles p ON p.person_id = a.person_id
JOIN account_settings s ON s.account_id = a.id
WHERE a.id = $1
LIMIT 1;

-- name: UpdatePersonProfile :execrows
UPDATE person_profiles
SET display_name = sqlc.arg(display_name),
    first_name = sqlc.arg(first_name),
    last_name = sqlc.arg(last_name),
    updated_at = now()
WHERE person_id = (SELECT person_id FROM accounts WHERE id = sqlc.arg(account_id));

-- name: UpdateAccountSettings :execrows
UPDATE account_settings
SET locale = sqlc.arg(locale),
    timezone = sqlc.arg(timezone),
    currency = sqlc.arg(currency),
    updated_at = now()
WHERE account_id = sqlc.arg(account_id);
