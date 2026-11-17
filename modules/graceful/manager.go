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

var ErrShuttingDown = errors.New("graceful: shutdown already started")

// GetManager returns the initialized process manager.
// The application must finish InitManager before starting its consumers.
func GetManager() *Manager {
	return manager
}

// InitManager initializes the process manager once, as in Gitea.
// Common settings are prepared by the CLI Before hook before this call.
func InitManager(ctx context.Context) {
	initManager(ctx)
}

func initManager(ctx context.Context) {
	initOnce.Do(func() {
		manager = newGracefulManager(
			ctx,
			setting.Common.Graceful.HammerTimeout,
		)
	})
}

type RunCanceler interface {
	Run()
	Cancel()
}

// runHook isolates a panic from another resource owner's callback, but
// deliberately does not claim that the cleanup itself succeeded.
func runHook(name string, fn func()) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf(
				"graceful: panic in %s: %v\n%s",
				name,
				p,
				debug.Stack(),
			)
		}
	}()

	fn()
}

// RunAtShutdown registers a bounded stop-request callback during running.
// BDSPro waits for these callbacks before entering termination.
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
		panic("graceful: shutdown callback registered after shutdown")
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

// RunAtTerminate allows late cleanup registration until the draining phase
// completes. In particular, PostgreSQL/Redis may finish initializing during
// an in-progress shutdown while startup remains registered as an activity.
func (g *Manager) RunAtTerminate(fn func()) {
	if fn == nil {
		panic("graceful: nil terminate callback")
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	if g.state == stateTerminate {
		panic(
			"graceful: termination callback registered after termination",
		)
	}

	g.toRunAtTerminate = append(
		g.toRunAtTerminate,
		func() {
			runHook("RunAtTerminate", fn)
		},
	)
}

// RunWithShutdownContext registers an activity and only releases registration
// after run returns; cancellation is not proof of completion.
func (g *Manager) RunWithShutdownContext(
	run func(context.Context),
) error {
	if run == nil {
		return errors.New("graceful: nil managed activity")
	}

	if err := g.RegisterServer(); err != nil {
		return err
	}

	defer g.ServerDone()

	run(g.ShutdownContext())
	return nil
}

// RunWithCancel atomically registers a custom stop request and its activity.
// Cancel must tolerate being called before Run actually starts.
func (g *Manager) RunWithCancel(rc RunCanceler) error {
	if rc == nil {
		return errors.New("graceful: nil RunCanceler")
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

// RegisterServer admits exactly one activity. Successful callers must pair
// with exactly one ServerDone, including on all error exits.
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

// doShutdown is the one-time running -> shutting-down transition.
func (g *Manager) doShutdown() {
	g.lock.Lock()

	if g.state != stateRunning {
		g.lock.Unlock()
		return
	}

	g.state = stateShuttingDown

	hooks := append(
		[]func(){},
		g.toRunAtShutdown...,
	)

	g.shutdownCallbackWaitGroup.Add(len(hooks))
	g.shutdownCtxCancel()
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

		close(g.drainDone)
		g.doTerminate()
	}()
}

// doHammerTime only escalates if server/worker draining is still in progress.
func (g *Manager) doHammerTime(delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-g.drainDone:
		return

	case <-timer.C:
		g.lock.Lock()

		if g.state == stateShuttingDown {
			select {
			case <-g.drainDone:
				// All registered activities finished.
			default:
				g.hammerCtxCancel()
			}
		}

		g.lock.Unlock()
	}
}

// doTerminate is the only transition into resource cleanup. It snapshots
// hooks under the same lock used by RunAtTerminate registration.
func (g *Manager) doTerminate() {
	g.lock.Lock()

	if g.state != stateShuttingDown {
		g.lock.Unlock()
		return
	}

	g.state = stateTerminate

	hooks := append(
		[]func(){},
		g.toRunAtTerminate...,
	)

	g.terminateWaitGroup.Add(len(hooks))
	g.terminateCtxCancel()
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
