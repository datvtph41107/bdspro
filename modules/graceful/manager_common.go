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

// Manager coordinates the lifecycle of the process.
// Each subsystem remains responsible for its resources.
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

	hammerAfter time.Duration
}

// newGracefulManager constructs and starts
// the process lifecycle manager.
func newGracefulManager(
	ctx context.Context,
	hammerAfter time.Duration,
) *Manager {
	if ctx == nil {
		panic("graceful: nil context")
	}

	g := &Manager{
		ctx:         ctx,
		hammerAfter: hammerAfter,
	}

	g.prepare(ctx)
	g.start()

	return g
}

// prepare creates independent lifecycle phase contexts.
// Cancelling the CLI context requests shutdown, but does
// not automatically cancel all phases at once.
func (g *Manager) prepare(ctx context.Context) {
	parent := context.WithoutCancel(ctx)

	g.shutdownCtx, g.shutdownCtxCancel =
		context.WithCancel(parent)

	g.hammerCtx, g.hammerCtxCancel =
		context.WithCancel(parent)

	g.terminateCtx, g.terminateCtxCancel =
		context.WithCancel(parent)

	g.managerCtx, g.managerCtxCancel =
		context.WithCancel(parent)

	if !g.setStateTransition(
		stateInit,
		stateRunning,
	) {
		panic("graceful: invalid manager state")
	}
}

// DoGracefulShutdown requests the shutdown phase.
func (g *Manager) DoGracefulShutdown() {
	g.doShutdown()
}

// DoImmediateHammer requests escalation.
// It does not kill arbitrary goroutines.
func (g *Manager) DoImmediateHammer() {
	g.hammerCtxCancel()
}
