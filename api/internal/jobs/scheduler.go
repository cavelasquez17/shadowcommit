package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Job is one named piece of recurring work.
type Job struct {
	Name     string
	Schedule Schedule
	Run      func(ctx context.Context, now time.Time) error
}

// Scheduler runs each job on its own goroutine. Jobs never overlap with themselves and a
// panic in one job is logged instead of taking the API down.
type Scheduler struct {
	jobs   []Job
	now    func() time.Time
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New builds a scheduler. Pass a custom clock for tests; nil uses time.Now.
func New(now func() time.Time, jobs ...Job) *Scheduler {
	if now == nil {
		now = time.Now
	}
	return &Scheduler{jobs: jobs, now: now}
}

// Start launches every job. It returns immediately.
func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	for _, j := range s.jobs {
		s.wg.Add(1)
		go s.loop(ctx, j)
	}
}

// Stop cancels every job and waits for in-flight runs to finish.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *Scheduler) loop(ctx context.Context, j Job) {
	defer s.wg.Done()
	for {
		now := s.now()
		wait := j.Schedule(now).Sub(now)
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.runOnce(ctx, j)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context, j Job) {
	start := s.now()
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return j.Run(ctx, start)
	}()
	if err != nil {
		log.Printf("job %s failed: %v", j.Name, err)
		return
	}
	log.Printf("job %s done in %s", j.Name, s.now().Sub(start).Round(time.Millisecond))
}
