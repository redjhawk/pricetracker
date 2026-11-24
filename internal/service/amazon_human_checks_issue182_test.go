package service

import (
	"context"
	"testing"
)

// Tests of issue #182 for a behaviour of issue #168 that shipped without one: an Amazon item stopped at
// its scheduled time and restarted by a Refresh price is not checked again by the next scheduled run.
// TC-001 to TC-003 are in TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning; TC-002, TC-005
// and TC-006 are checks recorded in the change folder's no-tests.md.

// TC-004
func TestAmazonRestartAfterStop_Issue182_TC004_NoScheduledCheckAfterRefreshPrice(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ctx := context.Background()
	const id = "B000001821"
	insertAt(t, database, id, "amazon", scheduleDay(7, 0))
	if err := database.StopAmazonItems(ctx, scheduleDay(7, 0)); err != nil {
		t.Fatal(err)
	}
	now := newClock(scheduleDay(7, 0))
	recorder := &amazonRecorder{}
	service := startHuman(t, database, dir, now, recorder, 182)
	if !stopped(t, database, id) {
		t.Fatal("the item is not stopped after the start")
	}
	if next := nextCheck(t, database, id); !next.Equal(scheduleDay(7, 0)) {
		t.Fatalf("next check at %s after the start, want 07:00", next.In(paris))
	}

	service.collectDue(ctx)
	if got := recorder.total(); got != 0 {
		t.Fatalf("the 07:00 run made %d checks of the stopped item, want 0", got)
	}
	if !stopped(t, database, id) {
		t.Fatal("the item is not stopped after the 07:00 run")
	}
	if next := nextCheck(t, database, id); !next.Equal(scheduleDay(7, 0)) {
		t.Fatalf("next check at %s after the 07:00 run, want 07:00", next.In(paris))
	}

	now.Set(scheduleDay(7, 30))
	if _, err := service.RefreshItem(ctx, id); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	if stopped(t, database, id) {
		t.Fatal("the item is still stopped after the Refresh price")
	}
	if got := recorder.calls(id); got != 1 {
		t.Fatalf("%d checks after the Refresh price, want 1", got)
	}

	now.Set(scheduleDay(7, 31))
	service.collectDue(ctx)
	service.collectDue(ctx)
	if got, total := recorder.calls(id), recorder.total(); got != 1 || total != 1 {
		t.Fatalf("%d checks of the item (%d in total) after the 07:31 runs, want 1", got, total)
	}
	if next := nextCheck(t, database, id); windowOf(next, paris) != evening || localDate(next, paris) != "2030-03-05" {
		t.Fatalf("next check at %s, want the evening window of 2030-03-05", next.In(paris))
	}
}
