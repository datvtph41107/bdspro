package structs

import "time"

// OAuthClientResponse is the existing /oauth/clients JSON representation.
// It intentionally never exposes the client secret or the owner account.
type OAuthClientResponse struct {
	ClientID    string    `json:"client_id"`
	Name        string    `json:"name"`
	RedirectURI string    `json:"redirect_uri"`
	CreatedAt   time.Time `json:"created_at"`
}
