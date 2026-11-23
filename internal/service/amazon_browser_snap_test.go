package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/store"
)

// Amazon checks of issue #226 through the service: a Chromium snap that cannot start is found first
// on PATH, and setting PRICEFOLLOWER_BROWSER_PATH to Google Chrome as documented makes the checks work.

const snapStartError = "cannot create user data directory: Permission denied"

// snapStubPath returns a directory holding only a chromium script that fails to start like a confined
// Chromium snap.
func snapStubPath(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	script := "#!/bin/sh\necho '" + snapStartError + "' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(directory, "chromium"), []byte(script), 0o755); err != nil { // #nosec G306 -- test executable
		t.Fatal(err)
	}
	return directory
}

// TC-004
func TestAmazonBrowserSnapDocs_TC004_BrowserPathSetAsDocumentedWorksDespiteTheSnap(t *testing.T) {
	browser := amazontest.Browser(t) // found before PATH changes
	_, server := newAmazonFixture(t, page(amazontest.ExpandedFixture(t, "mentions-captcha.html")))
	const id = "B0SNAP0226"
	const failedLine = "Amazon request failed"

	start := func(t *testing.T) (*Service, *store.Store, *clock, *retrievalLog) {
		t.Helper()
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertPricedAmazonItem(t, database, id, scheduleDay(20, 0))
		logs := captureRetrievalLog(t)
		now := newClock(scheduleDay(14, 0))
		return documentedService(t, database, dir, now, server.URL), database, now, logs
	}
	// refreshAt requests Refresh price at the time at.
	refreshAt := func(t *testing.T, service *Service, now *clock, at time.Time) {
		t.Helper()
		now.Set(at)
		refreshPrice(t, service, id)
	}
	assertChecked := func(t *testing.T, database *store.Store, logs *retrievalLog, at time.Time, failedLines int) {
		t.Helper()
		item := storedItem(t, database, id)
		if item.LastAttempt == nil || !item.LastAttempt.Timestamp.Equal(at) || item.LastAttempt.Result != "success" || item.LatestPrice == nil ||
			item.LatestPrice.AmountCents != 12999 {
			t.Fatalf("check at %s: attempt %+v, latest price %+v\n%s", at, item.LastAttempt, item.LatestPrice, shorten(logs.String()))
		}
		if got := amazonFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d after the success, want 0", got)
		}
		if got := strings.Count(logs.String(), failedLine); got != failedLines {
			t.Fatalf("%d failure lines, want %d:\n%s", got, failedLines, shorten(logs.String()))
		}
	}
	assertNotStarted := func(t *testing.T, database *store.Store, logs *retrievalLog, at time.Time, reason string) {
		t.Helper()
		item := storedItem(t, database, id)
		if item.LastAttempt == nil || item.LastAttempt.Result != "request_error" || item.Status != "retrieval_error" {
			t.Fatalf("check at %s: status %s, attempt %+v", at, item.Status, item.LastAttempt)
		}
		if item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
			t.Fatalf("the item lost its last successful price: %+v", item.LatestPrice)
		}
		if got := amazonFailures(t, database); got != 1 {
			t.Fatalf("consecutive failure count %d, want 1", got)
		}
		cause := logCause(t, checkLogLine(t, logs.String(), id, at))
		if !strings.Contains(cause, "browser could not start") || !strings.Contains(cause, reason) {
			t.Fatalf("logged cause %q does not say the browser could not start with %q", cause, reason)
		}
	}

	t.Run("snap found first, then PRICEFOLLOWER_BROWSER_PATH set without a restart", func(t *testing.T) {
		t.Setenv("PATH", snapStubPath(t))
		t.Setenv(amazon.BrowserPathVariable, "")
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		service, database, now, logs := start(t)

		// (a) The snap is used and cannot start.
		refreshAt(t, service, now, scheduleDay(14, 0))
		assertNotStarted(t, database, logs, scheduleDay(14, 0), snapStartError)

		// (b) The setting points to the test browser, standing for Google Chrome.
		t.Setenv(amazon.BrowserPathVariable, browser)
		refreshAt(t, service, now, scheduleDay(14, 5))
		assertChecked(t, database, logs, scheduleDay(14, 5), 1)

		// Refresh price again.
		refreshAt(t, service, now, scheduleDay(14, 10))
		assertChecked(t, database, logs, scheduleDay(14, 10), 1)
	})

	t.Run("only google-chrome-stable on PATH and the setting unset", func(t *testing.T) {
		directory := t.TempDir()
		if err := os.Symlink(browser, filepath.Join(directory, "google-chrome-stable")); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", directory)
		t.Setenv(amazon.BrowserPathVariable, "")
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		service, database, now, logs := start(t)
		refreshAt(t, service, now, scheduleDay(14, 0))
		assertChecked(t, database, logs, scheduleDay(14, 0), 0)
	})

	t.Run("setting already set with the snap on PATH", func(t *testing.T) {
		t.Setenv("PATH", snapStubPath(t))
		t.Setenv(amazon.BrowserPathVariable, browser)
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		service, database, now, logs := start(t)
		refreshAt(t, service, now, scheduleDay(14, 0))
		assertChecked(t, database, logs, scheduleDay(14, 0), 0)
	})

	t.Run("setting pointing to a missing file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "google-chrome-stable")
		t.Setenv("PATH", snapStubPath(t))
		t.Setenv(amazon.BrowserPathVariable, missing)
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		service, database, now, logs := start(t)
		refreshAt(t, service, now, scheduleDay(14, 0))
		assertNotStarted(t, database, logs, scheduleDay(14, 0), missing)
	})
}
