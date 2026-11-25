package store

import (
	"context"
	"database/sql"

	"pricefollower.local/internal/model"
)

// ensureAmazonRatingColumns adds the customer rating of Amazon items (issue #305); existing rows
// get NULL, an unknown rating. The columns are never part of an item response.
func (s *Store) ensureAmazonRatingColumns(ctx context.Context) error {
	if err := s.ensureColumn(ctx, "items", "amazon_rating_tenths", "INTEGER CHECK (amazon_rating_tenths IS NULL OR amazon_rating_tenths BETWEEN 0 AND 50)"); err != nil {
		return err
	}
	return s.ensureColumn(ctx, "items", "amazon_rating_count", "INTEGER CHECK (amazon_rating_count IS NULL OR amazon_rating_count >= 0)")
}

// recordAmazonRating replaces the stored rating with the one read from a product page; nil values
// are recorded as absent.
func recordAmazonRating(ctx context.Context, tx *sql.Tx, id string, rating model.CustomerRating) error {
	_, err := tx.ExecContext(ctx, "UPDATE items SET amazon_rating_tenths = ?, amazon_rating_count = ? WHERE id = ?", rating.StarsTenths, rating.Count, id)
	return err
}

// AmazonCustomerRating returns the latest recorded customer rating of an Amazon item; absent
// values are nil.
func (s *Store) AmazonCustomerRating(ctx context.Context, id string) (model.CustomerRating, error) {
	var tenths, count sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT amazon_rating_tenths, amazon_rating_count FROM items WHERE id = ?", id).Scan(&tenths, &count); err != nil {
		return model.CustomerRating{}, err
	}
	var rating model.CustomerRating
	if tenths.Valid {
		rating.StarsTenths = &tenths.Int64
	}
	if count.Valid {
		rating.Count = &count.Int64
	}
	return rating, nil
}
