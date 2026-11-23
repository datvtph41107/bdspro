package graceful

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

var (
	manager  *Manager
	initOnce sync.Once
)

var ErrShuttingDown = errors.New(
	"graceful: shutdown already started",
)

// GetManager returns the initialized process manager.
// Initialization is owned by the application entrypoint.
func GetManager() *Manager {
	return manager
}

// InitManager initializes the manager once per process.
func InitManager(ctx context.Context) {
	initOnce.Do(func() {
		manager = newGracefulManager(ctx, setting.Graceful.HammerTimeout)
	})
}

type RunCanceler interface {
	Run()
	Cancel()
}

// runHook prevents a panic in one callback from
// crashing the lifecycle coordinator.
func runHook(name string, fn func()) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf(
				"graceful: panic in %s: %v\n%s",
				name, p, debug.Stack(),
			)
		}
	}()
	fn()
}

// RunAtShutdown registers a stop callback.
// Registration is allowed only while running.
func (g *Manager) RunAtShutdown(
	ctx context.Context,
	fn func(),
) {
	if ctx == nil || fn == nil {
		panic("graceful: invalid shutdown callback")
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	if g.state != stateRunning {
		panic(
			"graceful: shutdown registration after shutdown",
		)
	}

	g.toRunAtShutdown = append(
		g.toRunAtShutdown,
		func() {
			if ctx.Err() == nil {
				runHook("RunAtShutdown", fn)
			}
		},
	)
}

// RunAtTerminate registers resource cleanup.
// Registration is permitted until termination begins.
func (g *Manager) RunAtTerminate(fn func()) {
	if fn == nil {
		panic("graceful: nil termination callback")
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	if g.state == stateTerminate {
		panic(
			"graceful: termination already started",
		)
	}

	g.toRunAtTerminate = append(
		g.toRunAtTerminate,
		func() {
			runHook("RunAtTerminate", fn)
		},
	)
}

// RunWithShutdownContext runs an activity that
// cooperatively stops when shutdown is requested.
func (g *Manager) RunWithShutdownContext(
	run func(context.Context),
) error {
	if run == nil {
		return errors.New(
			"graceful: nil managed activity",
		)
	}

	if err := g.RegisterServer(); err != nil {
		return err
	}
	defer g.ServerDone()

	run(g.ShutdownContext())
	return nil
}

// RunWithCancel manages a Run/Cancel subsystem.
// Its cancellation callback and activity admission
// are registered under the same lock.
func (g *Manager) RunWithCancel(
	rc RunCanceler,
) error {
	if rc == nil {
		return errors.New(
			"graceful: nil RunCanceler",
		)
	}

	g.lock.Lock()

	if g.state != stateRunning {
		g.lock.Unlock()
		return ErrShuttingDown
	}

	g.runningServerWaitGroup.Add(1)
	g.toRunAtShutdown = append(
		g.toRunAtShutdown,
		func() {
			runHook("RunWithCancel.Cancel", rc.Cancel)
		},
	)

	g.lock.Unlock()

	defer g.runningServerWaitGroup.Done()
	rc.Run()

	return nil
}

// RegisterServer admits one managed activity.
// Every successful call requires one ServerDone.
func (g *Manager) RegisterServer() error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.state != stateRunning {
		return ErrShuttingDown
	}

	g.runningServerWaitGroup.Add(1)
	return nil
}

func (g *Manager) ServerDone() {
	g.runningServerWaitGroup.Done()
}

func (g *Manager) Done() <-chan struct{} {
	return g.managerCtx.Done()
}

func (g *Manager) setStateTransition(
	from, to state,
) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.state != from {
		return false
	}

	g.state = to
	return true
}

// doShutdown transitions the process into shutdown,
// dispatches stop callbacks, and begins the hammer timer.
func (g *Manager) doShutdown() {
	g.lock.Lock()

	if g.state != stateRunning {
		g.lock.Unlock()
		return
	}

	g.state = stateShuttingDown
	g.shutdownCtxCancel()

	hooks := append(
		[]func(){},
		g.toRunAtShutdown...,
	)

	g.shutdownCallbackWaitGroup.Add(len(hooks))
	g.lock.Unlock()

	for _, hook := range hooks {
		go func(fn func()) {
			defer g.shutdownCallbackWaitGroup.Done()
			fn()
		}(hook)
	}

	if g.hammerAfter >= 0 {
		go g.doHammerTime(g.hammerAfter)
	}

	go func() {
		g.shutdownCallbackWaitGroup.Wait()
		g.runningServerWaitGroup.Wait()

		// Release hammer waiters after draining.
		g.DoImmediateHammer()

		g.doTerminate()
	}()
}

// doHammerTime requests escalation when the graceful
// deadline expires before process termination.
func (g *Manager) doHammerTime(delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-g.Done():
		return

	case <-timer.C:
		g.lock.Lock()
		shuttingDown := g.state == stateShuttingDown
		g.lock.Unlock()

		if shuttingDown {
			g.DoImmediateHammer()
		}
	}
}

// doTerminate begins cleanup and waits for callbacks
// before signalling process completion.
func (g *Manager) doTerminate() {
	g.lock.Lock()

	if g.state != stateShuttingDown {
		g.lock.Unlock()
		return
	}

	g.state = stateTerminate
	g.terminateCtxCancel()

	hooks := append(
		[]func(){},
		g.toRunAtTerminate...,
	)

	g.terminateWaitGroup.Add(len(hooks))
	g.lock.Unlock()

	for _, hook := range hooks {
		go func(fn func()) {
			defer g.terminateWaitGroup.Done()
			fn()
		}(hook)
	}

	g.terminateWaitGroup.Wait()
	g.managerCtxCancel()
}
