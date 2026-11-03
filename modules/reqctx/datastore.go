// Package reqctx owns the lifetime of request-scoped values and cleanup.
// It intentionally has no dependency on the HTTP router or business packages.
package reqctx

import (
	"context"
	"io"
	"sync"
)

type datastoreKey struct{}

type DataStore struct {
	mu       sync.RWMutex
	values   map[any]any
	cleanups []func()
	finished bool
}

// NewRequestContext preserves parent cancellation/deadline and returns a cleanup
// function that must be deferred by the request's outermost middleware.
// Cleanup executes once, last registered first, without holding the lock.
func NewRequestContext(parent context.Context) (context.Context, func()) {
	if parent == nil {
		panic("reqctx: nil parent context")
	}
	store := &DataStore{values: make(map[any]any)}
	return context.WithValue(parent, datastoreKey{}, store), store.Finish
}

func FromContext(ctx context.Context) (*DataStore, bool) {
	if ctx == nil {
		return nil, false
	}
	store, ok := ctx.Value(datastoreKey{}).(*DataStore)
	return store, ok
}

func MustFromContext(ctx context.Context) *DataStore {
	store, ok := FromContext(ctx)
	if !ok {
		panic("reqctx: request data store not initialized")
	}
	return store
}

// Set stores request-owned data. Do not pass this store across request lifetimes.
func (s *DataStore) Set(key, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finished {
		panic("reqctx: Set after Finish")
	}
	s.values[key] = value
}

func (s *DataStore) Get(key any) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[key]
	return v, ok
}

// AddCleanup registers an owned resource cleanup. Callers must not register
// cleanup after Finish because that would silently leak resources.
func (s *DataStore) AddCleanup(fn func()) {
	if fn == nil {
		panic("reqctx: nil cleanup")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finished {
		panic("reqctx: AddCleanup after Finish")
	}
	s.cleanups = append(s.cleanups, fn)
}

func (s *DataStore) AddCloser(c io.Closer) {
	if c == nil {
		panic("reqctx: nil closer")
	}
	s.AddCleanup(func() { _ = c.Close() })
}

func (s *DataStore) Finish() {
	s.mu.Lock()
	if s.finished {
		s.mu.Unlock()
		return
	}
	s.finished = true
	fns := s.cleanups
	s.cleanups = nil
	s.values = nil
	s.mu.Unlock()
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

