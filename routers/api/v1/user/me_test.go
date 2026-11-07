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
	"github.com/jackc/pgx/v5"
)

type currentProfileStoreFake struct {
	db.Store
	profile        db.GetAccountProfileRow
	err            error
	calls          int
	profileUpdate  db.UpdatePersonProfileParams
	settingsUpdate db.UpdateAccountSettingsParams
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

func TestMeUsesVerifiedAccountAndDoesNotLeakDriverErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	access, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	store := &currentProfileStoreFake{profile: db.GetAccountProfileRow{
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
