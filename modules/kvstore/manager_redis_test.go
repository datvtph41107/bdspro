package kvstore

import (
	"context"
	"testing"
)

func TestRedisManagerRejectsMissingAddress(t *testing.T) {
	m := NewRedisManager()
	defer m.Close()
	if _, err := m.Client(context.Background(), "  "); err == nil {
		t.Fatal("missing Redis address accepted")
	}
}

func TestRedisManagerDoesNotCacheFailedClient(t *testing.T) {
	m := NewRedisManager()
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Client(ctx, "127.0.0.1:6379"); err == nil {
		t.Fatal("canceled Redis probe accepted")
	}
	if len(m.clients) != 0 {
		t.Fatal("failed client was retained")
	}
}

func TestRedisManagerCloseIsIdempotent(t *testing.T) {
	m := NewRedisManager()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Client(context.Background(), "127.0.0.1:6379"); err == nil {
		t.Fatal("closed manager allowed a connection")
	}
}
