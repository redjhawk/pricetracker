package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"pricefollower.local/internal/model"
)

// InterruptedReviewMessage is stored for reviews still pending at startup.
const InterruptedReviewMessage = "The review was interrupted by a server restart."

func (s *Store) createAIReviewSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS ai_reviews (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed')),
  price_cents INTEGER CHECK (price_cents IS NULL OR price_cents >= 0),
  review_json TEXT,
  error_message TEXT,
  created_at TEXT NOT NULL,
  completed_at TEXT
);
CREATE INDEX IF NOT EXISTS ai_reviews_item_time ON ai_reviews(item_id, created_at DESC, id DESC);
`)
	if err != nil {
		return fmt.Errorf("create AI review schema: %w", err)
	}
	return nil
}

// FailInterruptedAIReviews fails reviews left pending by a previous process.
func (s *Store) FailInterruptedAIReviews(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE ai_reviews SET status = 'failed', error_message = ?, completed_at = ? WHERE status = 'pending'",
		InterruptedReviewMessage, now.UTC().Format(timestampLayout))
	return err
}

// StartAIReview stores a pending review attempt and returns its ID.
func (s *Store) StartAIReview(ctx context.Context, itemID string, now time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, "INSERT INTO ai_reviews (item_id, status, created_at) VALUES (?, 'pending', ?)", itemID, now.UTC().Format(timestampLayout))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// FinishAIReview completes a review attempt. A deleted item (no row) is not an error.
func (s *Store) FinishAIReview(ctx context.Context, id int64, status string, priceCents *int64, review *model.AIReviewContent, errorMessage string, now time.Time) error {
	var reviewJSON, message any
	if review != nil {
		encoded, err := json.Marshal(review)
		if err != nil {
			return err
		}
		reviewJSON = string(encoded)
	}
	if errorMessage != "" {
		message = errorMessage
	}
	_, err := s.db.ExecContext(ctx, "UPDATE ai_reviews SET status = ?, price_cents = ?, review_json = ?, error_message = ?, completed_at = ? WHERE id = ?",
		status, priceCents, reviewJSON, message, now.UTC().Format(timestampLayout), id)
	return err
}

// AIReviews returns the latest attempt of any status and up to 50 succeeded reviews, newest first.
func (s *Store) AIReviews(ctx context.Context, itemID string) (*model.AIReview, []model.AIReview, error) {
	const columns = "id, status, price_cents, review_json, error_message, created_at, completed_at"
	latest, err := scanAIReview(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM ai_reviews WHERE item_id = ? ORDER BY created_at DESC, id DESC LIMIT 1", itemID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, make([]model.AIReview, 0), nil
	}
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+columns+" FROM ai_reviews WHERE item_id = ? AND status = 'succeeded' ORDER BY created_at DESC, id DESC LIMIT 50", itemID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	succeeded := make([]model.AIReview, 0)
	for rows.Next() {
		review, err := scanAIReview(rows)
		if err != nil {
			return nil, nil, err
		}
		succeeded = append(succeeded, *review)
	}
	return latest, succeeded, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanAIReview(row scanner) (*model.AIReview, error) {
	var review model.AIReview
	var price sql.NullInt64
	var reviewJSON, message, completed sql.NullString
	var created string
	if err := row.Scan(&review.ID, &review.Status, &price, &reviewJSON, &message, &created, &completed); err != nil {
		return nil, err
	}
	if price.Valid {
		review.PriceCents = &price.Int64
	}
	if reviewJSON.Valid {
		review.Review = &model.AIReviewContent{}
		if err := json.Unmarshal([]byte(reviewJSON.String), review.Review); err != nil {
			return nil, fmt.Errorf("decode stored AI review %d: %w", review.ID, err)
		}
	}
	review.ErrorMessage = nullString(message)
	review.CreatedAt = parseTimestamp(created)
	review.CompletedAt = optionalTimestamp(completed)
	return &review, nil
}

// LatestPriceCents returns the most recent recorded price, or nil when none exists.
func (s *Store) LatestPriceCents(ctx context.Context, itemID string) (*int64, error) {
	var amount int64
	err := s.db.QueryRowContext(ctx, "SELECT amount_cents FROM price_observations WHERE item_id = ? ORDER BY observed_at DESC, id DESC LIMIT 1", itemID).Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &amount, nil
}

// PriceHistory returns the latest limit recorded prices, oldest first.
func (s *Store) PriceHistory(ctx context.Context, itemID string, limit int) ([]model.Observation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT amount_cents, currency, observed_at FROM (
  SELECT id, amount_cents, currency, observed_at FROM price_observations WHERE item_id = ? ORDER BY observed_at DESC, id DESC LIMIT ?
) ORDER BY observed_at ASC, id ASC`, itemID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := make([]model.Observation, 0)
	for rows.Next() {
		var observation model.Observation
		var timestamp string
		if err := rows.Scan(&observation.AmountCents, &observation.Currency, &timestamp); err != nil {
			return nil, err
		}
		observation.Timestamp = observationTimestamp(timestamp)
		history = append(history, observation)
	}
	return history, rows.Err()
}
