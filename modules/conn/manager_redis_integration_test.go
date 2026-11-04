package conn

import (
	"context"
	"os"
	"testing"
)

func TestRedisManagerReusesClient(t *testing.T) {
	address := os.Getenv("BDS_TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("set BDS_TEST_REDIS_ADDR to test a live Redis connection")
	}
	m := NewRedisManager()
	defer m.Close()
	first, err := m.Client(context.Background(), address)
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Client(context.Background(), address)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("one process created duplicate Redis clients for the same address")
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.Ping(context.Background()).Err(); err == nil {
		t.Fatal("Redis connection stayed usable after manager close")
	}
}
