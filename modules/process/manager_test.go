package process

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestCancelDoesNotFinish(t *testing.T) {
	pm := NewManager()
	ctx, cancel, finished := pm.AddContext(context.Background(), "job")
	if len(pm.Snapshot()) != 1 {
		t.Fatal("job not tracked")
	}
	cancel()
	if ctx.Err() == nil {
		t.Fatal("cancel did not stop context")
	}
	if len(pm.Snapshot()) != 1 {
		t.Fatal("cancel incorrectly finished job")
	}
	finished()
	finished()
	if len(pm.Snapshot()) != 0 {
		t.Fatal("finished did not remove job")
	}
}
func TestNestedParentAndCause(t *testing.T) {
	pm := NewManager()
	parent, _, parentDone := pm.AddContext(context.Background(), "parent")
	child, cancel, childDone := pm.AddContext(parent, "child")
	entries := pm.Snapshot()
	if len(entries) != 2 {
		t.Fatal("expected two tracked jobs")
	}
	if ParentPID(child) == "" {
		t.Fatal("missing child pid")
	}
	errCause := errors.New("stop")
	cancel(errCause)
	if !errors.Is(context.Cause(child), errCause) {
		t.Fatal("cancel cause lost")
	}
	childDone()
	parentDone()
}
func TestSystemProcessNotExternallyCanceled(t *testing.T) {
	pm := NewManager()
	ctx, _, done := pm.AddTypedContext(context.Background(), "system", SystemProcessType, true)
	if pm.Cancel(ParentPID(ctx)) {
		t.Fatal("system cancel allowed")
	}
	if ctx.Err() != nil {
		t.Fatal("system context canceled")
	}
	done()
}
func TestConcurrentFinish(t *testing.T) {
	pm := NewManager()
	_, _, done := pm.AddContext(context.Background(), "job")
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); done() }()
	}
	wg.Wait()
	if len(pm.Snapshot()) != 0 {
		t.Fatal("job leaked")
	}
}
