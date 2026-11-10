package store

import (
	"context"
	"fmt"
	"time"
)

// AmazonRequestState is the global count of consecutive failed Amazon requests, shared by every owner.
// The blocked and stopped_at columns of the former global stop are only read by the upgrade conversion.
type AmazonRequestState struct {
	ConsecutiveFailures int
}

func (s *Store) createAmazonRequestSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS amazon_request_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  consecutive_failures INTEGER NOT NULL DEFAULT 0,
  blocked INTEGER NOT NULL DEFAULT 0,
  stopped_at TEXT
);
INSERT OR IGNORE INTO amazon_request_state (id) VALUES (1);
`)
	if err != nil {
		return fmt.Errorf("create Amazon request state schema: %w", err)
	}
	if err := s.ensureColumn(ctx, "items", "amazon_stopped_at", "TEXT"); err != nil {
		return err
	}
	return s.convertGlobalAmazonStop(ctx)
}

// convertGlobalAmazonStop turns the former global stop into a stop of every Amazon item, with the
// former stop time (issue #111).
func (s *Store) convertGlobalAmazonStop(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(timestampLayout)
	if _, err := tx.ExecContext(ctx, `UPDATE items SET amazon_stopped_at = COALESCE((SELECT stopped_at FROM amazon_request_state WHERE id = 1), ?)
WHERE platform = 'amazon' AND amazon_stopped_at IS NULL AND (SELECT blocked FROM amazon_request_state WHERE id = 1) = 1`, now); err != nil {
		return fmt.Errorf("convert Amazon stop: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE amazon_request_state SET blocked = 0, consecutive_failures = 0 WHERE id = 1 AND blocked = 1"); err != nil {
		return fmt.Errorf("convert Amazon stop: %w", err)
	}
	return tx.Commit()
}

// AmazonRequestState returns the persisted Amazon request state.
func (s *Store) AmazonRequestState(ctx context.Context) (AmazonRequestState, error) {
	var state AmazonRequestState
	err := s.db.QueryRowContext(ctx, "SELECT consecutive_failures FROM amazon_request_state WHERE id = 1").Scan(&state.ConsecutiveFailures)
	return state, err
}

// SaveAmazonFailures persists the count of consecutive failed Amazon requests.
func (s *Store) SaveAmazonFailures(ctx context.Context, count int) error {
	_, err := s.db.ExecContext(ctx, "UPDATE amazon_request_state SET consecutive_failures = ? WHERE id = 1", count)
	return err
}

// StopAmazonItems stops every Amazon item of every owner that is not stopped yet.
func (s *Store) StopAmazonItems(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE items SET amazon_stopped_at = ? WHERE platform = 'amazon' AND amazon_stopped_at IS NULL", now.UTC().Format(timestampLayout))
	return err
}

// ClearAmazonStop restarts one Amazon item.
func (s *Store) ClearAmazonStop(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE items SET amazon_stopped_at = NULL WHERE id = ?", id)
	return err
}

// AmazonStopped reports whether the item is stopped; an unknown item is not stopped.
func (s *Store) AmazonStopped(ctx context.Context, id string) (bool, error) {
	var stopped bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM items WHERE id = ? AND amazon_stopped_at IS NOT NULL)", id).Scan(&stopped)
	return stopped, err
}
