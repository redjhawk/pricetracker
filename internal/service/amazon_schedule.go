package service

import "time"

// nextAmazonCheckAt returns a random instant in the first Amazon window (06:00–09:00, 19:00–23:00
// in location) that starts after after (issue #168).
// TDD red stub: not implemented.
func nextAmazonCheckAt(after time.Time, location *time.Location, random func(int64) int64) time.Time {
	return time.Time{}
}

// nextCheckAt returns the next scheduled check of an item of platform after after.
// TDD red stub: not implemented.
func (s *Service) nextCheckAt(platform string, after time.Time) time.Time {
	return time.Time{}
}
