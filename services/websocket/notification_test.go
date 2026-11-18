package websocket

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/services/pubsub"
)

type fakeCounter struct {
	count int64
	err   error
	calls int
}

func (f *fakeCounter) CountUnread(
	_ context.Context,
	_ int64,
) (int64, error) {
	f.calls++
	return f.count, f.err
}

func TestNotificationCountEvent(t *testing.T) {
	counter := &fakeCounter{count: 3}
	broker := pubsub.NewMemoryBroker()

	n, err := NewNotifier(counter, broker)
	if err != nil {
		t.Fatal(err)
	}

	data, err := n.NotificationCountEvent(
		context.Background(), 42,
	)
	if err != nil {
		t.Fatal(err)
	}

	want := "\n" +
		`{"eventType":"notification-count","eventData":{"count":3}}`

	if string(data) != want {
		t.Fatalf("got %q, want %q", data, want)
	}
}

func TestNotificationCountChangeFanout(t *testing.T) {
	counter := &fakeCounter{count: 7}
	broker := pubsub.NewMemoryBroker()

	first, cancelFirst := broker.Subscribe("user-42")
	defer cancelFirst()

	second, cancelSecond := broker.Subscribe("user-42")
	defer cancelSecond()

	n, err := NewNotifier(counter, broker)
	if err != nil {
		t.Fatal(err)
	}

	n.NotificationCountChange(context.Background(), 42)

	for _, ch := range []<-chan []byte{first, second} {
		select {
		case payload := <-ch:
			if !bytes.Contains(
				payload,
				[]byte(`"count":7`),
			) {
				t.Fatalf("unexpected payload: %s", payload)
			}
		default:
			t.Fatal("no realtime event received")
		}
	}

	if counter.calls != 1 {
		t.Fatalf("DB count called %d times", counter.calls)
	}
}

func TestNotificationCountChangeNoSubscribers(t *testing.T) {
	counter := &fakeCounter{count: 5}
	broker := pubsub.NewMemoryBroker()

	n, err := NewNotifier(counter, broker)
	if err != nil {
		t.Fatal(err)
	}

	n.NotificationCountChange(context.Background(), 42)

	if counter.calls != 0 {
		t.Fatalf("unexpected DB reads: %d", counter.calls)
	}
}

func TestNotificationCountChangeDatabaseFailure(t *testing.T) {
	counter := &fakeCounter{err: errors.New("db unavailable")}
	broker := pubsub.NewMemoryBroker()

	ch, cancel := broker.Subscribe("user-42")
	defer cancel()

	n, err := NewNotifier(counter, broker)
	if err != nil {
		t.Fatal(err)
	}

	n.NotificationCountChange(context.Background(), 42)

	select {
	case payload := <-ch:
		t.Fatalf("published after DB failure: %s", payload)
	default:
	}
}

func TestNotificationCountEventRejectsNegativeCount(t *testing.T) {
	counter := &fakeCounter{count: -1}
	n, err := NewNotifier(
		counter, pubsub.NewMemoryBroker(),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = n.NotificationCountEvent(
		context.Background(), 42,
	)
	if err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("expected negative count error, got %v", err)
	}
}
