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

// Manager coordinates shutdown phases; subsystem owners remain responsible for resource cleanup.
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
 shutdownCallbacks sync.WaitGroup
 terminating sync.WaitGroup
 shutdownHooks []func()
 terminateHooks []func()
 hammerAfter time.Duration
}

func newManager(parent context.Context, hammerAfter time.Duration) *Manager {
 phaseParent := context.WithoutCancel(parent)
 g := &Manager{state: stateRunning, hammerAfter: hammerAfter}
 g.shutdownCtx, g.shutdownCancel = context.WithCancel(phaseParent)
 g.hammerCtx, g.hammerCancel = context.WithCancel(phaseParent)
 g.terminateCtx, g.terminateCancel = context.WithCancel(phaseParent)
 g.managerCtx, g.managerCancel = context.WithCancel(phaseParent)
 go func() {
  select {
  case <-parent.Done(): g.DoGracefulShutdown()
  case <-g.Done():
  }
 }()
 return g
}

// DoGracefulShutdown is idempotent. Hammer is a separate timeout/escalation phase.
func (g *Manager) DoGracefulShutdown() {
 g.lock.Lock()
 if g.state != stateRunning {
  g.lock.Unlock()
  return
 }
 g.state = stateShuttingDown
 g.shutdownCancel()
 hooks := append([]func(){}, g.shutdownHooks...)
 g.shutdownCallbacks.Add(len(hooks))
 g.lock.Unlock()
 for _, hook := range hooks {
  go func(fn func()) {
   defer g.shutdownCallbacks.Done()
   fn()
  }(hook)
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
   case <-timer.C: g.DoImmediateHammer()
   case <-hammerDone:
   }
  }()
 }
 g.shutdownCallbacks.Wait()
 g.running.Wait()
 close(hammerDone)
 g.lock.Lock()
 g.state = stateTerminate
 hooks := append([]func(){}, g.terminateHooks...)
 g.terminating.Add(len(hooks))
 g.terminateCancel()
 g.lock.Unlock()
 for _, hook := range hooks {
  go func(fn func()) {
   defer g.terminating.Done()
   fn()
  }(hook)
 }
 g.terminating.Wait()
 g.managerCancel()
}

// DoImmediateHammer cancels the hammer context; it cannot forcibly kill goroutines.
func (g *Manager) DoImmediateHammer() { g.hammerCancel() }
