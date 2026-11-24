package service

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/leboncoin"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Amazon human-like checks of issue #168 through the service: browser collector, random windows
// and the one-time restart. The scheduler runs with a fake clock (Options.Now) and no pause.

// scheduleDay is a time of Tuesday 5 March 2030 in Paris (no clock change that week).
func scheduleDay(hour, minute int) time.Time {
	return localTime(paris, 2030, time.March, 5, hour, minute)
}

// clock is the fake clock of the schedule tests.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock(at time.Time) *clock { return &clock{now: at} }

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = at
}

// amazonRecorder is a fake Amazon collector recording every check. result nil is a success; with
// started set it reports each check start; with block set each check waits for block to be closed.
type amazonRecorder struct {
	mu      sync.Mutex
	clock   *clock
	ids     []string
	times   []time.Time
	result  func(model.Listing) model.CollectionResult
	started chan string
	block   chan struct{}
}

func (r *amazonRecorder) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	r.mu.Lock()
	r.ids = append(r.ids, item.ID)
	if r.clock != nil {
		r.times = append(r.times, r.clock.Now())
	}
	result, started, block := r.result, r.started, r.block
	r.mu.Unlock()
	if started != nil {
		started <- item.ID
	}
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
		}
	}
	if result != nil {
		return result(item)
	}
	return succeeded()
}

func (r *amazonRecorder) calls(id string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, checked := range r.ids {
		if checked == id {
			count++
		}
	}
	return count
}

func (r *amazonRecorder) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.ids)
}

func (r *amazonRecorder) checkTimes() []time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Time(nil), r.times...)
}

// waitFor polls condition for up to 5 seconds (the first check after Add starts in the background).
func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if condition() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

func waitStarted(t *testing.T, started <-chan string) string {
	t.Helper()
	select {
	case id := <-started:
		return id
	case <-time.After(5 * time.Second):
		t.Fatal("the Amazon check did not start")
		return ""
	}
}

func humanConfig(dir string) config.Config {
	return config.Config{DataDirectory: dir, Location: paris, CheckTimes: []string{"08:00", "20:00"}, StaleAfter: 36 * time.Hour, UserAgent: "test"}
}

// openHumanStore opens the database of dir; it is closed at the end of the test.
func openHumanStore(t *testing.T, dir string) *store.Store {
	t.Helper()
	database, err := store.Open(humanConfig(dir))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// startHuman starts a service with the Amazon collector, the fake clock, a seeded random source,
// no pause between scheduled checks and a LeBoncoin collector that never reaches the network.
func startHuman(t *testing.T, database *store.Store, dir string, now *clock, collector AmazonCollector, seed uint64) *Service {
	t.Helper()
	service := NewWithOptions(humanConfig(dir), database, Options{Amazon: collector, Now: now.Now, Random: seededRandom(seed),
		SchedulePause: func(context.Context) bool { return true }})
	t.Cleanup(service.Close)
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		return model.CollectionResult{Result: "success", AmountCents: 1200}, leboncoin.SessionOutcome{}
	})
	return service
}

// insertAt inserts an item of the open-mode owner with its next check at next.
func insertAt(t *testing.T, database *store.Store, id, platform string, next time.Time) {
	t.Helper()
	insertOwned(t, database, 0, id, platform, next)
}

func insertOwned(t *testing.T, database *store.Store, owner int64, id, platform string, next time.Time) {
	t.Helper()
	listing := model.Listing{ID: id, Platform: platform, ListingID: id, Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/test/" + id, OwnerID: owner}
	if platform == "amazon" {
		listing = model.Listing{ID: id, Platform: platform, ListingID: id, ASIN: id, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + id, OwnerID: owner}
	}
	if err := database.Insert(context.Background(), listing, listing.URL, next); err != nil {
		t.Fatal(err)
	}
}

func nextCheck(t *testing.T, database *store.Store, id string) time.Time {
	t.Helper()
	item, err := database.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if item.NextCheckAt == nil {
		t.Fatalf("item %s has no next check", id)
	}
	return *item.NextCheckAt
}

func amazonFailures(t *testing.T, database *store.Store) int {
	t.Helper()
	state, err := database.AmazonRequestState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return state.ConsecutiveFailures
}

var legacyAmazonIDs = []string{"B000000251", "B000000252", "B000000253"}

// legacyStoppedDatabase writes in dir a database as left before issue #168: Amazon items of owners
// stopped on 20 Nov 2026 07:26 UTC after 5 failures, with a price history, and a LeBoncoin item.
func legacyStoppedDatabase(t *testing.T, dir string, owners ...int64) {
	t.Helper()
	database, err := store.Open(humanConfig(dir))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if len(owners) == 0 {
		owners = []int64{0}
	}
	for index, id := range legacyAmazonIDs {
		insertOwned(t, database, owners[index%len(owners)], id, "amazon", time.Date(2026, time.November, 20, 19, 0, 0, 0, time.UTC))
		if err := database.RecordCollection(ctx, id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "not_found"},
			time.Date(2026, time.November, 19, 7, 0, 0, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}
	insertAt(t, database, "lbc", "leboncoin", localTime(paris, 2031, time.January, 1, 8, 0))
	if err := database.StopAmazonItems(ctx, time.Date(2026, time.November, 20, 7, 26, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := database.SaveAmazonFailures(ctx, 5); err != nil {
		t.Fatal(err)
	}
	database.Close()
	dropRestartMarker(t, dir)
}

// dropRestartMarker removes the one-time restart marker of issue #168, as in a database written before it.
func dropRestartMarker(t *testing.T, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var exists bool
	if err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM pragma_table_info('amazon_request_state') WHERE name = 'human_checks_restarted')").Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		if _, err := db.Exec("ALTER TABLE amazon_request_state DROP COLUMN human_checks_restarted"); err != nil {
			t.Fatal(err)
		}
	}
}

// TC-001
func TestAmazonHumanChecks_TC001_EveryAmazonCheckGoesThroughTheBrowser(t *testing.T) {
	t.Run("default collector", func(t *testing.T) {
		dir := t.TempDir()
		service := New(humanConfig(dir), openHumanStore(t, dir))
		t.Cleanup(service.Close)
		if _, ok := service.collectors["amazon"].(*amazon.BrowserCollector); !ok {
			t.Fatalf("the Amazon collector is %T, not the browser collector", service.collectors["amazon"])
		}
		for platform := range service.collectors {
			if platform != "amazon" {
				t.Errorf("unexpected collector for %q", platform)
			}
		}
	})

	t.Run("no plain HTTP request", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir()) // no browser on this machine: the check fails and reaches no network
		t.Setenv("PRICEFOLLOWER_BROWSER_PATH", "")
		var plainRequests atomic.Int32
		withTransport(t, func(r *http.Request) (*http.Response, error) {
			plainRequests.Add(1)
			return &http.Response{StatusCode: 503, Header: http.Header{"Content-Type": []string{"text/html"}}, Body: io.NopCloser(strings.NewReader("<html>busy</html>")), Request: r}, nil
		})
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		service := New(humanConfig(dir), database)
		t.Cleanup(service.Close)
		insertAt(t, database, "B000000001", "amazon", scheduleDay(20, 0))
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		service.Collect(ctx, "B000000001")
		if plainRequests.Load() != 0 {
			t.Fatalf("an Amazon check sent %d plain HTTP requests", plainRequests.Load())
		}
		if item, err := database.Get(ctx, "B000000001"); err != nil || item.LastAttempt == nil || item.LastAttempt.Result != "request_error" {
			t.Fatalf("a check without a browser must fail: %+v %v", item.LastAttempt, err)
		}
	})

	t.Run("every entry point", func(t *testing.T) {
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		now := newClock(scheduleDay(7, 0))
		recorder := &amazonRecorder{}
		service := startHuman(t, database, dir, now, recorder, 1)
		var leboncoinChecks atomic.Int32
		service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
			leboncoinChecks.Add(1)
			return model.CollectionResult{Result: "success", AmountCents: 1200}, leboncoin.SessionOutcome{}
		})
		ctx := context.Background()
		added, err := service.Add(ctx, "https://www.amazon.fr/dp/B000000001", "")
		if err != nil {
			t.Fatal(err)
		}
		waitFor(t, "the first check after add", func() bool { return recorder.calls(added.ID) >= 1 })
		waitIdle(t, service)
		if got := recorder.calls(added.ID); got != 1 {
			t.Fatalf("first check after add: %d browser checks", got)
		}
		if err := database.SetNextCheck(ctx, added.ID, now.Now().Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
		service.collectDue(ctx)
		if got := recorder.calls(added.ID); got != 2 {
			t.Fatalf("scheduled check: %d browser checks in total", got)
		}
		if _, err := service.RefreshItem(ctx, added.ID); err != nil {
			t.Fatal(err)
		}
		waitIdle(t, service)
		if got := recorder.calls(added.ID); got != 3 {
			t.Fatalf("Refresh price: %d browser checks in total", got)
		}
		insertAt(t, database, "lbc", "leboncoin", scheduleDay(20, 0))
		if _, _, err := service.RefreshAll(ctx); err != nil {
			t.Fatal(err)
		}
		waitIdle(t, service)
		if recorder.calls(added.ID) != 4 || recorder.total() != 4 || leboncoinChecks.Load() != 1 {
			t.Fatalf("Refresh all prices: %d browser checks of the Amazon item, %d in total, %d LeBoncoin checks",
				recorder.calls(added.ID), recorder.total(), leboncoinChecks.Load())
		}
	})
}

// A browser that cannot start fails the check, counts the failure and releases the item.
// TC-007 (service part; the browser cases are in internal/amazon)
func TestAmazonHumanChecks_TC007_FailedBrowserStartReleasesTheItem(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	browser := amazon.NewBrowserCollector(amazon.BrowserOptions{ExecPath: filepath.Join(t.TempDir(), "missing-chromium"),
		ProfileDir: filepath.Join(t.TempDir(), "profile"), Pause: func() time.Duration { return 0 }})
	service := startHuman(t, database, dir, newClock(scheduleDay(14, 0)), browser, 7)
	ctx := context.Background()
	added, err := service.Add(ctx, "https://www.amazon.fr/dp/B000000007", "")
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the first check after add", func() bool {
		item, err := database.Get(ctx, added.ID)
		return err == nil && item.LastAttempt != nil
	})
	waitIdle(t, service)
	item, err := database.Get(ctx, added.ID)
	if err != nil || item.LastAttempt == nil || item.LastAttempt.Result != "request_error" || item.Status != "retrieval_error" || amazonFailures(t, database) != 1 {
		t.Fatalf("failed browser start: %+v %v", item.LastAttempt, err)
	}
	if _, err := service.RefreshItem(ctx, added.ID); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	if got := amazonFailures(t, database); got != 2 {
		t.Fatalf("the item was not released: a new check counted %d failures", got)
	}
}

// TC-015
func TestAmazonHumanChecks_TC015_OnlyDueItemsAreChecked(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ids := []string{"B000000151", "B000000152", "B000000153"}
	times := []time.Time{scheduleDay(6, 20), scheduleDay(7, 10), scheduleDay(8, 40)}
	for index, id := range ids {
		insertAt(t, database, id, "amazon", times[index])
	}
	now := newClock(scheduleDay(6, 0))
	recorder := &amazonRecorder{}
	service := startHuman(t, database, dir, now, recorder, 15)
	ctx := context.Background()
	service.collectDue(ctx)
	if recorder.total() != 0 {
		t.Fatalf("no item is due at 06:00, checked %v", recorder.ids)
	}
	for index, id := range ids {
		now.Set(times[index].Add(time.Minute))
		service.collectDue(ctx)
		if recorder.total() != index+1 || recorder.calls(id) != 1 {
			t.Fatalf("run at %s checked %v, want only %s", now.Now().In(paris), recorder.ids, id)
		}
		if next := nextCheck(t, database, id); windowOf(next, paris) != evening || localDate(next, paris) != "2030-03-05" {
			t.Fatalf("next check of %s at %s, want the evening window", id, next.In(paris))
		}
	}

	insertAt(t, database, "B000000154", "amazon", scheduleDay(8, 50))
	service.mu.Lock()
	service.inFlight["B000000154"] = now.Now()
	service.mu.Unlock()
	now.Set(scheduleDay(8, 51))
	service.collectDue(ctx)
	service.releaseCollection("B000000154")
	if recorder.calls("B000000154") != 0 || recorder.total() != len(ids) {
		t.Fatalf("an item already being checked was checked again by the scheduler: %v", recorder.ids)
	}
}

// TC-020
func TestAmazonHumanChecks_TC020_ScheduledBrowserChecksOverThreeDays(t *testing.T) {
	page, noOffer := amazontest.Fixture(t, "price-found.html"), amazontest.Fixture(t, "aod-none.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/dp/B0PRICE001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	})
	usedOffers := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, noOffer)
	}
	mux.HandleFunc("/gp/aod/ajax", usedOffers)
	mux.HandleFunc("/gp/aod/ajax/", usedOffers)
	server := amazontest.NewServer(t, mux)
	browser := amazon.NewBrowserCollector(amazon.BrowserOptions{ExecPath: amazontest.Browser(t), ProfileDir: filepath.Join(t.TempDir(), "profile"), NoSandbox: true,
		NavigationTimeout: 30 * time.Second, Pause: func() time.Duration { return 0 }, Origin: server.URL})
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	const id = "B0PRICE001"
	insertAt(t, database, id, "amazon", scheduleDay(6, 30))
	now := newClock(scheduleDay(0, 0))
	service := startHuman(t, database, dir, now, browser, 20)
	ctx := context.Background()
	end := scheduleDay(0, 0).AddDate(0, 0, 3)
	var checks []time.Time
	for len(checks) < 10 {
		next := nextCheck(t, database, id)
		if !next.Before(end) {
			break
		}
		now.Set(next.Add(time.Second))
		service.collectDue(ctx)
		if after := nextCheck(t, database, id); !after.After(next) {
			t.Fatalf("the scheduled check of %s did not run", next.In(paris))
		}
		checks = append(checks, next)
	}
	if got := server.Count("/dp/" + id); got != 6 || len(checks) != 6 {
		t.Fatalf("%d browser page loads for %d scheduled checks over 3 days, want 6", got, len(checks))
	}
	perDay := map[string][2]int{}
	for _, check := range checks {
		window := windowOf(check, paris)
		if window < 0 {
			t.Fatalf("check at %s is outside the windows", check.In(paris))
		}
		counts := perDay[localDate(check, paris)]
		counts[window]++
		perDay[localDate(check, paris)] = counts
	}
	for _, date := range []string{"2030-03-05", "2030-03-06", "2030-03-07"} {
		if perDay[date] != [2]int{1, 1} {
			t.Errorf("%s: checks per window %v, want one in each", date, perDay[date])
		}
	}
	if allEqual([]int{secondsOfDay(checks[1], paris), secondsOfDay(checks[3], paris), secondsOfDay(checks[5], paris)}) {
		t.Errorf("the evening check is at the same time every day: %v", checks)
	}
	item, err := database.GetWithHistory(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(item.PriceHistory) != 6 {
		t.Fatalf("price history has %d entries, want one per check", len(item.PriceHistory))
	}
	recorded := map[int64]bool{}
	for _, observation := range item.PriceHistory {
		if observation.AmountCents != 12999 || observation.Timestamp == nil {
			t.Fatalf("unexpected observation %+v", observation)
		}
		recorded[observation.Timestamp.Unix()] = true
	}
	for _, check := range checks {
		if !recorded[check.Add(time.Second).Unix()] {
			t.Errorf("no price recorded at the drawn time %s", check.In(paris))
		}
	}
}

// TC-021
func TestAmazonHumanChecks_TC021_StopDuringAWindowPreventsLaterChecks(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	var ids []string
	var times []time.Time
	for index := range 7 {
		id := "B00000021" + string(rune('1'+index))
		ids = append(ids, id)
		times = append(times, scheduleDay(6, 10).Add(time.Duration(index)*20*time.Minute))
		insertAt(t, database, id, "amazon", times[index])
	}
	now := newClock(scheduleDay(6, 0))
	recorder := &amazonRecorder{result: func(model.Listing) model.CollectionResult { return failed() }}
	service := startHuman(t, database, dir, now, recorder, 21)
	ctx := context.Background()
	for _, at := range times {
		now.Set(at.Add(time.Minute))
		service.collectDue(ctx)
	}
	for index, id := range ids {
		want := 0
		if index < amazonFailureLimit {
			want = 1
		}
		if got := recorder.calls(id); got != want {
			t.Errorf("item %d: %d checks, want %d", index+1, got, want)
		}
		if !stopped(t, database, id) {
			t.Errorf("item %d is not stopped after the 5th failure", index+1)
		}
	}
	for at := scheduleDay(19, 0); !at.After(scheduleDay(23, 15)); at = at.Add(15 * time.Minute) {
		now.Set(at)
		service.collectDue(ctx)
	}
	if recorder.total() != amazonFailureLimit {
		t.Fatalf("requests sent for stopped items: %v", recorder.ids)
	}
}

// TC-022
func TestAmazonHumanChecks_TC022_StoppedItemSkippedAtItsTime(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ctx := context.Background()
	insertAt(t, database, "B000000221", "amazon", scheduleDay(7, 0))
	if err := database.StopAmazonItems(ctx, scheduleDay(6, 0)); err != nil {
		t.Fatal(err)
	}
	insertAt(t, database, "B000000222", "amazon", scheduleDay(7, 0))
	insertAt(t, database, "B000000223", "amazon", scheduleDay(7, 0))
	if deleted, err := database.Delete(ctx, 0, "B000000223"); err != nil || !deleted {
		t.Fatalf("delete: %v %v", deleted, err)
	}
	recorder := &amazonRecorder{}
	service := startHuman(t, database, dir, newClock(scheduleDay(7, 1)), recorder, 22)
	service.collectDue(ctx)
	if recorder.calls("B000000222") != 1 || recorder.total() != 1 {
		t.Fatalf("checks %v, want only the active item", recorder.ids)
	}
	if !stopped(t, database, "B000000221") {
		t.Fatal("the stopped item was restarted by the schedule")
	}
	due, err := database.DueIDs(ctx, scheduleDay(23, 59))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range due {
		if id == "B000000223" {
			t.Fatal("the deleted item still has a scheduled check")
		}
	}
}

// TC-023
func TestAmazonHumanChecks_TC023_RestartBeforeTheDrawnTimeKeepsTheCheck(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	const id = "B000000231"
	insertAt(t, database, id, "amazon", scheduleDay(7, 40))
	now := newClock(scheduleDay(6, 30))
	recorder := &amazonRecorder{clock: now}
	service := startHuman(t, database, dir, now, recorder, 23)
	ctx := context.Background()
	service.collectDue(ctx)
	for index, restart := range []time.Time{scheduleDay(6, 35), scheduleDay(7, 20)} {
		service.Close()
		database.Close()
		now.Set(restart)
		database = openHumanStore(t, dir)
		service = startHuman(t, database, dir, now, recorder, uint64(230+index))
		service.collectDue(ctx)
		if got := nextCheck(t, database, id); !got.Equal(scheduleDay(7, 40)) {
			t.Fatalf("restart at %s moved the next check to %s", restart.In(paris), got.In(paris))
		}
	}
	if recorder.total() != 0 {
		t.Fatalf("checked before its time: %v", recorder.checkTimes())
	}
	for _, at := range []time.Time{scheduleDay(7, 41), scheduleDay(8, 0), scheduleDay(8, 59)} {
		now.Set(at)
		service.collectDue(ctx)
	}
	checked := recorder.checkTimes()
	if len(checked) != 1 || checked[0].Before(scheduleDay(7, 40)) || windowOf(checked[0], paris) != morning {
		t.Fatalf("morning checks at %v, want one at or after 07:40", checked)
	}
}

// TC-024
func TestAmazonHumanChecks_TC024_RestartAfterTheCheckDoesNotRepeatIt(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	const id = "B000000241"
	insertAt(t, database, id, "amazon", scheduleDay(6, 45))
	now := newClock(scheduleDay(6, 50))
	recorder := &amazonRecorder{}
	service := startHuman(t, database, dir, now, recorder, 24)
	ctx := context.Background()
	service.collectDue(ctx)
	eveningCheck := nextCheck(t, database, id)
	if recorder.calls(id) != 1 || windowOf(eveningCheck, paris) != evening || localDate(eveningCheck, paris) != "2030-03-05" {
		t.Fatalf("morning check: %d checks, next at %s", recorder.calls(id), eveningCheck.In(paris))
	}
	restart := func(at time.Time, runs ...time.Time) {
		t.Helper()
		before := nextCheck(t, database, id)
		service.Close()
		database.Close()
		now.Set(at)
		database = openHumanStore(t, dir)
		service = startHuman(t, database, dir, now, recorder, uint64(at.Unix()))
		for _, run := range runs {
			now.Set(run)
			service.collectDue(ctx)
		}
		if got := nextCheck(t, database, id); !got.Equal(before) {
			t.Fatalf("restart at %s moved the next check from %s to %s", at.In(paris), before.In(paris), got.In(paris))
		}
	}
	restart(scheduleDay(7, 0), scheduleDay(7, 0), scheduleDay(8, 59))
	if got := recorder.calls(id); got != 1 {
		t.Fatalf("the restart repeated the morning check: %d checks", got)
	}

	now.Set(eveningCheck.Add(time.Minute))
	service.collectDue(ctx)
	if got := recorder.calls(id); got != 2 {
		t.Fatalf("evening check: %d checks in total", got)
	}
	if tomorrow := nextCheck(t, database, id); windowOf(tomorrow, paris) != morning || localDate(tomorrow, paris) != "2030-03-06" {
		t.Fatalf("next check after the evening check at %s", tomorrow.In(paris))
	}
	restart(eveningCheck.Add(5*time.Minute), eveningCheck.Add(5*time.Minute), eveningCheck.Add(15*time.Minute))
	if got := recorder.calls(id); got != 2 {
		t.Fatalf("the restart repeated the evening check: %d checks", got)
	}
}

// The service started on a pre-#168 database schedules the restarted items in the next window and
// sends their checks: the Amazon gate is not stopped.
// TC-025 (schedule part; the data part is in internal/store)
func TestAmazonHumanChecks_TC025_RestartedItemsAreScheduledAndChecked(t *testing.T) {
	dir := t.TempDir()
	legacyStoppedDatabase(t, dir)
	database := openHumanStore(t, dir)
	now := newClock(scheduleDay(10, 0))
	recorder := &amazonRecorder{}
	service := startHuman(t, database, dir, now, recorder, 25)
	ctx := context.Background()
	for _, id := range legacyAmazonIDs {
		item, err := database.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if item.AmazonStoppedAt != nil || item.NextCheckAt == nil || windowOf(*item.NextCheckAt, paris) != evening || localDate(*item.NextCheckAt, paris) != "2030-03-05" {
			t.Fatalf("item %s after the upgrade: stopped %v, next check %v", id, item.AmazonStoppedAt, item.NextCheckAt)
		}
	}
	if got := amazonFailures(t, database); got != 0 {
		t.Fatalf("failure count after the upgrade: %d", got)
	}
	first := nextCheck(t, database, legacyAmazonIDs[0])
	now.Set(first.Add(time.Second))
	service.collectDue(ctx)
	if got := recorder.calls(legacyAmazonIDs[0]); got != 1 {
		t.Fatalf("the restarted item was not checked at its scheduled time: %d checks", got)
	}
}

// Protected mode: the restarted items of every owner get a scheduled check in a window.
// TC-028 (window part; the restart is in internal/store)
func TestAmazonHumanChecks_TC028_EveryOwnersItemsAreScheduled(t *testing.T) {
	dir := t.TempDir()
	legacyStoppedDatabase(t, dir, 1, 2)
	database := openHumanStore(t, dir)
	startHuman(t, database, dir, newClock(scheduleDay(10, 0)), &amazonRecorder{}, 28)
	for _, id := range legacyAmazonIDs {
		item, err := database.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if item.AmazonStoppedAt != nil || item.NextCheckAt == nil || windowOf(*item.NextCheckAt, paris) != evening {
			t.Fatalf("item %s: stopped %v, next check %v", id, item.AmazonStoppedAt, item.NextCheckAt)
		}
	}
}

// TC-031
func TestAmazonHumanChecks_TC031_LeboncoinKeepsItsFixedSchedule(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	insertAt(t, database, "lbc", "leboncoin", scheduleDay(8, 0))
	insertAt(t, database, "B000000311", "amazon", scheduleDay(6, 40))
	now := newClock(scheduleDay(0, 0))
	recorder := &amazonRecorder{clock: now}
	service := startHuman(t, database, dir, now, recorder, 31)
	var mu sync.Mutex
	var leboncoinTimes []time.Time
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		mu.Lock()
		leboncoinTimes = append(leboncoinTimes, now.Now())
		mu.Unlock()
		return model.CollectionResult{Result: "request_error", Message: "LeBoncoin fixture failure"}, leboncoin.SessionOutcome{}
	})
	ctx := context.Background()
	for at := scheduleDay(0, 0); at.Before(scheduleDay(0, 0).AddDate(0, 0, 1)); at = at.Add(time.Minute) {
		now.Set(at)
		service.collectDue(ctx)
	}
	mu.Lock()
	checked := append([]time.Time(nil), leboncoinTimes...)
	mu.Unlock()
	if len(checked) != 2 || !checked[0].Equal(scheduleDay(8, 0)) || !checked[1].Equal(scheduleDay(20, 0)) {
		t.Fatalf("LeBoncoin checks at %v, want 08:00 and 20:00", checked)
	}
	amazonTimes := recorder.checkTimes()
	if len(amazonTimes) != 2 || windowOf(amazonTimes[0], paris) != morning || windowOf(amazonTimes[1], paris) != evening {
		t.Fatalf("Amazon checks at %v, want one in each window", amazonTimes)
	}
	if got := amazonFailures(t, database); got != 0 || service.amazonGate.failures != 0 {
		t.Fatalf("LeBoncoin failures counted as Amazon failures: %d", got)
	}
}

// TC-032
func TestAmazonHumanChecks_TC032_NoDuplicateCheckWhileOneIsRunning(t *testing.T) {
	assertScheduled := func(t *testing.T, database *store.Store, id string, now time.Time) {
		t.Helper()
		if next := nextCheck(t, database, id); windowOf(next, paris) < 0 || !next.After(now) {
			t.Fatalf("next check of %s at %s, want a later window", id, next.In(paris))
		}
	}

	t.Run("Refresh price running", func(t *testing.T) {
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		const id = "B000000321"
		insertAt(t, database, id, "amazon", scheduleDay(7, 0))
		now := newClock(scheduleDay(7, 5))
		recorder := &amazonRecorder{started: make(chan string, 4), block: make(chan struct{})}
		service := startHuman(t, database, dir, now, recorder, 32)
		ctx := context.Background()
		if _, err := service.RefreshItem(ctx, id); err != nil {
			t.Fatal(err)
		}
		waitStarted(t, recorder.started)
		if err := database.SetNextCheck(ctx, id, now.Now().Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
		service.collectDue(ctx)
		close(recorder.block)
		waitIdle(t, service)
		if got := recorder.calls(id); got != 1 {
			t.Fatalf("%d checks, want 1", got)
		}
		assertScheduled(t, database, id, now.Now())
	})

	t.Run("first check after add running", func(t *testing.T) {
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		now := newClock(scheduleDay(7, 5))
		recorder := &amazonRecorder{started: make(chan string, 4), block: make(chan struct{})}
		service := startHuman(t, database, dir, now, recorder, 33)
		ctx := context.Background()
		added, err := service.Add(ctx, "https://www.amazon.fr/dp/B000000322", "")
		if err != nil {
			t.Fatal(err)
		}
		waitStarted(t, recorder.started)
		if err := database.SetNextCheck(ctx, added.ID, now.Now().Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
		service.collectDue(ctx)
		close(recorder.block)
		waitIdle(t, service)
		if got := recorder.calls(added.ID); got != 1 {
			t.Fatalf("%d checks, want 1", got)
		}
		assertScheduled(t, database, added.ID, now.Now())
	})
}
