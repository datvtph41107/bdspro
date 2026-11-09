package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

// TestGetListOptions verifies the single API pagination boundary used by
// Team and TeamMember lists. SQLC overflow protection remains in models/db.
func TestGetListOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		query   string
		page    int
		size    int
		invalid bool
	}{
		{name: "defaults", page: 1, size: 20},
		{name: "page and limit", query: "?page=3&limit=15", page: 3, size: 15},
		{name: "max page size", query: "?page=2&limit=101", page: 2, size: 100},
		{name: "invalid page text", query: "?page=abc", invalid: true},
		{name: "invalid page zero", query: "?page=0", invalid: true},
		{name: "invalid negative page", query: "?page=-1", invalid: true},
		{name: "invalid limit text", query: "?limit=abc", invalid: true},
		{name: "invalid limit zero", query: "?limit=0", invalid: true},
		{name: "invalid negative limit", query: "?limit=-1", invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orgs/1/teams"+tt.query, nil)
			ctx := context.GetAPIContext(ginCtx)
			options, err := GetListOptions(ctx)
			if tt.invalid {
				if !errors.Is(err, errInvalidListOptions) {
					t.Fatalf("invalid parameters accepted: options=%+v err=%v", options, err)
				}
				return
			}
			if err != nil || options.Page != tt.page || options.PageSize != tt.size {
				t.Fatalf("options=%+v err=%v, want page=%d size=%d", options, err, tt.page, tt.size)
			}
		})
	}
}
