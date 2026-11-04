package org

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTeamInviteHandlersRejectMalformedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTeamHandler(nil, "")
	for _, tt := range []struct {
		name string
		method string
		path string
		pattern string
		body string
		handle gin.HandlerFunc
	}{
		{"missing email", http.MethodPost, "/teams/1/invites", "/teams/:team_id/invites", "{}", h.CreateInvite},
		{"invalid email", http.MethodPost, "/teams/1/invites", "/teams/:team_id/invites", `{"email":"bad"}`, h.CreateInvite},
		{"invalid team", http.MethodPost, "/teams/invalid/invites", "/teams/:team_id/invites", `{"email":"member@example.test"}`, h.CreateInvite},
		{"missing token", http.MethodPost, "/teams/invites/accept", "/teams/invites/accept", "{}", h.AcceptInvite},
		{"invalid token length", http.MethodPost, "/teams/invites/accept", "/teams/invites/accept", `{"token":"wrong"}`, h.AcceptInvite},
		{"invalid invite ID", http.MethodDelete, "/teams/1/invites/wrong", "/teams/:team_id/invites/:invite_id", "", h.RevokeInvite},
	}{
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Handle(tt.method, tt.pattern, tt.handle)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
