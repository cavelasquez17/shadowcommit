package ranking

import (
	"testing"
	"time"
)

func TestHotScoreDecaysWithAge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	fresh := HotScore(10, now, now)
	dayOld := HotScore(10, now.Add(-24*time.Hour), now)
	weekOld := HotScore(10, now.Add(-7*24*time.Hour), now)
	if !(fresh > dayOld && dayOld > weekOld) {
		t.Fatalf("score must fall as a thread ages: fresh=%v day=%v week=%v", fresh, dayOld, weekOld)
	}
}

func TestHotScoreRewardsReactionsAtEqualAge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	created := now.Add(-3 * time.Hour)
	if HotScore(50, created, now) <= HotScore(1, created, now) {
		t.Fatal("more reactions at the same age must rank higher")
	}
}

func TestHotScoreHandlesZeroAndFutureTimes(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	if got := HotScore(0, time.Time{}, now); got != HotScore(0, now, now) {
		t.Errorf("zero createdAt should behave as brand new, got %v", got)
	}
	if got := HotScore(0, now.Add(time.Hour), now); got != HotScore(0, now, now) {
		t.Errorf("a future createdAt (clock skew) must not inflate the score, got %v", got)
	}
}
