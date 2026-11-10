package oauth

import (
	"net/http"

	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListAuthorizations is distinct from ListClients: ownership != consent.
func (h *Handler) ListAuthorizations(c *gin.Context) {
	grants, err := h.service.ListAuthorizations(c.Request.Context(), appctx.AccountID(c))
	if err != nil {
		serviceFailure(c, err)
		return
	}
	out := make([]gin.H, 0, len(grants))
	for _, grant := range grants {
		out = append(out, gin.H{
			"client_id": grant.ClientID.String(), "name": grant.Name, "scope": grant.Scope,
		})
	}
	c.Header("Cache-Control", "no-store")
	appctx.GetAPIContext(c).JSON(http.StatusOK, gin.H{"authorizations": out})
}

// RevokeAuthorization invalidates credentials for the signed-in account only.
func (h *Handler) RevokeAuthorization(c *gin.Context) {
	clientID, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		failure(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.service.RevokeAuthorization(c.Request.Context(), appctx.AccountID(c), clientID); err != nil {
		serviceFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}
