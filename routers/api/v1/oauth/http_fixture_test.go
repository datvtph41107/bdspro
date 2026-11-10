package oauth

import (
	"context"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// oauthSessionTestStore supplies real session admission semantics with an
// isolated fake store, without requiring PostgreSQL or bypassing RequireSession.
type oauthSessionTestStore struct {
	db.Store
	accounts map[uuid.UUID]int64
}

func (s oauthSessionTestStore) GetSession(_ context.Context, id uuid.UUID) (db.Session, error) {
	if accountID, ok := s.accounts[id]; ok {
		return db.Session{ID: id, AccountID: accountID, ExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	return db.Session{}, pgx.ErrNoRows
}

// oauthTestRouter registers real package actions with per-request capabilities.
// No OAuth Handler factory or Gin compatibility action exists in production.
func oauthTestRouter(service *oauth_service.Service, store db.Store, proofs *oauth_service.ConsentProofs, maker authcrypto.Maker) *web.Router {
	m := web.NewRouter()
	m.AfterRouting(appctx.APIContexter())
	engine := m.GinEngine()
	engine.Use(func(c *gin.Context) {
		ctx := oauth_service.WithService(c.Request.Context(), service, proofs)
		if store != nil {
			ctx = db.WithStore(ctx, store)
		}
		if maker != nil {
			ctx = authcrypto.WithMaker(ctx, maker)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	return m
}
