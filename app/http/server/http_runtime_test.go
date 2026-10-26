package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type arc02ContextKey struct{}

// Embedding db.Store leaves unrelated operations unimplemented: this test
// intercepts only the first database operation of createOrganization.
type arc02ContextSpyStore struct {
	db.Store
	received context.Context
}

func (s *arc02ContextSpyStore) GetAccountByID(ctx context.Context, _ int64) (db.Account, error) {
	s.received = ctx
	return db.Account{}, errors.New("arc02 test: stop before any database write")
}

func TestARC02IngressRejectsDeclaredOversizeBeforeHandler(t *testing.T) {
	server := newTestServer(t, nil)
	called := false
	server.router.POST("/test/arc02/ingress", func(c *gin.Context) {
		called = true
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/test/arc02/ingress", strings.NewReader("ok"))
	req.ContentLength = maxRequestBodyBytes + 1
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	require.False(t, called, "request body admission must stop the route handler")
	require.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestARC02StoreReceivesHTTPContext(t *testing.T) {
	spy := &arc02ContextSpyStore{}
	server := newTestServer(t, spy)
	access, _, err := server.tokenMaker.CreateToken(19, time.Minute, token.TokenTypeAccess)
	require.NoError(t, err)

	parent, cancel := context.WithCancel(context.WithValue(context.Background(), arc02ContextKey{}, "from-http-request"))
	defer cancel()

	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(`{"name":"ARC02"}`)).WithContext(parent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+access)

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code, "test Store deliberately stops the operation")
	require.NotNil(t, spy.received, "the handler must call Store")
	require.Equal(t, "from-http-request", spy.received.Value(arc02ContextKey{}))
	require.NotNil(t, spy.received.Done(), "database calls need HTTP request cancellation")

	cancel()
	select {
	case <-spy.received.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("Store context did not observe request cancellation")
	}
}
