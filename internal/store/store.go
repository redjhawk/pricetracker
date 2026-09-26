package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"pricefollower.local/config"
	"pricefollower.local/internal/model"
)

type Store struct {
	db     *sql.DB
	config config.Config
}

const timestampLayout = "2006-01-02T15:04:05.000Z"

func Open(cfg config.Config) (*Store, error) {
	if err := os.MkdirAll(cfg.DataDirectory, 0o750); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	db, err := sql.Open("sqlite", filepath.Join(cfg.DataDirectory, "pricefollower.sqlite"))
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	// A single connection keeps connection-level SQLite pragmas consistent and is ample for v1.
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{"PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000"} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to SQLite: %w", err)
	}
	s := &Store{db: db, config: cfg}
	if err := s.createSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) createSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS items (
  id TEXT PRIMARY KEY,
  asin TEXT NOT NULL,
  marketplace TEXT NOT NULL,
  canonical_url TEXT NOT NULL UNIQUE,
  url TEXT NOT NULL,
  title TEXT,
  thumbnail_url TEXT,
  next_check_at TEXT,
  added_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS price_observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  amount_cents INTEGER NOT NULL CHECK(amount_cents >= 0),
  currency TEXT NOT NULL DEFAULT 'EUR' CHECK(currency = 'EUR'),
  observed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS price_observations_item_time ON price_observations(item_id, observed_at DESC);
CREATE TABLE IF NOT EXISTS collection_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  result TEXT NOT NULL CHECK(result IN ('success', 'request_error', 'price_not_found', 'unavailable')),
  attempted_at TEXT NOT NULL,
  message TEXT
);
CREATE INDEX IF NOT EXISTS collection_attempts_item_time ON collection_attempts(item_id, attempted_at DESC);
CREATE INDEX IF NOT EXISTS items_next_check ON items(next_check_at);
`)
	if err != nil {
		return fmt.Errorf("create SQLite schema: %w", err)
	}
	return nil
}

func (s *Store) SeedDevelopment(ctx context.Context, nextCheck func(time.Time) time.Time) error {
	if !s.config.Development {
		return nil
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	type sample struct {
		id, asin, marketplace, title, thumbnail string
		prices                                  [][2]int
		attempt                                 string
		attemptHours                            int
	}
	samples := []sample{
		{"sample-sony-headphones", "B09XS7JWHH", "amazon.de", "Sony WH-1000XM5 Wireless Noise Canceling Headphones", "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=120&h=120&fit=crop&auto=format", [][2]int{{27900, 3}, {29900, 27}, {29900, 51}}, "success", 3},
		{"sample-kindle-paperwhite", "B0CFPJYX4M", "amazon.fr", "Kindle Paperwhite (16 Go) — Éclairage réglable chaud/froid", "https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=120&h=120&fit=crop&auto=format", [][2]int{{13999, 4}, {13999, 28}, {14999, 52}}, "success", 4},
		{"sample-lego-land-rover", "B07STGQK2S", "amazon.es", "LEGO Technic Land Rover Defender 42110", "https://images.unsplash.com/photo-1587654780291-39c9404d746b?w=120&h=120&fit=crop&auto=format", nil, "", 0},
		{"sample-nike-air-max", "B07D9FKTTN", "amazon.it", "Nike Air Max 270 Uomo", "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=120&h=120&fit=crop&auto=format", [][2]int{{9495, 60}, {10995, 84}, {10995, 108}}, "success", 60},
		{"sample-samsung-monitor", "B0CXMQZKNR", "amazon.de", "Samsung 27-Zoll-Monitor S27C432GAU", "https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=120&h=120&fit=crop&auto=format", [][2]int{{21900, 8}, {22900, 32}, {24900, 56}}, "request_error", 2},
		{"sample-title-unavailable", "B0D2XWJ7XY", "amazon.de", "", "", [][2]int{{4590, 120}}, "unavailable", 30},
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range samples {
		addedHours := 12
		if len(item.prices) > 0 {
			addedHours = item.prices[len(item.prices)-1][1]
		}
		var title, thumbnail any
		if item.title != "" {
			title = item.title
		}
		if item.thumbnail != "" {
			thumbnail = item.thumbnail
		}
		url := "https://" + item.marketplace + "/dp/" + item.asin
		if _, err := tx.ExecContext(ctx, `INSERT INTO items
(id, asin, marketplace, canonical_url, url, title, thumbnail_url, next_check_at, added_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, item.id, item.asin, item.marketplace, url, url, title, thumbnail,
			nextCheck(time.Now()).UTC().Format(timestampLayout), time.Now().Add(-time.Duration(addedHours)*time.Hour).UTC().Format(timestampLayout)); err != nil {
			return err
		}
		for _, price := range item.prices {
			stamp := time.Now().Add(-time.Duration(price[1]) * time.Hour).UTC().Format(timestampLayout)
			if _, err := tx.ExecContext(ctx, "INSERT INTO price_observations (item_id, amount_cents, currency, observed_at) VALUES (?, ?, 'EUR', ?)", item.id, price[0], stamp); err != nil {
				return err
			}
		}
		if item.attempt != "" {
			stamp := time.Now().Add(-time.Duration(item.attemptHours) * time.Hour).UTC().Format(timestampLayout)
			var message any
			if item.attempt == "request_error" {
				message = "Sample upstream request error"
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO collection_attempts (item_id, result, attempted_at, message) VALUES (?, ?, ?, ?)", item.id, item.attempt, stamp, message); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("Seeded %d sample items into %s\n", len(samples), filepath.Join(s.config.DataDirectory, "pricefollower.sqlite"))
	return nil
}

func (s *Store) List(ctx context.Context) ([]model.Item, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM items ORDER BY added_at DESC, id DESC")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, 100)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	items := make([]model.Item, 0, len(ids))
	for _, id := range ids {
		item, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) Get(ctx context.Context, id string) (model.Item, error) {
	var item model.Item
	var title, thumbnail, next sql.NullString
	var added string
	err := s.db.QueryRowContext(ctx, `SELECT id, title, asin, marketplace, url, thumbnail_url, next_check_at, added_at FROM items WHERE id = ?`, id).
		Scan(&item.ID, &title, &item.ASIN, &item.Marketplace, &item.URL, &thumbnail, &next, &added)
	if err != nil {
		return item, err
	}
	item.Title = nullString(title)
	item.ThumbnailURL = nullString(thumbnail)
	item.AddedAt = parseTimestamp(added)
	if next.Valid {
		parsed := parseTimestamp(next.String)
		item.NextCheckAt = &parsed
	}
	item.LastThreeDetections = make([]model.Observation, 0, 3)
	prices, err := s.db.QueryContext(ctx, `SELECT amount_cents, currency, observed_at FROM price_observations WHERE item_id = ? ORDER BY observed_at DESC, id DESC LIMIT 3`, id)
	if err != nil {
		return item, err
	}
	for prices.Next() {
		var observation model.Observation
		var timestamp string
		if err := prices.Scan(&observation.AmountCents, &observation.Currency, &timestamp); err != nil {
			prices.Close()
			return item, err
		}
		observation.Timestamp = parseTimestamp(timestamp)
		item.LastThreeDetections = append(item.LastThreeDetections, observation)
	}
	if err := prices.Err(); err != nil {
		prices.Close()
		return item, err
	}
	if err := prices.Close(); err != nil {
		return item, err
	}
	if len(item.LastThreeDetections) > 0 {
		latest := item.LastThreeDetections[0]
		item.LatestPrice = &latest
	}
	var result, attempted string
	var message sql.NullString
	err = s.db.QueryRowContext(ctx, `SELECT result, attempted_at, message FROM collection_attempts WHERE item_id = ? ORDER BY attempted_at DESC, id DESC LIMIT 1`, id).Scan(&result, &attempted, &message)
	if err == nil {
		attempt := &model.Attempt{Result: result, Timestamp: parseTimestamp(attempted), Message: nullString(message)}
		item.LastAttempt = attempt
	} else if !errors.Is(err, sql.ErrNoRows) {
		return item, err
	}
	item.Status = statusFor(item.LatestPrice, item.LastAttempt, s.config.StaleAfter)
	return item, nil
}

func statusFor(price *model.Observation, attempt *model.Attempt, staleAfter time.Duration) string {
	if attempt == nil {
		return "pending"
	}
	if attempt.Result == "unavailable" {
		return "unavailable"
	}
	if attempt.Result != "success" {
		return "retrieval_error"
	}
	if price == nil {
		return "pending"
	}
	if time.Since(price.Timestamp) > staleAfter {
		return "stale"
	}
	return "active"
}

func (s *Store) IsCanonicalTracked(ctx context.Context, canonicalURL string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM items WHERE canonical_url = ?", canonicalURL).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) Insert(ctx context.Context, item model.Listing, canonicalURL string, nextCheck time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO items (id, asin, marketplace, canonical_url, url, next_check_at, added_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.ASIN, item.Marketplace, canonicalURL, item.URL,
		nextCheck.UTC().Format(timestampLayout), time.Now().UTC().Format(timestampLayout))
	return err
}

func (s *Store) SetNextCheck(ctx context.Context, id string, next time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE items SET next_check_at = ? WHERE id = ?", next.UTC().Format(timestampLayout), id)
	return err
}

func (s *Store) DueIDs(ctx context.Context, now time.Time) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM items WHERE next_check_at IS NOT NULL AND next_check_at <= ? ORDER BY next_check_at ASC", now.UTC().Format(timestampLayout))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) Listing(ctx context.Context, id string) (model.Listing, error) {
	var item model.Listing
	err := s.db.QueryRowContext(ctx, "SELECT id, asin, marketplace, url FROM items WHERE id = ?", id).Scan(&item.ID, &item.ASIN, &item.Marketplace, &item.URL)
	return item, err
}

func (s *Store) RecordSuccess(ctx context.Context, id string, result model.CollectionResult, timestamp time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stamp := timestamp.UTC().Format(timestampLayout)
	if _, err := tx.ExecContext(ctx, "INSERT INTO price_observations (item_id, amount_cents, currency, observed_at) VALUES (?, ?, 'EUR', ?)", id, result.AmountCents, stamp); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO collection_attempts (item_id, result, attempted_at) VALUES (?, 'success', ?)", id, stamp); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE items SET title = COALESCE(?, title), thumbnail_url = COALESCE(?, thumbnail_url) WHERE id = ?", result.Title, result.ThumbnailURL, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RecordFailure(ctx context.Context, id, result, message string, timestamp time.Time) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO collection_attempts (item_id, result, attempted_at, message) VALUES (?, ?, ?, ?)", id, result, timestamp.UTC().Format(timestampLayout), message)
	return err
}

func (s *Store) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM items WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count > 0, err
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func parseTimestamp(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func IsUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
