package org

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTeamHandlersRejectMalformedInputsWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewTeamHandler(nil)
	tests := []struct {
		method string
		path string
		bind string
		body string
		serve gin.HandlerFunc
	}{
		{http.MethodGet, "/organizations/invalid/teams", "/organizations/:id/teams", "", handler.List},
		{http.MethodGet, "/organizations/1/teams?page=invalid", "/organizations/:id/teams", "", handler.List},
		{http.MethodPost, "/organizations/1/teams", "/organizations/:id/teams", "{}", handler.Create},
		{http.MethodGet, "/teams/invalid", "/teams/:team_id", "", handler.Get},
		{http.MethodPatch, "/teams/1", "/teams/:team_id", "{}", handler.Edit},
		{http.MethodPost, "/teams/1/members", "/teams/:team_id/members", "{}", handler.AddMember},
		{http.MethodDelete, "/teams/1/members/invalid", "/teams/:team_id/members/:person_id", "", handler.RemoveMember},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			router := gin.New()
			router.Handle(tt.method, tt.bind, tt.serve)
			var body *strings.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			} else {
				body = strings.NewReader("")
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=400: %s", response.Code, response.Body.String())
			}
		})
	}
}
