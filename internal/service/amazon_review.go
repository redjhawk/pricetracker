package service

import (
	"context"
	"errors"
	"log"
	"time"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/model"
)

var errProductNotRead = errors.New("product not read")

// reviewSearchItem is called right after a successful search item read. It starts an asynchronous
// review when the item has no succeeded review yet; a price change never starts a new one.
func (s *Service) reviewSearchItem(id string, result model.CollectionResult) {
	_, succeeded, err := s.store.AIReviews(s.workerContext, id, "amazon")
	if err != nil {
		log.Printf("load AI reviews of search item %s: %v", id, err)
		return
	}
	if len(succeeded) > 0 {
		return
	}
	token, outcome, err := s.reserveReview(id)
	if err != nil {
		log.Printf("reserve AI review for item %s: %v", id, err)
	}
	if outcome != reviewStarted {
		return
	}
	reviewID, err := s.store.StartAIReview(s.workerContext, id, time.Now().UTC())
	if err != nil {
		s.releaseReview(id)
		log.Printf("start AI review for item %s: %v", id, err)
		return
	}
	s.workers.Add(1)
	go s.runAmazonReview(id, reviewID, token, result.Product)
}

func (s *Service) runAmazonReview(id string, reviewID int64, token model.ClaudeToken, product *model.ProductDetails) {
	defer s.workers.Done()
	defer s.releaseReview(id)
	if product == nil {
		s.finishReview(id, reviewID, token, nil, nil, errProductNotRead)
		return
	}
	ctx := s.workerContext
	listing, err := s.store.Listing(ctx, id)
	if err != nil {
		s.finishReview(id, reviewID, token, nil, nil, errProductNotRead)
		return
	}
	history, err := s.store.PriceHistory(ctx, id, 100)
	if err != nil {
		log.Printf("load price history for AI review of item %s: %v", id, err)
	}
	reviewContext, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	content, err := s.claude.ReviewAmazon(reviewContext, *token.Value, claude.AmazonReviewInput{Product: *product, URL: listing.URL, PriceHistory: history})
	s.finishReview(id, reviewID, token, product.PriceCents, &content, err)
}
