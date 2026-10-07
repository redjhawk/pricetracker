package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// fakeBrowser is the fake clock and Amazon collector of a search worker; it records every action.
type fakeBrowser struct {
	now     time.Time
	events  []string
	fail    map[string]int // ASIN or "results" → number of failures left
	results int            // products on the results page
}

func (f *fakeBrowser) Now() time.Time { return f.now }

func (f *fakeBrowser) Sleep(ctx context.Context, d time.Duration) bool {
	f.events = append(f.events, "pause")
	f.now = f.now.Add(d)
	return ctx.Err() == nil
}

func (f *fakeBrowser) failing(key string) bool {
	if f.fail[key] > 0 {
		f.fail[key]--
		return true
	}
	return false
}

func (f *fakeBrowser) FetchSearch(_ context.Context, search amazon.SearchURLResult) ([]amazon.SearchProduct, model.CollectionResult) {
	f.events = append(f.events, "results")
	if f.failing("results") {
		return nil, failed()
	}
	products := make([]amazon.SearchProduct, 0, f.results)
	for index := 1; index <= f.results; index++ {
		asin := fmt.Sprintf("B%09d", index)
		url := "https://www.amazon.fr/dp/" + asin
		products = append(products, amazon.SearchProduct{ASIN: asin, Marketplace: "amazon.fr", URL: url, Canonical: amazon.ParseURL(url).Canonical})
	}
	return products, model.CollectionResult{Result: "success"}
}

func (f *fakeBrowser) CollectProduct(_ context.Context, item model.Listing) model.CollectionResult {
	f.events = append(f.events, "item "+item.ASIN)
	if f.failing(item.ASIN) {
		return failed()
	}
	return succeeded()
}

// newTestWorker returns a worker inside the 22:00 Paris window with one due search of three items.
func newTestWorker(t *testing.T) (*searchWorker, *fakeBrowser, *store.Store) {
	t.Helper()
	service, database := newSessionService(t)
	start := time.Date(2026, 10, 7, 22, 0, 0, 0, parisLocation)
	browser := &fakeBrowser{now: start, fail: map[string]int{}, results: 3}
	service.amazonGate.now = browser.Now
	if err := database.InsertSearch(context.Background(), store.Search{ID: "s1", URL: "https://www.amazon.fr/s?k=lego", AddedAt: start, NextRunAt: start}); err != nil {
		t.Fatal(err)
	}
	return &searchWorker{service: service, collector: browser, clock: browser, pause: func() time.Duration { return 30 * time.Second }}, browser, database
}

func runOnePass(t *testing.T, worker *searchWorker) store.Search {
	t.Helper()
	search, err := worker.service.store.Search(context.Background(), 0, "s1")
	if err != nil {
		t.Fatal(err)
	}
	worker.runPass(context.Background(), search)
	search, err = worker.service.store.Search(context.Background(), 0, "s1")
	if err != nil {
		t.Fatal(err)
	}
	return search
}

func TestSearchPassFollowsHumanOrder(t *testing.T) {
	worker, browser, database := newTestWorker(t)
	logs := captureLog(t)
	search := runOnePass(t, worker)
	want := "results,pause,item B000000001,pause,pause,item B000000002,pause,pause,item B000000003,pause"
	if got := strings.Join(browser.events, ","); got != want {
		t.Fatalf("actions\n got %s\nwant %s", got, want)
	}
	if !strings.Contains(logs.String(), "close item B000000001") {
		t.Fatalf("missing close log: %s", logs.String())
	}
	if search.PassPosition != 0 || search.ItemCount != 3 || !search.NextRunAt.Equal(time.Date(2026, 10, 8, 6, 0, 0, 0, parisLocation)) {
		t.Fatalf("unexpected search after pass %+v", search)
	}
	ids, _ := database.SearchItemIDs(context.Background(), "s1")
	if item, err := database.Get(context.Background(), ids[0]); err != nil || item.LatestPrice.AmountCents != 100 || item.Tracked {
		t.Fatalf("item not recorded %+v %v", item, err)
	}
}

func TestSearchPassRetriesFailedItemOnceAtTheEnd(t *testing.T) {
	worker, browser, _ := newTestWorker(t)
	browser.fail["B000000002"] = 2
	runOnePass(t, worker)
	got := strings.Join(browser.events, ",")
	if !strings.HasSuffix(got, "item B000000003,pause,pause,item B000000002,pause") || strings.Count(got, "item B000000002") != 2 {
		t.Fatalf("unexpected actions %s", got)
	}
}

func TestHumanPauseBounds(t *testing.T) {
	for index := 0; index < 1000; index++ {
		if pause := humanPause(); pause < 30*time.Second || pause > 120*time.Second {
			t.Fatalf("pause %s out of bounds", pause)
		}
	}
}

func TestSearchPassStopsOutsideWindowAndResumes(t *testing.T) {
	worker, browser, _ := newTestWorker(t)
	browser.now = time.Date(2026, 10, 8, 0, 59, 0, 0, parisLocation)
	search := runOnePass(t, worker) // results at 00:59:00, item 1 at 00:59:30, read until 01:00: the window is over
	if search.PassPosition != 2 || strings.Count(strings.Join(browser.events, ","), "item ") != 1 {
		t.Fatalf("unexpected stop %+v %v", search, browser.events)
	}
	browser.now, browser.events = time.Date(2026, 10, 8, 6, 0, 0, 0, parisLocation), nil
	runOnePass(t, worker)
	if got := strings.Join(browser.events, ","); !strings.HasPrefix(got, "pause,item B000000002") || strings.Contains(got, "results") {
		t.Fatalf("pass did not resume at item 2: %s", got)
	}
}

func TestFiveFailuresStopSearchesAndTrackedChecks(t *testing.T) {
	worker, browser, database := newTestWorker(t)
	browser.results = 6
	browser.fail = map[string]int{"B000000001": 9, "B000000002": 9, "B000000003": 9, "B000000004": 9, "B000000005": 9}
	logs := captureLog(t)
	search := runOnePass(t, worker)
	if !worker.service.amazonGate.current().Blocked || search.PassPosition != 6 || strings.Contains(strings.Join(browser.events, ","), "B000000006") {
		t.Fatalf("pass not stopped: %+v %v", search, browser.events)
	}
	if !strings.Contains(logs.String(), "status=503") || !strings.Contains(logs.String(), "response=<html>busy</html>") {
		t.Fatalf("server response not logged: %s", logs.String())
	}
	insertListing(t, database, "tracked", "amazon", "https://www.amazon.fr/dp/B012345678")
	calls := 0
	worker.service.collectors["amazon"] = fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { calls++; return succeeded() })
	worker.service.Collect(context.Background(), "tracked")
	if calls != 0 {
		t.Fatal("tracked check sent while Amazon requests are stopped")
	}
}
