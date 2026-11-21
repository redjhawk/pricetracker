package service

import (
	"context"
	"testing"
	"time"
)

// Amazon checks are sent only in the window they were due in, and the pre-#168 shared check times
// of Amazon items are redrawn once (issue #168, review round 1).

// TC-033
func TestAmazonHumanChecks_TC033_CheckMissedInAnEarlierWindowIsRedrawn(t *testing.T) {
	cases := []struct {
		name  string
		due   time.Time
		start time.Time
	}{
		{"due in the morning, started in the evening", scheduleDay(8, 30), scheduleDay(19, 5)},
		{"due three days ago, started in the morning", scheduleDay(21, 0).AddDate(0, 0, -3), scheduleDay(7, 0)},
		{"due yesterday evening, started in the morning", scheduleDay(21, 0).AddDate(0, 0, -1), scheduleDay(6, 0)},
	}
	for index, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			database := openHumanStore(t, dir)
			ids := []string{"B000000331", "B000000332", "B000000333"}
			for _, id := range ids {
				insertAt(t, database, id, "amazon", test.due)
			}
			now := newClock(test.start)
			recorder := &amazonRecorder{}
			service := startHuman(t, database, dir, now, recorder, uint64(330+index))
			service.collectDue(context.Background())
			if recorder.total() != 0 {
				t.Fatalf("checks sent outside the window they were due in: %v", recorder.ids)
			}
			for _, id := range ids {
				if next := nextCheck(t, database, id); !next.After(test.start) || windowOf(next, paris) < 0 {
					t.Fatalf("next check of %s at %s, want a later window", id, next.In(paris))
				}
			}
		})
	}

	t.Run("due in the window of now", func(t *testing.T) {
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertAt(t, database, "B000000334", "amazon", scheduleDay(19, 0))
		recorder := &amazonRecorder{}
		service := startHuman(t, database, dir, newClock(scheduleDay(23, 10)), recorder, 334)
		service.collectDue(context.Background())
		if recorder.calls("B000000334") != 1 {
			t.Fatalf("a check due in the current window (grace included) was not sent: %v", recorder.ids)
		}
	})
}

// TC-034
func TestAmazonHumanChecks_TC034_SharedCheckTimesAreRedrawnOnce(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ids := []string{"B000000341", "B000000342", "B000000343"}
	shared := scheduleDay(20, 0)
	for _, id := range ids {
		insertAt(t, database, id, "amazon", shared)
	}
	insertAt(t, database, "lbc", "leboncoin", shared)
	database.Close()
	dropRestartMarker(t, dir)

	database = openHumanStore(t, dir)
	now := newClock(scheduleDay(10, 0))
	service := startHuman(t, database, dir, now, &amazonRecorder{}, 34)
	drawn := map[time.Time]bool{}
	for _, id := range ids {
		next := nextCheck(t, database, id)
		if windowOf(next, paris) != evening || localDate(next, paris) != "2030-03-05" {
			t.Fatalf("next check of %s at %s, want the evening window", id, next.In(paris))
		}
		drawn[next] = true
	}
	if drawn[shared] || len(drawn) != len(ids) {
		t.Fatalf("the Amazon items kept a shared check time: %v", drawn)
	}
	if got := nextCheck(t, database, "lbc"); !got.Equal(shared) {
		t.Fatalf("the LeBoncoin check time changed to %s", got.In(paris))
	}

	kept := nextCheck(t, database, ids[0])
	service.Close()
	database.Close()
	database = openHumanStore(t, dir)
	startHuman(t, database, dir, now, &amazonRecorder{}, 35)
	if got := nextCheck(t, database, ids[0]); !got.Equal(kept) {
		t.Fatalf("a later start redrew the check time again: %s, want %s", got.In(paris), kept.In(paris))
	}
}
