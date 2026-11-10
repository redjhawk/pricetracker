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

	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

type fakeAmazon func(context.Context, model.Listing) model.CollectionResult

func (f fakeAmazon) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	return f(ctx, item)
}

func failed() model.CollectionResult {
	return model.CollectionResult{Result: "request_error", Message: "Amazon returned HTTP 503", RequestURL: "https://www.amazon.fr/dp/B012345678", HTTPStatus: 503, ResponseExcerpt: "<html>busy</html>"}
}

func succeeded() model.CollectionResult {
	return model.CollectionResult{Result: "success", AmountCents: 100, SecondHandStatus: "check_error"}
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &logs
}

func insertAmazonItems(t *testing.T, database *store.Store, ids ...string) {
	t.Helper()
	for _, id := range ids {
		insertListing(t, database, id, "amazon", "https://www.amazon.fr/dp/"+id)
	}
}

func stopped(t *testing.T, database *store.Store, id string) bool {
	t.Helper()
	isStopped, err := database.AmazonStopped(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return isStopped
}

// waitIdle waits until no collection is in flight.
func waitIdle(t *testing.T, service *Service) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		service.mu.Lock()
		idle := len(service.inFlight) == 0
		service.mu.Unlock()
		if idle {
			return
		}
	}
	t.Fatal("collections still running")
}

func TestAmazonGateStopsEveryAmazonItemAfterFiveConsecutiveFailures(t *testing.T) {
	service, database := newSessionService(t)
	insertAmazonItems(t, database, "a", "b")
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/x/123")
	gate, ctx := service.amazonGate, context.Background()
	for _, result := range []func() model.CollectionResult{failed, failed, failed, failed, succeeded, failed, failed, failed, failed} {
		if _, err := gate.do(ctx, "a", false, result); err != nil {
			t.Fatal(err)
		}
	}
	if stopped(t, database, "a") || stopped(t, database, "b") || gate.failures != 4 {
		t.Fatalf("stopped too early: failures=%d", gate.failures)
	}
	logs := captureLog(t)
	if _, err := gate.do(ctx, "a", false, failed); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "url=https://www.amazon.fr/dp/B012345678") || !strings.Contains(logs.String(), "status=503") ||
		!strings.Contains(logs.String(), "response=<html>busy</html>") || !strings.Contains(logs.String(), "Amazon checks stopped after 5") {
		t.Fatalf("unexpected logs %q", logs.String())
	}
	if !stopped(t, database, "a") || !stopped(t, database, "b") || stopped(t, database, "lbc") || gate.failures != 0 {
		t.Fatalf("5th failure must stop every Amazon item only: failures=%d", gate.failures)
	}
	called := false
	if _, err := gate.do(ctx, "b", false, func() model.CollectionResult { called = true; return succeeded() }); !errors.Is(err, errAmazonStopped) || called {
		t.Fatalf("stopped item was requested: %v %v", err, called)
	}
}

func TestAmazonGateIgnoresCancelledRequests(t *testing.T) {
	service, _ := newSessionService(t)
	ctx, cancel := context.WithCancel(context.Background())
	if _, err := service.amazonGate.do(ctx, "a", false, func() model.CollectionResult { cancel(); return failed() }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request error = %v", err)
	}
	if saved := newAmazonGate(service.store, time.Now); saved.failures != 0 || service.amazonGate.failures != 0 {
		t.Fatalf("cancelled request counted as a failure: %d", saved.failures)
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
			service.amazonGate.do(context.Background(), "a", false, func() model.CollectionResult {
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

func TestStoppedAmazonItemsAreSkippedByScheduleAndRefreshAll(t *testing.T) {
	service, database := newSessionService(t)
	ctx := context.Background()
	insertAmazonItems(t, database, "a")
	if err := database.StopAmazonItems(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	service.collectors["amazon"] = fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { calls.Add(1); return succeeded() })
	service.Collect(ctx, "a")
	if _, _, err := service.RefreshAll(ctx); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	item, err := database.Get(ctx, "a")
	if err != nil || calls.Load() != 0 || item.LastAttempt != nil || item.AmazonStoppedAt == nil {
		t.Fatalf("stopped item checked or recorded: calls=%d %+v %v", calls.Load(), item.LastAttempt, err)
	}
}

func TestRefreshPriceRestartsOnlyThatItem(t *testing.T) {
	service, database := newSessionService(t)
	ctx := context.Background()
	insertAmazonItems(t, database, "a", "b", "c")
	if err := database.StopAmazonItems(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	service.collectors["amazon"] = fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { calls.Add(1); return succeeded() })
	if _, err := service.RefreshItem(ctx, "a"); err != nil {
		t.Fatalf("refresh of a stopped item refused: %v", err)
	}
	waitIdle(t, service)
	item, err := database.Get(ctx, "a")
	if err != nil || item.AmazonStoppedAt != nil || item.LatestPrice == nil || !stopped(t, database, "b") || !stopped(t, database, "c") {
		t.Fatalf("restart: %+v %v", item, err)
	}
	service.Collect(ctx, "a") // a restarted item is checked by scheduled checks again
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestFailedRefreshPriceKeepsTheItemStopped(t *testing.T) {
	service, database := newSessionService(t)
	ctx := context.Background()
	insertAmazonItems(t, database, "a")
	if err := database.StopAmazonItems(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	service.collectors["amazon"] = fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { return failed() })
	if _, err := service.RefreshItem(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	item, err := database.Get(ctx, "a")
	if err != nil || item.AmazonStoppedAt == nil || item.LastAttempt == nil || item.LastAttempt.Result != "request_error" || service.amazonGate.failures != 1 {
		t.Fatalf("failed refresh: %+v failures=%d %v", item, service.amazonGate.failures, err)
	}
}
