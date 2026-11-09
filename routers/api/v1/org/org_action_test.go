package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

func TestGetUsesAssignedOrganizationWithoutDatabaseRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(appctx.APIContexter())
	r.GET("/orgs/:id", func(c *gin.Context) {
		api := appctx.GetAPIContext(c)
		api.Org = &appctx.APIOrganization{Access: &organization_model.Access{
			Organization: db.Organization{PartyID: 12, Name: "Acme", CreatedAt: time.Unix(100, 0).UTC()},
			MembershipID: 5,
			Role:         organization_model.RoleOwner,
		}}
		c.Next()
	}, appctx.Wrap(Get))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orgs/12", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		PartyID    int64 `json:"party_id"`
		Membership struct {
			ID int64 `json:"id"`
		} `json:"membership"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.PartyID != 12 || response.Membership.ID != 5 {
		t.Fatalf("wrong organization: %+v", response)
	}
}
