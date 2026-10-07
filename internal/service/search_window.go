package service

import "time"

// Amazon search items are read only in the Paris windows [22:00, 01:00) and [06:00, 08:00).
var parisLocation = mustLoadLocation("Europe/Paris")

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return location
}

func inSearchWindow(t time.Time) bool {
	local := t.In(parisLocation)
	minutes := local.Hour()*60 + local.Minute()
	return minutes >= 22*60 || minutes < 1*60 || (minutes >= 6*60 && minutes < 8*60)
}

// nextSearchWindowStart returns the first 06:00 or 22:00 Paris time strictly after t, in UTC.
func nextSearchWindowStart(t time.Time) time.Time {
	local := t.In(parisLocation)
	for day := 0; day < 2; day++ {
		date := local.AddDate(0, 0, day)
		for _, hour := range []int{6, 22} {
			start := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, parisLocation)
			if start.After(t) {
				return start.UTC()
			}
		}
	}
	return t.Add(24 * time.Hour) // not reached
}
