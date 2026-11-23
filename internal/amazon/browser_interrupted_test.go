package amazon

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
)

// Tests of issue #233: a check interrupted by its time limit or a server stop still clears the
// browser's HTTP cache, and waits for the browser to confirm it before closing it, so an answer offered
// as a download never stays in the profile.

const (
	interruptionTimeLimit  = "time limit"
	interruptionServerStop = "server stop"
	cacheLogLine           = "Amazon browser cache could not be cleared"
	clearMethod            = "Network.clearBrowserCache"
)

// interruptedContext returns the context of a check that ends at interruptAt: through its deadline
// (time limit) or through a cancellation (server stop).
func interruptedContext(kind string, interruptAt time.Time) (context.Context, context.CancelFunc) {
	if kind == interruptionTimeLimit {
		return context.WithDeadline(context.Background(), interruptAt)
	}
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(time.Until(interruptAt), cancel)
	return ctx, func() { timer.Stop(); cancel() }
}

// collectInterrupted runs the check of item with ctx and fails t unless it returns within 20 s of interruptAt.
func collectInterrupted(t *testing.T, collector *BrowserCollector, ctx context.Context, item model.Listing, interruptAt time.Time) model.CollectionResult {
	t.Helper()
	done := make(chan model.CollectionResult, 1)
	go func() { done <- collector.Collect(ctx, item) }()
	select {
	case got := <-done:
		return got
	case <-time.After(time.Until(interruptAt.Add(20 * time.Second))):
		t.Fatal("the interrupted check did not return within 20 s of the interruption")
		return model.CollectionResult{}
	}
}

// assertInterruptedFailure checks that got is the failure result of a check.
func assertInterruptedFailure(t *testing.T, got model.CollectionResult) {
	t.Helper()
	if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
		t.Fatalf("not a failure: %+v", got)
	}
}

// cacheLines returns the lines of output reporting a cache clear that failed.
func cacheLines(output string) []string {
	var found []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, cacheLogLine) {
			found = append(found, line)
		}
	}
	return found
}

func assertNoCacheLogLine(t *testing.T, output string) {
	t.Helper()
	if lines := cacheLines(output); len(lines) > 0 {
		t.Fatalf("cache clear failure logged:\n%s", strings.Join(lines, "\n"))
	}
}

// assertNoInterruptionGivenForCache checks that no line about the cache gives the end of the check's
// context as its cause.
func assertNoInterruptionGivenForCache(t *testing.T, output string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(strings.ToLower(line), "cache") && (strings.Contains(line, "context canceled") || strings.Contains(line, "context deadline exceeded")) {
			t.Fatalf("the interruption of the check is given as the cause of a cache clear: %s", line)
		}
	}
}

// assertNavigatedBeforeInterruption fails t when the fake browser received Page.navigate after
// interruptAt: the interruption point of the test was then missed.
func assertNavigatedBeforeInterruption(t *testing.T, record fakeRecord, interruptAt time.Time) int {
	t.Helper()
	index, at := record.first("Page.navigate", "received")
	if index < 0 {
		t.Fatalf("the fake browser did not receive Page.navigate:\n%s", record)
	}
	if at.After(interruptAt) {
		t.Fatalf("Page.navigate received at %s, after the interruption at %s: the interruption point was missed:\n%s",
			at.Format("15:04:05.000"), interruptAt.Format("15:04:05.000"), record)
	}
	return index
}

// assertClearConfirmedBeforeClose checks that the fake browser answered the cache clear before it
// received Browser.close, and returns the time it received the clear.
func assertClearConfirmedBeforeClose(t *testing.T, record fakeRecord) time.Time {
	t.Helper()
	received, receivedAt := record.first(clearMethod, "received")
	if received < 0 {
		t.Fatalf("the browser cache was not cleared:\n%s", record)
	}
	answered, _ := record.first(clearMethod, "answered")
	if answered < 0 {
		t.Fatalf("the browser was closed before it answered the cache clear:\n%s", record)
	}
	closed, _ := record.first("Browser.close", "received")
	if closed < 0 {
		t.Fatalf("the browser was not closed:\n%s", record)
	}
	if closed < answered {
		t.Fatalf("Browser.close sent before the answer to the cache clear:\n%s", record)
	}
	return receivedAt
}

// A check interrupted while waiting after the download was offered clears the cache and closes the
// browser only once the clear is confirmed.
// TC-001
func TestBrowserInterrupted_TC001_ClearConfirmedBeforeClose(t *testing.T) {
	item := fixtureItem("B0PRICE001")
	for _, kind := range []string{interruptionTimeLimit, interruptionServerStop} {
		t.Run(kind, func(t *testing.T) {
			collector := NewBrowserCollector(fakeBrowserOptions(t))
			for check := 1; check <= 2; check++ {
				interruptAt := time.Now().Add(2 * time.Second)
				recordPath := useFakeBrowser(t, fakeBrowserScript{SetUpUntil: interruptAt.Add(-200 * time.Millisecond).UnixNano(),
					Navigate: "error", Clear: "answer", ClearDelay: 300 * time.Millisecond})
				ctx, cancel := interruptedContext(kind, interruptAt)
				output := captureOutput(t)
				got := collectInterrupted(t, collector, ctx, item, interruptAt)
				logged := output()
				cancel()
				record := readFakeRecord(t, recordPath)
				navigated := assertNavigatedBeforeInterruption(t, record, interruptAt)
				if received, _ := record.first(clearMethod, "received"); received >= 0 && received < navigated {
					t.Fatalf("check %d: cache cleared before Page.navigate:\n%s", check, record)
				}
				assertClearConfirmedBeforeClose(t, record)
				if count := record.count(clearMethod, "received"); count != 1 {
					t.Fatalf("check %d: %d cache clears, want 1:\n%s", check, count, record)
				}
				assertInterruptedFailure(t, got)
				assertNoCacheLogLine(t, logged)
				assertNoInterruptionGivenForCache(t, logged)
			}
		})
	}
}

// A check interrupted while the page is still being requested clears the cache before the browser closes.
// TC-002
func TestBrowserInterrupted_TC002_PageStillRequestedIsCleared(t *testing.T) {
	item := fixtureItem("B0PRICE001")
	for _, test := range []struct {
		name, kind, navigate string
	}{
		{"navigation held, " + interruptionTimeLimit, interruptionTimeLimit, "hold"},
		{"navigation held, " + interruptionServerStop, interruptionServerStop, "hold"},
		{"late navigation answer, " + interruptionServerStop, interruptionServerStop, "late"},
	} {
		t.Run(test.name, func(t *testing.T) {
			interruptAt := time.Now().Add(2 * time.Second)
			recordPath := useFakeBrowser(t, fakeBrowserScript{SetUpUntil: interruptAt.Add(-200 * time.Millisecond).UnixNano(),
				Navigate: test.navigate, NavigateAt: interruptAt.Add(300 * time.Millisecond).UnixNano(), Clear: "answer", ClearDelay: 300 * time.Millisecond})
			ctx, cancel := interruptedContext(test.kind, interruptAt)
			defer cancel()
			output := captureOutput(t)
			got := collectInterrupted(t, NewBrowserCollector(fakeBrowserOptions(t)), ctx, item, interruptAt)
			logged := output()
			record := readFakeRecord(t, recordPath)
			assertNavigatedBeforeInterruption(t, record, interruptAt)
			if clearedAt := assertClearConfirmedBeforeClose(t, record); clearedAt.Before(interruptAt) {
				t.Fatalf("cache cleared at %s, before the interruption at %s:\n%s", clearedAt.Format("15:04:05.000"), interruptAt.Format("15:04:05.000"), record)
			}
			if count := record.count(clearMethod, "received"); count != 1 {
				t.Fatalf("%d cache clears, want 1:\n%s", count, record)
			}
			assertInterruptedFailure(t, got)
			assertNoCacheLogLine(t, logged)
		})
	}
}

// A clear that never answers or fails does not hold the check, and is logged with its real cause.
// TC-003
func TestBrowserInterrupted_TC003_UnconfirmedClearIsBoundedAndLogged(t *testing.T) {
	item := fixtureItem("B0PRICE001")
	for _, test := range []struct {
		name, navigate, clear, cause string
	}{
		{"no answer", "error", "never", ""},
		{"error answer", "error", "error", "cache clear refused by fake"},
		{"no answer, page still requested", "hold", "never", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			interruptAt := time.Now().Add(2 * time.Second)
			recordPath := useFakeBrowser(t, fakeBrowserScript{SetUpUntil: interruptAt.Add(-200 * time.Millisecond).UnixNano(),
				Navigate: test.navigate, Clear: test.clear, ClearError: test.cause})
			ctx, cancel := interruptedContext(interruptionServerStop, interruptAt)
			defer cancel()
			output := captureOutput(t)
			got := collectInterrupted(t, NewBrowserCollector(fakeBrowserOptions(t)), ctx, item, interruptAt)
			logged := output()
			record := readFakeRecord(t, recordPath)
			assertNavigatedBeforeInterruption(t, record, interruptAt)
			if index, _ := record.first("Browser.close", "received"); index < 0 {
				t.Fatalf("the browser was not closed:\n%s", record)
			}
			assertInterruptedFailure(t, got)
			lines := cacheLines(logged)
			if len(lines) != 1 {
				t.Fatalf("%d cache log lines, want 1:\n%s", len(lines), logged)
			}
			if test.cause != "" && !strings.Contains(lines[0], test.cause) {
				t.Fatalf("cache log line without the browser's error %q: %s", test.cause, lines[0])
			}
			if strings.Contains(lines[0], "context canceled") {
				t.Fatalf("the interruption of the check is given as the cause of the cache clear failure: %s", lines[0])
			}
		})
	}
}

// A real check interrupted around a download leaves no file in the profile or the Downloads folder.
// TC-004
func TestBrowserInterrupted_TC004_RealCheckInterruptedAroundADownloadWritesNoFile(t *testing.T) {
	const (
		whileRequested int32 = iota
		afterOffered
		normal
	)
	page := amazontest.Fixture(t, "price-found.html")
	item := fixtureItem("B0PRICE001")
	for _, test := range []struct {
		name             string
		mode             int32
		kind             string
		downloadsExisted bool
	}{
		{"while requested, " + interruptionTimeLimit, whileRequested, interruptionTimeLimit, false},
		{"while requested, " + interruptionServerStop, whileRequested, interruptionServerStop, false},
		{"after the download was offered, " + interruptionTimeLimit, afterOffered, interruptionTimeLimit, false},
		{"after the download was offered, " + interruptionServerStop, afterOffered, interruptionServerStop, false},
		{"after the download was offered, existing Downloads folder, " + interruptionServerStop, afterOffered, interruptionServerStop, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mode atomic.Int32
			mode.Store(test.mode)
			var interruptAt atomic.Int64 // Unix nanoseconds
			var mu sync.Mutex
			var arrivedAt, flushedAt time.Time
			writeDownload := func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Content-Disposition", `attachment; filename="page.html"`)
				io.WriteString(w, page+amazontest.DownloadMarker)
			}
			waitUntil := func(r *http.Request, at time.Time) bool {
				timer := time.NewTimer(time.Until(at))
				defer timer.Stop()
				select {
				case <-timer.C:
					return true
				case <-r.Context().Done():
					return false
				}
			}
			server := navigationServer(t, func(w http.ResponseWriter, r *http.Request) {
				current := mode.Load()
				if current == normal {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					io.WriteString(w, page)
					return
				}
				mu.Lock()
				if arrivedAt.IsZero() {
					arrivedAt = time.Now()
				}
				mu.Unlock()
				interruption := time.Unix(0, interruptAt.Load())
				if current == whileRequested {
					if waitUntil(r, interruption.Add(100*time.Millisecond)) {
						writeDownload(w)
					}
					return
				}
				if !waitUntil(r, interruption.Add(-100*time.Millisecond)) {
					return
				}
				writeDownload(w)
				http.NewResponseController(w).Flush()
				mu.Lock()
				if flushedAt.IsZero() {
					flushedAt = time.Now()
				}
				mu.Unlock()
				<-r.Context().Done()
			})
			options := testBrowserOptions(t, server.URL)
			home := amazontest.IsolatedHome(t)
			if test.downloadsExisted {
				if err := os.Mkdir(filepath.Join(home, "Downloads"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			assertNoFileWritten := func() {
				t.Helper()
				amazontest.AssertNoDownload(t, []string{"page.html"}, options.ProfileDir, home)
				amazontest.AssertDownloadsFolder(t, home, test.downloadsExisted)
			}
			collector := NewBrowserCollector(options)

			at := time.Now().Add(5 * time.Second)
			interruptAt.Store(at.UnixNano())
			ctx, cancel := interruptedContext(test.kind, at)
			output := captureOutput(t)
			got := collectInterrupted(t, collector, ctx, item, at)
			logged := output()
			cancel()
			mu.Lock()
			arrived, flushed := arrivedAt, flushedAt
			mu.Unlock()
			if arrived.IsZero() || arrived.After(at) {
				t.Fatalf("product page requested at %v, want before the interruption at %s: the interruption point was missed", arrived, at.Format("15:04:05.000"))
			}
			if test.mode == afterOffered && (flushed.IsZero() || flushed.After(at)) {
				t.Fatalf("download flushed at %v, want before the interruption at %s: the interruption point was missed", flushed, at.Format("15:04:05.000"))
			}
			assertInterruptedFailure(t, got)
			assertNoFileWritten()
			assertNoCacheLogLine(t, logged)

			mode.Store(normal)
			if next := browserCheck(t, collector, item); next.Result != "success" || next.AmountCents != 12999 {
				t.Fatalf("check after the interrupted check: %+v", next)
			}
			assertNoFileWritten()
		})
	}
}

// An uninterrupted check is unchanged: a download clears the cache before the browser closes, a
// non-2xx error page does not clear it.
// TC-006
func TestBrowserInterrupted_TC006_UninterruptedCheckIsUnchanged(t *testing.T) {
	item := fixtureItem("B0PRICE001")

	t.Run("download", func(t *testing.T) {
		recordPath := useFakeBrowser(t, fakeBrowserScript{Navigate: "error", Response: http.StatusOK, Clear: "answer"})
		output := captureOutput(t)
		got := browserCheck(t, NewBrowserCollector(fakeBrowserOptions(t)), item)
		logged := output()
		record := readFakeRecord(t, recordPath)
		assertClearConfirmedBeforeClose(t, record)
		assertNavigationFailure(t, got, item, logged)
		assertNoCacheLogLine(t, logged)
	})

	t.Run("error page", func(t *testing.T) {
		recordPath := useFakeBrowser(t, fakeBrowserScript{Navigate: "error", Response: http.StatusServiceUnavailable, Clear: "answer"})
		output := captureOutput(t)
		got := browserCheck(t, NewBrowserCollector(fakeBrowserOptions(t)), item)
		logged := output()
		record := readFakeRecord(t, recordPath)
		if count := record.count(clearMethod, "received"); count != 0 {
			t.Fatalf("%d cache clears for an error page, want none:\n%s", count, record)
		}
		if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" || got.HTTPStatus != http.StatusServiceUnavailable {
			t.Fatalf("error page HTTP 503: %+v", got)
		}
		if !strings.Contains(logged, "Amazon returned HTTP 503") {
			t.Fatalf("the failure is not logged with HTTP 503:\n%s", logged)
		}
	})
}
