package pubsub

import "sync"

// MemoryBroker fans out messages within one process.
//
// The mutex guards both the subscriber registry
// and channel close/send coordination.
type MemoryBroker struct {
	mu sync.RWMutex

	subs map[string]map[chan []byte]struct{}
}

var _ Broker = (*MemoryBroker)(nil)

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		subs: make(map[string]map[chan []byte]struct{}),
	}
}

func (b *MemoryBroker) Subscribe(
	topic string,
) (<-chan []byte, func()) {
	ch := make(chan []byte, subscriberBuffer)

	b.mu.Lock()
	if b.subs == nil {
		b.subs = make(map[string]map[chan []byte]struct{})
	}
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[chan []byte]struct{})
	}

	b.subs[topic][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once

	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			delete(b.subs[topic], ch)

			if len(b.subs[topic]) == 0 {
				delete(b.subs, topic)
			}

			close(ch)
		})
	}

	return ch, cancel
}

func (b *MemoryBroker) HasTopicSubscribers(
	topic string,
) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subs[topic]) > 0
}

func (b *MemoryBroker) Publish(
	topic string,
	msg []byte,
) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subs[topic] {
		// Each subscriber owns its message bytes.
		// No mutable slice is shared across subscribers.
		data := append([]byte(nil), msg...)

		select {
		case ch <- data:
		default:
			// Best-effort: a slow consumer may
			// miss this event.
		}
	}
}
