package queue

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// redisLinkProxy interrupts this producer's connection to a real disposable
// Redis instance without stopping the Redis server used by other CI tests.
type redisLinkProxy struct {
	listener net.Listener
	upstream string

	mu      sync.Mutex
	offline bool
	active  map[net.Conn]net.Conn
}

func newRedisLinkProxy(t *testing.T, upstream string) *redisLinkProxy {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	proxy := &redisLinkProxy{
		listener: listener,
		upstream: upstream,
		active:   make(map[net.Conn]net.Conn),
	}
	go proxy.accept()
	t.Cleanup(func() {
		_ = proxy.listener.Close()
		proxy.Disconnect()
	})
	return proxy
}

func (p *redisLinkProxy) accept() {
	for {
		client, err := p.listener.Accept()
		if err != nil {
			return
		}
		p.mu.Lock()
		offline := p.offline
		p.mu.Unlock()
		if offline {
			_ = client.Close()
			continue
		}
		go p.forward(client)
	}
}

func (p *redisLinkProxy) forward(client net.Conn) {
	remote, err := net.DialTimeout("tcp", p.upstream, 2*time.Second)
	if err != nil {
		_ = client.Close()
		return
	}
	p.mu.Lock()
	if p.offline {
		p.mu.Unlock()
		_ = client.Close()
		_ = remote.Close()
		return
	}
	p.active[client] = remote
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(remote, client)
		close(done)
	}()
	_, _ = io.Copy(client, remote)
	_ = client.Close()
	_ = remote.Close()
	<-done

	p.mu.Lock()
	delete(p.active, client)
	p.mu.Unlock()
}

func (p *redisLinkProxy) Disconnect() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offline = true
	for client, upstream := range p.active {
		_ = client.Close()
		_ = upstream.Close()
	}
}

func (p *redisLinkProxy) Reconnect() {
	p.mu.Lock()
	p.offline = false
	p.mu.Unlock()
}

// An Asynq client must return a transport error during a mid-process Redis
// network interruption, then use the same client for a later successful
// enqueue when connectivity returns. This does not simulate Redis power loss
// or promise exactly-once effects after an ambiguous enqueue response.
func TestRedisEnqueuerRecoversAfterMidProcessNetworkInterruption(t *testing.T) {
	address := os.Getenv("BDS_TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("requires disposable Redis")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing network fault injection outside disposable mode")
	}

	proxy := newRedisLinkProxy(t, address)
	options := asynq.RedisClientOpt{Addr: proxy.listener.Addr().String(), DB: 15}
	producer := NewRedisEnqueuer(options)
	defer producer.Close()

	queueName := "recover_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	firstID := "before_" + uuid.NewString()
	failedID := "during_" + uuid.NewString()
	afterID := "after_" + uuid.NewString()
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: address, DB: 15})
	defer inspector.Close()
	t.Cleanup(func() {
		for _, id := range []string{firstID, failedID, afterID} {
			_ = inspector.DeleteTask(queueName, id)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := producer.Enqueue(ctx, asynq.NewTask("task:test:redis-link", nil),
		asynq.Queue(queueName), asynq.TaskID(firstID)); err != nil {
		t.Fatalf("enqueue before network interruption: %v", err)
	}

	proxy.Disconnect()
	outageCtx, outageCancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer outageCancel()
	if err := producer.Enqueue(outageCtx, asynq.NewTask("task:test:redis-link", nil),
		asynq.Queue(queueName), asynq.TaskID(failedID)); err == nil {
		t.Fatal("queue acknowledged a task while its Redis connection was interrupted")
	}

	proxy.Reconnect()
	recoveryCtx, recoveryCancel := context.WithTimeout(ctx, 8*time.Second)
	defer recoveryCancel()
	var recoveryErr error
	for {
		recoveryErr = producer.Enqueue(recoveryCtx, asynq.NewTask("task:test:redis-link", nil),
			asynq.Queue(queueName), asynq.TaskID(afterID))
		if recoveryErr == nil || errors.Is(recoveryErr, asynq.ErrTaskIDConflict) {
			break
		}
		select {
		case <-recoveryCtx.Done():
			t.Fatalf("same Asynq producer did not recover: last error %v", recoveryErr)
		case <-time.After(100 * time.Millisecond):
		}
	}
	for _, id := range []string{firstID, afterID} {
		info, err := inspector.GetTaskInfo(queueName, id)
		if err != nil || info == nil || info.Type != "task:test:redis-link" {
			t.Fatalf("task %q was not durably queued: info=%+v err=%v", id, info, err)
		}
	}
}
