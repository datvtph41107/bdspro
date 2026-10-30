package api

import (
	"net/http"

	"github.com/google/uuid"
	"testing"

	db "github.com/datvtph41107/bdspro/internal/database"
)

func TestAccountProfileAndSettingsHTTP(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	firstID, firstToken := organizationTestVerifiedAccount(t, ctx, pool)
	secondID, secondToken := organizationTestVerifiedAccount(t, ctx, pool)
	if firstID == secondID {
		t.Fatal("accounts should be distinct")
	}

	status, account := authAcceptanceCall(t, server, "/users/me", http.MethodGet, nil, firstToken)
	if status != http.StatusOK || int64(account["account_id"].(float64)) != firstID {
		t.Fatalf("wrong current account: %d %v", status, account)
	}
	profile := account["profile"].(map[string]any)
	security := account["security"].(map[string]any)
	settings := account["settings"].(map[string]any)
	if profile["display_name"] != "" || security["has_password"] != false ||
		security["has_google"] != true || settings["locale"] != "vi" {
		t.Fatalf("unexpected default profile: %v", account)
	}
	if _, present := account["password_hash"]; present {
		t.Fatal("response exposed credential")
	}

	status, account = authAcceptanceCall(t, server, "/users/me/profile", http.MethodPatch,
		map[string]string{"display_name": "Nguyen A", "first_name": "A", "last_name": "Nguyen"}, firstToken)
	if status != http.StatusOK || account["profile"].(map[string]any)["display_name"] != "Nguyen A" {
		t.Fatalf("cannot update owned profile: %d %v", status, account)
	}

	status, updated := authAcceptanceCall(t, server, "/users/me/settings", http.MethodPatch,
		map[string]string{"locale": "en", "timezone": "UTC", "currency": "USD"}, firstToken)
	if status != http.StatusOK || updated["timezone"] != "UTC" || updated["currency"] != "USD" {
		t.Fatalf("cannot update settings: %d %v", status, updated)
	}

	status, saved := authAcceptanceCall(t, server, "/users/me", http.MethodGet, nil, firstToken)
	if status != http.StatusOK || saved["profile"].(map[string]any)["first_name"] != "A" ||
		saved["settings"].(map[string]any)["locale"] != "en" {
		t.Fatalf("profile was not persisted: %d %v", status, saved)
	}
	status, other := authAcceptanceCall(t, server, "/users/me", http.MethodGet, nil, secondToken)
	if status != http.StatusOK || int64(other["account_id"].(float64)) != secondID ||
		other["profile"].(map[string]any)["display_name"] != "" {
		t.Fatalf("account data crossed boundaries: %d %v", status, other)
	}

	status, invalid := authAcceptanceCall(t, server, "/users/me/settings", http.MethodPatch,
		map[string]string{"locale": "xx", "timezone": "UTC", "currency": "USD"}, firstToken)
	if status != http.StatusBadRequest || invalid["code"] != "invalid_request" {
		t.Fatalf("invalid settings accepted: %d %v", status, invalid)
	}

	status, _ = authAcceptanceCall(t, server, "/users/me", http.MethodGet, nil, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous profile read: %d", status)
	}
	stored, err := db.NewStore(pool).GetAccountProfile(ctx, firstID)
	if err != nil || stored.DisplayName != "Nguyen A" || stored.Locale != "en" {
		t.Fatalf("SQLC profile does not match HTTP profile: %+v err=%v", stored, err)
	}
}

func TestPasswordRegistrationCreatesProfile(t *testing.T) {
	ctx, pool, _, _ := authAcceptanceServer(t)
	store := db.NewStore(pool)
	account, err := store.RegisterTx(ctx, db.RegisterTxParams{
		Email: "test-profile-" + uuid.NewString() + "@example.test",
		PasswordHash: "test-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.GetAccountProfile(ctx, account.ID)
	if err != nil || !profile.HasPassword || profile.HasPhone || profile.HasGoogle {
		t.Fatalf("incorrect password registration profile: %+v err=%v", profile, err)
	}
}
