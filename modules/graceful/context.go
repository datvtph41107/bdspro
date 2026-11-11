package graceful

import "context"

// Context cancellation order: Shutdown, Hammer, Terminate, Done.
func (g *Manager) ShutdownContext() context.Context  { return g.shutdownCtx }
func (g *Manager) HammerContext() context.Context    { return g.hammerCtx }
func (g *Manager) TerminateContext() context.Context { return g.terminateCtx }
func (g *Manager) IsShutdown() <-chan struct{}       { return g.shutdownCtx.Done() }
func (g *Manager) IsHammer() <-chan struct{}         { return g.hammerCtx.Done() }
