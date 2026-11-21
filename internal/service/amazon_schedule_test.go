package service

import (
	"context"
	mathrand "math/rand/v2"
	"sync"
	"testing"
	"time"
	_ "time/tzdata" // Europe/Paris on every machine

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Amazon schedule of issue #168: one check at a random time in each window (06:00–09:00 and
// 19:00–23:00 in the schedule timezone) per day.

var paris = loadLocation("Europe/Paris")

func loadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return location
}

func localTime(location *time.Location, year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, location)
}

func localDate(t time.Time, location *time.Location) string {
	return t.In(location).Format("2006-01-02")
}

// seededRandom is a deterministic, goroutine-safe random source for Options.Random.
func seededRandom(seed uint64) func(int64) int64 {
	var mu sync.Mutex
	source := mathrand.New(mathrand.NewPCG(seed, seed+1))
	return func(n int64) int64 {
		mu.Lock()
		defer mu.Unlock()
		return source.Int64N(n)
	}
}

const (
	morning = 0
	evening = 1
)

// windowOf returns the Amazon window holding t in location (bounds included), or -1.
func windowOf(t time.Time, location *time.Location) int {
	local := t.In(location)
	seconds := local.Hour()*3600 + local.Minute()*60 + local.Second()
	switch {
	case seconds >= 6*3600 && seconds <= 9*3600:
		return morning
	case seconds >= 19*3600 && seconds <= 23*3600:
		return evening
	}
	return -1
}

func secondsOfDay(t time.Time, location *time.Location) int {
	local := t.In(location)
	return local.Hour()*3600 + local.Minute()*60 + local.Second()
}

func allEqual(values []int) bool {
	for _, value := range values {
		if value != values[0] {
			return false
		}
	}
	return true
}

// TC-009
func TestAmazonSchedule_TC009_TwoChecksPerDayOneInEachWindow(t *testing.T) {
	start := localTime(paris, 2026, time.November, 23, 0, 0)
	end := start.AddDate(0, 0, 7)
	random := seededRandom(9)
	perDay := map[string][2]int{}
	count := 0
	for after := start; ; count++ {
		next := nextAmazonCheckAt(after, paris, random)
		if !next.After(after) {
			t.Fatalf("next check %s is not after %s", next, after)
		}
		if next.Location() != time.UTC {
			t.Errorf("next check %s is not stored in UTC", next)
		}
		if !next.Before(end) {
			break
		}
		window := windowOf(next, paris)
		if window < 0 {
			t.Fatalf("check at %s (Paris) is outside the windows", next.In(paris))
		}
		checks := perDay[localDate(next, paris)]
		checks[window]++
		perDay[localDate(next, paris)] = checks
		after = next
		if count > 20 {
			t.Fatal("too many checks")
		}
	}
	if count != 14 || len(perDay) != 7 {
		t.Fatalf("%d checks over %d days, want 14 over 7: %v", count, len(perDay), perDay)
	}
	for date, checks := range perDay {
		if checks != [2]int{1, 1} {
			t.Errorf("%s: %d morning and %d evening checks", date, checks[morning], checks[evening])
		}
	}

	t.Run("extreme random values stay inside the windows", func(t *testing.T) {
		minimum := func(int64) int64 { return 0 }
		first := nextAmazonCheckAt(start, paris, minimum)
		second := nextAmazonCheckAt(first, paris, minimum)
		third := nextAmazonCheckAt(second, paris, minimum)
		for index, test := range []struct{ got, want time.Time }{
			{first, localTime(paris, 2026, time.November, 23, 6, 0)},
			{second, localTime(paris, 2026, time.November, 23, 19, 0)},
			{third, localTime(paris, 2026, time.November, 24, 6, 0)},
		} {
			if !test.got.Equal(test.want) {
				t.Errorf("minimum value, check %d at %s, want the window start %s", index+1, test.got.In(paris), test.want)
			}
		}
		maximum := func(n int64) int64 { return n - 1 }
		first = nextAmazonCheckAt(start, paris, maximum)
		second = nextAmazonCheckAt(first, paris, maximum)
		third = nextAmazonCheckAt(second, paris, maximum)
		for index, test := range []struct {
			got      time.Time
			window   int
			date     string
			earliest time.Time
		}{
			{first, morning, "2026-11-23", localTime(paris, 2026, time.November, 23, 8, 59)},
			{second, evening, "2026-11-23", localTime(paris, 2026, time.November, 23, 22, 59)},
			{third, morning, "2026-11-24", localTime(paris, 2026, time.November, 24, 8, 59)},
		} {
			if windowOf(test.got, paris) != test.window || localDate(test.got, paris) != test.date || test.got.Before(test.earliest) {
				t.Errorf("maximum value, check %d at %s, want the end of the window", index+1, test.got.In(paris))
			}
		}
	})
}

// TC-010
func TestAmazonSchedule_TC010_TimesChangeFromDayToDay(t *testing.T) {
	random := seededRandom(10)
	var mornings, evenings []int
	after := localTime(paris, 2026, time.November, 23, 0, 0)
	for len(evenings) < 7 {
		next := nextAmazonCheckAt(after, paris, random)
		if !next.After(after) || len(mornings)+len(evenings) > 14 {
			t.Fatalf("next check %s after %s", next, after)
		}
		switch windowOf(next, paris) {
		case morning:
			mornings = append(mornings, secondsOfDay(next, paris))
		case evening:
			evenings = append(evenings, secondsOfDay(next, paris))
		default:
			t.Fatalf("check at %s is outside the windows", next.In(paris))
		}
		after = next
	}
	if allEqual(mornings) || allEqual(evenings) {
		t.Fatalf("the same time every day: mornings %v, evenings %v", mornings, evenings)
	}
	wholeHours := true
	for _, seconds := range append(mornings, evenings...) {
		if seconds%3600 != 0 {
			wholeHours = false
		}
	}
	if wholeHours {
		t.Fatalf("only whole hours are drawn: %v %v", mornings, evenings)
	}

	t.Run("the time follows the random value", func(t *testing.T) {
		calls := int64(0)
		fraction := func(n int64) int64 { // 0, 1/10, 2/10, … of the window
			k := calls % 10
			calls++
			return (n - 1) * k / 10
		}
		after := localTime(paris, 2026, time.November, 23, 0, 0)
		for index := 0; index < 10; index++ {
			next := nextAmazonCheckAt(after, paris, fraction)
			local := next.In(paris)
			start, length := 6*time.Hour, 3*time.Hour
			switch windowOf(next, paris) {
			case morning:
			case evening:
				start, length = 19*time.Hour, 4*time.Hour
			default:
				t.Fatalf("check at %s is outside the windows", local)
			}
			midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, paris)
			offset, want := next.Sub(midnight)-start, length*time.Duration(index)/10
			if difference := offset - want; difference < -time.Minute || difference > time.Minute {
				t.Errorf("draw %d at %s: %s into the window, want %s", index, local, offset, want)
			}
			after = next
		}
	})
}

// TC-011
func TestAmazonSchedule_TC011_ConfiguredCheckTimesApplyToLeboncoinOnly(t *testing.T) {
	newService := func(checkTimes []string) *Service {
		t.Helper()
		cfg := config.Config{DataDirectory: t.TempDir(), Location: paris, CheckTimes: checkTimes, StaleAfter: 36 * time.Hour, UserAgent: "test"}
		database, err := store.Open(cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { database.Close() })
		service := NewWithOptions(cfg, database, Options{Amazon: fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { return succeeded() }),
			Random: func(int64) int64 { return 0 }})
		t.Cleanup(service.Close)
		return service
	}
	ten := localTime(paris, 2026, time.November, 24, 10, 0)

	configured := newService([]string{"12:00", "15:00"})
	if next := configured.nextCheckAt("amazon", ten); windowOf(next, paris) != evening || localDate(next, paris) != "2026-11-24" {
		t.Errorf("Amazon next check at %s, want the evening window of the same day", next.In(paris))
	}
	if next := configured.nextCheckAt("leboncoin", ten); !next.Equal(localTime(paris, 2026, time.November, 24, 12, 0)) {
		t.Errorf("LeBoncoin next check at %s, want 12:00", next.In(paris))
	}

	defaults := newService([]string{"08:00", "20:00"})
	for _, test := range []struct{ after, want time.Time }{
		{ten, localTime(paris, 2026, time.November, 24, 20, 0)},
		{localTime(paris, 2026, time.November, 24, 21, 0), localTime(paris, 2026, time.November, 25, 8, 0)},
	} {
		if next := defaults.nextCheckAt("leboncoin", test.after); !next.Equal(test.want) {
			t.Errorf("LeBoncoin next check after %s at %s, want %s", test.after, next.In(paris), test.want)
		}
	}
	if next := defaults.nextCheckAt("amazon", localTime(paris, 2026, time.November, 24, 21, 0)); windowOf(next, paris) != morning || localDate(next, paris) != "2026-11-25" {
		t.Errorf("Amazon next check at %s, want the next morning window", next.In(paris))
	}
}

// TC-012
func TestAmazonSchedule_TC012_NextWindowFromAnyTimeOfDay(t *testing.T) {
	random := seededRandom(12)
	tests := []struct {
		after  time.Time
		window int
		date   string
	}{
		{localTime(paris, 2026, time.November, 24, 5, 0), morning, "2026-11-24"},
		{localTime(paris, 2026, time.November, 24, 7, 30), evening, "2026-11-24"},
		{localTime(paris, 2026, time.November, 24, 14, 0), evening, "2026-11-24"},
		{localTime(paris, 2026, time.November, 24, 20, 30), morning, "2026-11-25"},
		{localTime(paris, 2026, time.November, 24, 23, 30), morning, "2026-11-25"},
		{localTime(paris, 2026, time.November, 30, 23, 30), morning, "2026-12-01"},
		{localTime(paris, 2026, time.December, 31, 23, 30), morning, "2027-01-01"},
	}
	for _, test := range tests {
		next := nextAmazonCheckAt(test.after, paris, random)
		if !next.After(test.after) || windowOf(next, paris) != test.window || localDate(next, paris) != test.date {
			t.Errorf("after %s: next check at %s, want window %d of %s", test.after, next.In(paris), test.window, test.date)
		}
	}
}

// TC-013
func TestAmazonSchedule_TC013_DaylightSavingDays(t *testing.T) {
	tests := []struct {
		name     string
		location *time.Location
		day      time.Time
	}{
		{"spring clock change", paris, localTime(paris, 2027, time.March, 28, 0, 0)},
		{"autumn clock change", paris, localTime(paris, 2026, time.October, 25, 0, 0)},
		{"UTC timezone", time.UTC, localTime(time.UTC, 2026, time.October, 25, 0, 0)},
	}
	sources := map[string]func(int64) int64{
		"seeded":  seededRandom(13),
		"minimum": func(int64) int64 { return 0 },
		"maximum": func(n int64) int64 { return n - 1 },
	}
	for _, test := range tests {
		for name, random := range sources {
			t.Run(test.name+" "+name, func(t *testing.T) {
				date, nextDate := test.day.Format("2006-01-02"), test.day.AddDate(0, 0, 1).Format("2006-01-02")
				first := nextAmazonCheckAt(test.day, test.location, random)
				second := nextAmazonCheckAt(first, test.location, random)
				third := nextAmazonCheckAt(second, test.location, random)
				for index, check := range []struct {
					got    time.Time
					window int
					date   string
				}{{first, morning, date}, {second, evening, date}, {third, morning, nextDate}} {
					if windowOf(check.got, test.location) != check.window || localDate(check.got, test.location) != check.date {
						t.Errorf("check %d at %s, want window %d of %s", index+1, check.got.In(test.location), check.window, check.date)
					}
				}
			})
		}
	}
}

// TC-014
func TestAmazonSchedule_TC014_EachItemDrawsItsOwnTime(t *testing.T) {
	spread := func(t *testing.T, times []time.Time) {
		t.Helper()
		distinct := map[int64]bool{}
		earliest, latest := times[0], times[0]
		for _, next := range times {
			if windowOf(next, paris) != morning || localDate(next, paris) != localDate(times[0], paris) {
				t.Fatalf("next check at %s, want the morning window", next.In(paris))
			}
			distinct[next.Unix()] = true
			if next.Before(earliest) {
				earliest = next
			}
			if next.After(latest) {
				latest = next
			}
		}
		if len(distinct) < 2 || latest.Sub(earliest) <= 30*time.Minute {
			t.Fatalf("%d distinct times spanning %s: the items share one time", len(distinct), latest.Sub(earliest))
		}
	}

	t.Run("schedule function", func(t *testing.T) {
		random := seededRandom(14)
		from := localTime(paris, 2026, time.November, 24, 5, 0)
		times := make([]time.Time, 0, 10)
		for range 10 {
			times = append(times, nextAmazonCheckAt(from, paris, random))
		}
		spread(t, times)
		if single := nextAmazonCheckAt(from, paris, seededRandom(1)); windowOf(single, paris) != morning || localDate(single, paris) != "2026-11-24" {
			t.Fatalf("single item next check at %s", single.In(paris))
		}
	})

	t.Run("items added at the same time", func(t *testing.T) {
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		now := newClock(scheduleDay(5, 0))
		recorder := &amazonRecorder{}
		service := startHuman(t, database, dir, now, recorder, 14)
		ids := make([]string, 0, 10)
		for index := range 10 {
			item, err := service.Add(context.Background(), "https://www.amazon.fr/dp/B00000014"+string(rune('0'+index)), "")
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, item.ID)
		}
		waitFor(t, "the first checks after add", func() bool { return recorder.total() >= len(ids) })
		waitIdle(t, service)
		times := make([]time.Time, 0, len(ids))
		for _, id := range ids {
			times = append(times, nextCheck(t, database, id))
		}
		spread(t, times)
	})
}
