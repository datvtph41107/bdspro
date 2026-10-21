package api

import (
	"strings"
	"context"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/hibiken/asynq"
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
	server, err := NewServer(config, store, testTaskDistributor{})
	require.NoError(t, err)
	return server
}

type testTaskDistributor struct {}
func (testTaskDistributor) DistributeTaskSendVerifyEmail(_ context.Context, _ *worker.PayloadSendVerifyEmail, _ ...asynq.Option) error {
	return nil
}
