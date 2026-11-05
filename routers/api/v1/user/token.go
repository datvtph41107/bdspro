package user

import (
	"net/http"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *AuthHandler) Renew(c *gin.Context) {
	var req structs.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "refresh", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	result, err := h.auth.Renew(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, structs.RenewAccessTokenResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})
}

func (h *AuthHandler) Revoke(c *gin.Context) {
	var req structs.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	if !h.limiter.Allow(c, "revoke", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	if err := h.auth.Revoke(c.Request.Context(), req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetConnections is DRX's connection-history API scoped to the verified Account.
func (h *AuthHandler) GetConnections(c *gin.Context) {
	connections, err := h.auth.GetConnections(c.Request.Context(), middleware.AccountID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	if connections == nil {
		connections = []db.ListAccountConnectionHistoriesRow{}
	}
	c.JSON(http.StatusOK, gin.H{"connections": connections})
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	sessions, err := h.auth.ListSessions(c.Request.Context(), middleware.AccountID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	if sessions == nil {
		sessions = []db.ListActiveSessionsRow{}
	}
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func (h *AuthHandler) RevokeOwnSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Reject(c, http.StatusBadRequest, "invalid_session_id", "invalid session id")
		return
	}
	if err := h.auth.RevokeOwnSession(c.Request.Context(), middleware.AccountID(c), id); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) RevokeAllSessions(c *gin.Context) {
	if err := h.auth.RevokeAllSessions(c.Request.Context(), middleware.AccountID(c)); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
