package websocket

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/services/pubsub"
)

func TestStreamSubscribesBeforeSnapshot(t *testing.T) {
	broker := pubsub.NewMemoryBroker()

	stream, err := NewStream(
		broker,
		func(ctx context.Context, accountID int64) ([]byte, error) {
			if !broker.HasTopicSubscribers(
				pubsub.UserTopic(accountID),
			) {
				t.Fatal("snapshot called before subscribe")
			}

			return EncodeEvent(
				"snapshot", "",
				map[string]int{"version": 1},
			)
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	var messages [][]byte

	err = stream.Serve(ctx, 42, "session-A",
		func(_ context.Context, data []byte) error {
			messages = append(messages,
				append([]byte(nil), data...))
			cancel()
			return nil
		},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stream error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages: %d", len(messages))
	}
	if broker.HasTopicSubscribers("user-42") {
		t.Fatal("subscription was not cancelled")
	}
}

func TestStreamRejectsInvalidIdentity(t *testing.T) {
	broker := pubsub.NewMemoryBroker()

	stream, err := NewStream(
		broker,
		func(context.Context, int64) ([]byte, error) {
			t.Fatal("snapshot must not run")
			return nil, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = stream.Serve(
		context.Background(), 0, "",
		func(context.Context, []byte) error {
			t.Fatal("send must not run")
			return nil
		},
	)
	if !errors.Is(err, ErrStreamIdentity) {
		t.Fatalf("error: %v", err)
	}
}

func TestStreamFiltersOtherSessions(t *testing.T) {
	broker := pubsub.NewMemoryBroker()

	ctx, cancel := context.WithTimeout(
		context.Background(), 3*time.Second,
	)
	defer cancel()

	started := make(chan struct{})
	var once sync.Once
	var received []string

	stream, err := NewStream(
		broker,
		func(context.Context, int64) ([]byte, error) {
			return EncodeEvent("snapshot", "", nil)
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- stream.Serve(
			ctx, 42, "session-A",
			func(_ context.Context, data []byte) error {
				received = append(received, string(data))
				once.Do(func() { close(started) })
				if len(received) == 2 {
					cancel()
				}
				return nil
			},
		)
	}()

	<-started

	wrong, err := EncodeEvent("private", "session-B", nil)
	if err != nil {
		t.Fatal(err)
	}
	correct, err := EncodeEvent("private", "session-A", nil)
	if err != nil {
		t.Fatal(err)
	}

	broker.Publish("user-42", wrong)
	broker.Publish("user-42", correct)

	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("stream error: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("received %d events", len(received))
	}
}
