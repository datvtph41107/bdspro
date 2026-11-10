package convert

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/google/uuid"
)

func TestOAuthClientConversionPreservesPublicContract(t *testing.T) {
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	client := &auth_model.OAuthClient{
		ID: uuid.New(), OwnerAccountID: 42, Name: "Integration",
		RedirectURI: "https://client.example.test/callback",
		CreatedAt:   now,
	}
	dto := ToOAuthClient(client)
	if dto.ClientID != client.ID.String() || dto.Name != client.Name ||
		dto.RedirectURI != client.RedirectURI || dto.CreatedAt != client.CreatedAt {
		t.Fatalf("OAuth client DTO changed: %+v", dto)
	}
	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 || fields["client_id"] != dto.ClientID ||
		fields["name"] != dto.Name || fields["redirect_uri"] != dto.RedirectURI ||
		!strings.Contains(string(encoded), "created_at") {
		t.Fatalf("OAuth HTTP JSON contract changed: %s", encoded)
	}
	if strings.Contains(string(encoded), "OwnerAccountID") ||
		strings.Contains(string(encoded), "secret_hash") {
		t.Fatalf("internal OAuth fields exposed: %s", encoded)
	}
	list := ToOAuthClients([]*auth_model.OAuthClient{client})
	if len(list) != 1 || list[0] != dto {
		t.Fatalf("OAuth client list conversion changed: %+v", list)
	}
	if ToOAuthClients(nil) == nil {
		t.Fatal("empty OAuth client API list must encode []")
	}
}
