package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type currentProfileStoreFake struct {
	db.Store
	profile        db.GetAccountProfileRow
	err            error
	calls          int
	profileUpdate  db.UpdatePersonProfileParams
	settingsUpdate db.UpdateAccountSettingsParams
	sessionID      uuid.UUID
	sessionBlocked bool
	sessionErr     error
	lockCalls      int
	txCalls        int
	commitErr      error
}

func (s *currentProfileStoreFake) GetAccountProfile(_ context.Context, id int64) (db.GetAccountProfileRow, error) {
	s.calls++
	if s.err != nil {
		return db.GetAccountProfileRow{}, s.err
	}
	if id != s.profile.AccountID {
		return db.GetAccountProfileRow{}, pgx.ErrNoRows
	}
	return s.profile, nil
}

func (s *currentProfileStoreFake) UpdatePersonProfile(_ context.Context, arg db.UpdatePersonProfileParams) (int64, error) {
	s.profileUpdate = arg
	if arg.AccountID != s.profile.AccountID {
		return 0, nil
	}
	if arg.DisplayName.Valid {
		s.profile.DisplayName = arg.DisplayName.String
	}
	if arg.FirstName.Valid {
		s.profile.FirstName = arg.FirstName.String
	}
	if arg.LastName.Valid {
		s.profile.LastName = arg.LastName.String
	}
	return 1, nil
}

func (s *currentProfileStoreFake) UpdateAccountSettings(_ context.Context, arg db.UpdateAccountSettingsParams) (int64, error) {
	s.settingsUpdate = arg
	if arg.AccountID != s.profile.AccountID {
		return 0, nil
	}
	if arg.Locale.Valid {
		s.profile.Locale = arg.Locale.String
	}
	if arg.Timezone.Valid {
		s.profile.Timezone = arg.Timezone.String
	}
	if arg.Currency.Valid {
		s.profile.Currency = arg.Currency.String
	}
	return 1, nil
}

// Recording SQLC DBTX keeps the HTTP test faithful to the transactional
// service: it does not bypass the acting-session lookup or emulate a second
// non-transactional profile read.
type profileHTTPDBTX struct{ store *currentProfileStoreFake }

type profileHTTPRow struct {
	profile db.GetAccountProfileRow
	session uuid.UUID
	err     error
}

func (r profileHTTPRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) == 1 {
		*dest[0].(*uuid.UUID) = r.session
		return nil
	}
	if len(dest) != 14 {
		return errors.New("invalid profile query result")
	}
	p := r.profile
	*dest[0].(*int64) = p.AccountID
	*dest[1].(*int64) = p.PersonID
	*dest[2].(*string) = p.Email
	*dest[3].(*bool) = p.IsEmailVerified
	*dest[4].(*time.Time) = p.CreatedAt
	*dest[5].(*string) = p.DisplayName
	*dest[6].(*string) = p.FirstName
	*dest[7].(*string) = p.LastName
	*dest[8].(*string) = p.Locale
	*dest[9].(*string) = p.Timezone
	*dest[10].(*string) = p.Currency
	*dest[11].(*bool) = p.HasPassword
	*dest[12].(*bool) = p.HasPhone
	*dest[13].(*bool) = p.HasOauth2
	return nil
}

func (q profileHTTPDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "-- name: LockActingSession"):
		q.store.lockCalls++
		if q.store.sessionErr != nil {
			return profileHTTPRow{err: q.store.sessionErr}
		}
		if args[0] != q.store.sessionID || args[1] != q.store.profile.AccountID || q.store.sessionBlocked {
			return profileHTTPRow{err: pgx.ErrNoRows}
		}
		return profileHTTPRow{session: q.store.sessionID}
	case strings.Contains(sql, "-- name: GetAccountProfile"):
		p, err := q.store.GetAccountProfile(ctx, args[0].(int64))
		return profileHTTPRow{profile: p, err: err}
	default:
		return profileHTTPRow{err: errors.New("unexpected SQLC profile query")}
	}
}

func (q profileHTTPDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	var affected int64
	var err error
	switch {
	case strings.Contains(sql, "-- name: UpdatePersonProfile"):
		affected, err = q.store.UpdatePersonProfile(ctx, db.UpdatePersonProfileParams{
			DisplayName: args[0].(pgtype.Text),
			FirstName: args[1].(pgtype.Text),
			LastName: args[2].(pgtype.Text),
			AccountID: args[3].(int64),
		})
	case strings.Contains(sql, "-- name: UpdateAccountSettings"):
		affected, err = q.store.UpdateAccountSettings(ctx, db.UpdateAccountSettingsParams{
			Locale: args[0].(pgtype.Text),
			Timezone: args[1].(pgtype.Text),
			Currency: args[2].(pgtype.Text),
			AccountID: args[3].(int64),
		})
	default:
		return pgconn.CommandTag{}, errors.New("unexpected SQLC profile update")
	}
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	if affected == 1 {
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	return pgconn.NewCommandTag("UPDATE 0"), nil
}

func (profileHTTPDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected profile list query")
}

func (s *currentProfileStoreFake) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	s.txCalls++
	// Stage changes in a copy to model transaction rollback and commit.
	staged := *s
	err := fn(db.New(profileHTTPDBTX{store: &staged}))
	// Attempted reads/locks are observable even if the DB transaction
	// rolls back; data changes are published only after commit.
	s.lockCalls = staged.lockCalls
	s.calls = staged.calls
	if err != nil {
		return err
	}
	if s.commitErr != nil {
		return s.commitErr
	}
	s.profile = staged.profile
	s.profileUpdate = staged.profileUpdate
	s.settingsUpdate = staged.settingsUpdate
	return nil
}

func TestMeUsesVerifiedAccountAndDoesNotLeakDriverErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	access, _, err := maker.CreateSessionAccessToken(42, time.Minute, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	store := &currentProfileStoreFake{sessionID: sessionID, profile: db.GetAccountProfileRow{
		AccountID: 42, PersonID: 67, Email: "verified@example.test",
		IsEmailVerified: true, CreatedAt: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC),
		DisplayName: "Verified", FirstName: "Lin", LastName: "Ng",
		Locale: "vi", Timezone: "Asia/Ho_Chi_Minh", Currency: "VND",
		HasPassword: true, HasPhone: true, HasOauth2: false,
	}}
	h := NewProfileHandler(store)
	r := gin.New()
	r.GET("/users/me", appctx.Authenticate(maker), appctx.Wrap(h.Show))
	r.PATCH("/users/me/profile", appctx.Authenticate(maker), appctx.Wrap(h.Update))
	r.GET("/users/me/settings", appctx.Authenticate(maker), appctx.Wrap(h.Settings))
	r.PATCH("/users/me/settings", appctx.Authenticate(maker), appctx.Wrap(h.UpdateSettings))

	request := func(method, path, body, bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := request("GET", "/users/me", "", ""); w.Code != http.StatusUnauthorized || store.calls != 0 {
		t.Fatalf("unauthenticated call reached data store: status=%d calls=%d", w.Code, store.calls)
	}
	w := request("GET", "/users/me", "", access)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /users/me: %d %s", w.Code, w.Body.String())
	}
	var got struct {
		AccountID int64 `json:"account_id"`
		PersonID  int64 `json:"person_id"`
		Profile   struct {
			DisplayName string `json:"display_name"`
		} `json:"profile"`
		Security struct {
			HasPassword bool `json:"has_password"`
			HasOauth2   bool `json:"has_oauth2"`
		} `json:"security"`
		Settings struct {
			Locale string `json:"locale"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.AccountID != 42 || got.PersonID != 67 || got.Profile.DisplayName != "Verified" || !got.Security.HasPassword || got.Security.HasOauth2 || got.Settings.Locale != "vi" {
		t.Fatalf("unexpected account profile: %+v", got)
	}
	w = request("PATCH", "/users/me/profile", `{"display_name":"","account_id":1000,"person_id":999}`, access)
	if w.Code != http.StatusOK || store.profileUpdate.AccountID != 42 || store.profile.DisplayName != "" || store.profile.FirstName != "Lin" {
		t.Fatalf("partial patch or identity scope violation: status=%d body=%s arg=%+v", w.Code, w.Body.String(), store.profileUpdate)
	}
	w = request("PATCH", "/users/me/profile", `{}`, access)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "account.invalid_profile") {
		t.Fatalf("empty profile patch accepted: status=%d body=%s", w.Code, w.Body.String())
	}
	w = request("PATCH", "/users/me/settings", `{"timezone":"UTC","account_id":1234}`, access)
	if w.Code != http.StatusOK || store.settingsUpdate.AccountID != 42 || store.profile.Locale != "vi" || store.profile.Timezone != "UTC" {
		t.Fatalf("settings patch or identity scope violation: status=%d body=%s", w.Code, w.Body.String())
	}
	w = request("GET", "/users/me/settings", "", access)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"timezone":"UTC"`) {
		t.Fatalf("current settings: %d %s", w.Code, w.Body.String())
	}
	w = request("PATCH", "/users/me/settings", `{"currency":"UNSUPPORTED"}`, access)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "account.invalid_settings") {
		t.Fatalf("invalid currency accepted: %d %s", w.Code, w.Body.String())
	}
	// A signed JWT is not sufficient to authorize a PATCH after session
	// revocation. Even a request which passed HTTP Authenticate must fail
	// at the transaction boundary without modifying profile or settings.
	stale, _, err := maker.CreateSessionAccessToken(42, time.Minute, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	before := store.profile
	w = request("PATCH", "/users/me/profile", `{"display_name":"hijacked"}`, stale)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "auth.session_unavailable") || store.profile != before {
		t.Fatalf("stale session changed profile: status=%d body=%s", w.Code, w.Body.String())
	}
	w = request("PATCH", "/users/me/settings", `{"locale":"en"}`, stale)
	if w.Code != http.StatusUnauthorized || store.profile != before {
		t.Fatalf("stale session changed settings: status=%d body=%s", w.Code, w.Body.String())
	}
	store.sessionBlocked = true
	w = request("PATCH", "/users/me/profile", `{"first_name":"changed"}`, access)
	if w.Code != http.StatusUnauthorized || store.profile != before {
		t.Fatalf("revoked session changed profile: status=%d body=%s", w.Code, w.Body.String())
	}
	store.sessionBlocked = false
	store.commitErr = errors.New("private: failed commit")
	w = request("PATCH", "/users/me/settings", `{"locale":"en"}`, access)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "private: failed commit") || store.profile != before {
		t.Fatalf("failed commit exposed or applied profile: status=%d body=%s", w.Code, w.Body.String())
	}
	store.commitErr = nil
	if store.lockCalls < 3 || store.txCalls < 4 {
		t.Fatalf("PATCH did not use transaction session checks: locks=%d tx=%d", store.lockCalls, store.txCalls)
	}

	store.err = pgx.ErrNoRows
	w = request("GET", "/users/me", "", access)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "auth.account_unavailable") {
		t.Fatalf("deleted account returned profile: %d %s", w.Code, w.Body.String())
	}
	store.err = errors.New("private-password=db-secret")
	w = request("GET", "/users/me", "", access)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "db-secret") {
		t.Fatalf("driver leak: %d %s", w.Code, w.Body.String())
	}
}
