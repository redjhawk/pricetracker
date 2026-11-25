package service

import (
	"context"
	"slices"
	"time"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/model"
)

const amazonReviewHistoryLimit = 100

// amazonReviewAvailability returns why an Amazon item cannot be reviewed now: "stopped",
// "no_current_price", or "" when a review can be requested. A retrieval error keeps the last price.
func amazonReviewAvailability(stoppedAt *time.Time, latestPrice *model.Observation, lastAttempt *model.Attempt) string {
	switch {
	case stoppedAt != nil:
		return "stopped"
	case latestPrice == nil, lastAttempt != nil && (lastAttempt.Result == "price_not_found" || lastAttempt.Result == "unavailable"):
		return "no_current_price"
	}
	return ""
}

// requestAmazonReview starts a manual review of an Amazon item from its recorded data; it sends
// no Amazon request. Stopped items and items without a current price are refused before the token.
func (s *Service) requestAmazonReview(ctx context.Context, listing model.Listing) (reviewOutcome, error) {
	item, err := s.store.GetWithHistory(ctx, listing.ID)
	if err != nil {
		return reviewNoToken, err
	}
	switch amazonReviewAvailability(item.AmazonStoppedAt, item.LatestPrice, item.LastAttempt) {
	case "stopped":
		return reviewNoToken, &Error{Status: 409, Code: "AI_REVIEW_ITEM_STOPPED", Message: "Amazon checks are stopped: restart the item with Refresh price to request an AI review."}
	case "no_current_price":
		return reviewNoToken, &Error{Status: 409, Code: "AI_REVIEW_NO_CURRENT_PRICE", Message: "No current price: an AI review needs a current Amazon price."}
	}
	rating, err := s.store.AmazonCustomerRating(ctx, listing.ID)
	if err != nil {
		return reviewNoToken, err
	}
	input := amazonReviewInput(item, rating, listing.PurchaseGoal)
	token, outcome, err := s.reserveReview(listing.ID)
	if err != nil || outcome != reviewStarted {
		return outcome, err
	}
	reviewID, err := s.store.StartAIReview(s.workerContext, listing.ID, time.Now().UTC())
	if err != nil {
		s.releaseReview(listing.ID)
		return reviewNoToken, err
	}
	s.workers.Add(1)
	go s.runAmazonReview(listing.ID, reviewID, token, input)
	return reviewStarted, nil
}

// amazonReviewInput is the snapshot of the recorded data of an item read with its history.
func amazonReviewInput(item model.Item, rating model.CustomerRating, goal string) claude.AmazonReviewInput {
	input := claude.AmazonReviewInput{
		Marketplace:  item.Marketplace,
		URL:          item.URL,
		PriceCents:   item.LatestPrice.AmountCents,
		PriceHistory: oldestFirst(item.PriceHistory),
		Rating:       rating,
		PurchaseGoal: goal,
	}
	if item.Title != nil {
		input.Title = *item.Title
	}
	if offer := item.SecondHandOffer; offer != nil {
		input.SecondHandStatus = offer.Status
		input.SecondHandLatest = offer.LatestDetection
		input.SecondHandHistory = oldestFirst(offer.PriceHistory)
	}
	return input
}

// oldestFirst returns the latest amazonReviewHistoryLimit entries of a newest-first history, oldest first.
func oldestFirst[T any](newestFirst []T) []T {
	latest := slices.Clone(newestFirst[:min(len(newestFirst), amazonReviewHistoryLimit)])
	slices.Reverse(latest)
	return latest
}

func (s *Service) runAmazonReview(id string, reviewID int64, token model.ClaudeToken, input claude.AmazonReviewInput) {
	defer s.workers.Done()
	defer s.releaseReview(id)
	reviewContext, cancel := context.WithTimeout(s.workerContext, 150*time.Second)
	defer cancel()
	content, err := s.claude.ReviewAmazon(reviewContext, *token.Value, input)
	s.finishReview(id, reviewID, token, &input.PriceCents, &content, err)
}
