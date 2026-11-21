package store

import (
	"context"
	"fmt"
	"log"
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
	if err := s.convertGlobalAmazonStop(ctx); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "amazon_request_state", "human_checks_restarted", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "amazon_request_state", "human_checks_redraw", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return s.restartAmazonItemsOnce(ctx)
}

// restartAmazonItemsOnce restarts, once per database, the Amazon items stopped before the human-like
// checks (issue #168): their stop is cleared and their next check unset, so the service draws a time
// in the next Amazon window. The failure count is reset when items were restarted. The other Amazon
// items still have a configured check time, shared by all of them: they are marked for one redraw.
func (s *Store) restartAmazonItemsOnce(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var restarted bool
	if err := tx.QueryRowContext(ctx, "SELECT human_checks_restarted FROM amazon_request_state WHERE id = 1").Scan(&restarted); err != nil {
		return fmt.Errorf("read Amazon restart marker: %w", err)
	}
	if restarted {
		return nil
	}
	result, err := tx.ExecContext(ctx, "UPDATE items SET amazon_stopped_at = NULL, next_check_at = NULL WHERE platform = 'amazon' AND amazon_stopped_at IS NOT NULL")
	if err != nil {
		return fmt.Errorf("restart stopped Amazon items: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count > 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE amazon_request_state SET consecutive_failures = 0 WHERE id = 1"); err != nil {
			return fmt.Errorf("restart stopped Amazon items: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE amazon_request_state SET human_checks_redraw = 1
WHERE id = 1 AND EXISTS (SELECT 1 FROM items WHERE platform = 'amazon' AND next_check_at IS NOT NULL)`); err != nil {
		return fmt.Errorf("mark Amazon check times for a redraw: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE amazon_request_state SET human_checks_restarted = 1 WHERE id = 1"); err != nil {
		return fmt.Errorf("record Amazon restart: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if count > 0 {
		log.Printf("restarted %d stopped Amazon items once (issue #168)", count)
	}
	return nil
}

// AmazonIDsToSchedule returns the Amazon items that need a drawn next check: those without one
// (restarted by issue #168) and, until ClearAmazonRedraw, every Amazon item marked by the upgrade.
func (s *Store) AmazonIDsToSchedule(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM items WHERE platform = 'amazon'
AND (next_check_at IS NULL OR (SELECT human_checks_redraw FROM amazon_request_state WHERE id = 1) = 1) ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ClearAmazonRedraw records that the Amazon check times marked by the upgrade were redrawn.
func (s *Store) ClearAmazonRedraw(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "UPDATE amazon_request_state SET human_checks_redraw = 0 WHERE id = 1")
	return err
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
