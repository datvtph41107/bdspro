package reqctx

import (
	"context"
	"io"
	"sync"
)

type ContextData map[string]any

type ContextDataProvider interface {
	GetData() ContextData
}

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

func FromContext(ctx context.Context) RequestDataStore {
	if ctx == nil {
		return nil
	}
	store, ok := ctx.Value(dataStoreKey{}).(*requestDataStore)
	if !ok || store == nil {
		return nil
	}
	return store
}

func NewRequestContext(parent context.Context) (context.Context, func()) {
	if parent == nil {
		panic("reqctx: nil parent context")
	}
	store := &requestDataStore{data: make(ContextData)}
	return context.WithValue(parent, dataStoreKey{}, store), store.finish
}
