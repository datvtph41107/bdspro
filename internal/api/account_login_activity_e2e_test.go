package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/util"
	"github.com/google/uuid"
)

func TestPasswordLoginActivityIsAccountScoped(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	hash, err := util.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	store := db.NewStore(pool)
	firstEmail := uuid.NewString() + "@example.test"
	first, err := store.RegisterTx(ctx, db.RegisterTxParams{Email: firstEmail, PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	secondEmail := uuid.NewString() + "@example.test"
	second, err := store.RegisterTx(ctx, db.RegisterTxParams{Email: secondEmail, PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	status, failed := authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{
		"email": firstEmail, "password": "incorrect",
	}, "")
	if status != http.StatusUnauthorized || failed["code"] != "auth.invalid_credentials" {
		t.Fatalf("password rejection leaked or failed: %d %v", status, failed)
	}
	status, valid := authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{
		"email": firstEmail, "password": "correct-password",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("login unavailable: %d %v", status, valid)
	}
	token := valid["access_token"].(string)
	status, history := authAcceptanceCall(t, server, "/users/me/activity", http.MethodGet, nil, token)
	if status != http.StatusOK {
		t.Fatalf("cannot list activity: %d %v", status, history)
	}
	events := history["events"].([]any)
	if len(events) != 2 {
		t.Fatalf("password history = %v", events)
	}
	if events[0].(map[string]any)["outcome"] != "succeeded" ||
		events[1].(map[string]any)["outcome"] != "rejected" {
		t.Fatalf("activity does not preserve outcomes: %v", events)
	}
	for _, event := range events {
		fields := event.(map[string]any)
		for _, forbidden := range []string{"account_id", "password", "refresh_token", "email"} {
			if _, exists := fields[forbidden]; exists {
				t.Fatalf("activity leaked %s: %v", forbidden, fields)
			}
		}
	}
	status, _ = authAcceptanceCall(t, server, "/users/me/activity", http.MethodGet, nil, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous activity request: %d", status)
	}
	status, secondLogin := authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{
		"email": secondEmail, "password": "correct-password",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("second account login unavailable: %d %v", status, secondLogin)
	}
	secondToken := secondLogin["access_token"].(string)
	// A forged account_id query parameter must never override the bearer identity.
	status, otherHistory := authAcceptanceCall(t, server,
		"/users/me/activity?account_id="+fmt.Sprint(first.ID), http.MethodGet, nil, secondToken)
	if status != http.StatusOK {
		t.Fatalf("other activity request failed: %d %v", status, otherHistory)
	}
	otherEvents := otherHistory["events"].([]any)
	if len(otherEvents) != 1 || otherEvents[0].(map[string]any)["outcome"] != "succeeded" {
		t.Fatalf("account histories were mixed: %v", otherHistory)
	}
	var wrongAccountVisible int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM account_login_events WHERE account_id=$1", second.ID).Scan(&wrongAccountVisible)
	if err != nil || wrongAccountVisible != 1 {
		t.Fatalf("second account did not retain only its own login: count=%d error=%v", wrongAccountVisible, err)
	}
	var ownCount int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM account_login_events WHERE account_id=$1", first.ID).Scan(&ownCount)
	if err != nil || ownCount != 2 {
		t.Fatalf("password attempts not durably persisted: %d %v", ownCount, err)
	}
}

func TestLoginEventRetentionQuery(t *testing.T) {
	ctx, pool, _, _ := authAcceptanceServer(t)
	store := db.NewStore(pool)
	// The retention SQL is a maintenance operation, not a background job.
	var oldID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO account_login_events (outcome, client_ip, user_agent, occurred_at)
		VALUES ('rejected', '192.0.2.1', 'retention-test', now() - interval '91 days')
		RETURNING id
	`).Scan(&oldID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteExpiredLoginEvents(context.Background()); err != nil {
		t.Fatal(err)
	}
	var remains int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM account_login_events WHERE id=$1", oldID).Scan(&remains); err != nil {
		t.Fatal(err)
	}
	if remains != 0 {
		t.Fatalf("expired security event was not deleted")
	}
}
