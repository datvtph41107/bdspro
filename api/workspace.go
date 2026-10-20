package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// The account is authenticated by requireSession. Workspace discovery uses
// the account's actual Person and live PostgreSQL membership/roles.
func (server *Server) listMyWorkspaces(ctx *gin.Context) {
	accountID, ok := server.currentAccount(ctx)
	if !ok {
		return
	}
	if server.organizations == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "workspace lookup unavailable")
		return
	}
	workspaces, err := server.organizations.ListWorkspaces(ctx.Request.Context(), accountID)
	if err != nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "workspace lookup unavailable")
		return
	}
	replyJSON(ctx, http.StatusOK, gin.H{"workspaces": workspaces})
}
