package store

import (
	"context"

	"pricefollower.local/internal/model"
)

// AmazonCustomerRating returns the latest recorded customer rating of an Amazon item; absent
// values are nil. Not implemented yet (issue #305).
func (s *Store) AmazonCustomerRating(ctx context.Context, id string) (model.CustomerRating, error) {
	return model.CustomerRating{}, nil
}
