package convert

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/google/uuid"
)

func TestLoginConversionPreservesTokenContractOnlyInPublicFields(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	id := uuid.New()
	login := auth_service.LoginResult{
		Account:     &account_model.Account{ID: 23, PersonID: 31, Email: "person@example.test", CreatedAt: now},
		Session:     &auth_model.Session{ID: id, AccountID: 23, CreatedAt: now, ExpiresAt: now.Add(time.Hour)},
		AccessToken: "jwt-access", AccessExpiresAt: now.Add(15 * time.Minute),
		RefreshToken: "jwt-refresh", RefreshExpiresAt: now.Add(time.Hour),
	}
	dto := ToLogin(login)
	if dto.SessionID != id.String() || dto.User.AccountID != 23 ||
		dto.User.PersonID != 31 || dto.AccessToken != login.AccessToken ||
		dto.RefreshToken != login.RefreshToken ||
		dto.AccessTokenExpiresAt != login.AccessExpiresAt ||
		dto.RefreshTokenExpiresAt != login.RefreshExpiresAt {
		t.Fatalf("public login response changed: %+v", dto)
	}
	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "RefreshTokenDigest") ||
		strings.Contains(string(encoded), "IsBlocked") ||
		!strings.Contains(string(encoded), `"refresh_token":"jwt-refresh"`) {
		t.Fatalf("public login response leaked session authority: %s", encoded)
	}
}

func TestListSessionAndConnectionWireContracts(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	id := uuid.New()
	sessions := ToSessions([]auth_model.SessionInfo{{
		ID: id, UserAgent: "browser", ClientIP: "203.0.113.3",
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}})
	if len(sessions) != 1 || sessions[0].ID != id || sessions[0].ClientIP != "203.0.113.3" {
		t.Fatalf("session DTO changed: %+v", sessions)
	}
	if ToSessions(nil) == nil || ToConnections(nil) == nil {
		t.Fatal("API must continue to return [] for empty lists")
	}
	encoded, err := json.Marshal(sessions[0])
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) != 5 || payload["id"] != id.String() ||
		payload["user_agent"] != "browser" || payload["client_ip"] != "203.0.113.3" {
		t.Fatalf("session list API fields changed: %s", encoded)
	}
	connections := ToConnections([]account_model.Connection{{
		ID: 1, SessionID: id, Action: "login", IPAddress: "203.0.113.3",
		UserAgent: "browser", CreatedAt: now,
	}})
	if len(connections) != 1 || connections[0].SessionID != id.String() ||
		connections[0].IPAddress != "203.0.113.3" {
		t.Fatalf("connection DTO changed: %+v", connections)
	}
}
