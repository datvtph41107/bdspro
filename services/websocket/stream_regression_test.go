package websocket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/services/pubsub"
)

// readyEventBroker simulates a notification published immediately after the
// subscription becomes visible. No goroutine timing assumptions are needed.
type readyEventBroker struct {
	pubsub.Broker
	event []byte
}

func (b readyEventBroker) Subscribe(topic string) (<-chan []byte, func()) {
	ch, cancel := b.Broker.Subscribe(topic)
	b.Broker.Publish(topic, b.event)
	return ch, cancel
}

func TestStreamDoesNotDiscardBufferedEvent(t *testing.T) {
	event, err := EncodeEvent("chat-message", "", map[string]string{"id": "m1"})
	if err != nil {
		t.Fatal(err)
	}
	broker := readyEventBroker{Broker: pubsub.NewMemoryBroker(), event: event}
	stream, err := NewStream(broker, func(context.Context, int64) ([]byte, error) {
		return EncodeEvent("snapshot", "", map[string]string{"ready": "yes"})
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var messages []string
	err = stream.Serve(ctx, 42, "trusted-session", func(_ context.Context, body []byte) error {
		messages = append(messages, string(body))
		if len(messages) == 2 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Serve: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("received %d, want snapshot and buffered event", len(messages))
	}
	if messages[0] != `{"eventType":"snapshot","eventData":{"ready":"yes"}}` {
		t.Fatalf("snapshot = %s", messages[0])
	}
	if messages[1] != `{"eventType":"chat-message","eventData":{"id":"m1"}}` {
		t.Fatalf("buffered event = %s", messages[1])
	}
}
