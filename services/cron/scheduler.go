package cron

import (
	"context"
	"errors"
	"fmt"
	"sync"

	robfig "github.com/robfig/cron/v3"
)

type Job func(context.Context)

type Scheduler struct {
	ctx       context.Context
	scheduler *robfig.Cron

	mu       sync.Mutex
	names    map[string]struct{}
	started  bool
	stopping bool

	stopOnce sync.Once
	stopped  chan struct{}
}

func New(ctx context.Context) *Scheduler {
	if ctx == nil {
		panic("cron: nil context")
	}

	return &Scheduler{
		ctx:       ctx,
		scheduler: robfig.New(),
		names:     make(map[string]struct{}),
		stopped:   make(chan struct{}),
	}
}

func (s *Scheduler) Register(
	name, spec string,
	job Job,
) error {
	if name == "" || spec == "" || job == nil {
		return errors.New("cron: invalid registration")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started || s.stopping {
		return errors.New("cron: registration closed")
	}

	if _, exists := s.names[name]; exists {
		return fmt.Errorf(
			"cron: duplicate job %q", name,
		)
	}

	_, err := s.scheduler.AddFunc(spec, func() {
		if s.ctx.Err() != nil {
			return
		}

		job(s.ctx)
	})
	if err != nil {
		return fmt.Errorf(
			"cron: register %s: %w", name, err,
		)
	}

	s.names[name] = struct{}{}
	return nil
}

func (s *Scheduler) HasJobs() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.names) > 0
}

// Run owns the actual drain wait.
func (s *Scheduler) Run() {
	s.mu.Lock()

	if !s.stopping && !s.started {
		s.started = true
		s.scheduler.Start()
	}

	s.mu.Unlock()

	<-s.stopped
}

// Cancel requests shutdown without waiting for jobs.
// Run remains blocked until running jobs have returned.
func (s *Scheduler) Cancel() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.stopping = true
		started := s.started
		s.mu.Unlock()

		if !started {
			close(s.stopped)
			return
		}

		drained := s.scheduler.Stop()

		go func() {
			<-drained.Done()
			close(s.stopped)
		}()
	})
}
