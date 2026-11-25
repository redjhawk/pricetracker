package amazon

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Tests of issue #240: a failed cache clear is logged with a neutral line, "Amazon browser cache could not be
// cleared: <cause>", whatever started the clear (a page not opened, or a check interrupted while the page was
// still requested).

const (
	neutralCachePrefix = cacheLogLine + ": "
	oldCacheWording    = "after a page not opened"
	clearRefusedCause  = "cache clear refused by fake"
	clearUnboundCause  = "no answer within"
)

// logMessage matches the standard log date and time prefix (log.LstdFlags) and captures the message after it.
var logMessage = regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} (.*)$`)

// assertNeutralCacheLine checks that output holds exactly one cache log line, whose message (after the
// standard log date and time) is "Amazon browser cache could not be cleared: " followed by a non-empty
// cause containing cause and neither "context canceled" nor "context deadline exceeded".
func assertNeutralCacheLine(t *testing.T, output, cause string) {
	t.Helper()
	lines := cacheLines(output)
	if len(lines) != 1 {
		t.Fatalf("%d cache log lines, want 1:\n%s", len(lines), output)
	}
	match := logMessage.FindStringSubmatch(lines[0])
	if match == nil {
		t.Fatalf("cache log line without the standard log date and time: %q", lines[0])
	}
	message := match[1]
	if !strings.HasPrefix(message, neutralCachePrefix) {
		t.Fatalf("cache log message %q, want it to start with %q", message, neutralCachePrefix)
	}
	got := strings.TrimPrefix(message, neutralCachePrefix)
	if strings.TrimSpace(got) == "" {
		t.Fatalf("cache log line without a cause: %q", lines[0])
	}
	if !strings.Contains(got, cause) {
		t.Fatalf("cache log cause %q, want it to contain %q", got, cause)
	}
	if strings.Contains(got, "context canceled") || strings.Contains(got, "context deadline exceeded") {
		t.Fatalf("the interruption of the check is given as the cause of the cache clear failure: %q", lines[0])
	}
}

// assertNoOldCacheWording fails t when a line of output contains "after a page not opened".
func assertNoOldCacheWording(t *testing.T, output string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, oldCacheWording) {
			t.Fatalf("log line with the old wording %q: %s", oldCacheWording, line)
		}
	}
}

// assertBrowserClosed fails t when the fake browser did not receive Browser.close.
func assertBrowserClosed(t *testing.T, record fakeRecord) {
	t.Helper()
	if index, _ := record.first("Browser.close", "received"); index < 0 {
		t.Fatalf("the browser was not closed:\n%s", record)
	}
}

// A failed clear after a page the browser did not open is logged with the neutral line and its real cause.
// TC-001
func TestBrowserCacheLog_TC001_PageNotOpened(t *testing.T) {
	item := fixtureItem("B0PRICE001")
	for _, test := range []struct {
		name, clear, clearError, cause string
	}{
		{"live check, error answer", "error", clearRefusedCause, clearRefusedCause},
		{"live check, no answer", "never", "", clearUnboundCause},
	} {
		t.Run(test.name, func(t *testing.T) {
			recordPath := useFakeBrowser(t, fakeBrowserScript{Navigate: "error", Response: http.StatusOK, Clear: test.clear, ClearError: test.clearError})
			output := captureOutput(t)
			got := browserCheck(t, NewBrowserCollector(fakeBrowserOptions(t)), item)
			logged := output()
			record := readFakeRecord(t, recordPath)
			assertBrowserClosed(t, record)
			assertNavigationFailure(t, got, item, logged)
			assertNeutralCacheLine(t, logged, test.cause)
			assertNoOldCacheWording(t, logged)
		})
	}

	t.Run("interrupted after the offer, error answer", func(t *testing.T) {
		interruptAt := time.Now().Add(2 * time.Second)
		recordPath := useFakeBrowser(t, fakeBrowserScript{SetUpUntil: interruptAt.Add(-200 * time.Millisecond).UnixNano(),
			Navigate: "error", Clear: "error", ClearError: clearRefusedCause})
		ctx, cancel := interruptedContext(interruptionServerStop, interruptAt)
		defer cancel()
		output := captureOutput(t)
		got := collectInterrupted(t, NewBrowserCollector(fakeBrowserOptions(t)), ctx, item, interruptAt)
		logged := output()
		record := readFakeRecord(t, recordPath)
		assertNavigatedBeforeInterruption(t, record, interruptAt)
		assertBrowserClosed(t, record)
		assertInterruptedFailure(t, got)
		assertNeutralCacheLine(t, logged, clearRefusedCause)
		assertNoOldCacheWording(t, logged)
	})

	t.Run("repeated", func(t *testing.T) {
		collector := NewBrowserCollector(fakeBrowserOptions(t))
		total := 0
		for check := 1; check <= 2; check++ {
			recordPath := useFakeBrowser(t, fakeBrowserScript{Navigate: "error", Response: http.StatusOK, Clear: "error", ClearError: clearRefusedCause})
			output := captureOutput(t)
			got := browserCheck(t, collector, item)
			logged := output()
			record := readFakeRecord(t, recordPath)
			assertBrowserClosed(t, record)
			assertNavigationFailure(t, got, item, logged)
			assertNeutralCacheLine(t, logged, clearRefusedCause)
			assertNoOldCacheWording(t, logged)
			total += len(cacheLines(logged))
		}
		if total != 2 {
			t.Fatalf("%d cache log lines over two failed clears, want 2", total)
		}
	})
}

// A failed clear after a check interrupted while the page was still requested is logged with the same
// neutral line, with the clear's cause and never the interruption of the check.
// TC-002
func TestBrowserCacheLog_TC002_PageStillRequested(t *testing.T) {
	item := fixtureItem("B0PRICE001")
	for _, kind := range []string{interruptionTimeLimit, interruptionServerStop} {
		for _, failure := range []struct {
			name, clear, clearError, cause string
		}{
			{"error answer", "error", clearRefusedCause, clearRefusedCause},
			{"no answer", "never", "", clearUnboundCause},
		} {
			t.Run(kind+", "+failure.name, func(t *testing.T) {
				interruptAt := time.Now().Add(2 * time.Second)
				recordPath := useFakeBrowser(t, fakeBrowserScript{SetUpUntil: interruptAt.Add(-200 * time.Millisecond).UnixNano(),
					Navigate: "hold", Clear: failure.clear, ClearError: failure.clearError})
				ctx, cancel := interruptedContext(kind, interruptAt)
				defer cancel()
				output := captureOutput(t)
				got := collectInterrupted(t, NewBrowserCollector(fakeBrowserOptions(t)), ctx, item, interruptAt)
				logged := output()
				record := readFakeRecord(t, recordPath)
				assertNavigatedBeforeInterruption(t, record, interruptAt)
				if count := record.count(clearMethod, "received"); count != 1 {
					t.Fatalf("%d cache clears, want 1:\n%s", count, record)
				}
				assertBrowserClosed(t, record)
				assertInterruptedFailure(t, got)
				assertNeutralCacheLine(t, logged, failure.cause)
				assertNoOldCacheWording(t, logged)
				assertNoInterruptionGivenForCache(t, logged)
			})
		}
	}
}

// A successful clear logs no cache line, whatever started it.
// TC-003
func TestBrowserCacheLog_TC003_SuccessfulClearLogsNothing(t *testing.T) {
	item := fixtureItem("B0PRICE001")

	t.Run("live check, page not opened, repeated", func(t *testing.T) {
		collector := NewBrowserCollector(fakeBrowserOptions(t))
		for check := 1; check <= 2; check++ {
			recordPath := useFakeBrowser(t, fakeBrowserScript{Navigate: "error", Response: http.StatusOK, Clear: "answer"})
			output := captureOutput(t)
			got := browserCheck(t, collector, item)
			logged := output()
			record := readFakeRecord(t, recordPath)
			assertClearConfirmedBeforeClose(t, record)
			assertNavigationFailure(t, got, item, logged)
			assertNoCacheLogLine(t, logged)
			assertNoOldCacheWording(t, logged)
		}
	})

	for _, kind := range []string{interruptionTimeLimit, interruptionServerStop} {
		for _, test := range []struct{ name, navigate string }{
			{"interrupted after the offer", "error"},
			{"page still requested", "hold"},
		} {
			t.Run(test.name+", "+kind, func(t *testing.T) {
				interruptAt := time.Now().Add(2 * time.Second)
				recordPath := useFakeBrowser(t, fakeBrowserScript{SetUpUntil: interruptAt.Add(-200 * time.Millisecond).UnixNano(),
					Navigate: test.navigate, Clear: "answer", ClearDelay: 300 * time.Millisecond})
				ctx, cancel := interruptedContext(kind, interruptAt)
				defer cancel()
				output := captureOutput(t)
				got := collectInterrupted(t, NewBrowserCollector(fakeBrowserOptions(t)), ctx, item, interruptAt)
				logged := output()
				record := readFakeRecord(t, recordPath)
				assertNavigatedBeforeInterruption(t, record, interruptAt)
				assertClearConfirmedBeforeClose(t, record)
				assertInterruptedFailure(t, got)
				assertNoCacheLogLine(t, logged)
				assertNoOldCacheWording(t, logged)
			})
		}
	}
}
