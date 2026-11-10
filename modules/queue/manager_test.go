package queue

import (
	"slices"
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

type orderedManagedQueue struct {
	name  string
	order *[]string
}

func (q *orderedManagedQueue) Shutdown() {
	*q.order = append(*q.order, q.name)
}

func TestManagerShutsDownQueuesInReverseRegistrationOrder(t *testing.T) {
	m := NewManager()
	order := []string{}
	for _, name := range []string{"first", "second", "third"} {
		if err := m.Register(name, &orderedManagedQueue{name: name, order: &order}); err != nil {
			t.Fatal(err)
		}
	}
	m.Shutdown()
	want := []string{"third", "second", "first"}
	if !slices.Equal(order, want) {
		t.Fatalf("shutdown order = %v, want %v", order, want)
	}
}
