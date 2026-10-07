package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// AmazonRequestState is the global stop state of Amazon requests, shared by every owner.
type AmazonRequestState struct {
	ConsecutiveFailures int
	Blocked             bool
	StoppedAt           *time.Time // kept after a restart until the next Amazon success
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
	return nil
}

// AmazonRequestState returns the persisted Amazon request state.
func (s *Store) AmazonRequestState(ctx context.Context) (AmazonRequestState, error) {
	var state AmazonRequestState
	var stoppedAt sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT consecutive_failures, blocked, stopped_at FROM amazon_request_state WHERE id = 1").
		Scan(&state.ConsecutiveFailures, &state.Blocked, &stoppedAt)
	state.StoppedAt = optionalTimestamp(stoppedAt)
	return state, err
}

// SaveAmazonRequestState replaces the persisted Amazon request state.
func (s *Store) SaveAmazonRequestState(ctx context.Context, state AmazonRequestState) error {
	_, err := s.db.ExecContext(ctx, "UPDATE amazon_request_state SET consecutive_failures = ?, blocked = ?, stopped_at = ? WHERE id = 1",
		state.ConsecutiveFailures, state.Blocked, optionalTimestampText(state.StoppedAt))
	return err
}
