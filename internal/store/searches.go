package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrSearchCaptured reports that a search already has its frozen set of items.
var ErrSearchCaptured = errors.New("Amazon search items already captured")

// Search is one saved Amazon search of an owner (0 is the open-mode owner).
type Search struct {
	ID               string
	OwnerID          int64
	URL              string
	AddedAt          time.Time
	CapturedAt       *time.Time // nil until the first results page succeeds
	NextRunAt        time.Time
	PassPosition     int // 0 = the next pass starts with the results page; n = next item position
	LastErrorAt      *time.Time
	LastErrorMessage *string
	ItemCount        int
}

// CapturedItem is one product of a results page, in page order.
type CapturedItem struct {
	ID           string // used only when the product is not an item of the owner yet
	ASIN         string
	Marketplace  string
	CanonicalURL string
	URL          string
	Title        *string
	PriceCents   *int64
}

func (s *Store) createSearchSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS amazon_searches (
  id TEXT PRIMARY KEY,
  owner_id INTEGER NOT NULL,
  url TEXT NOT NULL,
  added_at TEXT NOT NULL,
  captured_at TEXT,
  next_run_at TEXT NOT NULL,
  pass_position INTEGER NOT NULL DEFAULT 0,
  last_error_at TEXT,
  last_error_message TEXT,
  UNIQUE (owner_id, url)
);
CREATE TABLE IF NOT EXISTS amazon_search_items (
  search_id TEXT NOT NULL REFERENCES amazon_searches(id) ON DELETE CASCADE,
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  PRIMARY KEY (search_id, item_id),
  UNIQUE (search_id, position)
);
CREATE INDEX IF NOT EXISTS amazon_search_items_item ON amazon_search_items(item_id);
`)
	if err != nil {
		return fmt.Errorf("create Amazon search schema: %w", err)
	}
	return nil
}

const searchColumns = `id, owner_id, url, added_at, captured_at, next_run_at, pass_position, last_error_at, last_error_message,
(SELECT COUNT(*) FROM amazon_search_items WHERE search_id = amazon_searches.id)`

func scanSearch(scan func(...any) error) (Search, error) {
	var search Search
	var added, nextRun string
	var captured, errorAt, errorMessage sql.NullString
	err := scan(&search.ID, &search.OwnerID, &search.URL, &added, &captured, &nextRun, &search.PassPosition, &errorAt, &errorMessage, &search.ItemCount)
	search.AddedAt = parseTimestamp(added)
	search.NextRunAt = parseTimestamp(nextRun)
	search.CapturedAt = optionalTimestamp(captured)
	search.LastErrorAt = optionalTimestamp(errorAt)
	search.LastErrorMessage = nullString(errorMessage)
	return search, err
}

func (s *Store) querySearches(ctx context.Context, query string, args ...any) ([]Search, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+searchColumns+" FROM amazon_searches "+query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	searches := make([]Search, 0)
	for rows.Next() {
		search, err := scanSearch(rows.Scan)
		if err != nil {
			return nil, err
		}
		searches = append(searches, search)
	}
	return searches, rows.Err()
}

// InsertSearch saves a new search; a duplicate URL for the owner is a unique-constraint error.
func (s *Store) InsertSearch(ctx context.Context, search Search) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO amazon_searches (id, owner_id, url, added_at, next_run_at) VALUES (?, ?, ?, ?, ?)",
		search.ID, search.OwnerID, search.URL, search.AddedAt.UTC().Format(timestampLayout), search.NextRunAt.UTC().Format(timestampLayout))
	return err
}

// Searches returns the owner's searches, most recent first.
func (s *Store) Searches(ctx context.Context, ownerID int64) ([]Search, error) {
	return s.querySearches(ctx, "WHERE owner_id = ? ORDER BY added_at DESC, id DESC", ownerID)
}

// Search returns one search of the owner, or sql.ErrNoRows.
func (s *Store) Search(ctx context.Context, ownerID int64, id string) (Search, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+searchColumns+" FROM amazon_searches WHERE owner_id = ? AND id = ?", ownerID, id)
	return scanSearch(row.Scan)
}

// DueSearches returns the searches of every owner whose next pass is due, earliest first.
func (s *Store) DueSearches(ctx context.Context, now time.Time) ([]Search, error) {
	return s.querySearches(ctx, "WHERE next_run_at <= ? ORDER BY next_run_at, added_at", now.UTC().Format(timestampLayout))
}

// SearchItemIDs returns the item ids of a search in Amazon order.
func (s *Store) SearchItemIDs(ctx context.Context, searchID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT item_id FROM amazon_search_items WHERE search_id = ? ORDER BY position", searchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, 30)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CaptureSearch stores the frozen set of search items once. A product the owner already has
// (tracked or in another search) is shared and keeps its title and history.
func (s *Store) CaptureSearch(ctx context.Context, searchID string, ownerID int64, items []CapturedItem, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var capturedAt sql.NullString
	if err := tx.QueryRowContext(ctx, "SELECT captured_at FROM amazon_searches WHERE id = ? AND owner_id = ?", searchID, ownerID).Scan(&capturedAt); err != nil {
		return err
	}
	if capturedAt.Valid {
		return ErrSearchCaptured
	}
	stamp := now.UTC().Format(timestampLayout)
	for index, captured := range items {
		inserted, err := tx.ExecContext(ctx, `INSERT INTO items (id, asin, platform, listing_id, marketplace, canonical_url, url, title, next_check_at, added_at, owner_id, tracked)
VALUES (?, ?, 'amazon', ?, ?, ?, ?, ?, NULL, ?, ?, 0) ON CONFLICT(owner_id, canonical_url) DO NOTHING`,
			captured.ID, captured.ASIN, captured.ASIN, captured.Marketplace, captured.CanonicalURL, captured.URL, captured.Title, stamp, ownerID)
		if err != nil {
			return err
		}
		count, err := inserted.RowsAffected()
		if err != nil {
			return err
		}
		var itemID string
		if err := tx.QueryRowContext(ctx, "SELECT id FROM items WHERE owner_id = ? AND canonical_url = ?", ownerID, captured.CanonicalURL).Scan(&itemID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO amazon_search_items (search_id, item_id, position) VALUES (?, ?, ?)", searchID, itemID, index+1); err != nil {
			return err
		}
		if count > 0 && captured.PriceCents != nil {
			if _, err := tx.ExecContext(ctx, "INSERT INTO price_observations (item_id, amount_cents, currency, observed_at) VALUES (?, ?, 'EUR', ?)", itemID, *captured.PriceCents, stamp); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE amazon_searches SET captured_at = ?, last_error_at = NULL, last_error_message = NULL WHERE id = ?", stamp, searchID); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteSearch removes a search and its items that are neither tracked nor in another search.
func (s *Store) DeleteSearch(ctx context.Context, ownerID int64, id string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM amazon_searches WHERE id = ? AND owner_id = ?", id, ownerID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM items WHERE tracked = 0
AND id IN (SELECT item_id FROM amazon_search_items WHERE search_id = ?)
AND id NOT IN (SELECT item_id FROM amazon_search_items WHERE search_id != ?)`, id, id); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM amazon_searches WHERE id = ?", id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// TrackItem moves an untracked item of the owner to the tracked list and reports whether it did.
func (s *Store) TrackItem(ctx context.Context, ownerID int64, itemID string, nextCheck time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, "UPDATE items SET tracked = 1, next_check_at = ? WHERE id = ? AND owner_id = ? AND tracked = 0",
		nextCheck.UTC().Format(timestampLayout), itemID, ownerID)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count > 0, err
}

// UntrackedItemID returns the id of the owner's search-only item for the canonical URL, or "".
func (s *Store) UntrackedItemID(ctx context.Context, ownerID int64, canonicalURL string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM items WHERE owner_id = ? AND canonical_url = ? AND tracked = 0", ownerID, canonicalURL).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// SetSearchProgress saves where the current pass resumes and when the next pass is due.
func (s *Store) SetSearchProgress(ctx context.Context, id string, position int, nextRunAt time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE amazon_searches SET pass_position = ?, next_run_at = ? WHERE id = ?", position, nextRunAt.UTC().Format(timestampLayout), id)
	return err
}

// SetSearchError records the latest failed results-page request.
func (s *Store) SetSearchError(ctx context.Context, id string, at time.Time, message string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE amazon_searches SET last_error_at = ?, last_error_message = ? WHERE id = ?", at.UTC().Format(timestampLayout), message, id)
	return err
}

// ClearSearchError forgets the latest failed results-page request after a success.
func (s *Store) ClearSearchError(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE amazon_searches SET last_error_at = NULL, last_error_message = NULL WHERE id = ?", id)
	return err
}
