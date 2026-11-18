package pubsub

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	redis "github.com/go-redis/redis/v8"
)

func redisClientForTest(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("set REDIS_ADDR=127.0.0.1:6379 to run Redis integration tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("Redis unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRedisBrokerCrossClientFanout(t *testing.T) {
	clientA := redisClientForTest(t)
	clientB := redisClientForTest(t)
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	a, err := NewRedisBroker(ctx, clientA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewRedisBroker(ctx, clientB)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
	topic := t.Name()
	first, cancelFirst := a.Subscribe(topic)
	defer cancelFirst()
	second, cancelSecond := b.Subscribe(topic)
	defer cancelSecond()
	third, cancelThird := b.Subscribe(topic)
	defer cancelThird()
	a.Publish(topic, []byte("hello"))
	for i, ch := range []<-chan []byte{first, second, third} {
		select {
		case got, ok := <-ch:
			if !ok || !bytes.Equal(got, []byte("hello")) {
				t.Fatalf("subscriber %d: %q closed=%v", i, got, !ok)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("subscriber %d did not receive", i)
		}
	}
	cancelThird()
	if _, ok := <-third; ok {
		t.Fatal("cancel did not close channel")
	}
	cancelThird()
	if !b.HasTopicSubscribers(topic) {
		t.Fatal("Redis hint must be conservative")
	}
}

func TestRedisBrokerCloseDoesNotOwnClient(t *testing.T) {
	client := redisClientForTest(t)
	b, err := NewRedisBroker(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	ch, cancel := b.Subscribe(t.Name())
	defer cancel()
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := <-ch; ok {
		t.Fatal("expected subscriber channel closed")
	}
	ctx, cancelPing := context.WithTimeout(context.Background(), time.Second)
	defer cancelPing()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("broker closed borrowed Redis client: %v", err)
	}
}
