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
  platform TEXT NOT NULL DEFAULT 'amazon',
  listing_id TEXT NOT NULL DEFAULT '',
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
CREATE TABLE IF NOT EXISTS second_hand_offer_checks (
  item_id TEXT PRIMARY KEY REFERENCES items(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK(status IN ('available', 'not_found', 'check_error')),
  checked_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS second_hand_offer_observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  amount_cents INTEGER NOT NULL CHECK(amount_cents >= 0),
  currency TEXT NOT NULL DEFAULT 'EUR' CHECK(currency = 'EUR'),
  condition TEXT NOT NULL CHECK(condition IN ('like_new', 'very_good', 'good', 'acceptable', 'unknown')),
  condition_label TEXT NOT NULL,
  observed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS second_hand_offer_item_time ON second_hand_offer_observations(item_id, observed_at DESC);
`)
	if err != nil {
		return fmt.Errorf("create SQLite schema: %w", err)
	}
	for _, column := range []struct{ name, declaration string }{
		{"platform", "TEXT NOT NULL DEFAULT 'amazon'"},
		{"listing_id", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := s.ensureItemColumn(ctx, column.name, column.declaration); err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE items SET listing_id = asin WHERE listing_id = '' AND platform = 'amazon'"); err != nil {
		return fmt.Errorf("migrate item listing identifiers: %w", err)
	}
	return nil
}

func (s *Store) ensureItemColumn(ctx context.Context, name, declaration string) error {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info(items)")
	if err != nil {
		return fmt.Errorf("inspect items schema: %w", err)
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var columnName, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return fmt.Errorf("read items schema: %w", err)
		}
		if columnName == name {
			rows.Close()
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read items schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close items schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, "ALTER TABLE items ADD COLUMN "+name+" "+declaration); err != nil {
		return fmt.Errorf("add items.%s column: %w", name, err)
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
		if err := s.seedSecondHandExamples(ctx); err != nil {
			return err
		}
		return s.seedLeBoncoinExamples(ctx, nextCheck)
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
(id, asin, platform, listing_id, marketplace, canonical_url, url, title, thumbnail_url, next_check_at, added_at)
VALUES (?, ?, 'amazon', ?, ?, ?, ?, ?, ?, ?, ?)`, item.id, item.asin, item.asin, item.marketplace, url, url, title, thumbnail,
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
	if err := s.seedSecondHandExamples(ctx); err != nil {
		return err
	}
	if err := s.seedLeBoncoinExamples(ctx, nextCheck); err != nil {
		return err
	}
	fmt.Printf("Seeded %d sample items into %s\n", len(samples), filepath.Join(s.config.DataDirectory, "pricefollower.sqlite"))
	return nil
}

func (s *Store) seedLeBoncoinExamples(ctx context.Context, nextCheck func(time.Time) time.Time) error {
	now := time.Now().UTC()
	examples := []struct {
		id, listingID, category, title, thumbnail string
		priceCents                                int64
		hoursAgo                                  int
	}{
		{"sample-lbc-pool", "3259094860", "jardin_plantes", "Bestway Piscine Hors Sol 404 x 201 x 100 cm Rectangulaire Tubulaire", "https://img.leboncoin.fr/api/v1/lbcpb1/images/e8/2f/29/e82f29bc9cb04ef0378ff3e29b8256f3e4c6a7fa.jpg?rule=ad-thumb", 9900, 4},
		{"sample-lbc-donation", "3277184962", "bricolage", "Donne dalle noir très lourd", "https://img.leboncoin.fr/api/v1/lbcpb1/images/02/d6/4c/02d64c32f9ed60343cb5130c1080d2e69c68ec27.jpg?rule=ad-thumb", 0, 8},
	}
	for _, item := range examples {
		url := "https://www.leboncoin.fr/ad/" + item.category + "/" + item.listingID
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO items
(id, asin, platform, listing_id, marketplace, canonical_url, url, title, thumbnail_url, next_check_at, added_at)
VALUES (?, '', 'leboncoin', ?, 'leboncoin.fr', ?, ?, ?, ?, ?, ?)`, item.id, item.listingID, url, url, item.title, item.thumbnail,
			nextCheck(now).UTC().Format(timestampLayout), now.Add(-time.Duration(item.hoursAgo)*time.Hour).Format(timestampLayout)); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO price_observations (item_id, amount_cents, currency, observed_at)
SELECT ?, ?, 'EUR', ? WHERE NOT EXISTS (SELECT 1 FROM price_observations WHERE item_id = ?)`, item.id, item.priceCents,
			now.Add(-time.Duration(item.hoursAgo)*time.Hour).Format(timestampLayout), item.id); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO collection_attempts (item_id, result, attempted_at, message)
SELECT ?, 'success', ?, NULL WHERE NOT EXISTS (SELECT 1 FROM collection_attempts WHERE item_id = ?)`, item.id,
			now.Add(-time.Duration(item.hoursAgo)*time.Hour).Format(timestampLayout), item.id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) seedSecondHandExamples(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM second_hand_offer_checks").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	type sampleCheck struct {
		id     string
		status string
	}
	checks := []sampleCheck{
		{"sample-sony-headphones", "not_found"},
		{"sample-kindle-paperwhite", "available"},
		{"sample-lego-land-rover", "not_found"},
		{"sample-nike-air-max", "available"},
		{"sample-samsung-monitor", "check_error"},
		{"sample-title-unavailable", "check_error"},
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	for _, check := range checks {
		checkedAgo := 3 * time.Hour
		if check.status == "available" {
			checkedAgo = 4 * time.Hour
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO second_hand_offer_checks (item_id, status, checked_at)
SELECT id, ?, ? FROM items WHERE id = ?`, check.status, now.Add(-checkedAgo).Format(timestampLayout), check.id); err != nil {
			return err
		}
	}
	offers := []struct {
		amount    int
		condition string
		label     string
		hoursAgo  int
	}{
		{13499, "very_good", "Très bon état", 4},
		{12999, "good", "Bon état", 28},
		{12999, "good", "Bon état", 52},
	}
	for _, offer := range offers {
		if _, err := tx.ExecContext(ctx, `INSERT INTO second_hand_offer_observations
(item_id, amount_cents, currency, condition, condition_label, observed_at)
SELECT id, ?, 'EUR', ?, ?, ? FROM items WHERE id = 'sample-kindle-paperwhite'`, offer.amount, offer.condition, offer.label,
			now.Add(-time.Duration(offer.hoursAgo)*time.Hour).Format(timestampLayout)); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO second_hand_offer_observations
(item_id, amount_cents, currency, condition, condition_label, observed_at)
SELECT id, 8995, 'EUR', 'good', 'Buone condizioni', ? FROM items WHERE id = 'sample-nike-air-max'`, now.Add(-5*time.Hour).Format(timestampLayout))
	if err != nil {
		return err
	}
	return tx.Commit()
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
	var asin, title, thumbnail, next sql.NullString
	var added string
	err := s.db.QueryRowContext(ctx, `SELECT id, title, platform, listing_id, asin, marketplace, url, thumbnail_url, next_check_at, added_at FROM items WHERE id = ?`, id).
		Scan(&item.ID, &title, &item.Platform, &item.ListingID, &asin, &item.Marketplace, &item.URL, &thumbnail, &next, &added)
	if err != nil {
		return item, err
	}
	if item.Platform == "amazon" && asin.Valid {
		item.ASIN = &asin.String
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
	if item.Platform == "amazon" {
		item.SecondHandOffer = &model.SecondHandOffer{Status: "pending", LastThreeDetections: make([]model.SecondHandObservation, 0, 3)}
		var usedStatus, usedChecked string
		err = s.db.QueryRowContext(ctx, "SELECT status, checked_at FROM second_hand_offer_checks WHERE item_id = ?", id).Scan(&usedStatus, &usedChecked)
		if err == nil {
			item.SecondHandOffer.Status = usedStatus
			checked := parseTimestamp(usedChecked)
			item.SecondHandOffer.LastCheckedAt = &checked
		} else if !errors.Is(err, sql.ErrNoRows) {
			return item, err
		}
		usedRows, err := s.db.QueryContext(ctx, `SELECT amount_cents, currency, condition, condition_label, observed_at FROM second_hand_offer_observations
WHERE item_id = ? ORDER BY observed_at DESC, id DESC LIMIT 3`, id)
		if err != nil {
			return item, err
		}
		for usedRows.Next() {
			var detection model.SecondHandObservation
			var observed string
			if err := usedRows.Scan(&detection.AmountCents, &detection.Currency, &detection.Condition, &detection.ConditionLabel, &observed); err != nil {
				usedRows.Close()
				return item, err
			}
			detection.Timestamp = parseTimestamp(observed)
			item.SecondHandOffer.LastThreeDetections = append(item.SecondHandOffer.LastThreeDetections, detection)
		}
		if err := usedRows.Err(); err != nil {
			usedRows.Close()
			return item, err
		}
		if err := usedRows.Close(); err != nil {
			return item, err
		}
		if len(item.SecondHandOffer.LastThreeDetections) > 0 {
			latest := item.SecondHandOffer.LastThreeDetections[0]
			item.SecondHandOffer.LatestDetection = &latest
		}
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
	_, err := s.db.ExecContext(ctx, `INSERT INTO items (id, asin, platform, listing_id, marketplace, canonical_url, url, next_check_at, added_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.ASIN, item.Platform, item.ListingID, item.Marketplace, canonicalURL, item.URL,
		nextCheck.UTC().Format(timestampLayout), time.Now().UTC().Format(timestampLayout))
	return err
}

func (s *Store) SetNextCheck(ctx context.Context, id string, next time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE items SET next_check_at = ? WHERE id = ?", next.UTC().Format(timestampLayout), id)
	return err
}

func (s *Store) SetNextChecks(ctx context.Context, ids []string, next time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stamp := next.UTC().Format(timestampLayout)
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, "UPDATE items SET next_check_at = ? WHERE id = ?", stamp, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) IDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM items ORDER BY added_at DESC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, 100)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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
	err := s.db.QueryRowContext(ctx, "SELECT id, platform, listing_id, asin, marketplace, url FROM items WHERE id = ?", id).
		Scan(&item.ID, &item.Platform, &item.ListingID, &item.ASIN, &item.Marketplace, &item.URL)
	return item, err
}

func (s *Store) RecordSuccess(ctx context.Context, id string, result model.CollectionResult, timestamp time.Time) error {
	return s.RecordCollection(ctx, id, result, timestamp)
}

func (s *Store) RecordCollection(ctx context.Context, id string, result model.CollectionResult, timestamp time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stamp := timestamp.UTC().Format(timestampLayout)
	var attemptMessage any
	if result.Result == "success" {
		if _, err := tx.ExecContext(ctx, "INSERT INTO price_observations (item_id, amount_cents, currency, observed_at) VALUES (?, ?, 'EUR', ?)", id, result.AmountCents, stamp); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE items SET title = COALESCE(?, title), thumbnail_url = COALESCE(?, thumbnail_url) WHERE id = ?", result.Title, result.ThumbnailURL, id); err != nil {
			return err
		}
	} else {
		attemptMessage = result.Message
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO collection_attempts (item_id, result, attempted_at, message) VALUES (?, ?, ?, ?)", id, result.Result, stamp, attemptMessage); err != nil {
		return err
	}
	var platform string
	if err := tx.QueryRowContext(ctx, "SELECT platform FROM items WHERE id = ?", id).Scan(&platform); err != nil {
		return err
	}
	if platform == "amazon" {
		usedStatus := result.SecondHandStatus
		if usedStatus != "available" && usedStatus != "not_found" && usedStatus != "check_error" {
			usedStatus = "check_error"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO second_hand_offer_checks (item_id, status, checked_at) VALUES (?, ?, ?)
ON CONFLICT(item_id) DO UPDATE SET status = excluded.status, checked_at = excluded.checked_at`, id, usedStatus, stamp); err != nil {
			return err
		}
		if usedStatus == "available" {
			if _, err := tx.ExecContext(ctx, `INSERT INTO second_hand_offer_observations
(item_id, amount_cents, currency, condition, condition_label, observed_at) VALUES (?, ?, 'EUR', ?, ?, ?)`, id,
				result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel, stamp); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) RecordFailure(ctx context.Context, id, result, message string, timestamp time.Time) error {
	return s.RecordCollection(ctx, id, model.CollectionResult{Result: result, Message: message, SecondHandStatus: "check_error"}, timestamp)
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
