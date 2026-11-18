package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLoginPasswordDoesNotAccessDatabaseWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := LoginPassword(ctx, "user@example.com", "password", "agent", "127.0.0.1", nil, time.Minute, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestLoginPasswordRejectsNilContext(t *testing.T) {
	_, err := LoginPassword(nil, "user@example.com", "password", "agent", "127.0.0.1", nil, time.Minute, time.Hour)
	if err == nil {
		t.Fatal("nil context accepted")
	}
}
