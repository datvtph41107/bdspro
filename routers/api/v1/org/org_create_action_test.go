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
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	"github.com/datvtph41107/bdspro/routers/common"
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
	r := web.NewRouter()
	r.AfterRouting(appctx.APIContexter())
	r.GinEngine().Use(common.RequestIngress(), func(c *gin.Context) {
		c.Request = c.Request.WithContext(db.WithStore(c.Request.Context(), store))
		c.Next()
	})
	r.Post("/orgs", func(api *appctx.APIContext) {
		api.Doer = &authcrypto.Payload{AccountID: 42, SessionID: sessionID}
		api.PublishVerifiedActor(auth_model.Actor{AccountID: 42, SessionID: sessionID})
	}, func(api *appctx.APIContext) {
		form, err := middleware.BindJSON[structs.CreateOrgOption](api)
		if err != nil {
			api.Validation(err, form)
			return
		}
		web.SetForm(api, form)
	}, Create)

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
