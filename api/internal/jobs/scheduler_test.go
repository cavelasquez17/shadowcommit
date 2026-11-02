package jobs

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func soon(time.Time) time.Time { return time.Now().Add(5 * time.Millisecond) }

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestSchedulerRunsJobRepeatedlyUntilStopped(t *testing.T) {
	var n atomic.Int32
	s := New(nil, Job{Name: "count", Schedule: soon, Run: func(context.Context, time.Time) error {
		n.Add(1)
		return nil
	}})
	s.Start()
	waitFor(t, func() bool { return n.Load() >= 3 })
	s.Stop()

	after := n.Load()
	time.Sleep(30 * time.Millisecond)
	if n.Load() != after {
		t.Errorf("job kept running after Stop: %d -> %d", after, n.Load())
	}
}

func TestSchedulerSurvivesPanicsAndErrors(t *testing.T) {
	var panics, errs, healthy atomic.Int32
	s := New(nil,
		Job{Name: "panics", Schedule: soon, Run: func(context.Context, time.Time) error {
			panics.Add(1)
			panic("boom")
		}},
		Job{Name: "errors", Schedule: soon, Run: func(context.Context, time.Time) error {
			errs.Add(1)
			return errors.New("nope")
		}},
		Job{Name: "healthy", Schedule: soon, Run: func(context.Context, time.Time) error {
			healthy.Add(1)
			return nil
		}},
	)
	s.Start()
	defer s.Stop()
	// A panicking job must keep being rescheduled, and must not stop its siblings.
	waitFor(t, func() bool { return panics.Load() >= 2 && errs.Load() >= 2 && healthy.Load() >= 2 })
}

func TestSchedulerStopWithoutStartIsSafe(t *testing.T) {
	New(nil).Stop()
}
