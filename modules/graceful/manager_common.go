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

func (g *Manager) DoGracefulShutdown() {
	g.doShutdown()
}

func (g *Manager) DoImmediateHammer() {
	g.hammerCtxCancel()
}
