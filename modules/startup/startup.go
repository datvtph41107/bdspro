// Package startup runs fail-fast, ordered initialization for process entrypoints.
// Resource construction and cleanup remain owned by the participating packages.
package startup

import (
	"context"
	"errors"
	"fmt"
)

// Step is a required initialization and its optional rollback. Init must
// clean up its own partially created state if it returns an error.
// Rollback runs only after Init succeeds and must tolerate a later graceful
// termination hook calling the same resource owner's Close function again.
type Step struct {
	Name string
	Init func(context.Context) error
	Rollback func()
}

// Run executes required stages in declaration order, reporting the failed
// stage as a wrapped error. On failure it rolls back completed stages in
// reverse order. Unlike Gitea's mustInit, Run never calls log.Fatal; cmd
// remains responsible for graceful process shutdown.
func Run(ctx context.Context, steps ...Step) (err error) {
	if ctx == nil {
		return errors.New("startup: nil context")
	}
	if len(steps) == 0 {
		return errors.New("startup: no initialization steps")
	}
	for i, step := range steps {
		if step.Name == "" || step.Init == nil {
			return fmt.Errorf("startup: invalid step at index %d", i)
		}
	}

	completed := make([]Step, 0, len(steps))
	defer func() {
		if err == nil {
			return
		}
		for i := len(completed) - 1; i >= 0; i-- {
			if completed[i].Rollback != nil {
				completed[i].Rollback()
			}
		}
	}()

	for _, step := range steps {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("startup before %s: %w", step.Name, ctxErr)
		}
		if initErr := step.Init(ctx); initErr != nil {
			return fmt.Errorf("startup %s: %w", step.Name, initErr)
		}
		completed = append(completed, step)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("startup after %s: %w", steps[len(steps)-1].Name, ctxErr)
	}
	return nil
}
