// Package jobs runs the API's recurring background work (a tiny in-process cron).
package jobs

import "time"

// Schedule returns the next moment a job should fire, strictly after now.
type Schedule func(now time.Time) time.Time

// Daily fires every day at the given hour (0-23) in now's location.
func Daily(hour int) Schedule {
	return func(now time.Time) time.Time {
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next
	}
}

// Weekly fires once a week, on the given weekday at the given hour.
func Weekly(day time.Weekday, hour int) Schedule {
	return func(now time.Time) time.Time {
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
		next = next.AddDate(0, 0, (int(day)-int(now.Weekday())+7)%7)
		if !next.After(now) {
			next = next.AddDate(0, 0, 7)
		}
		return next
	}
}
