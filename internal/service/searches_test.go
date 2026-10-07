package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

func TestAddingASearchItemOnTheAmazonTabTracksIt(t *testing.T) {
	worker, _, database := newTestWorker(t)
	service := worker.service
	ctx := context.Background()
	runOnePass(t, worker)
	ids, _ := database.SearchItemIDs(ctx, "s1")
	var calls atomic.Int32
	service.collectors["amazon"] = fakeAmazon(func(context.Context, model.Listing) model.CollectionResult { calls.Add(1); return succeeded() })
	item, err := service.Add(ctx, "https://www.amazon.fr/dp/B000000002", "")
	if err != nil || item.ID != ids[1] || !item.Tracked || item.NextCheckAt == nil {
		t.Fatalf("unexpected added item %+v %v", item, err)
	}
	for deadline := time.Now().Add(5 * time.Second); calls.Load() == 0 && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond) // the immediate collection runs in the background
	}
	if calls.Load() != 1 {
		t.Fatalf("%d collections, want 1", calls.Load())
	}
	if history, _ := database.GetWithHistory(ctx, ids[1]); len(history.PriceHistory) != 2 {
		t.Fatalf("history not kept %+v", history.PriceHistory)
	}
	if _, err := service.Add(ctx, "https://www.amazon.fr/dp/B000000002", ""); err == nil {
		t.Fatal("second add accepted")
	}
}

func TestSearchStates(t *testing.T) {
	worker, browser, _ := newTestWorker(t)
	service := worker.service
	service.now = browser.Now
	ctx := context.Background()
	if searches, _ := service.Searches(ctx); searches[0].State != "waiting" || searches[0].WaitingUntil != nil {
		t.Fatalf("unexpected state in window %+v", searches[0])
	}
	service.setRunningSearch("s1")
	if searches, _ := service.Searches(ctx); searches[0].State != "running" {
		t.Fatalf("unexpected running state %+v", searches[0])
	}
	service.setRunningSearch("")
	runOnePass(t, worker)
	if searches, _ := service.Searches(ctx); searches[0].State != "done" || searches[0].ItemCount != 3 {
		t.Fatalf("unexpected done state %+v", searches[0])
	}
	browser.now = time.Date(2026, 10, 8, 12, 0, 0, 0, parisLocation)
	service.store.SetSearchProgress(ctx, "s1", 2, browser.now)
	searches, _ := service.Searches(ctx)
	if searches[0].State != "waiting" || !searches[0].WaitingUntil.Equal(time.Date(2026, 10, 8, 22, 0, 0, 0, parisLocation)) {
		t.Fatalf("unexpected waiting state %+v", searches[0])
	}
}
