package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRedisBrokerDisconnectClosesSubscribers(t *testing.T) {
	client := redisClientForTest(t)

	b, err := NewRedisBroker(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ch, cancel := b.Subscribe(t.Name())
	defer cancel()

	// Simulate a permanent failure of this test's
	// dedicated Redis client. The production RedisManager
	// remains unrelated to this test.
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unexpected message after disconnect")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("subscriber not closed after Redis failure")
	}
}

func TestRedisBrokerConcurrentCloseCancel(t *testing.T) {
	client := redisClientForTest(t)

	b, err := NewRedisBroker(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}

	const n = 100

	cancels := make([]func(), 0, n)

	for i := 0; i < n; i++ {
		_, cancel := b.Subscribe(t.Name())
		cancels = append(cancels, cancel)
	}

	var wg sync.WaitGroup

	for _, cancel := range cancels {
		wg.Add(1)
		go func(cancel func()) {
			defer wg.Done()
			cancel()
			cancel()
		}(cancel)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			b.Publish(t.Name(), []byte("event"))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := b.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	}()

	wg.Wait()

	// Calling Close more than once must be safe.
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	// Broker must not close its borrowed Redis client.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("borrowed Redis client closed: %v", err)
	}
}

func TestRedisBrokerSubscribeAfterClose(t *testing.T) {
	client := redisClientForTest(t)

	b, err := NewRedisBroker(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	ch, cancel := b.Subscribe(t.Name())
	defer cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("closed broker delivered a message")
		}
	case <-time.After(time.Second):
		t.Fatal("closed broker did not close subscriber channel")
	}
}
