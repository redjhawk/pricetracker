package service

import (
	"context"
	"testing"
	"time"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// productBrowser adds product details to the fake browser's successful reads.
type productBrowser struct{ *fakeBrowser }

func (p productBrowser) CollectProduct(ctx context.Context, item model.Listing) model.CollectionResult {
	result := p.fakeBrowser.CollectProduct(ctx, item)
	if result.Result == "success" {
		result.Product = &model.ProductDetails{Title: "Product " + item.ASIN, PriceCents: &result.AmountCents}
	}
	return result
}

func newReviewWorker(t *testing.T, withToken bool) (*searchWorker, *fakeBrowser, *fakeReviewer) {
	t.Helper()
	worker, browser, database := newTestWorker(t)
	worker.collector = productBrowser{browser}
	reviewer := &fakeReviewer{}
	worker.service.claude = reviewer
	if withToken {
		token := "sk-ant-oat01-synthetic"
		if _, _, err := database.SaveSettings(context.Background(), 0, nil, &store.TokenChange{Value: &token}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	return worker, browser, reviewer
}

func waitNoReview(t *testing.T, service *Service, ids []string) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		running := false
		for _, id := range ids {
			running = running || service.reviewRunning(id)
		}
		if !running {
			return
		}
	}
	t.Fatal("reviews still running")
}

func TestSearchItemsAreReviewedOnceAfterTheirRead(t *testing.T) {
	worker, browser, reviewer := newReviewWorker(t, true)
	ctx := context.Background()
	runOnePass(t, worker)
	ids, _ := worker.service.store.SearchItemIDs(ctx, "s1")
	waitNoReview(t, worker.service, ids)
	if reviewer.calls() != 3 {
		t.Fatalf("%d reviews, want 3", reviewer.calls())
	}
	item, _ := worker.service.store.Get(ctx, ids[0])
	state, err := worker.service.AIReview(ctx, item)
	review, isAmazon := state.Latest.Review.(*model.AmazonAIReviewContent)
	if err != nil || !isAmazon || review.Price.Rating != "good_deal" || *state.Latest.PriceCents != 100 || !state.TokenConfigured {
		t.Fatalf("unexpected review state %+v %v", state, err)
	}
	// A second pass reads the items again but never reviews an already reviewed item.
	if err := worker.service.store.SetSearchProgress(ctx, "s1", 0, browser.now); err != nil {
		t.Fatal(err)
	}
	runOnePass(t, worker)
	waitNoReview(t, worker.service, ids)
	if reviewer.calls() != 3 {
		t.Fatalf("%d reviews after the second pass, want 3", reviewer.calls())
	}
}

func TestFailedAmazonReviewIsRetriedAndNeverCountsAsAmazonFailure(t *testing.T) {
	worker, browser, reviewer := newReviewWorker(t, true)
	reviewer.err = claude.ErrRejected
	ctx := context.Background()
	runOnePass(t, worker)
	ids, _ := worker.service.store.SearchItemIDs(ctx, "s1")
	waitNoReview(t, worker.service, ids)
	if state := worker.service.amazonGate.current(); state.ConsecutiveFailures != 0 {
		t.Fatalf("review failure counted as Amazon failure: %+v", state)
	}
	item, _ := worker.service.store.Get(ctx, ids[0])
	if state, err := worker.service.AIReview(ctx, item); err != nil || state.LastAttempt.Status != "failed" || state.Latest != nil {
		t.Fatalf("unexpected failed state %+v %v", state, err)
	}
	reviewer.err = nil
	worker.service.store.SetSearchProgress(ctx, "s1", 0, browser.now)
	runOnePass(t, worker)
	waitNoReview(t, worker.service, ids)
	if reviewer.calls() != 6 {
		t.Fatalf("%d reviews, want 6", reviewer.calls())
	}
}

func TestSearchItemsWithoutTokenAreNotReviewed(t *testing.T) {
	worker, _, reviewer := newReviewWorker(t, false)
	ctx := context.Background()
	runOnePass(t, worker)
	ids, _ := worker.service.store.SearchItemIDs(ctx, "s1")
	item, _ := worker.service.store.Get(ctx, ids[0])
	state, err := worker.service.AIReview(ctx, item)
	if err != nil || reviewer.calls() != 0 || state == nil || state.LastAttempt != nil || item.LatestPrice == nil {
		t.Fatalf("unexpected state without token %+v %v", state, err)
	}
	insertListing(t, worker.service.store, "tracked", "amazon", "https://www.amazon.fr/dp/B012345678")
	tracked, _ := worker.service.store.Get(ctx, "tracked")
	if state, err := worker.service.AIReview(ctx, tracked); err != nil || state != nil {
		t.Fatalf("tracked Amazon item has a review state %+v %v", state, err)
	}
}
