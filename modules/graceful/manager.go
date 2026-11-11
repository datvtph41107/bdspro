package graceful

import (
	"context"
	"sync"
	"time"
)

var (
	managerMu sync.RWMutex
	manager *Manager
)

// InitManager is called once by cmd/web.go before starting subsystems.
func InitManager(ctx context.Context) {
	managerMu.Lock()
	defer managerMu.Unlock()
	if manager != nil {
		panic("graceful: InitManager called more than once")
	}
	manager = newManager(ctx, 15*time.Second)
}

// GetManager fails immediately if the application's init order is incorrect.
func GetManager() *Manager {
	managerMu.RLock()
	defer managerMu.RUnlock()
	if manager == nil {
		panic("graceful: GetManager called before InitManager")
	}
	return manager
}

type RunCanceler interface {
	Run()
	Cancel()
}

func (g *Manager) RunAtShutdown(ctx context.Context, fn func()) {
	g.lock.Lock()
	if g.state != stateRunning {
		g.lock.Unlock()
		if ctx.Err() == nil {
			go fn()
		}
		return
	}
	g.shutdownHooks = append(g.shutdownHooks, func() {
		if ctx.Err() == nil {
			fn()
		}
	})
	g.lock.Unlock()
}

func (g *Manager) RunAtTerminate(fn func()) {
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.state != stateRunning {
		panic("graceful: registering termination callback after shutdown")
	}
	g.terminateHooks = append(g.terminateHooks, fn)
}

func (g *Manager) RunWithShutdownContext(fn func(context.Context)) {
	g.RegisterServer()
	defer g.ServerDone()
	fn(g.ShutdownContext())
}

func (g *Manager) RunWithCancel(rc RunCanceler) {
	g.RunAtShutdown(context.Background(), rc.Cancel)
	g.RegisterServer()
	defer g.ServerDone()
	rc.Run()
}

func (g *Manager) RegisterServer() {
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.state != stateRunning {
		panic("graceful: server registered after shutdown")
	}
	g.running.Add(1)
}

func (g *Manager) ServerDone() { g.running.Done() }
func (g *Manager) Done() <-chan struct{} { return g.managerCtx.Done() }
