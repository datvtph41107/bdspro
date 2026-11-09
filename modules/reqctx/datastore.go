// Package reqctx owns short-lived request state and cleanup lifecycle.
// It is HTTP-framework independent, following Gitea's reqctx boundary.
package reqctx

import (
	"context"
	"io"
	"sync"
)

// ContextData is request-local, not safe for concurrent mutation.
type ContextData map[string]any

// ContextDataProvider lets web form binding and context share one datastore.
type ContextDataProvider interface {
	GetData() ContextData
}

// RequestDataStore is created once by HTTP ingress and disposed at request end.
type RequestDataStore interface {
	ContextDataProvider
	AddCleanup(func())
	AddCloser(io.Closer)
}

type dataStoreKey struct{}

type requestDataStore struct {
	data ContextData

	mu       sync.Mutex
	cleanups []func()
	closed   bool
}

func (s *requestDataStore) GetData() ContextData {
	return s.data
}

func (s *requestDataStore) AddCleanup(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		// A late registration must not silently leak its resource.
		fn()
		return
	}
	s.cleanups = append(s.cleanups, fn)
	s.mu.Unlock()
}

func (s *requestDataStore) AddCloser(closer io.Closer) {
	if closer != nil {
		s.AddCleanup(func() { _ = closer.Close() })
	}
}

func (s *requestDataStore) finish() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	cleanups := s.cleanups
	s.cleanups = nil
	s.mu.Unlock()
	for _, fn := range cleanups {
		fn()
	}
}

// FromContext retrieves the existing request store without creating another.
func FromContext(ctx context.Context) RequestDataStore {
	if ctx == nil {
		return nil
	}
	store, _ := ctx.Value(dataStoreKey{}).(*requestDataStore)
	return store
}

// NewRequestContext returns an isolated request context and one idempotent
// cleanup callback. A caller must invoke finished after its handler chain.
func NewRequestContext(parent context.Context) (context.Context, func()) {
	if parent == nil {
		panic("reqctx: nil parent context")
	}
	store := &requestDataStore{data: make(ContextData)}
	return context.WithValue(parent, dataStoreKey{}, store), store.finish
}
