package process

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type IDType string
type CancelCauseFunc func(cause ...error)
type FinishedFunc func()

const (
	NormalProcessType = "normal"
	SystemProcessType = "system"
)

type processKey struct{}
type Entry struct {
	PID         IDType
	ParentPID   IDType
	Description string
	Type        string
	Started     time.Time
}
type activeProcess struct {
	Entry
	cancel CancelCauseFunc
}
type Manager struct {
	mu        sync.Mutex
	next      uint64
	processes map[IDType]*activeProcess
}

var (
	managerOnce sync.Once
	manager     *Manager
)

func NewManager() *Manager { return &Manager{processes: make(map[IDType]*activeProcess)} }
func GetManager() *Manager {
	managerOnce.Do(func() { manager = NewManager() })
	return manager
}
func ParentPID(ctx context.Context) IDType {
	id, _ := ctx.Value(processKey{}).(IDType)
	return id
}
func (pm *Manager) AddContext(parent context.Context, description string) (context.Context, CancelCauseFunc, FinishedFunc) {
	return pm.AddTypedContext(parent, description, NormalProcessType, true)
}
func (pm *Manager) AddTypedContext(parent context.Context, description, processType string, currentlyRunning bool) (context.Context, CancelCauseFunc, FinishedFunc) {
	ctx, cancelCause := context.WithCancelCause(parent)
	cancel := func(cause ...error) {
		if len(cause) > 0 {
			cancelCause(cause[0])
			return
		}
		cancelCause(context.Canceled)
	}
	parentPID := ParentPID(parent)
	pm.mu.Lock()
	pm.next++
	id := IDType(fmt.Sprintf("p-%d", pm.next))
	if _, ok := pm.processes[parentPID]; !ok {
		parentPID = ""
	}
	p := &activeProcess{Entry: Entry{PID: id, ParentPID: parentPID, Description: description, Type: processType, Started: time.Now()}, cancel: cancel}
	pm.processes[id] = p
	pm.mu.Unlock()
	ctx = context.WithValue(ctx, processKey{}, id)
	var once sync.Once
	finished := func() {
		once.Do(func() {
			cancel()
			pm.mu.Lock()
			if pm.processes[id] == p {
				delete(pm.processes, id)
			}
			pm.mu.Unlock()
		})
	}
	_ = currentlyRunning // Reserved for pprof goroutine labeling, as in Gitea.
	return ctx, cancel, finished
}
func (pm *Manager) Cancel(id IDType) bool {
	pm.mu.Lock()
	p, ok := pm.processes[id]
	pm.mu.Unlock()
	if !ok || p.Type == SystemProcessType {
		return false
	}
	p.cancel()
	return true
}
func (pm *Manager) Snapshot() []Entry {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	entries := make([]Entry, 0, len(pm.processes))
	for _, p := range pm.processes {
		entries = append(entries, p.Entry)
	}
	return entries
}
