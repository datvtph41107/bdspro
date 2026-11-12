// Package reqctx owns request-local values, context lifetime, and cleanup.
package reqctx

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
)

type ContextData map[string]any

func (d ContextData) GetData() ContextData { return d }

type ContextDataProvider interface {
	GetData() ContextData
}

type RequestDataStore interface {
	ContextDataProvider
	SetContextValue(key, value any)
	GetContextValue(key any) any
	AddCleanup(func())
	AddCloser(io.Closer)
}

type RequestContext interface {
	context.Context
	RequestDataStore
}

type dataStoreKey struct{}

type requestDataStore struct {
	data ContextData

	mu sync.RWMutex
	values map[any]any
	cleanups []func()
	closed bool
}

func (s *requestDataStore) GetData() ContextData {
	return s.data
}

func (s *requestDataStore) SetContextValue(key, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		panic("reqctx: write to closed request datastore")
	}
	s.values[key] = value
}

func (s *requestDataStore) GetContextValue(key any) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values[key]
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

type requestContext struct {
	context.Context
	RequestDataStore
}

func (c *requestContext) Value(key any) any {
	if key == (dataStoreKey{}) {
		return c.RequestDataStore
	}
	if value := c.GetContextValue(key); value != nil {
		return value
	}
	return c.Context.Value(key)
}

func FromContext(ctx context.Context) RequestContext {
	if ctx == nil {
		return nil
	}
	if rc, ok := ctx.(RequestContext); ok {
		return rc
	}
	store, ok := ctx.Value(dataStoreKey{}).(*requestDataStore)
	if !ok || store == nil {
		return nil
	}
	return &requestContext{Context: ctx, RequestDataStore: store}
}

func MustContextValue[T any](ctx context.Context, key any) T {
	value, ok := ctx.Value(key).(T)
	if !ok {
		panic(fmt.Sprintf("reqctx: context value for %T has type %T, expected %v", key, ctx.Value(key), reflect.TypeFor[T]()))
	}
	return value
}

func NewRequestContext(parent context.Context) (context.Context, func()) {
	if parent == nil {
		panic("reqctx: nil parent context")
	}
	if FromContext(parent) != nil {
		panic("reqctx: request context already initialized")
	}
	store := &requestDataStore{
		data: make(ContextData),
		values: make(map[any]any),
	}
	return &requestContext{Context: parent, RequestDataStore: store}, store.finish
}
