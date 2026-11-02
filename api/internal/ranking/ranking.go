// Package ranking holds the feed ordering formula, shared by the request handlers and
// the scheduled job that keeps scores fresh.
package ranking

import (
	"math"
	"time"
)

// HotScore is a Hacker News style decay: reactions matter, but recency matters more.
// now is a parameter so callers (and tests) control the clock.
func HotScore(reactions int, createdAt, now time.Time) float64 {
	if createdAt.IsZero() {
		createdAt = now
	}
	hours := now.Sub(createdAt).Hours()
	if hours < 0 {
		hours = 0
	}
	return float64(reactions+1) / math.Pow(hours+2, 1.5)
}
