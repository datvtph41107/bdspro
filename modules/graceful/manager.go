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

// InitManager must be called by the application entrypoint before subsystem initialization.
func InitManager(ctx context.Context) {
 InitManagerWithTimeout(ctx, 15*time.Second)
}

// InitManagerWithTimeout starts the one process lifecycle manager.
// It must run before creating any long-lived subsystem.
func InitManagerWithTimeout(ctx context.Context, hammerAfter time.Duration) {
 managerMu.Lock()
 defer managerMu.Unlock()
 if manager != nil { panic("graceful: InitManager called more than once") }
 manager = newManager(ctx, hammerAfter)
}
func GetManager() *Manager {
 managerMu.RLock()
 defer managerMu.RUnlock()
 if manager == nil { panic("graceful: GetManager called before InitManager") }
 return manager
}
type RunCanceler interface { Run(); Cancel() }
func (g *Manager) RunAtShutdown(ctx context.Context, fn func()) {
 g.lock.Lock()
 defer g.lock.Unlock()
 if g.state != stateRunning { panic("graceful: shutdown callback registered after shutdown") }
 g.shutdownHooks = append(g.shutdownHooks, func() {
  if ctx.Err() == nil { fn() }
 })
}
func (g *Manager) RunAtTerminate(fn func()) {
 g.lock.Lock()
 defer g.lock.Unlock()
 if g.state != stateRunning { panic("graceful: termination callback registered after shutdown") }
 g.terminateHooks = append(g.terminateHooks, fn)
}
func (g *Manager) RunWithShutdownContext(fn func(context.Context)) {
 g.RegisterServer()
 defer g.ServerDone()
 fn(g.ShutdownContext())
}
func (g *Manager) RunWithCancel(rc RunCanceler) {
 g.lock.Lock()
 if g.state != stateRunning {
  g.lock.Unlock()
  panic("graceful: RunWithCancel registered after shutdown")
 }
 g.running.Add(1)
 g.shutdownHooks = append(g.shutdownHooks, rc.Cancel)
 g.lock.Unlock()
 defer g.running.Done()
 rc.Run()
}
func (g *Manager) RegisterServer() {
 g.lock.Lock()
 defer g.lock.Unlock()
 if g.state != stateRunning { panic("graceful: server registered after shutdown") }
 g.running.Add(1)
}
func (g *Manager) ServerDone() { g.running.Done() }
func (g *Manager) Done() <-chan struct{} { return g.managerCtx.Done() }
