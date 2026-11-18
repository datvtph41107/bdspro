package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
)

func TestRefreshSessionRejectsInsufficientProof(t *testing.T) {
	_, err := RefreshSession(
		context.Background(), uuid.New(), "not-a-refresh-token",
		"test-agent", "127.0.0.1", nil, time.Minute, time.Hour,
	)
	if !errors.Is(err, util.ErrUnauthenticated) {
		t.Fatalf("invalid refresh proof must reject before DB: %v", err)
	}

	_, err = RefreshSession(
		context.Background(), uuid.Nil, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"test-agent", "127.0.0.1", nil, time.Minute, time.Hour,
	)
	if !errors.Is(err, util.ErrUnauthenticated) {
		t.Fatalf("nil session must reject before DB: %v", err)
	}
}

func TestRefreshSessionHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RefreshSession(ctx, uuid.New(), "", "", "", nil, time.Minute, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context: %v", err)
	}
}
