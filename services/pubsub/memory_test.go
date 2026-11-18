package pubsub

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func receive(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()

	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatal("subscriber channel closed")
		}
		return msg

	case <-time.After(time.Second):
		t.Fatal("message not received")
		return nil
	}
}

func TestMemoryBrokerFanout(t *testing.T) {
	b := NewMemoryBroker()

	a, cancelA := b.Subscribe("user-42")
	defer cancelA()

	c, cancelC := b.Subscribe("user-42")
	defer cancelC()

	b.Publish("user-42", []byte("hello"))

	for _, ch := range []<-chan []byte{a, c} {
		if got := receive(t, ch); !bytes.Equal(got, []byte("hello")) {
			t.Fatalf("message = %q", got)
		}
	}
}

func TestMemoryBrokerTopicIsolation(t *testing.T) {
	b := NewMemoryBroker()

	a, cancelA := b.Subscribe("user-42")
	defer cancelA()

	other, cancelOther := b.Subscribe("user-99")
	defer cancelOther()

	b.Publish("user-42", []byte("private"))
	receive(t, a)

	select {
	case got := <-other:
		t.Fatalf("message leaked to other topic: %q", got)
	default:
	}
}

func TestMemoryBrokerCancel(t *testing.T) {
	b := NewMemoryBroker()

	ch, cancel := b.Subscribe("user-42")

	if !b.HasTopicSubscribers("user-42") {
		t.Fatal("expected active subscriber")
	}

	cancel()
	cancel()

	if b.HasTopicSubscribers("user-42") {
		t.Fatal("subscriber still registered")
	}

	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed")
	}

	b.Publish("user-42", []byte("after cancel"))
}

func TestMemoryBrokerSlowSubscriber(t *testing.T) {
	b := NewMemoryBroker()

	_, cancel := b.Subscribe("user-42")
	defer cancel()

	for i := 0; i < 1000; i++ {
		b.Publish("user-42", []byte("message"))
	}
	// Test completes without blocking on a full buffer.
}

func TestMemoryBrokerCopiesPayload(t *testing.T) {
	b := NewMemoryBroker()

	a, cancelA := b.Subscribe("user-42")
	defer cancelA()

	c, cancelC := b.Subscribe("user-42")
	defer cancelC()

	payload := []byte("hello")
	b.Publish("user-42", payload)

	payload[0] = 'X'

	first := receive(t, a)
	second := receive(t, c)

	if !bytes.Equal(first, []byte("hello")) ||
		!bytes.Equal(second, []byte("hello")) {
		t.Fatalf("publisher mutated message")
	}

	first[0] = 'Y'

	if !bytes.Equal(second, []byte("hello")) {
		t.Fatal("subscribers share mutable payload")
	}
}

func TestMemoryBrokerConcurrentPublishCancel(t *testing.T) {
	b := NewMemoryBroker()

	const count = 100

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, cancel := b.Subscribe("user-42")

			for j := 0; j < 20; j++ {
				b.Publish("user-42", []byte("event"))
			}

			cancel()
			cancel()
		}()
	}

	wg.Wait()

	if b.HasTopicSubscribers("user-42") {
		t.Fatal("subscriber leaked")
	}
}
