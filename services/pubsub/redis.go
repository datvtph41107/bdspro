package pubsub

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	redis "github.com/go-redis/redis/v8"
)

const (
	redisTopicPrefix      = "bdspro-ws-topic:"
	redisSubscribeTimeout = 2 * time.Second
	redisPublishTimeout   = 2 * time.Second
)

// RedisBroker borrows the Redis client; modules/conn remains its owner.
// Redis Pub/Sub is transient: disconnected or slow subscribers may miss events.
type RedisBroker struct {
	client *redis.Client
	ctx    context.Context
	mu     sync.RWMutex
	topics map[string]*redisTopic
	closed bool
	wg     sync.WaitGroup
}

type redisTopic struct {
	ps     *redis.PubSub
	cancel context.CancelFunc
	subs   map[*redisSubscriber]struct{}
}

type redisSubscriber struct {
	ch   chan []byte
	once sync.Once
}

func (s *redisSubscriber) close() { s.once.Do(func() { close(s.ch) }) }

var _ Broker = (*RedisBroker)(nil)

func NewRedisBroker(ctx context.Context, client *redis.Client) (*RedisBroker, error) {
	if ctx == nil {
		return nil, errors.New("pubsub: nil context")
	}
	if client == nil {
		return nil, errors.New("pubsub: nil Redis client")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &RedisBroker{ctx: ctx, client: client, topics: make(map[string]*redisTopic)}, nil
}

// Subscribe confirms the Redis SUBSCRIBE acknowledgement before returning.
// On failure it returns a closed channel, as required by the Broker contract.
func (b *RedisBroker) Subscribe(topic string) (<-chan []byte, func()) {
	sub := &redisSubscriber{ch: make(chan []byte, subscriberBuffer)}
	b.mu.Lock()
	if b.closed || b.ctx.Err() != nil {
		b.mu.Unlock()
		sub.close()
		return sub.ch, func() {}
	}
	if state := b.topics[topic]; state != nil {
		state.subs[sub] = struct{}{}
		b.mu.Unlock()
		return sub.ch, b.cancelFunc(topic, state, sub)
	}

	// Serialized first-subscription setup: avoids duplicate SUBSCRIBE races.
	// Redis network readiness may delay other subscriptions for <= 2s.
	topicCtx, cancelTopic := context.WithCancel(b.ctx)
	ps := b.client.Subscribe(topicCtx, redisTopicPrefix+topic)
	ackCtx, cancelAck := context.WithTimeout(topicCtx, redisSubscribeTimeout)
	_, err := ps.Receive(ackCtx)
	cancelAck()
	if err != nil {
		b.mu.Unlock()
		cancelTopic()
		_ = ps.Close()
		sub.close()
		log.Error("pubsub: Redis subscribe failed: %v", err)
		return sub.ch, func() {}
	}
	state := &redisTopic{ps: ps, cancel: cancelTopic, subs: map[*redisSubscriber]struct{}{sub: {}}}
	b.topics[topic] = state
	b.wg.Add(1)
	b.mu.Unlock()
	go b.readLoop(topicCtx, topic, state)
	return sub.ch, b.cancelFunc(topic, state, sub)
}

func (b *RedisBroker) cancelFunc(topic string, state *redisTopic, sub *redisSubscriber) func() {
	return func() {
		var ps *redis.PubSub
		b.mu.Lock()
		if b.topics[topic] == state {
			delete(state.subs, sub)
			if len(state.subs) == 0 {
				delete(b.topics, topic)
				state.cancel()
				ps = state.ps
			}
		}
		sub.close() // close and send are serialized under the same broker lock.
		b.mu.Unlock()
		if ps != nil {
			_ = ps.Close()
		}
	}
}

func (b *RedisBroker) readLoop(ctx context.Context, topic string, state *redisTopic) {
	defer b.wg.Done()
	defer b.dropTopic(topic, state)
	for {
		msg, err := state.ps.ReceiveMessage(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Error("pubsub: Redis receive failed: %v", err)
			}
			return
		}
		b.mu.RLock()
		if b.topics[topic] != state {
			b.mu.RUnlock()
			return
		}
		for sub := range state.subs {
			data := []byte(msg.Payload) // each subscriber owns an independent backing array.
			select {
			case sub.ch <- data:
			default:
				log.Trace("pubsub: dropped Redis message for slow subscriber")
			}
		}
		b.mu.RUnlock()
	}
}

func (b *RedisBroker) dropTopic(topic string, state *redisTopic) {
	b.mu.Lock()
	if b.topics[topic] == state {
		delete(b.topics, topic)
		for sub := range state.subs {
			sub.close()
		}
	}
	state.cancel()
	b.mu.Unlock()
	_ = state.ps.Close()
}

func (b *RedisBroker) Publish(topic string, msg []byte) {
	if b == nil {
		return
	}
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()
	if closed {
		return
	}
	ctx, cancel := context.WithTimeout(b.ctx, redisPublishTimeout)
	defer cancel()
	if err := b.client.Publish(ctx, redisTopicPrefix+topic, append([]byte(nil), msg...)).Err(); err != nil {
		log.Error("pubsub: Redis publish failed: %v", err)
	}
}

// A Redis topic may have subscribers in another process. Never optimize
// publication away based only on subscribers in this process.
func (b *RedisBroker) HasTopicSubscribers(string) bool { return true }

// Close stops all subscriptions/read loops but never closes the borrowed client.
func (b *RedisBroker) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		b.wg.Wait()
		return nil
	}
	b.closed = true
	states := make([]*redisTopic, 0, len(b.topics))
	for topic, state := range b.topics {
		delete(b.topics, topic)
		for sub := range state.subs {
			sub.close()
		}
		state.cancel()
		states = append(states, state)
	}
	b.mu.Unlock()
	var err error
	for _, state := range states {
		err = errors.Join(err, state.ps.Close())
	}
	b.wg.Wait()
	return err
}
