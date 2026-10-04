package service

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/leboncoin"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

func (f *fakeClaude) Review(context.Context, string, claude.ReviewInput) (model.AIReviewContent, error) {
	return model.AIReviewContent{}, errors.New("not used by settings tests")
}

// fakeReviewer returns err (or a fixed review) and can block until release is closed.
type fakeReviewer struct {
	mu      sync.Mutex
	inputs  []claude.ReviewInput
	err     error
	release chan struct{}
}

func (f *fakeReviewer) Verify(context.Context, string) error { return nil }

func (f *fakeReviewer) Review(ctx context.Context, _ string, input claude.ReviewInput) (model.AIReviewContent, error) {
	f.mu.Lock()
	f.inputs = append(f.inputs, input)
	f.mu.Unlock()
	if f.release != nil {
		<-f.release
	}
	good := "good"
	return model.AIReviewContent{Price: model.AIRating{Rating: "fair", Explanation: "ok"}, Condition: model.AIConditionRating{Rating: &good, Explanation: "ok"}}, f.err
}

func (f *fakeReviewer) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.inputs)
}

// newReviewService returns a service whose LeBoncoin collector reports *price.
func newReviewService(t *testing.T, withToken bool) (*Service, *store.Store, *fakeReviewer, *int64) {
	t.Helper()
	service, database := newSessionService(t)
	reviewer := &fakeReviewer{}
	service.claude = reviewer
	price := int64(1200)
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		amount := price
		return model.CollectionResult{Result: "success", AmountCents: amount, Listing: &model.ListingDetails{Title: "Lamp", PriceCents: &amount}}, leboncoin.SessionOutcome{}
	})
	if withToken {
		token := "sk-ant-oat01-synthetic"
		if _, _, err := database.SaveSettings(context.Background(), nil, &store.TokenChange{Value: &token}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	return service, database, reviewer, &price
}

// waitReviews waits until the item has no running collection or review (and,
// with afterCollection, until its first collection is recorded), then returns
// the item's review state.
func waitReviews(t *testing.T, service *Service, id string, afterCollection bool) *model.AIReviewState {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		item, err := service.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		collected := item.LastAttempt != nil && item.LastAttempt.Result != "pending"
		if !service.reviewRunning(id) && (collected || !afterCollection && item.LastAttempt == nil) {
			state, err := service.AIReview(context.Background(), item)
			if err != nil {
				t.Fatal(err)
			}
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("review did not finish")
	return nil
}

func TestNewItemIsReviewedOnlyWithToken(t *testing.T) {
	service, _, reviewer, _ := newReviewService(t, true)
	item, err := service.Add(context.Background(), "https://www.leboncoin.fr/ad/test/123")
	if err != nil {
		t.Fatal(err)
	}
	state := waitReviews(t, service, item.ID, true)
	if state.Latest == nil || state.Latest.Status != "succeeded" || *state.Latest.PriceCents != 1200 || len(state.History) != 1 || !state.TokenConfigured {
		t.Fatalf("unexpected state %+v", state)
	}
	if reviewer.inputs[0].Listing.Title != "Lamp" || len(reviewer.inputs[0].PriceHistory) != 1 {
		t.Fatalf("unexpected review input %+v", reviewer.inputs[0])
	}

	without, _, reviewerWithout, _ := newReviewService(t, false)
	item, err = without.Add(context.Background(), "https://www.leboncoin.fr/ad/test/123")
	if err != nil {
		t.Fatal(err)
	}
	state = waitReviews(t, without, item.ID, true)
	if state.TokenConfigured || state.LastAttempt != nil || reviewerWithout.calls() != 0 {
		t.Fatalf("review without token: %+v", state)
	}
}

func TestPriceChangeTriggersReview(t *testing.T) {
	service, database, reviewer, price := newReviewService(t, true)
	insertListing(t, database, "old", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	service.Collect(context.Background(), "old") // first price of a pre-existing item: no review
	service.Collect(context.Background(), "old") // same price: no review
	if state := waitReviews(t, service, "old", true); state.LastAttempt != nil || reviewer.calls() != 0 {
		t.Fatalf("unexpected review %+v", state)
	}
	*price = 900
	service.Collect(context.Background(), "old")
	state := waitReviews(t, service, "old", true)
	if state.Latest == nil || *state.Latest.PriceCents != 900 || reviewer.calls() != 1 {
		t.Fatalf("price change not reviewed: %+v", state)
	}
}

func TestRejectedReviewFailsAndMarksToken(t *testing.T) {
	service, database, reviewer, _ := newReviewService(t, true)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	reviewer.err = claude.ErrRejected
	if _, running, err := service.RequestAIReview(context.Background(), "lbc"); err != nil || running {
		t.Fatalf("request: %v %v", running, err)
	}
	state := waitReviews(t, service, "lbc", false)
	if state.LastAttempt == nil || state.LastAttempt.Status != "failed" || *state.LastAttempt.ErrorMessage != "Claude rejected the token. Replace it in Settings." || state.Latest != nil {
		t.Fatalf("unexpected state %+v", state)
	}
	if token, _ := database.ClaudeToken(context.Background()); token.LastRejectedAt == nil {
		t.Fatal("token rejection not recorded")
	}
	reviewer.err = nil
	service.RequestAIReview(context.Background(), "lbc")
	waitReviews(t, service, "lbc", false)
	if token, _ := database.ClaudeToken(context.Background()); token.LastRejectedAt != nil {
		t.Fatal("successful review did not clear the rejection")
	}
}

func TestDuplicateRequestRunsOneReview(t *testing.T) {
	service, database, reviewer, _ := newReviewService(t, true)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	reviewer.release = make(chan struct{})
	if _, running, err := service.RequestAIReview(context.Background(), "lbc"); err != nil || running {
		t.Fatalf("first request: %v %v", running, err)
	}
	if _, running, err := service.RequestAIReview(context.Background(), "lbc"); err != nil || !running {
		t.Fatalf("second request: %v %v", running, err)
	}
	item, _ := service.Get(context.Background(), "lbc")
	if state, _ := service.AIReview(context.Background(), item); !state.Running || state.LastAttempt.Status != "pending" {
		t.Fatalf("running state %+v", state)
	}
	close(reviewer.release)
	waitReviews(t, service, "lbc", false)
	if reviewer.calls() != 1 {
		t.Fatalf("reviews %d", reviewer.calls())
	}
}

func TestRequestAIReviewErrorsAndUnretrievedListing(t *testing.T) {
	service, database, _, _ := newReviewService(t, false)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	insertListing(t, database, "amz", "amazon", "https://www.amazon.fr/dp/B012345678")
	if _, _, err := service.RequestAIReview(context.Background(), "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing item: %v", err)
	}
	_, _, err := service.RequestAIReview(context.Background(), "amz")
	assertServiceError(t, err, 422, "AI_REVIEW_UNSUPPORTED")
	_, _, err = service.RequestAIReview(context.Background(), "lbc")
	assertServiceError(t, err, 409, "CLAUDE_TOKEN_MISSING")
	if amazon, _ := service.Get(context.Background(), "amz"); amazon.Platform == "amazon" {
		if state, _ := service.AIReview(context.Background(), amazon); state != nil {
			t.Fatal("Amazon item has an AI review state")
		}
	}

	withToken, _, _, _ := newReviewService(t, true)
	withToken.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		return model.CollectionResult{Result: "unavailable", Message: "gone"}, leboncoin.SessionOutcome{}
	})
	item, err := withToken.Add(context.Background(), "https://www.leboncoin.fr/ad/test/123")
	if err != nil {
		t.Fatal(err)
	}
	state := waitReviews(t, withToken, item.ID, true)
	if state.LastAttempt == nil || *state.LastAttempt.ErrorMessage != listingNotRetrievedMessage {
		t.Fatalf("unexpected state %+v", state)
	}
}

// REV-001: the review is shown as running no later than the collection result.
func TestReviewRunningWhenCollectionIsRecorded(t *testing.T) {
	service, database, reviewer, price := newReviewService(t, true)
	reviewer.release = make(chan struct{})
	defer close(reviewer.release)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	service.Collect(context.Background(), "lbc") // first price of a pre-existing item: no review
	*price = 900
	stop := make(chan struct{})
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-stop:
				done <- false
				return
			default:
			}
			// Read the stored result first, then the running flag.
			item, err := database.Get(context.Background(), "lbc")
			if err == nil && item.LatestPrice != nil && item.LatestPrice.AmountCents == 900 && !service.reviewRunning("lbc") {
				done <- true
				return
			}
		}
	}()
	service.Collect(context.Background(), "lbc")
	close(stop)
	if <-done {
		t.Fatal("price change recorded before the review was running")
	}
	if !service.reviewRunning("lbc") {
		t.Fatal("review not running after the price change")
	}
}

// REV-003: startReview reports each outcome distinctly.
func TestStartReviewOutcomes(t *testing.T) {
	without, database, _, _ := newReviewService(t, false)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	if outcome, err := without.startReview("lbc", nil, true); err != nil || outcome != reviewNoToken {
		t.Fatalf("no token: %v %v", outcome, err)
	}
	service, database, reviewer, _ := newReviewService(t, true)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	reviewer.release = make(chan struct{})
	if outcome, err := service.startReview("lbc", nil, true); err != nil || outcome != reviewStarted {
		t.Fatalf("start: %v %v", outcome, err)
	}
	if outcome, err := service.startReview("lbc", nil, true); err != nil || outcome != reviewAlreadyRunning {
		t.Fatalf("already running: %v %v", outcome, err)
	}
	close(reviewer.release)
	waitReviews(t, service, "lbc", false)
}
