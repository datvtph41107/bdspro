package auth

import (
	"time"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
)

type SessionInfo struct {
	ID        uuid.UUID `json:"id"`
	UserAgent string    `json:"user_agent"`
	ClientIP  string    `json:"client_ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Current   bool      `json:"current"`
}

var ErrSessionNotFound = util.WithPublic(util.ErrNotExist, "active owned session not found", "resource_not_found", "session not found")
