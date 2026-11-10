package convert

import (
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/structs"
)

// ToOAuthClient takes a model/auth entity, never a generated SQLC row.
// Exactly like Gitea's ToOAuth2Application, API serialization is owned here.
func ToOAuthClient(client *auth_model.OAuthClient) structs.OAuthClientResponse {
	return structs.OAuthClientResponse{
		ClientID:    client.ID.String(),
		Name:        client.Name,
		RedirectURI: client.RedirectURI,
		CreatedAt:   client.CreatedAt,
	}
}

func ToOAuthClients(clients []*auth_model.OAuthClient) []structs.OAuthClientResponse {
	out := make([]structs.OAuthClientResponse, 0, len(clients))
	for _, client := range clients {
		out = append(out, ToOAuthClient(client))
	}
	return out
}
