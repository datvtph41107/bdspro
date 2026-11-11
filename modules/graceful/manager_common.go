package graceful

import (
	"context"
	"sync"
	"time"
)

type state uint8

const (
	stateInit state = iota
	stateRunning
	stateShuttingDown
	stateTerminate
)

// Manager owns application shutdown phases, not individual subsystem resources.
type Manager struct {
	lock sync.Mutex
	state state

	shutdownCtx context.Context
	shutdownCancel context.CancelFunc
	hammerCtx context.Context
	hammerCancel context.CancelFunc
	terminateCtx context.Context
	terminateCancel context.CancelFunc
	managerCtx context.Context
	managerCancel context.CancelFunc

	running sync.WaitGroup
	terminating sync.WaitGroup
	shutdownHooks []func()
	terminateHooks []func()
	hammerAfter time.Duration
}

func newManager(parent context.Context, hammerAfter time.Duration) *Manager {
	g := &Manager{state: stateRunning, hammerAfter: hammerAfter}
	phaseParent := context.WithoutCancel(parent)
	g.shutdownCtx, g.shutdownCancel = context.WithCancel(phaseParent)
	g.hammerCtx, g.hammerCancel = context.WithCancel(phaseParent)
	g.terminateCtx, g.terminateCancel = context.WithCancel(phaseParent)
	g.managerCtx, g.managerCancel = context.WithCancel(phaseParent)
	go func() {
		select {
		case <-parent.Done():
			g.DoGracefulShutdown()
		case <-g.Done():
		}
	}()
	return g
}

// DoGracefulShutdown transitions once and initiates the shutdown sequence.
func (g *Manager) DoGracefulShutdown() {
	g.lock.Lock()
	if g.state != stateRunning {
		g.lock.Unlock()
		g.DoImmediateHammer()
		return
	}
	g.state = stateShuttingDown
	hooks := append([]func(){}, g.shutdownHooks...)
	g.shutdownCancel()
	g.lock.Unlock()

	for _, hook := range hooks {
		go hook()
	}
	go g.finishShutdown()
}

func (g *Manager) finishShutdown() {
	hammerDone := make(chan struct{})
	if g.hammerAfter >= 0 {
		go func() {
			timer := time.NewTimer(g.hammerAfter)
			defer timer.Stop()
			select {
			case <-timer.C:
				g.DoImmediateHammer()
			case <-hammerDone:
			}
		}()
	}

	g.running.Wait()
	close(hammerDone)
	g.DoImmediateHammer()

	g.lock.Lock()
	g.state = stateTerminate
	hooks := append([]func(){}, g.terminateHooks...)
	g.terminateCancel()
	g.lock.Unlock()

	for _, hook := range hooks {
		g.terminating.Add(1)
		go func(fn func()) {
			defer g.terminating.Done()
			fn()
		}(hook)
	}
	g.terminating.Wait()
	g.managerCancel()
}

func (g *Manager) DoImmediateHammer() {
	g.hammerCancel()
}
