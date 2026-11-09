package org

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// The new function-based action must preserve SQLC's account-before-session
// lock order, not merely reproduce a 201 JSON response.
func TestCreateOrgActionPreservesCommitOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sessionID := uuid.New()
	store := &organizationHTTPStoreFake{sessionID: sessionID}
	r := gin.New()
	r.Use(appctx.RequestIngress(), appctx.APIContexter(), func(c *gin.Context) {
		scope := db.WithStore(c.Request.Context(), store)
		scope = auth_model.WithActor(scope, auth_model.Actor{
			AccountID: 42,
			SessionID: sessionID,
		})
		c.Request = c.Request.WithContext(scope)
		api := appctx.GetAPIContext(c)
		api.Doer = &authcrypto.Payload{AccountID: 42, SessionID: sessionID}
		api.IsSigned = true
		c.Next()
	})
	r.POST("/orgs", func(c *gin.Context) {
		var form structs.CreateOrgOption
		if err := c.ShouldBindJSON(&form); err != nil {
			appctx.Validation(c, err, &form)
			return
		}
		web.SetForm(appctx.GetAPIContext(c), &form)
		c.Next()
	}, appctx.Wrap(Create))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orgs", strings.NewReader(`{"name":"  Example  "}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated ||
		!strings.Contains(w.Body.String(), `"name":"Example"`) ||
		!strings.Contains(w.Body.String(), `"party_id":123`) {
		t.Fatalf("status=%d response=%s", w.Code, w.Body.String())
	}
	if !reflect.DeepEqual(store.lockOrder, []string{"account", "session"}) {
		t.Fatalf("changed lock order: %v", store.lockOrder)
	}
}
