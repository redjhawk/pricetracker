package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/leboncoin"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Amazon item AI review and purchase goal (issue #305) through the service: real SQLite, the fake
// Claude reviewer and a fake Amazon collector counting every Amazon check.

const amazonGoal = "gift for a 10-year-old"

var (
	firstCheckAt  = time.Date(2026, time.November, 1, 8, 0, 0, 0, time.UTC)
	secondCheckAt = time.Date(2026, time.November, 2, 8, 0, 0, 0, time.UTC)
)

func validAmazonReview() model.AmazonAIReviewContent {
	return model.AmazonAIReviewContent{
		Price:          model.AIRating{Rating: "good_deal", Explanation: "Below the usual price."},
		FairPrice:      model.AmazonFairPrice{MinCents: 11000, MaxCents: 13000},
		Recommendation: model.AIRating{Rating: "wait", Explanation: "The price often drops."},
	}
}

func int64Pointer(value int64) *int64 { return &value }

func rating(tenths, count int64) *model.CustomerRating {
	return &model.CustomerRating{StarsTenths: int64Pointer(tenths), Count: int64Pointer(count)}
}

// newAmazonReviewService returns a service on the database of dir with the fake reviewer, a
// counting fake Amazon collector (success by default) and, withToken, a saved Claude token.
func newAmazonReviewService(t *testing.T, dir string, withToken bool) (*Service, *store.Store, *fakeReviewer, *amazonRecorder) {
	t.Helper()
	cfg := config.Config{DataDirectory: dir, Location: time.UTC, CheckTimes: []string{"08:00", "20:00"}, StaleAfter: 36 * time.Hour, UserAgent: "test"}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	recorder := &amazonRecorder{}
	service := NewWithOptions(cfg, database, Options{Amazon: recorder})
	t.Cleanup(service.Close)
	reviewer := &fakeReviewer{}
	service.claude = reviewer
	price := int64(1200)
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		amount := price
		return model.CollectionResult{Result: "success", AmountCents: amount, Listing: &model.ListingDetails{Title: "Lamp", PriceCents: &amount}}, leboncoin.SessionOutcome{}
	})
	if withToken {
		saveToken(t, database, "sk-ant-oat01-synthetic")
	}
	return service, database, reviewer, recorder
}

func saveToken(t *testing.T, database *store.Store, token string) {
	t.Helper()
	if _, _, err := database.SaveSettings(context.Background(), 0, nil, &store.TokenChange{Value: &token}, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func amazonCheck(cents, usedCents int64, customer *model.CustomerRating) model.CollectionResult {
	title := "Synthetic headphones"
	result := model.CollectionResult{Result: "success", AmountCents: cents, Title: &title, SecondHandStatus: "not_found", CustomerRating: customer}
	if usedCents > 0 {
		result.SecondHandStatus, result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel = "available", usedCents, "very_good", "Used - Very good"
	}
	return result
}

func record(t *testing.T, database *store.Store, id string, result model.CollectionResult, at time.Time) {
	t.Helper()
	if err := database.RecordCollection(context.Background(), id, result, at); err != nil {
		t.Fatal(err)
	}
}

// seedItemA inserts item A of the test cases: price history 129,99 € then 119,99 €, Amazon-sold
// second-hand offer 95,00 € "Used - Very good" after 99,00 €, rating 4.5 with 1234 ratings and the goal.
func seedItemA(t *testing.T, database *store.Store, id string) {
	t.Helper()
	insertListing(t, database, id, "amazon", "https://www.amazon.fr/dp/B012345678?item="+id)
	if _, err := database.SetPurchaseGoal(context.Background(), id, amazonGoal); err != nil {
		t.Fatal(err)
	}
	record(t, database, id, amazonCheck(12999, 9900, rating(45, 1234)), firstCheckAt)
	record(t, database, id, amazonCheck(11999, 9500, rating(45, 1234)), secondCheckAt)
}

// storeAmazonReview stores a finished review of the item; content nil stores a failed attempt.
func storeAmazonReview(t *testing.T, database *store.Store, id string, priceCents int64, content *model.AmazonAIReviewContent, at time.Time) int64 {
	t.Helper()
	ctx := context.Background()
	reviewID, err := database.StartAIReview(ctx, id, at)
	if err != nil {
		t.Fatal(err)
	}
	if content == nil {
		err = database.FinishAIReview(ctx, reviewID, "failed", nil, nil, "Claude could not be reached. Try again later.", at.Add(time.Minute))
	} else {
		err = database.FinishAIReview(ctx, reviewID, "succeeded", &priceCents, content, "", at.Add(time.Minute))
	}
	if err != nil {
		t.Fatal(err)
	}
	return reviewID
}

func reviewState(t *testing.T, service *Service, id string) *model.AIReviewState {
	t.Helper()
	item, err := service.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.AIReview(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatalf("item %s has no AI review state", id)
	}
	return state
}

// amazonContent returns the Amazon review of a stored review, failing the test for another shape.
func amazonContent(t *testing.T, review *model.AIReview) model.AmazonAIReviewContent {
	t.Helper()
	if review == nil {
		t.Fatal("no review")
	}
	content, ok := review.Review.(*model.AmazonAIReviewContent)
	if !ok || content == nil {
		t.Fatalf("review %d is %T, want *model.AmazonAIReviewContent", review.ID, review.Review)
	}
	return *content
}

func requestAmazonReview(t *testing.T, service *Service, id string) {
	t.Helper()
	if _, running, err := service.RequestAIReview(context.Background(), id); err != nil || running {
		t.Fatalf("request AI review of %s: running=%v err=%v", id, running, err)
	}
}

// sqlCount runs a COUNT query on the service's database file.
func sqlCount(t *testing.T, service *Service, query string, args ...any) int {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(service.config.DataDirectory, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func unavailableReason(state *model.AIReviewState) string {
	if state.UnavailableReason == nil {
		return ""
	}
	return *state.UnavailableReason
}

// amazonSnapshot is what an Amazon review must never change.
type amazonSnapshot struct {
	checks, attempts, failures int
	history                    []model.Observation
	next                       time.Time
}

func snapshotAmazon(t *testing.T, service *Service, database *store.Store, recorder *amazonRecorder, id string) amazonSnapshot {
	t.Helper()
	history, err := database.PriceHistory(context.Background(), id, 1000)
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := database.NextCheck(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return amazonSnapshot{checks: recorder.total(), attempts: sqlCount(t, service, "SELECT COUNT(*) FROM collection_attempts WHERE item_id = ?", id),
		failures: amazonFailures(t, database), history: history, next: next}
}

func assertAmazonUnchanged(t *testing.T, before, after amazonSnapshot) {
	t.Helper()
	if after.checks != before.checks || after.attempts != before.attempts || after.failures != before.failures || !after.next.Equal(before.next) || len(after.history) != len(before.history) {
		t.Fatalf("an AI review changed the Amazon state: before %+v, after %+v", before, after)
	}
	for index := range before.history {
		if after.history[index].AmountCents != before.history[index].AmountCents || !after.history[index].Timestamp.Equal(*before.history[index].Timestamp) {
			t.Fatalf("price history changed: before %+v, after %+v", before.history, after.history)
		}
	}
}

// TC-004: review availability of an Amazon item is available, stopped or no current price.
func TestAmazonReviewAvailability_TC004(t *testing.T) {
	stoppedAt := time.Date(2026, time.November, 20, 7, 26, 0, 0, time.UTC)
	price := &model.Observation{AmountCents: 11999, Currency: "EUR", Timestamp: &secondCheckAt}
	attempt := func(result string) *model.Attempt { return &model.Attempt{Result: result, Timestamp: secondCheckAt} }
	for _, test := range []struct {
		name        string
		stoppedAt   *time.Time
		latestPrice *model.Observation
		lastAttempt *model.Attempt
		want        string
	}{
		{"priced and running", nil, price, attempt("success"), ""},
		{"no price recorded yet", nil, nil, nil, "no_current_price"},
		{"first check failed without a price", nil, nil, attempt("request_error"), "no_current_price"},
		{"price not found after an earlier price", nil, price, attempt("price_not_found"), "no_current_price"},
		{"listing unavailable", nil, price, attempt("unavailable"), "no_current_price"},
		{"retrieval error after a price keeps it current", nil, price, attempt("request_error"), ""},
		{"stopped with a price", &stoppedAt, price, attempt("request_error"), "stopped"},
		{"stopped without a current price", &stoppedAt, price, attempt("price_not_found"), "stopped"},
		{"stopped never priced", &stoppedAt, nil, nil, "stopped"},
	} {
		// The decision has no token input: it is the same with or without a saved Claude token.
		if got := amazonReviewAvailability(test.stoppedAt, test.latestPrice, test.lastAttempt); got != test.want {
			t.Errorf("%s: got %q, want %q", test.name, got, test.want)
		}
	}
}

func TestAmazonReview_TC007_RefreshProducesStoredReviewFromRecordedData(t *testing.T) {
	service, database, reviewer, recorder := newAmazonReviewService(t, t.TempDir(), true)
	ctx := context.Background()
	seedItemA(t, database, "A")
	before := snapshotAmazon(t, service, database, recorder, "A")
	reviewer.release = make(chan struct{})
	requestAmazonReview(t, service, "A")
	if state := reviewState(t, service, "A"); !state.Running || state.LastAttempt == nil || state.LastAttempt.Status != "pending" {
		t.Fatalf("running state %+v", state)
	}
	close(reviewer.release)
	state := waitReviews(t, service, "A", false)
	if state.Running || state.Latest == nil || state.Latest.Status != "succeeded" || state.Latest.PriceCents == nil || *state.Latest.PriceCents != 11999 ||
		state.Latest.CreatedAt.IsZero() || state.Latest.CompletedAt == nil || len(state.History) != 1 {
		t.Fatalf("first review state %+v", state)
	}
	if content := amazonContent(t, state.Latest); content != validAmazonReview() {
		t.Fatalf("stored review %+v", content)
	}

	second := validAmazonReview()
	second.Price = model.AIRating{Rating: "fair", Explanation: "Usual price."}
	second.Recommendation = model.AIRating{Rating: "buy", Explanation: "Good time to buy."}
	reviewer.amazonReview = &second
	requestAmazonReview(t, service, "A")
	state = waitReviews(t, service, "A", false)
	if len(state.History) != 2 || state.Latest == nil || state.History[0].ID != state.Latest.ID || state.History[0].CreatedAt.Before(state.History[1].CreatedAt) {
		t.Fatalf("history %+v", state.History)
	}
	if amazonContent(t, &state.History[0]) != second || amazonContent(t, &state.History[1]) != validAmazonReview() {
		t.Fatalf("history not newest first: %+v", state.History)
	}
	for _, review := range state.History {
		if review.PriceCents == nil || *review.PriceCents != 11999 || review.CreatedAt.IsZero() {
			t.Fatalf("history entry %+v", review)
		}
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), amazonGoal) || strings.Contains(string(encoded), "1234") {
		t.Fatalf("reviews contain the goal or the customer rating: %s", encoded)
	}

	// Recorded data sent to Claude (TC-002).
	if reviewer.amazonCalls() != 2 {
		t.Fatalf("%d Amazon reviews sent", reviewer.amazonCalls())
	}
	input := reviewer.amazonInput(0)
	if input.Title != "Synthetic headphones" || input.Marketplace != "amazon.fr" || input.URL != "https://www.amazon.fr/dp/B012345678?item=A" ||
		input.PriceCents != 11999 || input.PurchaseGoal != amazonGoal {
		t.Fatalf("review input %+v", input)
	}
	if len(input.PriceHistory) != 2 || input.PriceHistory[0].AmountCents != 12999 || input.PriceHistory[1].AmountCents != 11999 {
		t.Fatalf("price history sent %+v", input.PriceHistory)
	}
	if input.SecondHandStatus != "available" || input.SecondHandLatest == nil || input.SecondHandLatest.AmountCents != 9500 || input.SecondHandLatest.ConditionLabel != "Used - Very good" {
		t.Fatalf("second-hand offer sent %+v %+v", input.SecondHandStatus, input.SecondHandLatest)
	}
	if len(input.SecondHandHistory) != 2 || input.SecondHandHistory[0].AmountCents != 9900 || input.SecondHandHistory[1].AmountCents != 9500 {
		t.Fatalf("second-hand history sent %+v", input.SecondHandHistory)
	}
	if input.Rating.StarsTenths == nil || *input.Rating.StarsTenths != 45 || input.Rating.Count == nil || *input.Rating.Count != 1234 {
		t.Fatalf("rating sent %+v", input.Rating)
	}
	assertAmazonUnchanged(t, before, snapshotAmazon(t, service, database, recorder, "A"))

	t.Run("retrieval error after a price", func(t *testing.T) {
		seedItemA(t, database, "R")
		record(t, database, "R", model.CollectionResult{Result: "request_error", Message: "Amazon could not be reached for a price check.", SecondHandStatus: "check_error"}, secondCheckAt.Add(time.Hour))
		calls := reviewer.amazonCalls()
		requestAmazonReview(t, service, "R")
		state := waitReviews(t, service, "R", false)
		if state.Latest == nil || state.Latest.PriceCents == nil || *state.Latest.PriceCents != 11999 || reviewer.amazonCalls() != calls+1 {
			t.Fatalf("review after a retrieval error %+v", state)
		}
		input := reviewer.amazonInput(calls)
		if input.PriceCents != 11999 || input.SecondHandLatest == nil || input.SecondHandLatest.AmountCents != 9500 ||
			input.Rating.StarsTenths == nil || *input.Rating.StarsTenths != 45 || input.Rating.Count == nil || *input.Rating.Count != 1234 {
			t.Fatalf("input after a retrieval error %+v", input)
		}
	})
	t.Run("unknown and other owner's items", func(t *testing.T) {
		if _, _, err := service.RequestAIReview(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("unknown item: %v", err)
		}
		other := model.Listing{ID: "other", Platform: "amazon", ListingID: "B012345678", ASIN: "B012345678", Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/B012345678", OwnerID: 7}
		if err := database.Insert(ctx, other, other.URL, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		record(t, database, "other", amazonCheck(11999, 0, nil), secondCheckAt)
		calls := reviewer.amazonCalls()
		if _, _, err := service.RequestAIReview(ctx, "other"); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("other owner's item: %v", err)
		}
		if reviewer.amazonCalls() != calls || sqlCount(t, service, "SELECT COUNT(*) FROM ai_reviews WHERE item_id = 'other'") != 0 {
			t.Fatal("another owner's item was reviewed")
		}
	})
}

// TC-008: repeated review requests during a run start one review.
func TestAmazonReview_TC008_RepeatedRequestsStartOneReview(t *testing.T) {
	service, database, reviewer, _ := newAmazonReviewService(t, t.TempDir(), true)
	seedItemA(t, database, "A")
	reviewer.release = make(chan struct{})
	requestAmazonReview(t, service, "A")
	for index := 0; index < 2; index++ {
		if _, running, err := service.RequestAIReview(context.Background(), "A"); err != nil || !running {
			t.Fatalf("sequential request %d: running=%v err=%v", index, running, err)
		}
	}
	var group sync.WaitGroup
	results := make(chan bool, 2)
	for index := 0; index < 2; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, running, err := service.RequestAIReview(context.Background(), "A")
			results <- err == nil && running
		}()
	}
	group.Wait()
	close(results)
	for alreadyRunning := range results {
		if !alreadyRunning {
			t.Fatal("a concurrent request did not report the running review")
		}
	}
	close(reviewer.release)
	state := waitReviews(t, service, "A", false)
	if reviewer.amazonCalls() != 1 || len(state.History) != 1 || sqlCount(t, service, "SELECT COUNT(*) FROM ai_reviews WHERE item_id = 'A'") != 1 {
		t.Fatalf("%d Claude requests, %d reviews", reviewer.amazonCalls(), len(state.History))
	}
}

// TC-009: a failed Amazon review keeps the previous review and is not an Amazon failure.
func TestAmazonReview_TC009_FailedReviewKeepsPreviousAndIsNotAnAmazonFailure(t *testing.T) {
	logs := captureLogs(t)
	const token = "sk-ant-oat01-synthetic"
	service, database, reviewer, _ := newAmazonReviewService(t, t.TempDir(), true)
	ctx := context.Background()
	seedItemA(t, database, "A")
	seedItemA(t, database, "B")
	previous := validAmazonReview()
	previousID := storeAmazonReview(t, database, "A", 11999, &previous, secondCheckAt.Add(time.Hour))
	if err := database.SaveAmazonFailures(ctx, 4); err != nil {
		t.Fatal(err)
	}
	service.amazonGate.failures = 4
	for _, failure := range []struct {
		name     string
		err      error
		message  string
		rejected bool
	}{
		{"token rejected (401)", claude.ErrRejected, "Claude rejected the token. Replace it in Settings.", true},
		{"usage limit (429)", claude.ErrUsageLimit, "Claude usage limit reached. Try again later.", true},
		{"server error (500)", claude.ErrUnreachable, "Claude could not be reached. Try again later.", false},
		{"network error", claude.ErrUnreachable, "Claude could not be reached. Try again later.", false},
		{"invalid review", claude.ErrBadResponse, "Claude returned an unusable review. Try again.", false},
	} {
		if failure.rejected {
			if err := database.ClearClaudeTokenRejected(ctx, 0, mustToken(t, database).Revision); err != nil {
				t.Fatal(err)
			}
		}
		reviewer.amazonErr = failure.err
		requestAmazonReview(t, service, "A")
		state := waitReviews(t, service, "A", false)
		if state.LastAttempt == nil || state.LastAttempt.Status != "failed" || state.LastAttempt.CompletedAt == nil || state.LastAttempt.ErrorMessage == nil ||
			*state.LastAttempt.ErrorMessage != failure.message {
			t.Fatalf("%s: last attempt %+v", failure.name, state.LastAttempt)
		}
		if state.Latest == nil || state.Latest.ID != previousID || len(state.History) != 1 || unavailableReason(state) != "" {
			t.Fatalf("%s: previous review not kept: %+v", failure.name, state)
		}
		if failure.rejected && mustToken(t, database).LastRejectedAt == nil {
			t.Fatalf("%s: rejected-token warning not set", failure.name)
		}
		if amazonFailures(t, database) != 4 || service.amazonGate.failures != 4 || stopped(t, database, "A") || stopped(t, database, "B") {
			t.Fatalf("%s: counted as an Amazon failure (%d) or stopped an item", failure.name, amazonFailures(t, database))
		}
		item, _ := service.Get(ctx, "A")
		encoded, _ := json.Marshal(model.ItemDetails{Item: item, AIReview: state})
		if strings.Contains(string(encoded), token) || strings.Contains(logs.String(), token) {
			t.Fatalf("%s: token leaked", failure.name)
		}
	}
	if reviewer.amazonCalls() != 5 {
		t.Fatalf("%d Claude requests, want 5", reviewer.amazonCalls())
	}

	t.Run("no previous review", func(t *testing.T) {
		reviewer.amazonErr = claude.ErrUnreachable
		requestAmazonReview(t, service, "B")
		state := waitReviews(t, service, "B", false)
		if state.LastAttempt == nil || state.LastAttempt.Status != "failed" || state.Latest != nil || len(state.History) != 0 {
			t.Fatalf("failed first review %+v", state)
		}
	})
	t.Run("retry after the failures", func(t *testing.T) {
		reviewer.amazonErr = nil
		requestAmazonReview(t, service, "A")
		state := waitReviews(t, service, "A", false)
		if state.Latest == nil || state.Latest.ID == previousID || len(state.History) != 2 || state.LastAttempt.Status != "succeeded" {
			t.Fatalf("retry %+v", state)
		}
		if _, err := service.RefreshItem(ctx, "A"); err != nil {
			t.Fatal(err)
		}
		waitIdle(t, service)
		if amazonFailures(t, database) != 0 {
			t.Fatalf("a successful Amazon check left %d failures", amazonFailures(t, database))
		}
	})
}

func mustToken(t *testing.T, database *store.Store) model.ClaudeToken {
	t.Helper()
	token, err := database.ClaudeToken(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// TC-010: without a Claude token, Amazon items are tracked and not reviewed.
func TestAmazonReview_TC010_WithoutTokenTrackedAndNotReviewed(t *testing.T) {
	service, database, reviewer, recorder := newAmazonReviewService(t, t.TempDir(), false)
	ctx := context.Background()
	seedItemA(t, database, "A")
	earlier := validAmazonReview()
	earlierID := storeAmazonReview(t, database, "A", 11999, &earlier, secondCheckAt.Add(time.Hour))
	state := reviewState(t, service, "A")
	if state.TokenConfigured || state.Latest == nil || state.Latest.ID != earlierID || unavailableReason(state) != "" {
		t.Fatalf("details without a token %+v", state)
	}
	_, _, err := service.RequestAIReview(ctx, "A")
	var typed *Error
	if !errors.As(err, &typed) || typed.Status != 409 || typed.Code != "CLAUDE_TOKEN_MISSING" || typed.Message != "Configure a Claude token in Settings." {
		t.Fatalf("request without a token: %v", err)
	}
	if state := reviewState(t, service, "A"); state.LastAttempt == nil || state.LastAttempt.ID != earlierID {
		t.Fatalf("a review attempt was recorded: %+v", state.LastAttempt)
	}
	if _, err := service.RefreshItem(ctx, "A"); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	if item, _ := service.Get(ctx, "A"); recorder.calls("A") != 1 || item.LatestPrice == nil || item.LatestPrice.AmountCents != 100 {
		t.Fatalf("check without a token: %d calls, %+v", recorder.calls("A"), item.LatestPrice)
	}
	if _, _, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: tokenInput("sk-ant-oat01-synthetic")}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if state := reviewState(t, service, "A"); reviewer.amazonCalls() != 0 || state.Running || state.Latest == nil || state.Latest.ID != earlierID || state.LastAttempt.ID != earlierID {
		t.Fatalf("saving a token started a review: %d calls, %+v", reviewer.amazonCalls(), state)
	}
}

func TestAmazonReview_TC013_ChecksRecordCustomerRating(t *testing.T) {
	service, database, reviewer, recorder := newAmazonReviewService(t, t.TempDir(), true)
	ctx := context.Background()
	setResult := func(result model.CollectionResult) {
		recorder.mu.Lock()
		recorder.result = func(model.Listing) model.CollectionResult { return result }
		recorder.mu.Unlock()
	}
	setResult(amazonCheck(11999, 0, rating(45, 1234)))
	added, err := service.Add(ctx, "https://www.amazon.fr/dp/B012345678", "")
	if err != nil {
		t.Fatal(err)
	}
	id := added.ID
	waitFor(t, "the first check", func() bool {
		return sqlCount(t, service, "SELECT COUNT(*) FROM collection_attempts WHERE item_id = ?", id) == 1
	})
	waitIdle(t, service)
	assertRatingOf := func(step string, tenths, count *int64) {
		t.Helper()
		stored, err := database.AmazonCustomerRating(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if !equalOptional(stored.StarsTenths, tenths) || !equalOptional(stored.Count, count) {
			t.Fatalf("%s: rating %s/%s, want %s/%s", step, showOptional(stored.StarsTenths), showOptional(stored.Count), showOptional(tenths), showOptional(count))
		}
	}
	item, _ := service.Get(ctx, id)
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 11999 || item.SecondHandOffer == nil || item.SecondHandOffer.Status != "not_found" {
		t.Fatalf("first check %+v", item)
	}
	assertRatingOf("first check", int64Pointer(45), int64Pointer(1234))

	setResult(model.CollectionResult{Result: "request_error", Message: "Amazon could not be reached for a price check.", SecondHandStatus: "check_error"})
	if _, err := service.RefreshItem(ctx, id); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	assertRatingOf("failed check", int64Pointer(45), int64Pointer(1234))

	requestAmazonReview(t, service, id)
	waitReviews(t, service, id, false)
	if reviewer.amazonCalls() != 1 {
		t.Fatalf("%d reviews", reviewer.amazonCalls())
	}
	if sent := reviewer.amazonInput(0).Rating; !equalOptional(sent.StarsTenths, int64Pointer(45)) || !equalOptional(sent.Count, int64Pointer(1234)) {
		t.Fatalf("first review rating %+v", sent)
	}

	setResult(amazonCheck(11999, 0, &model.CustomerRating{}))
	if _, _, err := service.RefreshAll(ctx); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	assertRatingOf("page without ratings", nil, nil)
	if item, _ := service.Get(ctx, id); item.LastAttempt == nil || item.LastAttempt.Result != "success" || item.LatestPrice.AmountCents != 11999 {
		t.Fatalf("check of a page without ratings %+v", item.LastAttempt)
	}
	requestAmazonReview(t, service, id)
	waitReviews(t, service, id, false)
	if reviewer.amazonCalls() != 2 {
		t.Fatalf("%d reviews", reviewer.amazonCalls())
	}
	if sent := reviewer.amazonInput(1).Rating; sent.StarsTenths != nil || sent.Count != nil {
		t.Fatalf("second review rating %+v, want unknown", sent)
	}

	// No answer without a rating invents one.
	setResult(model.CollectionResult{Result: "unavailable", Message: "The listing is unavailable.", SecondHandStatus: "check_error"})
	service.Collect(ctx, id)
	assertRatingOf("listing unavailable", nil, nil)
	notFound := model.CollectionResult{Result: "price_not_found", Message: "Amazon did not show a detectable euro price.", SecondHandStatus: "check_error", CustomerRating: &model.CustomerRating{}}
	setResult(notFound)
	service.Collect(ctx, id)
	assertRatingOf("price not found", nil, nil)
	// A scheduled check records the rating the same way.
	setResult(amazonCheck(11999, 0, rating(40, 10)))
	service.Collect(ctx, id)
	assertRatingOf("scheduled check", int64Pointer(40), int64Pointer(10))

	items, err := service.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	details, _ := service.Get(ctx, id)
	for what, value := range map[string]any{"list": items, "item": details} {
		encoded, _ := json.Marshal(value)
		if strings.Contains(strings.ToLower(string(encoded)), "rating") {
			t.Fatalf("the customer rating is returned in the %s: %s", what, encoded)
		}
	}
}

func equalOptional(got, want *int64) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}

func showOptional(value *int64) string {
	if value == nil {
		return "absent"
	}
	encoded, _ := json.Marshal(*value)
	return string(encoded)
}

func TestAmazonReview_TC014_EventsStartNoReviewAndEachReviewKeepsItsPrice(t *testing.T) {
	service, database, reviewer, recorder := newAmazonReviewService(t, t.TempDir(), true)
	ctx := context.Background()
	seedItemA(t, database, "A")
	atHundred := validAmazonReview()
	atHundredID := storeAmazonReview(t, database, "A", 10000, &atHundred, secondCheckAt.Add(time.Hour))
	recorder.mu.Lock()
	recorder.result = func(model.Listing) model.CollectionResult { return amazonCheck(8000, 7000, rating(45, 1234)) }
	recorder.mu.Unlock()

	service.Collect(ctx, "A")
	if _, _, _, err := service.SetPurchaseGoal(ctx, "A", "for a teenager"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: tokenInput("sk-ant-oat01-replacement")}); err != nil {
		t.Fatal(err)
	}
	if err := database.StopAmazonItems(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RefreshItem(ctx, "A"); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, service)
	time.Sleep(100 * time.Millisecond)
	if stopped(t, database, "A") {
		t.Fatal("Refresh price did not restart the item")
	}
	item, _ := service.Get(ctx, "A")
	state := reviewState(t, service, "A")
	if reviewer.amazonCalls() != 0 || state.Running || state.Latest == nil || state.Latest.ID != atHundredID || *state.Latest.PriceCents != 10000 {
		t.Fatalf("an event started a review: %d calls, %+v", reviewer.amazonCalls(), state)
	}
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 8000 {
		t.Fatalf("current price %+v", item.LatestPrice)
	}
	requestAmazonReview(t, service, "A")
	state = waitReviews(t, service, "A", false)
	if len(state.History) != 2 || state.History[0].PriceCents == nil || *state.History[0].PriceCents != 8000 || state.History[1].ID != atHundredID || *state.History[1].PriceCents != 10000 {
		t.Fatalf("history %+v", state.History)
	}
	if input := reviewer.amazonInput(0); input.PriceCents != 8000 || input.PurchaseGoal != "for a teenager" || input.SecondHandLatest == nil || input.SecondHandLatest.AmountCents != 7000 {
		t.Fatalf("review input %+v", input)
	}
}

// legacyAmazonDatabase writes in dir a database at the schema before issue #305 (no customer
// rating columns) with a priced Amazon item and a LeBoncoin item with a goal and reviews.
func legacyAmazonDatabase(t *testing.T, dir string) {
	t.Helper()
	cfg := config.Config{DataDirectory: dir, Location: time.UTC, CheckTimes: []string{"08:00", "20:00"}, StaleAfter: 36 * time.Hour, UserAgent: "test"}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	insertListing(t, database, "amz", "amazon", "https://www.amazon.fr/dp/B012345678")
	record(t, database, "amz", amazonCheck(11999, 0, nil), secondCheckAt)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	if _, err := database.SetPurchaseGoal(context.Background(), "lbc", "Light distro"); err != nil {
		t.Fatal(err)
	}
	good := "good"
	price := int64(1200)
	reviewID, _ := database.StartAIReview(context.Background(), "lbc", secondCheckAt)
	if err := database.FinishAIReview(context.Background(), reviewID, "succeeded", &price, &model.AIReviewContent{Price: model.AIRating{Rating: "fair", Explanation: "ok"},
		Condition: model.AIConditionRating{Rating: &good, Explanation: "ok"}}, "", secondCheckAt); err != nil {
		t.Fatal(err)
	}
	saveToken(t, database, "sk-ant-oat01-synthetic")
	database.Close()
	db, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, column := range []string{"amazon_rating_tenths", "amazon_rating_count"} {
		var exists bool
		if err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM pragma_table_info('items') WHERE name = ?)", column).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			if _, err := db.Exec("ALTER TABLE items DROP COLUMN " + column); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestAmazonReview_TC015_ItemsFromBeforeTheUpgrade(t *testing.T) {
	dir := t.TempDir()
	legacyAmazonDatabase(t, dir)
	service, database, reviewer, _ := newAmazonReviewService(t, dir, false)
	ctx := context.Background()
	time.Sleep(100 * time.Millisecond)
	if reviewer.amazonCalls() != 0 || reviewer.calls() != 0 {
		t.Fatal("start-up started a review")
	}
	amazonItem, err := service.Get(ctx, "amz")
	if err != nil || amazonItem.PurchaseGoal != "" {
		t.Fatalf("Amazon goal after the upgrade %q %v", amazonItem.PurchaseGoal, err)
	}
	if stored, err := database.AmazonCustomerRating(ctx, "amz"); err != nil || stored.StarsTenths != nil || stored.Count != nil {
		t.Fatalf("Amazon rating after the upgrade %+v %v", stored, err)
	}
	state := reviewState(t, service, "amz")
	if state.Latest != nil || state.LastAttempt != nil || len(state.History) != 0 || !state.TokenConfigured || unavailableReason(state) != "" {
		t.Fatalf("Amazon review state after the upgrade %+v", state)
	}
	lbc := reviewState(t, service, "lbc")
	if item, _ := service.Get(ctx, "lbc"); item.PurchaseGoal != "Light distro" || lbc.Latest == nil || len(lbc.History) != 1 {
		t.Fatalf("LeBoncoin item changed by the upgrade: %q %+v", item.PurchaseGoal, lbc)
	}
	requestAmazonReview(t, service, "amz")
	state = waitReviews(t, service, "amz", false)
	if reviewer.amazonCalls() != 1 || state.Latest == nil || len(state.History) != 1 {
		t.Fatalf("first review after the upgrade: %d calls, %+v", reviewer.amazonCalls(), state)
	}
	if sent := reviewer.amazonInput(0).Rating; sent.StarsTenths != nil || sent.Count != nil {
		t.Fatalf("rating sent after the upgrade %+v", sent)
	}
}

func TestAmazonReview_TC016_DeleteRemovesGoalRatingAndReviews(t *testing.T) {
	service, database, reviewer, _ := newAmazonReviewService(t, t.TempDir(), true)
	ctx := context.Background()
	seedItemA(t, database, "A")
	seedItemA(t, database, "O")
	content := validAmazonReview()
	storeAmazonReview(t, database, "A", 11999, &content, secondCheckAt.Add(time.Hour))
	storeAmazonReview(t, database, "A", 11999, &content, secondCheckAt.Add(2*time.Hour))
	storeAmazonReview(t, database, "A", 11999, nil, secondCheckAt.Add(3*time.Hour))
	otherID := storeAmazonReview(t, database, "O", 11999, &content, secondCheckAt.Add(time.Hour))
	if found, err := service.Remove(ctx, "A"); err != nil || !found {
		t.Fatalf("remove: %v %v", found, err)
	}
	if _, err := service.Get(ctx, "A"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("details of a deleted item: %v", err)
	}
	if count := sqlCount(t, service, "SELECT (SELECT COUNT(*) FROM ai_reviews WHERE item_id = 'A') + (SELECT COUNT(*) FROM items WHERE id = 'A')"); count != 0 {
		t.Fatalf("%d rows of the deleted item kept", count)
	}
	if state := reviewState(t, service, "O"); state.Latest == nil || state.Latest.ID != otherID || amazonContent(t, state.Latest) != content {
		t.Fatalf("other item's review changed: %+v", state)
	}

	t.Run("deleted while a review runs", func(t *testing.T) {
		seedItemA(t, database, "C")
		reviewer.release = make(chan struct{})
		requestAmazonReview(t, service, "C")
		if found, err := service.Remove(ctx, "C"); err != nil || !found {
			t.Fatalf("remove: %v %v", found, err)
		}
		close(reviewer.release)
		waitFor(t, "the review of the deleted item to end", func() bool { return !service.reviewRunning("C") })
		if count := sqlCount(t, service, "SELECT COUNT(*) FROM ai_reviews WHERE item_id = 'C'"); count != 0 {
			t.Fatalf("%d reviews stored for the deleted item", count)
		}
	})
}

func TestAmazonReview_TC017_LeboncoinReviewsUnchanged(t *testing.T) {
	service, database, reviewer, _ := newAmazonReviewService(t, t.TempDir(), true)
	ctx := context.Background()
	seedItemA(t, database, "A")
	lbc, err := service.Add(ctx, "https://www.leboncoin.fr/ad/test/123", "Light distro")
	if err != nil {
		t.Fatal(err)
	}
	state := waitReviews(t, service, lbc.ID, true)
	if reviewer.calls() != 1 || state.Latest == nil || unavailableReason(state) != "" {
		t.Fatalf("LeBoncoin item not reviewed on add: %d %+v", reviewer.calls(), state)
	}
	if _, ok := state.Latest.Review.(*model.AIReviewContent); !ok {
		t.Fatalf("LeBoncoin review is %T", state.Latest.Review)
	}
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		amount := int64(900)
		return model.CollectionResult{Result: "success", AmountCents: amount, Listing: &model.ListingDetails{Title: "Lamp", PriceCents: &amount}}, leboncoin.SessionOutcome{}
	})
	service.Collect(ctx, lbc.ID)
	state = waitReviews(t, service, lbc.ID, true)
	if reviewer.calls() != 2 || len(state.History) != 2 || *state.Latest.PriceCents != 900 || reviewer.inputs[1].PurchaseGoal != "Light distro" {
		t.Fatalf("LeBoncoin item not reviewed on price change: %d %+v", reviewer.calls(), state)
	}
	requestAmazonReview(t, service, "A")
	waitReviews(t, service, "A", false)
	if after := reviewState(t, service, lbc.ID); reviewer.calls() != 2 || len(after.History) != 2 || after.Latest.ID != state.Latest.ID {
		t.Fatalf("an Amazon review changed the LeBoncoin reviews: %+v", after)
	}
}

func TestAmazonReview_TC018_TokenReplacedDuringRunningReview(t *testing.T) {
	service, database, reviewer, _ := newAmazonReviewService(t, t.TempDir(), true)
	ctx := context.Background()
	seedItemA(t, database, "A")
	seedItemA(t, database, "B")

	reviewer.amazonErr = claude.ErrRejected
	reviewer.release = make(chan struct{})
	requestAmazonReview(t, service, "A")
	if _, _, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: tokenInput("sk-ant-oat01-replacement")}); err != nil {
		t.Fatal(err)
	}
	close(reviewer.release)
	state := waitReviews(t, service, "A", false)
	if state.LastAttempt == nil || state.LastAttempt.Status != "failed" || *state.LastAttempt.ErrorMessage != "Claude rejected the token. Replace it in Settings." {
		t.Fatalf("review with the replaced token %+v", state.LastAttempt)
	}
	if reviewer.amazonCalls() != 1 || reviewer.amazonTokens[0] != "sk-ant-oat01-synthetic" {
		t.Fatalf("Claude requests %d with tokens %v", reviewer.amazonCalls(), reviewer.amazonTokens)
	}
	if token := mustToken(t, database); token.LastRejectedAt != nil {
		t.Fatal("the rejection of the replaced token marked the new token")
	}

	reviewer.amazonErr = nil
	reviewer.release = make(chan struct{})
	requestAmazonReview(t, service, "B")
	if _, _, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: &TokenInput{Remove: true}}); err != nil {
		t.Fatal(err)
	}
	close(reviewer.release)
	state = waitReviews(t, service, "B", false)
	if state.Latest == nil || state.Latest.Status != "succeeded" || state.TokenConfigured {
		t.Fatalf("review with the removed token %+v", state)
	}
	time.Sleep(100 * time.Millisecond)
	if reviewer.amazonCalls() != 2 || reviewer.amazonTokens[1] != "sk-ant-oat01-replacement" {
		t.Fatalf("a token change started a review: %d calls", reviewer.amazonCalls())
	}
}
