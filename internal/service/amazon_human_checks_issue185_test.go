package service

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tests of issue #185 for behaviours of issue #168 that shipped without one: a scheduled Amazon check
// whose window has passed, and the one-time redraw of the Amazon check times after the upgrade.

// redrawMark reads the upgrade redraw mark (amazon_request_state.human_checks_redraw) of dir's database.
func redrawMark(t *testing.T, dir string) int {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var mark int
	if err := db.QueryRow("SELECT human_checks_redraw FROM amazon_request_state WHERE id = 1").Scan(&mark); err != nil {
		t.Fatal(err)
	}
	return mark
}

// setRedrawMark sets the upgrade redraw mark of dir's database, as the first start after issue #168 does.
func setRedrawMark(t *testing.T, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	result, err := db.Exec("UPDATE amazon_request_state SET human_checks_redraw = 1 WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		t.Fatalf("set the redraw mark: %d rows, %v", count, err)
	}
}

// An Amazon item due at 07:30 is not checked by the run at 20:00: it gets a time in the next morning
// window and the missed window is logged once. A control item due at 19:30 is checked.
// TC-001 (issue #185)
func TestAmazonMissedWindow_Issue185_TC001_SkippedRedrawnAndLogged(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	const missed, control = "B000001851", "B000001852"
	insertAt(t, database, missed, "amazon", scheduleDay(7, 30))
	insertAt(t, database, control, "amazon", scheduleDay(19, 30))
	now := newClock(scheduleDay(20, 0))
	recorder := &amazonRecorder{}
	service := startHuman(t, database, dir, now, recorder, 185)
	logs := captureLog(t)
	ctx := context.Background()

	service.collectDue(ctx)
	if recorder.calls(missed) != 0 {
		t.Fatalf("the item whose morning window has passed was checked at 20:00: %v", recorder.ids)
	}
	if recorder.calls(control) != 1 {
		t.Fatalf("the item due at 19:30 was not checked at 20:00: %v", recorder.ids)
	}
	next := nextCheck(t, database, missed)
	if next.Equal(scheduleDay(7, 30)) || windowOf(next, paris) != morning || localDate(next, paris) != "2030-03-06" {
		t.Fatalf("next check of the missed item at %s, want the morning window of 2030-03-06", next.In(paris))
	}
	line := fmt.Sprintf("scheduled Amazon check of item %s missed its window; next check %s", missed, next.UTC().Format(time.RFC3339))
	if !strings.Contains(logs.String(), line) {
		t.Fatalf("log has no line %q:\n%s", line, logs.String())
	}
	if count := strings.Count(logs.String(), "missed its window"); count != 1 {
		t.Fatalf("%d missed-window lines, want 1 (none for the control item):\n%s", count, logs.String())
	}

	service.collectDue(ctx)
	if recorder.calls(missed) != 0 {
		t.Fatalf("the missed item was checked by the second run at 20:00: %v", recorder.ids)
	}
	if again := nextCheck(t, database, missed); !again.Equal(next) {
		t.Fatalf("the second run moved the missed item's next check from %s to %s", next.In(paris), again.In(paris))
	}
	if count := strings.Count(logs.String(), "missed its window"); count != 1 {
		t.Fatalf("%d missed-window lines after the second run, want 1:\n%s", count, logs.String())
	}
}

// With the upgrade redraw mark set, the start draws an evening time for every Amazon item that held
// the former shared time 08:00 of the next day, then clears the mark; a second start redraws nothing.
// TC-002 (issue #185)
func TestAmazonUpgradeRedraw_Issue185_TC002_EveryItemRedrawnOnce(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ids := []string{"B000001853", "B000001854", "B000001855"}
	former := localTime(paris, 2030, time.March, 6, 8, 0)
	for _, id := range ids {
		insertAt(t, database, id, "amazon", former)
	}
	database.Close()
	setRedrawMark(t, dir)
	database = openHumanStore(t, dir)
	if mark := redrawMark(t, dir); mark != 1 {
		t.Fatalf("redraw mark %d before the start, want 1", mark)
	}
	for _, id := range ids {
		if next := nextCheck(t, database, id); !next.Equal(former) {
			t.Fatalf("next check of %s at %s before the start, want %s", id, next.In(paris), former.In(paris))
		}
	}

	service := startHuman(t, database, dir, newClock(scheduleDay(10, 0)), &amazonRecorder{}, 1851)
	drawn := map[string]time.Time{}
	for _, id := range ids {
		next := nextCheck(t, database, id)
		if next.Equal(former) || windowOf(next, paris) != evening || localDate(next, paris) != "2030-03-05" {
			t.Fatalf("next check of %s at %s after the start, want the evening window of 2030-03-05", id, next.In(paris))
		}
		drawn[id] = next
	}
	// FR-AMZ-HCHK-005: each item's time is drawn on its own, so no two items share one (issue #260).
	for i, id := range ids {
		for _, other := range ids[i+1:] {
			if drawn[id].Equal(drawn[other]) {
				t.Fatalf("items %s and %s both have the next check %s after the start, want a time drawn for each item", id, other, drawn[id].In(paris))
			}
		}
	}
	if mark := redrawMark(t, dir); mark != 0 {
		t.Fatalf("redraw mark %d after the start, want 0", mark)
	}

	service.Close()
	database.Close()
	database = openHumanStore(t, dir)
	startHuman(t, database, dir, newClock(scheduleDay(10, 5)), &amazonRecorder{}, 1852)
	for _, id := range ids {
		if next := nextCheck(t, database, id); !next.Equal(drawn[id]) {
			t.Fatalf("the second start moved the next check of %s from %s to %s", id, drawn[id].In(paris), next.In(paris))
		}
	}
	if mark := redrawMark(t, dir); mark != 0 {
		t.Fatalf("redraw mark %d after the second start, want 0", mark)
	}
}
