package claude

import (
	"context"

	"pricefollower.local/internal/model"
)

// AmazonReviewInput is everything sent to Claude for one Amazon item review, from recorded data.
type AmazonReviewInput struct {
	Title, Marketplace, URL string
	PriceCents              int64                         // current price (reviewed price)
	PriceHistory            []model.Observation           // oldest first, latest 100
	SecondHandStatus        string                        // available | not_found | check_error | pending
	SecondHandLatest        *model.SecondHandObservation  // latest recorded detection, nil when none
	SecondHandHistory       []model.SecondHandObservation // oldest first, latest 100
	Rating                  model.CustomerRating
	PurchaseGoal            string
}

// ReviewAmazon asks Claude for a structured review of an Amazon item. Not implemented yet (issue #305).
func (c *Client) ReviewAmazon(ctx context.Context, token string, input AmazonReviewInput) (model.AmazonAIReviewContent, error) {
	return model.AmazonAIReviewContent{}, ErrBadResponse
}
