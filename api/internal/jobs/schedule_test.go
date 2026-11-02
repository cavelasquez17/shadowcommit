package jobs

import (
	"testing"
	"time"
)

func at(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.UTC)
}

func TestDaily(t *testing.T) {
	t.Parallel()
	s := Daily(3)
	cases := []struct{ now, want time.Time }{
		{at(2026, 11, 2, 1), at(2026, 11, 2, 3)},   // before today's run
		{at(2026, 11, 2, 3), at(2026, 11, 3, 3)},   // exactly on it: strictly after
		{at(2026, 11, 2, 15), at(2026, 11, 3, 3)},  // after today's run
		{at(2026, 11, 30, 23), at(2026, 12, 1, 3)}, // month rollover
	}
	for _, c := range cases {
		if got := s(c.now); !got.Equal(c.want) {
			t.Errorf("Daily(3)(%v) = %v, want %v", c.now, got, c.want)
		}
	}
}

func TestWeekly(t *testing.T) {
	t.Parallel()
	s := Weekly(time.Monday, 9) // 2026-11-02 is a Monday
	cases := []struct{ now, want time.Time }{
		{at(2026, 11, 2, 8), at(2026, 11, 2, 9)},  // Monday before 9
		{at(2026, 11, 2, 9), at(2026, 11, 9, 9)},  // exactly on it
		{at(2026, 11, 2, 10), at(2026, 11, 9, 9)}, // Monday after 9
		{at(2026, 11, 4, 12), at(2026, 11, 9, 9)}, // midweek
		{at(2026, 11, 8, 23), at(2026, 11, 9, 9)}, // Sunday night
		{at(2026, 12, 28, 10), at(2027, 1, 4, 9)}, // year rollover
	}
	for _, c := range cases {
		got := s(c.now)
		if !got.Equal(c.want) || got.Weekday() != time.Monday {
			t.Errorf("Weekly(Mon,9)(%v) = %v, want %v", c.now, got, c.want)
		}
	}
}
