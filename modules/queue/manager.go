package queue

import (
	"errors"
	"fmt"
	"sync"
)

// Manager owns worker lifecycle for one process. No package-level singleton
// or implicit registration occurs during import.
type Manager struct {
	mu      sync.Mutex
	order   []string
	workers map[string]ManagedQueue
	closed  bool
}

func NewManager() *Manager {
	return &Manager{workers: make(map[string]ManagedQueue)}
}

func (m *Manager) Register(name string, worker ManagedQueue) error {
	if m == nil || worker == nil || name == "" {
		return errors.New("queue manager requires name and worker")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrQueueClosed
	}
	if _, exists := m.workers[name]; exists {
		return fmt.Errorf("queue %q already registered", name)
	}
	m.workers[name] = worker
	m.order = append(m.order, name)
	return nil
}

func (m *Manager) Get(name string) ManagedQueue {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.workers[name]
}

// Shutdown stops registered workers once, in reverse registration order,
// without holding the manager mutex while a worker drains.
func (m *Manager) Shutdown() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	workers := make([]ManagedQueue, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		workers = append(workers, m.workers[m.order[i]])
	}
	m.mu.Unlock()
	for _, worker := range workers {
		worker.Shutdown()
	}
}
