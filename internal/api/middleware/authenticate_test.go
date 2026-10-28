package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
)

func TestAuthenticateAccessOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := token.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	access, _, err := maker.CreateToken(42, time.Minute, token.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := maker.CreateToken(42, time.Minute, token.TokenTypeRefresh)
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/me", Authenticate(maker), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"account_id": AccountID(ctx)})
	})

	for _, tc := range []struct {
		name, authorization string
		status              int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic abc", http.StatusUnauthorized},
		{"invalid JWT", "Bearer garbage", http.StatusUnauthorized},
		{"refresh is not access", "Bearer " + refresh, http.StatusUnauthorized},
		{"valid access", "Bearer " + access, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tc.authorization != "" {
				request.Header.Set("Authorization", tc.authorization)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != tc.status {
				t.Fatalf("status %d; expected %d; response %s", recorder.Code, tc.status, recorder.Body.String())
			}
			if tc.status == http.StatusOK {
				var data struct {
					AccountID int64 `json:"account_id"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &data); err != nil || data.AccountID != 42 {
					t.Fatalf("wrong verified account: %+v; %v", data, err)
				}
			}
		})
	}
}
