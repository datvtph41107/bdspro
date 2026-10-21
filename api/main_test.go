package api

import (
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	config := util.Config{
		TokenSymmetricKey:    strings.Repeat("s", 32),
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
	}
	server, err := NewServer(config, store)
	require.NoError(t, err)
	return server
}
