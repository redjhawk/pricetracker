package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

var searchTime = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)

// searchFixture opens a store with tracked item tracked-B000000001 and searches s1 and s2 of owner 0.
func searchFixture(t *testing.T) (*Store, context.Context) {
	t.Helper()
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	url := "https://www.amazon.fr/dp/B000000001"
	listing := model.Listing{ID: "tracked-B000000001", Platform: "amazon", ListingID: "B000000001", ASIN: "B000000001", Marketplace: "amazon.fr", URL: url}
	must(t, database.Insert(ctx, listing, url, searchTime))
	for _, id := range []string{"s1", "s2"} {
		must(t, database.InsertSearch(ctx, Search{ID: id, URL: "https://www.amazon.fr/s?k=" + id, AddedAt: searchTime, NextRunAt: searchTime}))
	}
	return database, ctx
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func products(numbers ...int) []CapturedItem {
	items := make([]CapturedItem, 0, len(numbers))
	for _, number := range numbers {
		asin := fmt.Sprintf("B%09d", number)
		title := "Product " + asin
		price := int64(1000 * number)
		url := "https://www.amazon.fr/dp/" + asin
		items = append(items, CapturedItem{ID: "item-" + asin, ASIN: asin, Marketplace: "amazon.fr", CanonicalURL: url, URL: url, Title: &title, PriceCents: &price})
	}
	return items
}

func TestCaptureSearchKeepsOrderSharesItemsAndIsFrozen(t *testing.T) {
	database, ctx := searchFixture(t)
	numbers := make([]int, 0, 30)
	for number := 1; number <= 30; number++ {
		numbers = append(numbers, number)
	}
	must(t, database.CaptureSearch(ctx, "s1", 0, products(numbers...), searchTime))
	if err := database.CaptureSearch(ctx, "s1", 0, products(1), searchTime); !errors.Is(err, ErrSearchCaptured) {
		t.Fatalf("second capture error = %v", err)
	}
	must(t, database.CaptureSearch(ctx, "s2", 0, products(2, 99), searchTime))
	ids, err := database.SearchItemIDs(ctx, "s1")
	if err != nil || len(ids) != 30 || ids[0] != "tracked-B000000001" || ids[1] != "item-B000000002" || ids[29] != "item-B000000030" {
		t.Fatalf("unexpected s1 items %v %v", ids, err)
	}
	if ids, err := database.SearchItemIDs(ctx, "s2"); err != nil || len(ids) != 2 || ids[0] != "item-B000000002" {
		t.Fatalf("unexpected s2 items %v %v", ids, err)
	}
	shared, err := database.Get(ctx, "item-B000000002")
	if err != nil || shared.Tracked || shared.Status != "pending" || shared.LatestPrice.AmountCents != 2000 || len(shared.LastThreeDetections) != 1 || shared.NextCheckAt != nil {
		t.Fatalf("unexpected shared item %+v %v", shared, err)
	}
	if tracked, err := database.Get(ctx, "tracked-B000000001"); err != nil || !tracked.Tracked || tracked.LatestPrice != nil {
		t.Fatalf("tracked item changed by capture %+v %v", tracked, err)
	}
	if search, err := database.Search(ctx, 0, "s1"); err != nil || search.CapturedAt == nil || search.ItemCount != 30 {
		t.Fatalf("unexpected search %+v %v", search, err)
	}
	if _, err := database.Search(ctx, 1, "s1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("other owner sees search: %v", err)
	}
}

func TestTrackedQueriesIgnoreSearchItemsUntilTracked(t *testing.T) {
	database, ctx := searchFixture(t)
	must(t, database.CaptureSearch(ctx, "s1", 0, products(2), searchTime))
	if items, err := database.List(ctx, 0); err != nil || len(items) != 1 || items[0].ID != "tracked-B000000001" {
		t.Fatalf("unexpected list %+v %v", items, err)
	}
	if ids, err := database.IDs(ctx, 0); err != nil || len(ids) != 1 {
		t.Fatalf("unexpected ids %v %v", ids, err)
	}
	if due, err := database.DueIDs(ctx, searchTime.Add(time.Hour)); err != nil || len(due) != 1 || due[0] != "tracked-B000000001" {
		t.Fatalf("unexpected due ids %v %v", due, err)
	}
	if exists, err := database.IsCanonicalTracked(ctx, 0, "https://www.amazon.fr/dp/B000000002"); err != nil || exists {
		t.Fatalf("search item reported tracked: %v %v", exists, err)
	}
	if moved, err := database.TrackItem(ctx, 0, "item-B000000002", searchTime); err != nil || !moved {
		t.Fatalf("track item: %v %v", moved, err)
	}
	if moved, err := database.TrackItem(ctx, 0, "item-B000000002", searchTime); err != nil || moved {
		t.Fatalf("second track: %v %v", moved, err)
	}
	if exists, err := database.IsCanonicalTracked(ctx, 0, "https://www.amazon.fr/dp/B000000002"); err != nil || !exists {
		t.Fatalf("moved item not tracked: %v %v", exists, err)
	}
}

func TestDeleteTrackedItemStillInSearchOnlyUntracks(t *testing.T) {
	database, ctx := searchFixture(t)
	must(t, database.CaptureSearch(ctx, "s1", 0, products(1, 2), searchTime))
	must(t, ignoreResult(database.TrackItem(ctx, 0, "item-B000000002", searchTime)))
	must(t, ignoreResult(database.DeleteSearch(ctx, 0, "s1")))
	// tracked-B000000001 was tracked before the search: deleting the search keeps it, deleting it now removes it.
	if deleted, err := database.Delete(ctx, 0, "tracked-B000000001"); err != nil || !deleted {
		t.Fatalf("delete plain item: %v %v", deleted, err)
	}
	must(t, database.CaptureSearch(ctx, "s2", 0, products(2), searchTime))
	if deleted, err := database.Delete(ctx, 0, "item-B000000002"); err != nil || !deleted {
		t.Fatalf("delete: %v %v", deleted, err)
	}
	if item, err := database.Get(ctx, "item-B000000002"); err != nil || item.Tracked || item.NextCheckAt != nil {
		t.Fatalf("item not kept untracked %+v %v", item, err)
	}
	if deleted, err := database.Delete(ctx, 0, "item-B000000002"); err != nil || deleted {
		t.Fatalf("untracked item deleted: %v %v", deleted, err)
	}
}

func ignoreResult(_ bool, err error) error { return err }

func TestDeleteSearchRemovesOnlyOrphanUntrackedItems(t *testing.T) {
	database, ctx := searchFixture(t)
	must(t, database.CaptureSearch(ctx, "s1", 0, products(1, 2, 3), searchTime))
	must(t, database.CaptureSearch(ctx, "s2", 0, products(3), searchTime))
	if deleted, err := database.DeleteSearch(ctx, 1, "s1"); err != nil || deleted {
		t.Fatalf("other owner deleted search: %v %v", deleted, err)
	}
	if deleted, err := database.DeleteSearch(ctx, 0, "s1"); err != nil || !deleted {
		t.Fatalf("delete search: %v %v", deleted, err)
	}
	for id, kept := range map[string]bool{"tracked-B000000001": true, "item-B000000002": false, "item-B000000003": true} {
		if _, err := database.Get(ctx, id); (err == nil) != kept {
			t.Fatalf("item %s kept=%v: %v", id, kept, err)
		}
	}
	if searches, err := database.Searches(ctx, 0); err != nil || len(searches) != 1 || searches[0].ID != "s2" {
		t.Fatalf("unexpected searches %+v %v", searches, err)
	}
}

func TestSearchProgressDueAndErrors(t *testing.T) {
	database, ctx := searchFixture(t)
	must(t, database.SetSearchProgress(ctx, "s2", 0, searchTime.Add(2*time.Hour)))
	must(t, database.SetSearchProgress(ctx, "s1", 4, searchTime))
	must(t, database.SetSearchError(ctx, "s1", searchTime, "HTTP 503"))
	due, err := database.DueSearches(ctx, searchTime.Add(time.Hour))
	if err != nil || len(due) != 1 || due[0].ID != "s1" || due[0].PassPosition != 4 || *due[0].LastErrorMessage != "HTTP 503" {
		t.Fatalf("unexpected due searches %+v %v", due, err)
	}
	must(t, database.ClearSearchError(ctx, "s1"))
	if search, err := database.Search(ctx, 0, "s1"); err != nil || search.LastErrorAt != nil || search.LastErrorMessage != nil {
		t.Fatalf("error not cleared %+v %v", search, err)
	}
	if err := database.InsertSearch(ctx, Search{ID: "dup", URL: due[0].URL, AddedAt: searchTime, NextRunAt: searchTime}); !IsUniqueConstraint(err) {
		t.Fatalf("duplicate URL error = %v", err)
	}
}

func TestUpgradeAddsTrackedColumnKeepingItemsTracked(t *testing.T) {
	dir := t.TempDir()
	legacy, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	must(t, err)
	_, err = legacy.Exec(`CREATE TABLE items (id TEXT PRIMARY KEY, asin TEXT NOT NULL, platform TEXT NOT NULL DEFAULT 'amazon', listing_id TEXT NOT NULL DEFAULT '', marketplace TEXT NOT NULL, canonical_url TEXT NOT NULL, url TEXT NOT NULL, title TEXT, thumbnail_url TEXT, next_check_at TEXT, added_at TEXT NOT NULL, purchase_goal TEXT NOT NULL DEFAULT '', owner_id INTEGER NOT NULL DEFAULT 0, UNIQUE (owner_id, canonical_url));
INSERT INTO items (id, asin, marketplace, canonical_url, url, added_at) VALUES ('kept', 'B012345678', 'amazon.fr', 'https://www.amazon.fr/dp/B012345678', 'https://www.amazon.fr/dp/B012345678', '2026-01-01T00:00:00.000Z');`)
	must(t, err)
	legacy.Close()
	items, err := openTestStore(t, dir).List(context.Background(), 0)
	if err != nil || len(items) != 1 || !items[0].Tracked {
		t.Fatalf("unexpected upgraded items %+v %v", items, err)
	}
}
