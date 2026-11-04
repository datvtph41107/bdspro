package queue

import (
	"sync"
	"testing"
)

type managedTestQueue struct {
	mu    sync.Mutex
	stops int
}

func (m *managedTestQueue) Shutdown() {
	m.mu.Lock()
	m.stops++
	m.mu.Unlock()
}

func TestManagerRegistersAndShutsDownOnce(t *testing.T) {
	manager := NewManager()
	worker := &managedTestQueue{}
	if err := manager.Register("emails", worker); err != nil {
		t.Fatal(err)
	}
	if manager.Get("emails") != worker {
		t.Fatal("registration missing")
	}
	if err := manager.Register("emails", worker); err == nil {
		t.Fatal("duplicate name accepted")
	}
	manager.Shutdown()
	manager.Shutdown()
	if worker.stops != 1 {
		t.Fatalf("shutdown called %d times", worker.stops)
	}
	if err := manager.Register("other", worker); err == nil {
		t.Fatal("manager accepted worker after shutdown")
	}
}
