package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	mathrand "math/rand/v2"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	_ "time/tzdata" // Europe/Paris on every machine

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

// Amazon human-like checks of issue #168 through the HTTP API, with a fake Amazon collector and a
// fake clock (service.NewWithOptions).

var parisZone = func() *time.Location {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		panic(err)
	}
	return location
}()

// humanTime is a time of Tuesday 5 March 2030 in Paris.
func humanTime(hour, minute int) time.Time {
	return time.Date(2030, time.March, 5, hour, minute, 0, 0, parisZone)
}

const (
	morningWindow = 0
	eveningWindow = 1
)

// amazonWindow returns the Amazon window holding t in Paris (06:00–09:00 or 19:00–23:00), or -1.
func amazonWindow(t time.Time) int {
	local := t.In(parisZone)
	seconds := local.Hour()*3600 + local.Minute()*60 + local.Second()
	switch {
	case seconds >= 6*3600 && seconds <= 9*3600:
		return morningWindow
	case seconds >= 19*3600 && seconds <= 23*3600:
		return eveningWindow
	}
	return -1
}

func parisDate(t time.Time) string { return t.In(parisZone).Format("2006-01-02") }

type humanClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *humanClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *humanClock) Set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = at
}

// amazonChecks is a fake Amazon collector recording every check. result nil is a success; with
// started set it reports each check start; with block set each check waits for block to be closed.
type amazonChecks struct {
	mu      sync.Mutex
	ids     []string
	result  func(model.Listing) model.CollectionResult
	started chan string
	block   chan struct{}
}

func (f *amazonChecks) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	f.mu.Lock()
	f.ids = append(f.ids, item.ID)
	result, started, block := f.result, f.started, f.block
	f.mu.Unlock()
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
	return model.CollectionResult{Result: "success", AmountCents: 12999, SecondHandStatus: "not_found"}
}

func (f *amazonChecks) calls(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, checked := range f.ids {
		if checked == id {
			count++
		}
	}
	return count
}

func (f *amazonChecks) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.ids)
}

func failedCheck(model.Listing) model.CollectionResult {
	return model.CollectionResult{Result: "request_error", Message: "Amazon could not be reached for a price check.", SecondHandStatus: "check_error",
		RequestURL: "https://www.amazon.fr/dp/B000000008", HTTPStatus: 503, ResponseExcerpt: "<html>busy</html>"}
}

func humanConfig(dir string) config.Config {
	return config.Config{DataDirectory: dir, Location: parisZone, CheckTimes: []string{"08:00", "20:00"}, StaleAfter: 36 * time.Hour, UserAgent: "test"}
}

func openHumanStore(t *testing.T, dir string) *store.Store {
	t.Helper()
	database, err := store.Open(humanConfig(dir))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// openHumanAPI serves the API of a service using checks for Amazon and the fake clock.
func openHumanAPI(t *testing.T, dir string, database *store.Store, clock *humanClock, checks *amazonChecks) (*Server, *service.Service) {
	t.Helper()
	var mu sync.Mutex
	source := mathrand.New(mathrand.NewPCG(168, 169))
	random := func(n int64) int64 {
		mu.Lock()
		defer mu.Unlock()
		return source.Int64N(n)
	}
	items := service.NewWithOptions(humanConfig(dir), database, service.Options{Amazon: checks, Now: clock.Now, Random: random,
		SchedulePause: func(context.Context) bool { return true }})
	t.Cleanup(items.Close)
	return New(humanConfig(dir), items), items
}

func insertHumanItem(t *testing.T, database *store.Store, id, platform string, next time.Time) {
	t.Helper()
	listing := model.Listing{ID: id, Platform: platform, ListingID: "123", Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/test/123"}
	if platform == "amazon" {
		listing = model.Listing{ID: id, Platform: platform, ListingID: id, ASIN: id, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + id}
	}
	if err := database.Insert(context.Background(), listing, listing.URL, next); err != nil {
		t.Fatal(err)
	}
}

func listItems(t *testing.T, server *Server) map[string]model.Item {
	t.Helper()
	recorder := callPath(server, http.MethodGet, "/api/v1/items", "")
	var body struct {
		Items []model.Item `json:"items"`
	}
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &body) != nil {
		t.Fatalf("GET /api/v1/items: %d %s", recorder.Code, recorder.Body.String())
	}
	items := map[string]model.Item{}
	for _, item := range body.Items {
		items[item.ID] = item
	}
	return items
}

func itemDetails(t *testing.T, server *Server, id string) model.Item {
	t.Helper()
	recorder := callPath(server, http.MethodGet, "/api/v1/items/"+id, "")
	var item model.Item
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &item) != nil {
		t.Fatalf("GET /api/v1/items/%s: %d %s", id, recorder.Code, recorder.Body.String())
	}
	return item
}

// waitChecked waits until the item's check is finished and returns its details.
func waitChecked(t *testing.T, server *Server, id string) model.Item {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if item := itemDetails(t, server, id); item.LastAttempt != nil && item.LastAttempt.Result != "pending" {
			return item
		}
	}
	t.Fatalf("the check of %s did not finish", id)
	return model.Item{}
}

func refresh(t *testing.T, server *Server, id string) {
	t.Helper()
	if recorder := callPath(server, http.MethodPost, "/api/v1/items/"+id+"/refresh", ""); recorder.Code != http.StatusAccepted {
		t.Fatalf("Refresh price of %s: %d %s", id, recorder.Code, recorder.Body.String())
	}
}

func addItem(t *testing.T, server *Server, url string) model.Item {
	t.Helper()
	recorder := callPath(server, http.MethodPost, "/api/v1/items", `{"url":"`+url+`"}`)
	var item model.Item
	if recorder.Code != http.StatusCreated || json.Unmarshal(recorder.Body.Bytes(), &item) != nil {
		t.Fatalf("add %s: %d %s", url, recorder.Code, recorder.Body.String())
	}
	return item
}

func waitCalls(t *testing.T, checks *amazonChecks, id string, want int) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if checks.calls(id) >= want {
			return
		}
	}
	t.Fatalf("item %s: %d Amazon checks, want %d", id, checks.calls(id), want)
}

func storedFailures(t *testing.T, database *store.Store) int {
	t.Helper()
	state, err := database.AmazonRequestState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return state.ConsecutiveFailures
}

func equalTime(got *time.Time, want time.Time) bool { return got != nil && got.Equal(want) }

type syncBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// TC-005
func TestAmazonHumanChecks_TC005_BrowserSuccessIsStoredLikeToday(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ctx := context.Background()
	insertHumanItem(t, database, "B000000005", "amazon", humanTime(20, 0))
	insertHumanItem(t, database, "B000000055", "amazon", humanTime(20, 0))
	if err := database.SaveAmazonFailures(ctx, 2); err != nil {
		t.Fatal(err)
	}
	clock := &humanClock{now: humanTime(14, 0)}
	var otherChecks atomic.Int32
	title := "Price found product"
	checks := &amazonChecks{result: func(item model.Listing) model.CollectionResult {
		if item.ID == "B000000005" {
			return model.CollectionResult{Result: "success", AmountCents: 12999, Title: &title, SecondHandStatus: "available",
				SecondHandAmountCents: 8950, SecondHandCondition: "very_good", SecondHandConditionLabel: "Très bon état"}
		}
		if otherChecks.Add(1) == 1 {
			return failedCheck(item)
		}
		return model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available.", SecondHandStatus: "not_found"}
	}}
	server, _ := openHumanAPI(t, dir, database, clock, checks)

	refresh(t, server, "B000000005")
	details := waitChecked(t, server, "B000000005")
	item := listItems(t, server)["B000000005"]
	checkedAt := humanTime(14, 0)
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 || !equalTime(item.LatestPrice.Timestamp, checkedAt) {
		t.Fatalf("latest price %+v, want 129.99 checked at %s", item.LatestPrice, checkedAt)
	}
	if item.LastAttempt == nil || item.LastAttempt.Result != "success" || !item.LastAttempt.Timestamp.Equal(checkedAt) || item.Status != "active" {
		t.Fatalf("last attempt %+v, status %s", item.LastAttempt, item.Status)
	}
	if offer := item.SecondHandOffer; offer == nil || offer.Status != "available" || offer.LatestDetection == nil ||
		offer.LatestDetection.AmountCents != 8950 || offer.LatestDetection.Condition != "very_good" || !offer.LatestDetection.Timestamp.Equal(checkedAt) {
		t.Fatalf("second-hand offer %+v", item.SecondHandOffer)
	}
	if item.Title == nil || *item.Title != title || len(details.PriceHistory) != 1 {
		t.Fatalf("title %v, price history %+v", item.Title, details.PriceHistory)
	}
	if got := storedFailures(t, database); got != 0 {
		t.Fatalf("failure count %d after a success, want 0", got)
	}

	clock.Set(humanTime(14, 5))
	refresh(t, server, "B000000055")
	waitChecked(t, server, "B000000055")
	if got := storedFailures(t, database); got != 1 {
		t.Fatalf("failure count %d after a failure, want 1", got)
	}
	clock.Set(humanTime(14, 10))
	refresh(t, server, "B000000055")
	unavailable := waitChecked(t, server, "B000000055")
	if unavailable.Status != "unavailable" || unavailable.LastAttempt.Result != "unavailable" || !unavailable.LastAttempt.Timestamp.Equal(humanTime(14, 10)) {
		t.Fatalf("unavailable listing: status %s, attempt %+v", unavailable.Status, unavailable.LastAttempt)
	}
	if got := storedFailures(t, database); got != 0 {
		t.Fatalf("failure count %d after an unavailable listing, want 0", got)
	}
}

// TC-008
func TestAmazonHumanChecks_TC008_BrowserFailuresKeepThePriceAndStopAfterFive(t *testing.T) {
	setup := func(t *testing.T, checks *amazonChecks) (*Server, *service.Service, *store.Store, *humanClock) {
		t.Helper()
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		for _, id := range []string{"B000000081", "B000000082"} {
			insertHumanItem(t, database, id, "amazon", humanTime(20, 0))
			if err := database.RecordCollection(context.Background(), id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "not_found"},
				humanTime(8, 0).AddDate(0, 0, -1)); err != nil {
				t.Fatal(err)
			}
		}
		clock := &humanClock{now: humanTime(14, 0)}
		server, items := openHumanAPI(t, dir, database, clock, checks)
		return server, items, database, clock
	}
	var logs syncBuffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })

	checks := &amazonChecks{result: failedCheck}
	server, items, _, clock := setup(t, checks)
	ctx := context.Background()
	steps := []struct {
		id        string
		refreshed bool
	}{{"B000000081", false}, {"B000000082", true}, {"B000000082", false}, {"B000000081", true}}
	for index, step := range steps {
		clock.Set(humanTime(14, index+1))
		if step.refreshed {
			refresh(t, server, step.id)
			if details := waitChecked(t, server, step.id); details.LastAttempt.Result != "request_error" || details.Status != "retrieval_error" {
				t.Fatalf("failed Refresh price %d not reported: status %s, attempt %+v", index+1, details.Status, details.LastAttempt)
			}
		} else {
			items.Collect(ctx, step.id)
		}
		for id, item := range listItems(t, server) {
			if item.AmazonStoppedAt != nil || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
				t.Fatalf("after failure %d, item %s: stopped %v, latest price %+v", index+1, id, item.AmazonStoppedAt, item.LatestPrice)
			}
		}
	}
	clock.Set(humanTime(14, 30))
	items.Collect(ctx, "B000000081")
	for id, item := range listItems(t, server) {
		if !equalTime(item.AmazonStoppedAt, humanTime(14, 30)) || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
			t.Fatalf("after the 5th failure, item %s: stopped %v, latest price %+v", id, item.AmazonStoppedAt, item.LatestPrice)
		}
	}
	if got := strings.Count(logs.String(), "Amazon request failed"); got != 5 {
		t.Fatalf("%d failures logged, want 5:\n%s", got, logs.String())
	}

	t.Run("a success resets the count", func(t *testing.T) {
		var count atomic.Int32
		checks := &amazonChecks{result: func(item model.Listing) model.CollectionResult {
			if count.Add(1) == 5 {
				return model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "not_found"}
			}
			return failedCheck(item)
		}}
		server, items, _, _ := setup(t, checks)
		for range 9 {
			items.Collect(context.Background(), "B000000081")
		}
		for id, item := range listItems(t, server) {
			if item.AmazonStoppedAt != nil {
				t.Fatalf("item %s stopped after 4 failures, a success and 4 failures", id)
			}
		}
	})
}

// TC-016
func TestAmazonHumanChecks_TC016_AddChecksAtOnceThenTheNextWindow(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	clock := &humanClock{now: humanTime(7, 0)}
	checks := &amazonChecks{}
	server, _ := openHumanAPI(t, dir, database, clock, checks)
	tests := []struct {
		at     time.Time
		asin   string
		window int
		date   string
	}{
		{humanTime(7, 0), "B000000016", eveningWindow, "2030-03-05"},
		{humanTime(5, 59), "B000000161", morningWindow, "2030-03-05"},
		{humanTime(23, 30), "B000000162", morningWindow, "2030-03-06"},
	}
	for _, test := range tests {
		clock.Set(test.at)
		added := addItem(t, server, "https://www.amazon.fr/dp/"+test.asin)
		waitCalls(t, checks, added.ID, 1)
		item := waitChecked(t, server, added.ID)
		if item.NextCheckAt == nil || amazonWindow(*item.NextCheckAt) != test.window || parisDate(*item.NextCheckAt) != test.date {
			t.Fatalf("added at %s: next check %v, want window %d of %s", test.at.In(parisZone), item.NextCheckAt, test.window, test.date)
		}
	}
	before := checks.total()
	duplicate := callPath(server, http.MethodPost, "/api/v1/items", `{"url":"https://www.amazon.fr/dp/B000000016"}`)
	assertError(t, duplicate, http.StatusConflict, "ITEM_ALREADY_TRACKED", "This listing is already being tracked.")
	time.Sleep(50 * time.Millisecond)
	if checks.total() != before {
		t.Fatal("adding a tracked listing again started a check")
	}
}

// TC-017
func TestAmazonHumanChecks_TC017_FailedFirstCheckKeepsTheSchedule(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ctx := context.Background()
	clock := &humanClock{now: humanTime(14, 0)}
	checks := &amazonChecks{result: failedCheck}
	server, _ := openHumanAPI(t, dir, database, clock, checks)
	added := addItem(t, server, "https://www.amazon.fr/dp/B000000017")
	item := waitChecked(t, server, added.ID)
	if item.Status != "retrieval_error" || item.LastAttempt.Result != "request_error" || storedFailures(t, database) != 1 {
		t.Fatalf("failed first check: status %s, attempt %+v, failures %d", item.Status, item.LastAttempt, storedFailures(t, database))
	}
	if item.NextCheckAt == nil || amazonWindow(*item.NextCheckAt) != eveningWindow || parisDate(*item.NextCheckAt) != "2030-03-05" {
		t.Fatalf("next check %v, want the evening window of the same day", item.NextCheckAt)
	}

	insertHumanItem(t, database, "B000000171", "amazon", humanTime(20, 0))
	if err := database.StopAmazonItems(ctx, humanTime(13, 0)); err != nil {
		t.Fatal(err)
	}
	clock.Set(humanTime(14, 30))
	another := addItem(t, server, "https://www.amazon.fr/dp/B000000172")
	waitCalls(t, checks, another.ID, 1)
	if item := waitChecked(t, server, another.ID); item.AmazonStoppedAt != nil || storedFailures(t, database) != 2 {
		t.Fatalf("item added while Amazon items are stopped: stopped %v, failures %d", item.AmazonStoppedAt, storedFailures(t, database))
	}
	if stopped := listItems(t, server)["B000000171"]; stopped.AmazonStoppedAt == nil {
		t.Fatal("the stopped item was restarted by the add")
	}
}

// TC-018
func TestAmazonHumanChecks_TC018_RefreshPriceDoesNotMoveTheSchedule(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	const id = "B000000018"
	scheduled := humanTime(20, 47)
	insertHumanItem(t, database, id, "amazon", scheduled)
	clock := &humanClock{now: humanTime(14, 0)}
	checks := &amazonChecks{started: make(chan string, 4), block: make(chan struct{})}
	server, _ := openHumanAPI(t, dir, database, clock, checks)

	refresh(t, server, id)
	select {
	case <-checks.started:
	case <-time.After(5 * time.Second):
		t.Fatal("Refresh price did not start the check at once")
	}
	if running := itemDetails(t, server, id); running.Status != "pending" || running.LastAttempt == nil || running.LastAttempt.Result != "pending" {
		t.Fatalf("the running refresh is not shown: status %s, attempt %+v", running.Status, running.LastAttempt)
	}
	refresh(t, server, id)
	close(checks.block)
	item := waitChecked(t, server, id)
	if checks.calls(id) != 1 {
		t.Fatalf("%d checks for two refreshes during one check, want 1", checks.calls(id))
	}
	if !equalTime(item.NextCheckAt, scheduled) {
		t.Fatalf("next check %v after Refresh price, want %s", item.NextCheckAt, scheduled)
	}

	if err := database.StopAmazonItems(context.Background(), humanTime(14, 30)); err != nil {
		t.Fatal(err)
	}
	clock.Set(humanTime(15, 0))
	refresh(t, server, id)
	item = waitChecked(t, server, id)
	if checks.calls(id) != 2 || item.AmazonStoppedAt != nil || !equalTime(item.NextCheckAt, scheduled) {
		t.Fatalf("Refresh price of a stopped item: %d checks, stopped %v, next check %v", checks.calls(id), item.AmazonStoppedAt, item.NextCheckAt)
	}
}

// TC-019
func TestAmazonHumanChecks_TC019_RefreshAllSkipsStoppedItemsAndKeepsTheSchedule(t *testing.T) {
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ctx := context.Background()
	next := map[string]time.Time{"B000000191": humanTime(19, 31), "B000000192": humanTime(21, 12), "lbc": humanTime(20, 0)}
	insertHumanItem(t, database, "B000000191", "amazon", next["B000000191"])
	if err := database.StopAmazonItems(ctx, humanTime(7, 0)); err != nil {
		t.Fatal(err)
	}
	insertHumanItem(t, database, "B000000192", "amazon", next["B000000192"])
	insertHumanItem(t, database, "lbc", "leboncoin", next["lbc"])
	var leboncoinRequests atomic.Int32
	original := http.DefaultTransport
	http.DefaultTransport = claudeTransport(func(r *http.Request) (*http.Response, error) {
		leboncoinRequests.Add(1)
		page := `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","price":[12]}}}}</script>`
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/html"}}, Body: io.NopCloser(strings.NewReader(page)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	clock := &humanClock{now: humanTime(14, 0)}
	checks := &amazonChecks{}
	server, _ := openHumanAPI(t, dir, database, clock, checks)

	recorder := callPath(server, http.MethodPost, "/api/v1/items/refresh", "")
	if recorder.Code != http.StatusAccepted || !strings.Contains(recorder.Body.String(), `"itemsQueued":3`) {
		t.Fatalf("Refresh all prices: %d %s", recorder.Code, recorder.Body.String())
	}
	waitChecked(t, server, "B000000192")
	waitChecked(t, server, "lbc")
	if checks.calls("B000000192") != 1 || checks.calls("B000000191") != 0 || leboncoinRequests.Load() == 0 {
		t.Fatalf("checks: active Amazon %d, stopped Amazon %d, LeBoncoin requests %d", checks.calls("B000000192"), checks.calls("B000000191"), leboncoinRequests.Load())
	}
	for id, item := range listItems(t, server) {
		if !equalTime(item.NextCheckAt, next[id]) {
			t.Errorf("item %s: next check %v after Refresh all prices, want %s", id, item.NextCheckAt, next[id])
		}
	}

	t.Run("no tracked item", func(t *testing.T) {
		emptyDir := t.TempDir()
		checks := &amazonChecks{}
		server, _ := openHumanAPI(t, emptyDir, openHumanStore(t, emptyDir), &humanClock{now: humanTime(14, 0)}, checks)
		recorder := callPath(server, http.MethodPost, "/api/v1/items/refresh", "")
		if recorder.Code != http.StatusAccepted || !strings.Contains(recorder.Body.String(), `"itemsQueued":0`) || checks.total() != 0 {
			t.Fatalf("Refresh all prices without items: %d %s", recorder.Code, recorder.Body.String())
		}
	})
}

// TC-029
func TestAmazonHumanChecks_TC029_NoStoppedItemAfterTheUpgrade(t *testing.T) {
	dir := t.TempDir()
	ids := []string{"B000000291", "B000000292", "B000000293"}
	legacy, err := store.Open(humanConfig(dir))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, id := range ids {
		insertHumanItem(t, legacy, id, "amazon", time.Date(2026, time.November, 20, 19, 0, 0, 0, time.UTC))
		if err := legacy.RecordCollection(ctx, id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "not_found"},
			time.Date(2026, time.November, 19, 7, 0, 0, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}
	if err := legacy.StopAmazonItems(ctx, time.Date(2026, time.November, 20, 7, 26, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := legacy.SaveAmazonFailures(ctx, 5); err != nil {
		t.Fatal(err)
	}
	legacy.Close()
	raw, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	var marked bool
	if err := raw.QueryRow("SELECT EXISTS (SELECT 1 FROM pragma_table_info('amazon_request_state') WHERE name = 'human_checks_restarted')").Scan(&marked); err != nil {
		t.Fatal(err)
	}
	if marked {
		if _, err := raw.Exec("ALTER TABLE amazon_request_state DROP COLUMN human_checks_restarted"); err != nil {
			t.Fatal(err)
		}
	}
	raw.Close()

	server, _ := openHumanAPI(t, dir, openHumanStore(t, dir), &humanClock{now: humanTime(10, 0)}, &amazonChecks{})
	body := callPath(server, http.MethodGet, "/api/v1/items", "").Body.String()
	if strings.Contains(body, "2026-11-20T07:26:00") || strings.Count(body, `"amazonStoppedAt":null`) != len(ids) {
		t.Fatalf("stopped items after the upgrade: %s", body)
	}
	items := listItems(t, server)
	for _, id := range ids {
		if item := items[id]; item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
			t.Fatalf("item %s lost its latest price: %+v", id, item.LatestPrice)
		}
		if details := itemDetails(t, server, id); len(details.PriceHistory) != 1 {
			t.Fatalf("item %s price history %+v", id, details.PriceHistory)
		}
	}
}
