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

// Manager coordinates application lifetime. It never owns DB or HTTP resources.
type Manager struct {
	ctx context.Context

	lock  sync.Mutex
	state state

	shutdownCtx  context.Context
	hammerCtx    context.Context
	terminateCtx context.Context
	managerCtx   context.Context

	shutdownCtxCancel  context.CancelFunc
	hammerCtxCancel    context.CancelFunc
	terminateCtxCancel context.CancelFunc
	managerCtxCancel   context.CancelFunc

	runningServerWaitGroup    sync.WaitGroup
	shutdownCallbackWaitGroup sync.WaitGroup
	terminateWaitGroup        sync.WaitGroup

	toRunAtShutdown  []func()
	toRunAtTerminate []func()

	// drainDone closes after callbacks and registered activities finish. A
	// hammer deadline is meaningful only until that point, not during DB cleanup.
	drainDone   chan struct{}
	hammerAfter time.Duration
}

// newGracefulManager separates construction, preparation, and OS-specific start.
func newGracefulManager(ctx context.Context, hammerAfter time.Duration) *Manager {
	g := &Manager{
		ctx:         ctx,
		hammerAfter: hammerAfter,
	}
	g.prepare(ctx)
	g.start()
	return g
}

// prepare has no goroutine, OS signal, network or resource side effects.
func (g *Manager) prepare(ctx context.Context) {
	// A CLI context cancellation is a request to enter shutdown, not a
	// command to cancel all lifecycle phases simultaneously.
	phaseParent := context.WithoutCancel(ctx)

	g.shutdownCtx, g.shutdownCtxCancel = context.WithCancel(phaseParent)
	g.hammerCtx, g.hammerCtxCancel = context.WithCancel(phaseParent)
	g.terminateCtx, g.terminateCtxCancel = context.WithCancel(phaseParent)
	g.managerCtx, g.managerCtxCancel = context.WithCancel(phaseParent)
	g.drainDone = make(chan struct{})

	if !g.setStateTransition(stateInit, stateRunning) {
		panic("graceful: invalid transition init -> running")
	}
}

// DoGracefulShutdown accepts one request to stop; the sequence is in manager.go.
func (g *Manager) DoGracefulShutdown() {
	g.doShutdown()
}

// DoImmediateHammer signals escalation to interested consumers.
// It is not SIGKILL; goroutines must respond cooperatively.
func (g *Manager) DoImmediateHammer() {
	g.hammerCtxCancel()
}
