package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/datvtph41107/bdspro/modules/optional"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrProfileInvalid = util.WithPublic(util.ErrInvalidArgument, "invalid profile or settings patch", "validation_failed", "invalid profile or settings fields")

type profileSettings struct {
	Profile  authmodel.Profile
	Settings authmodel.AccountSettings
}

func readProfileSettings(ctx context.Context, accountID int64) (profileSettings, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return profileSettings{}, err
	}
	row, err := q.GetAccountProfile(ctx, accountID)
	if err == nil {
		return profileSettings{
			Profile:  authmodel.Profile{DisplayName: row.DisplayName, FirstName: row.FirstName, LastName: row.LastName},
			Settings: authmodel.AccountSettings{Locale: row.Locale, Timezone: row.Timezone, Currency: row.Currency},
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return profileSettings{}, err
	}

	engine, err := db.GetEngine(ctx)
	if err != nil {
		return profileSettings{}, err
	}
	var result profileSettings
	err = engine.QueryRow(ctx, `SELECT COALESCE(p.display_name,''), COALESCE(p.first_name,''), COALESCE(p.last_name,''),
COALESCE(s.locale,'vi'), COALESCE(s.timezone,'Asia/Ho_Chi_Minh'), COALESCE(s.currency,'VND')
FROM accounts a LEFT JOIN person_profiles p ON p.person_id=a.person_id
LEFT JOIN account_settings s ON s.account_id=a.id WHERE a.id=$1`, accountID).Scan(
		&result.Profile.DisplayName, &result.Profile.FirstName, &result.Profile.LastName,
		&result.Settings.Locale, &result.Settings.Timezone, &result.Settings.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return profileSettings{}, util.ErrUnauthenticated
	}
	return result, err
}

func GetProfile(ctx context.Context, accountID int64) (authmodel.Profile, error) {
	result, err := readProfileSettings(ctx, accountID)
	return result.Profile, err
}

func GetAccountSettings(ctx context.Context, accountID int64) (authmodel.AccountSettings, error) {
	result, err := readProfileSettings(ctx, accountID)
	return result.Settings, err
}

func profileField(value optional.Option[string], required bool) (pgtype.Text, error) {
	if !value.Has() {
		return pgtype.Text{}, nil
	}
	s := strings.TrimSpace(value.Value())
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) > 100 || required && s == "" ||
		strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return pgtype.Text{}, ErrProfileInvalid
	}
	return pgtype.Text{String: s, Valid: true}, nil
}

func settingsField(value optional.Option[string], allowed ...string) (pgtype.Text, error) {
	if !value.Has() {
		return pgtype.Text{}, nil
	}
	for _, candidate := range allowed {
		if value.Value() == candidate {
			return pgtype.Text{String: candidate, Valid: true}, nil
		}
	}
	return pgtype.Text{}, ErrProfileInvalid
}

func ensureProfileSettings(ctx context.Context, accountID int64) error {
	engine, err := db.GetEngine(ctx)
	if err != nil {
		return err
	}
	if _, err := engine.Exec(ctx, "INSERT INTO person_profiles(person_id) SELECT person_id FROM accounts WHERE id=$1 ON CONFLICT (person_id) DO NOTHING", accountID); err != nil {
		return err
	}
	_, err = engine.Exec(ctx, "INSERT INTO account_settings(account_id) VALUES ($1) ON CONFLICT (account_id) DO NOTHING", accountID)
	return err
}

func UpdateProfile(ctx context.Context, actor authmodel.Actor, display, first, last optional.Option[string]) (authmodel.Profile, error) {
	d, err := profileField(display, true)
	if err != nil {
		return authmodel.Profile{}, err
	}
	f, err := profileField(first, false)
	if err != nil {
		return authmodel.Profile{}, err
	}
	l, err := profileField(last, false)
	if err != nil {
		return authmodel.Profile{}, err
	}
	if !d.Valid && !f.Valid && !l.Valid {
		return authmodel.Profile{}, ErrProfileInvalid
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (authmodel.Profile, error) {
		q, err := LockActorForMutation(txCtx, actor)
		if err != nil {
			return authmodel.Profile{}, err
		}
		if err := ensureProfileSettings(txCtx, actor.AccountID); err != nil {
			return authmodel.Profile{}, err
		}
		n, err := q.UpdatePersonProfile(txCtx, sqlc.UpdatePersonProfileParams{
			AccountID: actor.AccountID, DisplayName: d, FirstName: f, LastName: l,
		})
		if err != nil {
			return authmodel.Profile{}, err
		}
		if n != 1 {
			return authmodel.Profile{}, util.ErrUnauthenticated
		}
		return GetProfile(txCtx, actor.AccountID)
	})
}

func UpdateSettings(ctx context.Context, actor authmodel.Actor, locale, timezone, currency optional.Option[string]) (authmodel.AccountSettings, error) {
	l, err := settingsField(locale, "vi", "en")
	if err != nil {
		return authmodel.AccountSettings{}, err
	}
	z, err := settingsField(timezone, "Asia/Ho_Chi_Minh", "UTC")
	if err != nil {
		return authmodel.AccountSettings{}, err
	}
	c, err := settingsField(currency, "VND", "USD")
	if err != nil {
		return authmodel.AccountSettings{}, err
	}
	if !l.Valid && !z.Valid && !c.Valid {
		return authmodel.AccountSettings{}, ErrProfileInvalid
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (authmodel.AccountSettings, error) {
		q, err := LockActorForMutation(txCtx, actor)
		if err != nil {
			return authmodel.AccountSettings{}, err
		}
		if err := ensureProfileSettings(txCtx, actor.AccountID); err != nil {
			return authmodel.AccountSettings{}, err
		}
		n, err := q.UpdateAccountSettings(txCtx, sqlc.UpdateAccountSettingsParams{
			AccountID: actor.AccountID, Locale: l, Timezone: z, Currency: c,
		})
		if err != nil {
			return authmodel.AccountSettings{}, err
		}
		if n != 1 {
			return authmodel.AccountSettings{}, util.ErrUnauthenticated
		}
		return GetAccountSettings(txCtx, actor.AccountID)
	})
}

func GetCredentials(ctx context.Context, accountID int64) (authmodel.CredentialSummary, error) {
	u, err := GetCurrentUser(ctx, accountID)
	if err != nil {
		return authmodel.CredentialSummary{}, err
	}
	summary := authmodel.CredentialSummary{Credentials: u.Credentials}
	if u.Email != nil {
		local, domain, ok := strings.Cut(*u.Email, "@")
		masked := "***"
		if ok && local != "" {
			r, _ := utf8.DecodeRuneInString(local)
			masked = string(r) + "***@" + domain
		}
		summary.Email = &masked
	}
	if u.Phone != nil {
		masked := "***"
		if len(*u.Phone) > 4 {
			masked += (*u.Phone)[len(*u.Phone)-4:]
		}
		summary.Phone = &masked
	}
	return summary, nil
}

func ListConnections(ctx context.Context, accountID int64) ([]authmodel.Connection, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAccountConnectionHistories(ctx, accountID)
	if err != nil {
		return nil, err
	}
	items := make([]authmodel.Connection, 0, len(rows))
	for _, row := range rows {
		items = append(items, authmodel.Connection{
			ID: strconv.FormatInt(row.ID, 10), SessionID: row.SessionID, Action: row.Action,
			IPAddress: row.IpAddress, UserAgent: row.UserAgent, CreatedAt: row.CreatedAt,
		})
	}
	return items, nil
}
