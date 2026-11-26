package startup

import (
	"context"
	"errors"
	"fmt"
)

type Step struct {
	Name     string
	Init     func(context.Context) error
	Rollback func()
}

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
