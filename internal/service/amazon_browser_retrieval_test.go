package service

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Amazon browser retrieval of issue #204 through the service: real browser checks of local fixture
// pages, the failure causes in the log, the failure count and the documented installation.

const retrievalSecret = "fixture-secret-token-204" // session-token cookie value; never in the log

// retrievalLog is the standard logger output, captured until the end of the test.
type retrievalLog struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (l *retrievalLog) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffer.Write(data)
}

func (l *retrievalLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffer.String()
}

func captureRetrievalLog(t *testing.T) *retrievalLog {
	t.Helper()
	logs := &retrievalLog{}
	previous := log.Writer()
	log.SetOutput(logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	return logs
}

// amazonFixture stands in for Amazon: it answers /dp/<ASIN> with answer(ASIN), setting the secret
// session-token cookie, the used-offer requests with no offer and the page resources with nothing.
type amazonFixture struct {
	mu     sync.Mutex
	answer func(asin string) (int, string)
}

func (f *amazonFixture) set(answer func(asin string) (int, string)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answer = answer
}

func (f *amazonFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/dp/"):
		f.mu.Lock()
		answer := f.answer
		f.mu.Unlock()
		status, body := answer(strings.TrimPrefix(r.URL.Path, "/dp/"))
		http.SetCookie(w, &http.Cookie{Name: "session-token", Value: retrievalSecret, Path: "/", MaxAge: 3600})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		io.WriteString(w, body)
	case strings.HasPrefix(r.URL.Path, "/gp/aod/ajax"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, `<div id="aod-container"><span>No featured offers available</span></div>`)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// page answers every ASIN with body and HTTP 200.
func page(body string) func(string) (int, string) {
	return func(string) (int, string) { return http.StatusOK, body }
}

func newAmazonFixture(t *testing.T, answer func(asin string) (int, string)) (*amazonFixture, *amazontest.Server) {
	t.Helper()
	fixture := &amazonFixture{answer: answer}
	return fixture, amazontest.NewServer(t, fixture)
}

// closedOrigin returns the origin of a local port that refuses connections.
func closedOrigin(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	origin := "http://" + listener.Addr().String()
	listener.Close()
	return origin
}

// neverLoadingOrigin returns the origin of a server whose pages never finish loading.
func neverLoadingOrigin(t *testing.T) string {
	t.Helper()
	release := make(chan struct{})
	server := amazontest.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "<html><body><p>Loading")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() { close(release) }) // runs before the server is closed
	return server.URL
}

// retrievalBrowser returns a browser collector sending the Amazon URLs to origin.
func retrievalBrowser(t *testing.T, executable, origin string, navigationTimeout time.Duration) *amazon.BrowserCollector {
	t.Helper()
	return amazon.NewBrowserCollector(amazon.BrowserOptions{ExecPath: executable, ProfileDir: filepath.Join(t.TempDir(), "profile"), NoSandbox: true,
		NavigationTimeout: navigationTimeout, Pause: func() time.Duration { return 0 }, Origin: origin})
}

// collectorByItem routes the check of each item to its own collector.
type collectorByItem map[string]AmazonCollector

func (c collectorByItem) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	return c[item.ID].Collect(ctx, item)
}

// refreshPrice requests Refresh price of id and waits for its end.
func refreshPrice(t *testing.T, service *Service, id string) {
	t.Helper()
	if _, err := service.RefreshItem(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	waitChecksDone(t, service)
}

// scheduledCheck makes id due a minute before now (in its window) and runs the scheduler once.
func scheduledCheck(t *testing.T, service *Service, database *store.Store, id string, now time.Time) {
	t.Helper()
	if err := database.SetNextCheck(context.Background(), id, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	service.collectDue(context.Background())
	waitChecksDone(t, service)
}

// insertPricedAmazonItem inserts an Amazon item with a successful price of 19.99 the day before.
func insertPricedAmazonItem(t *testing.T, database *store.Store, id string, next time.Time) {
	t.Helper()
	insertAt(t, database, id, "amazon", next)
	if err := database.RecordCollection(context.Background(), id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "not_found"},
		scheduleDay(8, 0).AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
}

// checkLogLine returns the log line of the check of id at the time at: the line with its request URL
// and check time.
func checkLogLine(t *testing.T, logged, id string, at time.Time) string {
	t.Helper()
	url, when := "url=https://www.amazon.fr/dp/"+id+" ", "at="+at.UTC().Format(time.RFC3339)
	for _, line := range strings.Split(logged, "\n") {
		if strings.Contains(line, url) && strings.Contains(line, when) {
			return line
		}
	}
	t.Fatalf("no log line of the check of %s with %s and %s:\n%s", id, url, when, shorten(logged))
	return ""
}

var causePattern = regexp.MustCompile(`\bcause=(.*?)(?: response=.*)?$`)

// logCause returns the cause logged on a check line.
func logCause(t *testing.T, line string) string {
	t.Helper()
	match := causePattern.FindStringSubmatch(line)
	if match == nil || strings.TrimSpace(match[1]) == "" {
		t.Fatalf("the log line has no cause: %s", shorten(line))
	}
	return strings.TrimSpace(match[1])
}

// shorten cuts each line of logged to 300 bytes, for failure messages (response excerpts are long).
func shorten(logged string) string {
	lines := strings.Split(logged, "\n")
	for index, line := range lines {
		if len(line) > 300 {
			lines[index] = line[:300] + "…"
		}
	}
	return strings.Join(lines, "\n")
}

// TC-007
func TestAmazonBrowserRetrieval_TC007_EachFailureCauseHasItsOwnLogText(t *testing.T) {
	browser := amazontest.Browser(t)
	fixture := amazontest.Fixture
	answers := map[string]func() (int, string){
		"B0CAUSED01": func() (int, string) {
			return http.StatusServiceUnavailable, "<html><body><h1>Service Unavailable fixture</h1></body></html>"
		},
		"B0CAUSEE01": func() (int, string) { return http.StatusOK, fixture(t, "robot-check-en.html") },
		"B0CAUSEF01": func() (int, string) { return http.StatusOK, fixture(t, "price-not-found.html") },
		"B0CAUSEG01": func() (int, string) {
			return http.StatusNotFound, "<html><body><h1>Page introuvable</h1></body></html>"
		},
		"B0CAUSEH01": func() (int, string) { return http.StatusOK, fixture(t, "price-found.html") },
	}
	_, server := newAmazonFixture(t, func(asin string) (int, string) { return answers[asin]() })
	missing := filepath.Join(t.TempDir(), "missing-chromium")
	served := retrievalBrowser(t, browser, server.URL, 30*time.Second)
	collectors := collectorByItem{
		"B0CAUSEA01": retrievalBrowser(t, missing, server.URL, 30*time.Second),
		"B0CAUSEB01": retrievalBrowser(t, browser, closedOrigin(t), 30*time.Second),
		"B0CAUSEC01": retrievalBrowser(t, browser, neverLoadingOrigin(t), 2*time.Second),
		"B0CAUSED01": served, "B0CAUSEE01": served, "B0CAUSEF01": served, "B0CAUSEG01": served, "B0CAUSEH01": served,
	}
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	for id := range collectors {
		insertPricedAmazonItem(t, database, id, scheduleDay(20, 0))
	}
	now := newClock(scheduleDay(6, 0))
	service := startHuman(t, database, dir, now, collectors, 204)
	logs := captureRetrievalLog(t)

	// (f) comes between the failures, so the five of them never reach the stop limit.
	failures := []struct {
		name     string
		id       string
		cause    []string
		failures int
	}{
		{"(a) browser could not start", "B0CAUSEA01", []string{"browser could not start", missing}, 1},
		{"(b) page could not be opened", "B0CAUSEB01", []string{"could not be opened"}, 2},
		{"(c) page did not finish loading", "B0CAUSEC01", []string{"did not finish loading"}, 3},
		{"(f) no detectable price", "B0CAUSEF01", []string{"no detectable"}, 0},
		{"(d) non-success status", "B0CAUSED01", []string{"HTTP 503"}, 1},
		{"(e) robot check", "B0CAUSEE01", []string{"robot check"}, 2},
	}
	causes := map[string]string{}
	for index, failure := range failures {
		at := scheduleDay(6, 10+index)
		now.Set(at)
		refreshPrice(t, service, failure.id)
		line := checkLogLine(t, logs.String(), failure.id, at)
		cause := logCause(t, line)
		for _, want := range failure.cause {
			if !strings.Contains(strings.ToLower(cause), strings.ToLower(want)) {
				t.Errorf("%s: logged cause %q does not contain %q (line %s)", failure.name, cause, want, shorten(line))
			}
		}
		causes[failure.name[:3]] = cause
		if got := amazonFailures(t, database); got != failure.failures {
			t.Errorf("%s: consecutive failure count %d, want %d", failure.name, got, failure.failures)
		}
		item := storedItem(t, database, failure.id)
		if item.Status != "retrieval_error" || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 || item.AmazonStoppedAt != nil {
			t.Errorf("%s: status %s, latest price %+v, stopped %v", failure.name, item.Status, item.LatestPrice, item.AmazonStoppedAt)
		}
	}
	for _, group := range [][]string{{"(a)", "(b)", "(d)", "(e)", "(f)"}, {"(a)", "(c)", "(d)", "(e)", "(f)"}} {
		for i := range group {
			for j := i + 1; j < len(group); j++ {
				if causes[group[i]] == causes[group[j]] {
					t.Errorf("causes %s and %s have the same log text %q", group[i], group[j], causes[group[i]])
				}
			}
		}
	}

	for index, id := range []string{"B0CAUSEG01", "B0CAUSEH01"} {
		now.Set(scheduleDay(6, 30+index))
		refreshPrice(t, service, id)
		for _, line := range strings.Split(logs.String(), "\n") {
			if strings.Contains(line, "url=https://www.amazon.fr/dp/"+id+" ") {
				t.Errorf("a failure was logged for %s: %s", id, shorten(line))
			}
		}
	}
	if result := storedItem(t, database, "B0CAUSEG01"); result.Status != "unavailable" {
		t.Errorf("404 listing status %s, want unavailable", result.Status)
	}
	if result := storedItem(t, database, "B0CAUSEH01"); result.LastAttempt == nil || result.LastAttempt.Result != "success" {
		t.Errorf("successful check recorded as %+v", result.LastAttempt)
	}
	if strings.Contains(logs.String(), retrievalSecret) {
		t.Fatalf("a cookie value reached the log:\n%s", shorten(logs.String()))
	}
}

// TC-008
func TestAmazonBrowserRetrieval_TC008_FailuresAreCountedAndStopItemsMentionsAreNot(t *testing.T) {
	ids := []string{"B0COUNT001", "B0COUNT002"}
	setup := func(t *testing.T, collector AmazonCollector) (*Service, *store.Store, *clock, *retrievalLog) {
		t.Helper()
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		for _, id := range ids {
			insertPricedAmazonItem(t, database, id, scheduleDay(20, 0))
		}
		now := newClock(scheduleDay(7, 0))
		service := startHuman(t, database, dir, now, collector, 208)
		return service, database, now, captureRetrievalLog(t)
	}
	// check runs the check number step (from 1): Refresh price and scheduled checks in turn, on both items.
	check := func(t *testing.T, service *Service, database *store.Store, now *clock, step int) (string, time.Time) {
		t.Helper()
		id := ids[step%2]
		at := scheduleDay(7, 5*step)
		now.Set(at)
		if step%2 == 1 {
			refreshPrice(t, service, id)
		} else {
			scheduledCheck(t, service, database, id, at)
		}
		return id, at
	}
	assertFailures := func(t *testing.T, service *Service, database *store.Store, now *clock, logs *retrievalLog, cause string) {
		t.Helper()
		for step := 1; step <= amazonFailureLimit; step++ {
			id, at := check(t, service, database, now, step)
			if item := storedItem(t, database, id); item.Status != "retrieval_error" || item.LastAttempt == nil || !item.LastAttempt.Timestamp.Equal(at) {
				t.Fatalf("failure %d of %s: status %s, attempt %+v", step, id, item.Status, item.LastAttempt)
			}
			if line := checkLogLine(t, logs.String(), id, at); !strings.Contains(strings.ToLower(logCause(t, line)), strings.ToLower(cause)) {
				t.Fatalf("failure %d: logged cause does not contain %q: %s", step, cause, shorten(line))
			}
			for _, other := range ids {
				item := storedItem(t, database, other)
				if item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
					t.Fatalf("after failure %d, item %s lost its last price: %+v", step, other, item.LatestPrice)
				}
				if step < amazonFailureLimit && item.AmazonStoppedAt != nil {
					t.Fatalf("item %s stopped after %d failures", other, step)
				}
				if step == amazonFailureLimit && (item.AmazonStoppedAt == nil || !item.AmazonStoppedAt.Equal(at)) {
					t.Fatalf("item %s after the 5th failure: stopped %v, want %s", other, item.AmazonStoppedAt, at)
				}
			}
			if got := amazonFailures(t, database); step < amazonFailureLimit && got != step {
				t.Fatalf("consecutive failure count %d after failure %d", got, step)
			}
		}
		if !strings.Contains(logs.String(), "Amazon checks stopped after 5 consecutive failures") {
			t.Fatalf("the stop is not logged:\n%s", shorten(logs.String()))
		}
	}

	t.Run("robot check", func(t *testing.T) {
		_, server := newAmazonFixture(t, page(amazontest.Fixture(t, "robot-check-en.html")))
		service, database, now, logs := setup(t, retrievalBrowser(t, amazontest.Browser(t), server.URL, 30*time.Second))
		assertFailures(t, service, database, now, logs, "robot check")
	})

	t.Run("mentioning page after four robot checks", func(t *testing.T) {
		fixture, server := newAmazonFixture(t, page(amazontest.Fixture(t, "robot-check-en.html")))
		service, database, now, logs := setup(t, retrievalBrowser(t, amazontest.Browser(t), server.URL, 30*time.Second))
		for step := 1; step < amazonFailureLimit; step++ {
			check(t, service, database, now, step)
		}
		if got := amazonFailures(t, database); got != amazonFailureLimit-1 {
			t.Fatalf("consecutive failure count %d after four robot checks", got)
		}
		fixture.set(page(amazontest.ExpandedFixture(t, "mentions-captcha.html")))
		id, at := check(t, service, database, now, amazonFailureLimit)
		item := storedItem(t, database, id)
		if item.LastAttempt == nil || item.LastAttempt.Result != "success" || item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 ||
			!equalTime(item.LatestPrice.Timestamp, at) {
			t.Fatalf("check of the mentioning page: attempt %+v, latest price %+v", item.LastAttempt, item.LatestPrice)
		}
		if got := amazonFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d after the success, want 0", got)
		}
		for _, other := range ids {
			if storedItem(t, database, other).AmazonStoppedAt != nil {
				t.Fatalf("item %s is stopped", other)
			}
		}
		for _, line := range strings.Split(logs.String(), "\n") {
			if strings.Contains(line, "url=https://www.amazon.fr/dp/"+id+" ") && strings.Contains(line, "at="+at.UTC().Format(time.RFC3339)) {
				t.Fatalf("a failure was logged for the mentioning page: %s", shorten(line))
			}
		}
	})

	t.Run("HTTP 503", func(t *testing.T) {
		_, server := newAmazonFixture(t, func(string) (int, string) {
			return http.StatusServiceUnavailable, "<html><body><h1>Service Unavailable fixture</h1></body></html>"
		})
		service, database, now, logs := setup(t, retrievalBrowser(t, amazontest.Browser(t), server.URL, 30*time.Second))
		assertFailures(t, service, database, now, logs, "HTTP 503")
	})

	t.Run("connection refused", func(t *testing.T) {
		service, database, now, logs := setup(t, retrievalBrowser(t, amazontest.Browser(t), closedOrigin(t), 30*time.Second))
		assertFailures(t, service, database, now, logs, "could not be opened")
	})
}

func equalTime(got *time.Time, want time.Time) bool { return got != nil && got.Equal(want) }

// documentedService starts the service with the collector it builds from its configuration and the
// environment, Amazon URLs sent to origin; the profile is in the data directory dir.
func documentedService(t *testing.T, database *store.Store, dir string, now *clock, origin string) *Service {
	t.Helper()
	service := NewWithOptions(humanConfig(dir), database, Options{Now: now.Now, Random: seededRandom(212),
		SchedulePause: func(context.Context) bool { return true }, AmazonBrowserOrigin: origin})
	t.Cleanup(service.Close)
	return service
}

// browserLink returns a directory holding only a chromium link to the test browser.
func browserLink(t *testing.T, browser string) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Symlink(browser, filepath.Join(directory, "chromium")); err != nil {
		t.Fatal(err)
	}
	return directory
}

// TC-012
func TestAmazonBrowserRetrieval_TC012_ServerSetUpAsDocumentedChecksAmazon(t *testing.T) {
	browser := amazontest.Browser(t) // found before PATH changes
	_, server := newAmazonFixture(t, page(amazontest.ExpandedFixture(t, "mentions-captcha.html")))
	const id = "B0SETUP012"
	run := func(t *testing.T) (model.Item, string, string) {
		t.Helper()
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertAt(t, database, id, "amazon", scheduleDay(20, 0))
		logs := captureRetrievalLog(t)
		service := documentedService(t, database, dir, newClock(scheduleDay(14, 0)), server.URL)
		refreshPrice(t, service, id)
		return storedItem(t, database, id), logs.String(), dir
	}
	assertChecked := func(t *testing.T, item model.Item, logged, dir string) {
		t.Helper()
		if item.LastAttempt == nil || item.LastAttempt.Result != "success" || item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 {
			t.Fatalf("check on the documented installation: attempt %+v, latest price %+v\n%s", item.LastAttempt, item.LatestPrice, shorten(logged))
		}
		if strings.Contains(logged, "Amazon request failed") {
			t.Fatalf("a failure was logged:\n%s", shorten(logged))
		}
		if info, err := os.Stat(filepath.Join(dir, "amazon-browser")); err != nil || !info.IsDir() {
			t.Fatalf("no browser profile in the data directory: %v", err)
		}
	}

	t.Run("PRICEFOLLOWER_BROWSER_PATH set", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", browser)
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		item, logged, dir := run(t)
		assertChecked(t, item, logged, dir)
	})

	t.Run("chromium found on PATH", func(t *testing.T) {
		t.Setenv("PATH", browserLink(t, browser))
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", "")
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		item, logged, dir := run(t)
		assertChecked(t, item, logged, dir)
	})

	t.Run("PRICEFOLLOWER_BROWSER_PATH pointing to a missing file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing-chromium")
		t.Setenv("PATH", t.TempDir())
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", missing)
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		item, logged, _ := run(t)
		if item.LastAttempt == nil || item.LastAttempt.Result != "request_error" {
			t.Fatalf("check with a missing browser: attempt %+v", item.LastAttempt)
		}
		cause := logCause(t, checkLogLine(t, logged, id, scheduleDay(14, 0)))
		if !strings.Contains(cause, "browser could not start") || !strings.Contains(cause, missing) {
			t.Fatalf("logged cause %q does not say the browser could not start with the missing file %s", cause, missing)
		}
	})

	t.Run("root without PRICEFOLLOWER_BROWSER_NO_SANDBOX", func(t *testing.T) {
		if os.Geteuid() != 0 {
			t.Skip("the browser refuses to run without its sandbox only as root")
		}
		t.Setenv("PATH", t.TempDir())
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", browser)
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "")
		item, logged, _ := run(t)
		if item.LastAttempt == nil || item.LastAttempt.Result != "request_error" {
			t.Fatalf("check as root with the sandbox: attempt %+v", item.LastAttempt)
		}
		cause := logCause(t, checkLogLine(t, logged, id, scheduleDay(14, 0)))
		if !strings.Contains(cause, "browser could not start") || !strings.Contains(cause, "--no-sandbox") {
			t.Fatalf("logged cause %q does not carry the browser's own reason", cause)
		}
	})
}

// TC-013
func TestAmazonBrowserRetrieval_TC013_MissingBrowserIsReportedAtStart(t *testing.T) {
	browser := amazontest.Browser(t) // found before PATH changes
	_, server := newAmazonFixture(t, page(amazontest.ExpandedFixture(t, "mentions-captcha.html")))
	const startLine = "Amazon browser not found"
	const id = "B0START013"

	t.Run("no browser", func(t *testing.T) {
		pathDirectory := t.TempDir()
		t.Setenv("PATH", pathDirectory)
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", "")
		t.Setenv("PRICEFOLLOWER_BROWSER_NO_SANDBOX", "true")
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertAt(t, database, id, "amazon", scheduleDay(20, 0))
		logs := captureRetrievalLog(t)
		now := newClock(scheduleDay(14, 0))
		service := documentedService(t, database, dir, now, server.URL)
		if got := strings.Count(logs.String(), startLine); got != 1 {
			t.Fatalf("%d start lines saying the browser was not found, want 1:\n%s", got, logs.String())
		}

		refreshPrice(t, service, id)
		if item := storedItem(t, database, id); item.LastAttempt == nil || item.LastAttempt.Result != "request_error" {
			t.Fatalf("check without a browser: attempt %+v", item.LastAttempt)
		}
		cause := logCause(t, checkLogLine(t, logs.String(), id, scheduleDay(14, 0)))
		if !strings.Contains(cause, "browser could not start") || !strings.Contains(cause, "no Chromium or Google Chrome found") {
			t.Fatalf("logged cause %q does not say the browser could not start and why", cause)
		}

		// A browser installed after the start is used by the next check, without a restart.
		if err := os.Symlink(browser, filepath.Join(pathDirectory, "chromium")); err != nil {
			t.Fatal(err)
		}
		now.Set(scheduleDay(14, 5))
		refreshPrice(t, service, id)
		if item := storedItem(t, database, id); item.LastAttempt == nil || item.LastAttempt.Result != "success" || item.LatestPrice == nil ||
			item.LatestPrice.AmountCents != 12999 {
			t.Fatalf("check after the browser was installed: attempt %+v, latest price %+v\n%s", item.LastAttempt, item.LatestPrice, shorten(logs.String()))
		}
		if got := strings.Count(logs.String(), startLine); got != 1 {
			t.Fatalf("%d start lines saying the browser was not found, want 1", got)
		}
	})

	t.Run("browser on PATH", func(t *testing.T) {
		t.Setenv("PATH", browserLink(t, browser))
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", "")
		dir := t.TempDir()
		logs := captureRetrievalLog(t)
		documentedService(t, openHumanStore(t, dir), dir, newClock(scheduleDay(14, 0)), server.URL)
		if strings.Contains(logs.String(), startLine) {
			t.Fatalf("start line with a browser on PATH:\n%s", logs.String())
		}
	})

	t.Run("PRICEFOLLOWER_BROWSER_PATH set", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", browser)
		dir := t.TempDir()
		logs := captureRetrievalLog(t)
		documentedService(t, openHumanStore(t, dir), dir, newClock(scheduleDay(14, 0)), server.URL)
		if strings.Contains(logs.String(), startLine) {
			t.Fatalf("start line with PRICEFOLLOWER_BROWSER_PATH set:\n%s", logs.String())
		}
	})
}
