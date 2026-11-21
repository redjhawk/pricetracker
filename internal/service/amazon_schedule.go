package service

import "time"

// amazonWindows are the local windows of the scheduled Amazon checks, in minutes after midnight
// (06:00–09:00 and 19:00–23:00, issue #168).
var amazonWindows = []struct{ start, end int }{{6 * 60, 9 * 60}, {19 * 60, 23 * 60}}

// amazonWindowGrace lets a scheduled Amazon check delayed by the ticker, the pauses between checks
// or the gate queue run shortly after its window ends.
const amazonWindowGrace = 15 * time.Minute

// nextAmazonCheckAt returns a random instant, with one-second resolution, in the first Amazon window
// (in location) that starts after after. Window bounds are included.
func nextAmazonCheckAt(after time.Time, location *time.Location, random func(int64) int64) time.Time {
	local := after.In(location)
	for offset := 0; offset < 3; offset++ {
		for _, window := range amazonWindows {
			start := atLocalMinute(local, offset, window.start, location)
			if !start.After(after) {
				continue
			}
			end := atLocalMinute(local, offset, window.end, location)
			seconds := int64(end.Sub(start) / time.Second)
			return start.Add(time.Duration(random(seconds+1)) * time.Second).UTC()
		}
	}
	return after.Add(24 * time.Hour).UTC() // not reached: a window starts every day
}

// inAmazonWindow reports whether t is inside an Amazon window of its day, grace included.
func inAmazonWindow(t time.Time, location *time.Location) bool {
	local := t.In(location)
	for _, window := range amazonWindows {
		start := atLocalMinute(local, 0, window.start, location)
		end := atLocalMinute(local, 0, window.end, location).Add(amazonWindowGrace)
		if !t.Before(start) && !t.After(end) {
			return true
		}
	}
	return false
}

// atLocalMinute returns the instant at minute of the day days after the local date of day.
func atLocalMinute(day time.Time, days, minute int, location *time.Location) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day()+days, minute/60, minute%60, 0, 0, location)
}

// nextCheckAt returns the next scheduled check of an item of platform after after: a random time
// in the next Amazon window for Amazon, the next configured check time otherwise.
func (s *Service) nextCheckAt(platform string, after time.Time) time.Time {
	if platform == "amazon" {
		return nextAmazonCheckAt(after, s.config.Location, s.random)
	}
	return s.NextCheckAt(after)
}
