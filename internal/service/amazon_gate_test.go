package service

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

type fakeAmazon func(context.Context, model.Listing) model.CollectionResult

func (f fakeAmazon) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	return f(ctx, item)
}

func failed() model.CollectionResult {
	return model.CollectionResult{Result: "request_error", Message: "Amazon returned HTTP 503", RequestURL: "https://www.amazon.fr/s", HTTPStatus: 503, ResponseExcerpt: "<html>busy</html>"}
}

func succeeded() model.CollectionResult {
	return model.CollectionResult{Result: "success", AmountCents: 100}
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &logs
}

func TestAmazonGateStopsAfterFiveConsecutiveFailures(t *testing.T) {
	service, _ := newSessionService(t)
	gate := service.amazonGate
	for _, result := range []func() model.CollectionResult{failed, failed, failed, failed, succeeded, failed, failed, failed, failed} {
		if _, err := gate.do(context.Background(), result); err != nil {
			t.Fatal(err)
		}
	}
	if state := gate.current(); state.Blocked || state.ConsecutiveFailures != 4 {
		t.Fatalf("blocked too early: %+v", state)
	}
	logs := captureLog(t)
	if _, err := gate.do(context.Background(), failed); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "url=https://www.amazon.fr/s") || !strings.Contains(logs.String(), "status=503") ||
		!strings.Contains(logs.String(), "response=<html>busy</html>") || !strings.Contains(logs.String(), "Amazon requests stopped after 5") {
		t.Fatalf("unexpected logs %q", logs.String())
	}
	called := false
	if _, err := gate.do(context.Background(), func() model.CollectionResult { called = true; return succeeded() }); !errors.Is(err, errAmazonStopped) || called {
		t.Fatalf("blocked gate ran the request: %v %v", err, called)
	}
	// The stop survives a restart of the process.
	reopened := newAmazonGate(service.store, time.Now)
	if state := reopened.current(); !state.Blocked || state.StoppedAt == nil {
		t.Fatalf("stop not persisted: %+v", state)
	}
	reopened.restart()
	if state := reopened.current(); state.Blocked || state.StoppedAt == nil {
		t.Fatalf("restart must keep stoppedAt until a success: %+v", state)
	}
	if _, err := reopened.do(context.Background(), succeeded); err != nil || reopened.current().StoppedAt != nil {
		t.Fatalf("success did not clear stoppedAt: %v %+v", err, reopened.current())
	}
}

func TestAmazonGateIgnoresCancelledRequests(t *testing.T) {
	service, _ := newSessionService(t)
	gate := service.amazonGate
	for range 4 {
		gate.do(context.Background(), failed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if _, err := gate.do(ctx, func() model.CollectionResult { cancel(); return failed() }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request error = %v", err)
	}
	if state := gate.current(); state.Blocked || state.ConsecutiveFailures != 4 {
		t.Fatalf("cancelled request changed the state: %+v", state)
	}
	if saved := newAmazonGate(service.store, time.Now).current(); saved.ConsecutiveFailures != 4 {
		t.Fatalf("cancelled request changed the saved state: %+v", saved)
	}
}

func TestAmazonGateSerializesConcurrentRequests(t *testing.T) {
	service, _ := newSessionService(t)
	var inFlight, maximum atomic.Int32
	var group sync.WaitGroup
	for index := 0; index < 20; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			service.amazonGate.do(context.Background(), func() model.CollectionResult {
				if current := inFlight.Add(1); current > maximum.Load() {
					maximum.Store(current)
				}
				time.Sleep(time.Millisecond)
				inFlight.Add(-1)
				if index%2 == 0 {
					return failed()
				}
				return succeeded()
			})
		}(index)
	}
	group.Wait()
	if maximum.Load() != 1 {
		t.Fatalf("%d Amazon requests ran at once", maximum.Load())
	}
}

func TestStoppedAmazonRequestsSkipTrackedChecksAndRefresh(t *testing.T) {
	service, database := newSessionService(t)
	ctx := context.Background()
	insertListing(t, database, "amazon-item", "amazon", "https://www.amazon.fr/dp/B012345678")
	calls := 0
	service.collectors["amazon"] = fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { calls++; return succeeded() })
	now := time.Now()
	if err := database.SaveAmazonRequestState(ctx, store.AmazonRequestState{ConsecutiveFailures: 5, Blocked: true, StoppedAt: &now}); err != nil {
		t.Fatal(err)
	}
	service.amazonGate = newAmazonGate(database, time.Now)
	service.Collect(ctx, "amazon-item")
	item, err := database.Get(ctx, "amazon-item")
	if err != nil || calls != 0 || item.LastAttempt != nil {
		t.Fatalf("blocked check sent or recorded: calls=%d %+v %v", calls, item.LastAttempt, err)
	}
	var typed *Error
	if _, err := service.RefreshItem(ctx, "amazon-item"); !errors.As(err, &typed) || typed.Code != "AMAZON_REQUESTS_STOPPED" || typed.Status != 409 {
		t.Fatalf("refresh error = %v", err)
	}
}

func TestRefreshOfSearchItemIsRefused(t *testing.T) {
	cfg := config.Config{DataDirectory: t.TempDir(), Location: time.UTC, CheckTimes: []string{"08:00"}, StaleAfter: time.Hour, UserAgent: "test"}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	service := New(cfg, database)
	defer service.Close()
	ctx := context.Background()
	search := store.Search{ID: "s1", URL: "https://www.amazon.fr/s?k=x", AddedAt: time.Now(), NextRunAt: time.Now()}
	url := "https://www.amazon.fr/dp/B000000001"
	if err := database.InsertSearch(ctx, search); err != nil {
		t.Fatal(err)
	}
	if err := database.CaptureSearch(ctx, "s1", 0, []store.CapturedItem{{ID: "i1", ASIN: "B000000001", Marketplace: "amazon.fr", CanonicalURL: url, URL: url}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	var typed *Error
	if _, err := service.RefreshItem(ctx, "i1"); !errors.As(err, &typed) || typed.Code != "ITEM_NOT_TRACKED" {
		t.Fatalf("refresh error = %v", err)
	}
}

func TestSearchWindows(t *testing.T) {
	paris := func(month time.Month, day, hour, minute int) time.Time {
		return time.Date(2026, month, day, hour, minute, 0, 0, parisLocation)
	}
	for _, test := range []struct {
		at     time.Time
		inside bool
		next   time.Time
	}{
		{paris(10, 7, 21, 59), false, paris(10, 7, 22, 0)},
		{paris(10, 7, 22, 0), true, paris(10, 8, 6, 0)},
		{paris(10, 8, 0, 59), true, paris(10, 8, 6, 0)},
		{paris(10, 8, 1, 0), false, paris(10, 8, 6, 0)},
		{paris(10, 8, 5, 59), false, paris(10, 8, 6, 0)},
		{paris(10, 8, 7, 59), true, paris(10, 8, 22, 0)},
		{paris(10, 8, 8, 0), false, paris(10, 8, 22, 0)},
		{paris(3, 29, 1, 30), false, paris(3, 29, 6, 0)},  // spring DST change day
		{paris(10, 25, 0, 30), true, paris(10, 25, 6, 0)}, // autumn DST change day
		{paris(10, 24, 23, 0), true, paris(10, 25, 6, 0)},
	} {
		if inSearchWindow(test.at) != test.inside || !nextSearchWindowStart(test.at).Equal(test.next) {
			t.Errorf("%s: inside=%v next=%s", test.at, inSearchWindow(test.at), nextSearchWindowStart(test.at).In(parisLocation))
		}
	}
}
