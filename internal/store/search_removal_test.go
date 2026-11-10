package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pricefollower.local/internal/model"
)

func tableExists(t *testing.T, database *Store, name string) bool {
	t.Helper()
	var exists bool
	must(t, database.db.QueryRow("SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?)", name).Scan(&exists))
	return exists
}

func childRows(t *testing.T, database *Store, itemID string) int {
	t.Helper()
	var count int
	must(t, database.db.QueryRow(`SELECT (SELECT COUNT(*) FROM price_observations WHERE item_id = ?1)
  + (SELECT COUNT(*) FROM collection_attempts WHERE item_id = ?1) + (SELECT COUNT(*) FROM ai_reviews WHERE item_id = ?1)`, itemID).Scan(&count))
	return count
}

// An issue #56 database loses its searches, search-only items and Amazon reviews; tracked items stay.
func TestUpgradeRemovesAmazonSearches(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	database := openTestStore(t, dir)
	amazonItem := func(id, asin string) model.Listing {
		return model.Listing{ID: id, Platform: "amazon", ListingID: asin, ASIN: asin, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + asin}
	}
	lbc := model.Listing{ID: "lbc", Platform: "leboncoin", ListingID: "123", Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/x/123"}
	for _, listing := range []model.Listing{amazonItem("search-only", "B000000001"), amazonItem("tracked", "B000000002"), lbc} {
		must(t, database.Insert(ctx, listing, listing.URL, searchTime))
		must(t, database.RecordCollection(ctx, listing.ID, model.CollectionResult{Result: "success", AmountCents: 1000, SecondHandStatus: "check_error"}, searchTime))
		_, err := database.StartAIReview(ctx, listing.ID, searchTime)
		must(t, err)
	}
	database.Close()

	legacy, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	must(t, err)
	_, err = legacy.Exec(`CREATE TABLE amazon_searches (id TEXT PRIMARY KEY, owner_id INTEGER NOT NULL, url TEXT NOT NULL, added_at TEXT NOT NULL, captured_at TEXT, next_run_at TEXT NOT NULL, pass_position INTEGER NOT NULL DEFAULT 0, last_error_at TEXT, last_error_message TEXT, UNIQUE (owner_id, url));
CREATE TABLE amazon_search_items (search_id TEXT NOT NULL REFERENCES amazon_searches(id) ON DELETE CASCADE, item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE, position INTEGER NOT NULL, PRIMARY KEY (search_id, item_id), UNIQUE (search_id, position));
INSERT INTO amazon_searches (id, owner_id, url, added_at, next_run_at) VALUES ('s1', 0, 'https://www.amazon.fr/s?k=x', '2026-10-07T20:00:00.000Z', '2026-10-07T20:00:00.000Z');
INSERT INTO amazon_search_items (search_id, item_id, position) VALUES ('s1', 'search-only', 1), ('s1', 'tracked', 2);
UPDATE items SET tracked = 0, next_check_at = NULL WHERE id = 'search-only';`)
	must(t, err)
	legacy.Close()

	upgraded := openTestStore(t, dir)
	if tableExists(t, upgraded, "amazon_searches") || tableExists(t, upgraded, "amazon_search_items") {
		t.Fatal("search tables kept")
	}
	if _, err := upgraded.Get(ctx, "search-only"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("search-only item kept: %v", err)
	}
	if count := childRows(t, upgraded, "search-only"); count != 0 {
		t.Fatalf("%d rows of the search-only item kept", count)
	}
	tracked, err := upgraded.Get(ctx, "tracked")
	if err != nil || len(tracked.LastThreeDetections) != 1 || childRows(t, upgraded, "tracked") != 2 {
		t.Fatalf("tracked item %+v %v", tracked, err)
	}
	if latest, _, err := upgraded.AIReviews(ctx, "tracked"); err != nil || latest != nil {
		t.Fatalf("Amazon review kept: %+v %v", latest, err)
	}
	if latest, _, err := upgraded.AIReviews(ctx, "lbc"); err != nil || latest == nil {
		t.Fatalf("LeBoncoin review lost: %v", err)
	}
	upgraded.Close()
	if items, err := openTestStore(t, dir).List(ctx, 0); err != nil || len(items) != 2 {
		t.Fatalf("reopened items %d %v", len(items), err)
	}
}

func TestFreshDatabaseHasNoSearchTables(t *testing.T) {
	if database := openTestStore(t, t.TempDir()); tableExists(t, database, "amazon_searches") || tableExists(t, database, "amazon_search_items") {
		t.Fatal("search tables created")
	}
}
