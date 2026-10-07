package service

import (
	"context"
	"errors"
	"log"
	"time"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

const listingNotRetrievedMessage = "The listing could not be retrieved from LeBoncoin."

// RequestAIReview starts a manual review of the current LeBoncoin listing.
// It reports alreadyRunning instead of starting a second review for the item.
func (s *Service) RequestAIReview(ctx context.Context, id string) (time.Time, bool, error) {
	listing, err := s.ownedListing(ctx, id)
	if err != nil {
		return time.Time{}, false, err
	}
	if listing.Platform != "leboncoin" {
		return time.Time{}, false, &Error{Status: 422, Code: "AI_REVIEW_UNSUPPORTED", Message: "AI reviews are available for LeBoncoin items only."}
	}
	requestedAt := time.Now().UTC()
	outcome, err := s.startReview(id, nil, true)
	if err != nil {
		return time.Time{}, false, err
	}
	switch outcome {
	case reviewStarted:
		return requestedAt, false, nil
	case reviewAlreadyRunning:
		return requestedAt, true, nil
	default:
		return time.Time{}, false, &Error{Status: 409, Code: "CLAUDE_TOKEN_MISSING", Message: "Configure a Claude token in Settings."}
	}
}

// AIReview returns the aiReview field for the item details: always for LeBoncoin items; for Amazon
// items only when they belong to a search or have reviews; otherwise nil.
func (s *Service) AIReview(ctx context.Context, item model.Item) (*model.AIReviewState, error) {
	lastAttempt, succeeded, err := s.store.AIReviews(ctx, item.ID, item.Platform)
	if err != nil {
		return nil, err
	}
	if item.Platform == "amazon" && lastAttempt == nil {
		inSearch, err := s.store.IsSearchItem(ctx, item.ID)
		if err != nil || !inSearch {
			return nil, err
		}
	}
	token, err := s.store.ClaudeToken(ctx, ownerFrom(ctx))
	if err != nil {
		return nil, err
	}
	state := &model.AIReviewState{TokenConfigured: token.Value != nil, Running: s.reviewRunning(item.ID), LastAttempt: lastAttempt, History: succeeded}
	if len(succeeded) > 0 {
		state.Latest = &succeeded[0]
	}
	return state, nil
}

func (s *Service) reviewRunning(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reviewing[id]
}

type reviewOutcome int

const (
	reviewStarted reviewOutcome = iota
	reviewAlreadyRunning
	reviewNoToken
)

// startReview starts an asynchronous review unless no token is saved or a
// review is already running for the item. With fetchFresh the worker fetches
// the listing itself; otherwise it reviews details (nil fails the review).
func (s *Service) startReview(id string, details *model.ListingDetails, fetchFresh bool) (reviewOutcome, error) {
	token, outcome, err := s.reserveReview(id)
	if err != nil || outcome != reviewStarted {
		return outcome, err
	}
	if err := s.launchReview(id, token, details, fetchFresh); err != nil {
		return reviewNoToken, err
	}
	return reviewStarted, nil
}

// reserveReview marks the item as reviewing (reviewStarted) when its owner has
// a token saved and no review runs. The token is read before taking s.mu; only
// the running check-and-set is under the lock. The caller must launch or
// release a reservation.
func (s *Service) reserveReview(id string) (model.ClaudeToken, reviewOutcome, error) {
	listing, err := s.store.Listing(s.workerContext, id)
	if err != nil {
		return model.ClaudeToken{}, reviewNoToken, err
	}
	token, err := s.store.ClaudeToken(s.workerContext, listing.OwnerID)
	if err != nil {
		return model.ClaudeToken{}, reviewNoToken, err
	}
	if token.Value == nil {
		return model.ClaudeToken{}, reviewNoToken, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reviewing[id] {
		return model.ClaudeToken{}, reviewAlreadyRunning, nil
	}
	s.reviewing[id] = true
	return token, reviewStarted, nil
}

// releaseReview releases a review reservation, whether or not the review ran,
// and starts one more review when the purchase goal changed meanwhile.
func (s *Service) releaseReview(id string) {
	s.mu.Lock()
	again := s.reviewAgain[id]
	delete(s.reviewAgain, id)
	delete(s.reviewing, id)
	s.mu.Unlock()
	if !again {
		return
	}
	if _, err := s.startReview(id, nil, true); err != nil {
		log.Printf("restart AI review for item %s: %v", id, err)
	}
}

// launchReview stores the pending attempt and runs a reserved review.
func (s *Service) launchReview(id string, token model.ClaudeToken, details *model.ListingDetails, fetchFresh bool) error {
	reviewID, err := s.store.StartAIReview(s.workerContext, id, time.Now().UTC())
	if err != nil {
		s.releaseReview(id)
		return err
	}
	s.workers.Add(1)
	go s.runReview(id, reviewID, token, details, fetchFresh)
	return nil
}

func (s *Service) runReview(id string, reviewID int64, token model.ClaudeToken, details *model.ListingDetails, fetchFresh bool) {
	defer s.workers.Done()
	defer s.releaseReview(id)
	review, err := s.reviewListing(s.workerContext, id, *token.Value, details, fetchFresh)
	s.finishReview(id, reviewID, token, review.priceCents, &review.content, err)
}

// finishReview stores the outcome of a review of either platform; content is used only when err is nil.
func (s *Service) finishReview(id string, reviewID int64, token model.ClaudeToken, priceCents *int64, content any, err error) {
	// The final write uses a fresh context so a shutdown still records the outcome.
	finishContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := time.Now().UTC()
	if err == nil {
		if err := s.store.FinishAIReview(finishContext, reviewID, "succeeded", priceCents, content, "", now); err != nil {
			log.Printf("save AI review for item %s: %v", id, err)
		}
		if err := s.store.ClearClaudeTokenRejected(finishContext, token.OwnerID, token.Revision); err != nil {
			log.Printf("clear Claude token rejection: %v", err)
		}
		log.Printf("AI review succeeded item=%s", id)
		return
	}
	message := reviewFailureMessage(s.workerContext, err)
	if errors.Is(err, claude.ErrRejected) || errors.Is(err, claude.ErrUsageLimit) {
		if err := s.store.MarkClaudeTokenRejected(finishContext, token.OwnerID, token.Revision, now); err != nil {
			log.Printf("mark Claude token rejected: %v", err)
		}
	}
	if err := s.store.FinishAIReview(finishContext, reviewID, "failed", nil, nil, message, now); err != nil {
		log.Printf("save failed AI review for item %s: %v", id, err)
	}
	log.Printf("AI review failed item=%s reason=%q", id, message)
}

type reviewResult struct {
	content    model.AIReviewContent
	priceCents *int64
}

var errListingNotRetrieved = errors.New("listing not retrieved")

func (s *Service) reviewListing(ctx context.Context, id, token string, details *model.ListingDetails, fetchFresh bool) (reviewResult, error) {
	listing, err := s.store.Listing(ctx, id)
	if err != nil {
		return reviewResult{}, errListingNotRetrieved
	}
	if fetchFresh {
		// Same fetch as a price check, without recording a price or attempt.
		details = s.collectLeboncoin(ctx, listing).Listing
	}
	if details == nil {
		return reviewResult{}, errListingNotRetrieved
	}
	history, err := s.store.PriceHistory(ctx, id, 100)
	if err != nil {
		log.Printf("load price history for AI review of item %s: %v", id, err)
	}
	reviewContext, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	content, err := s.claude.Review(reviewContext, token, claude.ReviewInput{Listing: *details, URL: listing.URL, PriceHistory: history, PurchaseGoal: listing.PurchaseGoal})
	if err != nil {
		return reviewResult{}, err
	}
	return reviewResult{content: content, priceCents: details.PriceCents}, nil
}

func reviewFailureMessage(ctx context.Context, err error) string {
	switch {
	case ctx.Err() != nil:
		return store.InterruptedReviewMessage
	case errors.Is(err, errListingNotRetrieved):
		return listingNotRetrievedMessage
	case errors.Is(err, errProductNotRead):
		return "The product could not be read from Amazon."
	case errors.Is(err, claude.ErrRejected):
		return "Claude rejected the token. Replace it in Settings."
	case errors.Is(err, claude.ErrUsageLimit):
		return "Claude usage limit reached. Try again later."
	case errors.Is(err, claude.ErrBadResponse):
		return "Claude returned an unusable review. Try again."
	default:
		return "Claude could not be reached. Try again later."
	}
}
