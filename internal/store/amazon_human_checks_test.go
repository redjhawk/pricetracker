package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

// One-time restart of the Amazon items stopped before issue #168.

var issue168Stop = time.Date(2026, time.November, 20, 7, 26, 0, 0, time.UTC)

// dropRestartMarker removes the one-time restart marker of issue #168, as in a database written before it.
func dropRestartMarker(t *testing.T, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	must(t, err)
	defer db.Close()
	var exists bool
	must(t, db.QueryRow("SELECT EXISTS (SELECT 1 FROM pragma_table_info('amazon_request_state') WHERE name = 'human_checks_restarted')").Scan(&exists))
	if exists {
		_, err := db.Exec("ALTER TABLE amazon_request_state DROP COLUMN human_checks_restarted")
		must(t, err)
	}
}

func insertOwnedItem(t *testing.T, database *Store, owner int64, id, platform string) {
	t.Helper()
	listing := model.Listing{ID: id, Platform: platform, ListingID: id, Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/x/" + id, OwnerID: owner}
	if platform == "amazon" {
		listing = model.Listing{ID: id, Platform: platform, ListingID: id, ASIN: id, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + id, OwnerID: owner}
	}
	must(t, database.Insert(context.Background(), listing, listing.URL, searchTime))
}

var legacyAmazonIDs = []string{"B000000001", "B000000002", "B000000003"}

// legacyDatabase writes in dir a database as left before issue #168, with the Amazon items of
// owners (legacyAmazonIDs) and a LeBoncoin item (123), each with a price history and attempts. With
// stop, the Amazon items are stopped on 20 Nov 2026 07:26 UTC; the failure count is failures.
// It returns the items as they were saved.
func legacyDatabase(t *testing.T, dir string, stop bool, failures int, owners ...int64) map[string]model.Item {
	t.Helper()
	database := openTestStore(t, dir)
	ctx := context.Background()
	if len(owners) == 0 {
		owners = []int64{0}
	}
	for index, id := range legacyAmazonIDs {
		insertOwnedItem(t, database, owners[index%len(owners)], id, "amazon")
		must(t, database.RecordCollection(ctx, id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "available",
			SecondHandAmountCents: 1500, SecondHandCondition: "good", SecondHandConditionLabel: "Bon état"}, searchTime.Add(time.Hour)))
		must(t, database.RecordCollection(ctx, id, model.CollectionResult{Result: "request_error", Message: "Amazon could not be reached for a price check.", SecondHandStatus: "check_error"},
			issue168Stop.Add(-time.Minute)))
	}
	insertPlatformItem(t, database, "123", "leboncoin")
	must(t, database.RecordCollection(ctx, "123", model.CollectionResult{Result: "success", AmountCents: 4500}, searchTime.Add(time.Hour)))
	if stop {
		must(t, database.StopAmazonItems(ctx, issue168Stop))
	}
	must(t, database.SaveAmazonFailures(ctx, failures))
	saved := snapshot(t, database, append([]string{"123"}, legacyAmazonIDs...)...)
	database.Close()
	dropRestartMarker(t, dir)
	return saved
}

func snapshot(t *testing.T, database *Store, ids ...string) map[string]model.Item {
	t.Helper()
	result := map[string]model.Item{}
	for _, id := range ids {
		item, err := database.GetWithHistory(context.Background(), id)
		must(t, err)
		result[id] = item
	}
	return result
}

func failureCount(t *testing.T, database *Store) int {
	t.Helper()
	state, err := database.AmazonRequestState(context.Background())
	must(t, err)
	return state.ConsecutiveFailures
}

// TC-025 (data; the schedule part is in internal/service)
func TestAmazonHumanChecks_TC025_StoppedItemsAreRestartedOnFirstStart(t *testing.T) {
	dir := t.TempDir()
	before := legacyDatabase(t, dir, true, 5)
	for _, id := range legacyAmazonIDs {
		if before[id].AmazonStoppedAt == nil || !before[id].AmazonStoppedAt.Equal(issue168Stop) {
			t.Fatalf("legacy item %s is not stopped", id)
		}
	}

	upgraded := openTestStore(t, dir)
	if got := failureCount(t, upgraded); got != 0 {
		t.Fatalf("failure count after the restart: %d", got)
	}
	after := snapshot(t, upgraded, append([]string{"123"}, legacyAmazonIDs...)...)
	for _, id := range legacyAmazonIDs {
		item := after[id]
		if item.AmazonStoppedAt != nil {
			t.Errorf("item %s still stopped since %s", id, item.AmazonStoppedAt)
		}
		if !reflect.DeepEqual(item.PriceHistory, before[id].PriceHistory) || !reflect.DeepEqual(item.LastAttempt, before[id].LastAttempt) ||
			!reflect.DeepEqual(item.SecondHandOffer, before[id].SecondHandOffer) {
			t.Errorf("item %s history changed:\nbefore %+v %+v\nafter  %+v %+v", id, before[id].PriceHistory, before[id].LastAttempt, item.PriceHistory, item.LastAttempt)
		}
	}
	if !reflect.DeepEqual(after["123"], before["123"]) {
		t.Errorf("the LeBoncoin item changed:\nbefore %+v\nafter  %+v", before["123"], after["123"])
	}
}

// TC-026
func TestAmazonHumanChecks_TC026_TheRestartRunsOnlyOnce(t *testing.T) {
	dir := t.TempDir()
	legacyDatabase(t, dir, true, 5)
	ids := legacyAmazonIDs
	ctx := context.Background()
	database := openTestStore(t, dir) // first start: the restart runs
	if stoppedAt(t, database, ids[0]) != nil {
		t.Fatal("the stopped items were not restarted on the first start")
	}
	stoppedAgain := time.Date(2026, time.November, 25, 7, 40, 0, 0, time.UTC)
	must(t, database.StopAmazonItems(ctx, stoppedAgain))
	must(t, database.SaveAmazonFailures(ctx, 2))
	database.Close()

	for start := 2; start <= 3; start++ {
		reopened := openTestStore(t, dir)
		for _, id := range ids {
			if at := stoppedAt(t, reopened, id); at == nil || !at.Equal(stoppedAgain) {
				t.Fatalf("start %d: item %s stop %v, want the stop of %s kept", start, id, at, stoppedAgain)
			}
		}
		if got := failureCount(t, reopened); got != 2 {
			t.Fatalf("start %d: failure count %d, want 2 kept", start, got)
		}
		reopened.Close()
	}
}

// TC-027
func TestAmazonHumanChecks_TC027_NoStoppedItemStillRecordsTheRestart(t *testing.T) {
	ctx := context.Background()
	t.Run("no stopped item", func(t *testing.T) {
		dir := t.TempDir()
		legacyDatabase(t, dir, false, 3)
		ids := legacyAmazonIDs
		database := openTestStore(t, dir)
		for _, id := range ids {
			item, err := database.Get(ctx, id)
			must(t, err)
			if item.AmazonStoppedAt != nil || item.NextCheckAt == nil || !item.NextCheckAt.Equal(searchTime) {
				t.Fatalf("item %s changed by the first start: stopped %v, next check %v", id, item.AmazonStoppedAt, item.NextCheckAt)
			}
		}
		if got := failureCount(t, database); got != 3 {
			t.Fatalf("failure count %d, want 3 unchanged", got)
		}
		must(t, database.StopAmazonItems(ctx, issue168Stop.Add(time.Hour)))
		database.Close()
		if reopened := openTestStore(t, dir); stoppedAt(t, reopened, ids[0]) == nil {
			t.Fatal("a stop after the change was cleared by a later start")
		}
	})

	t.Run("no item", func(t *testing.T) {
		dir := t.TempDir()
		database := openTestStore(t, dir)
		database.Close()
		dropRestartMarker(t, dir)
		database = openTestStore(t, dir)
		insertPlatformItem(t, database, "B000000009", "amazon")
		must(t, database.StopAmazonItems(ctx, issue168Stop.Add(time.Hour)))
		database.Close()
		if reopened := openTestStore(t, dir); stoppedAt(t, reopened, "B000000009") == nil {
			t.Fatal("a stop after the change was cleared by a later start")
		}
	})

	t.Run("fresh database", func(t *testing.T) {
		dir := t.TempDir()
		database := openTestStore(t, dir)
		insertPlatformItem(t, database, "B000000009", "amazon")
		must(t, database.StopAmazonItems(ctx, issue168Stop.Add(time.Hour)))
		database.Close()
		for range 2 {
			reopened := openTestStore(t, dir)
			if stoppedAt(t, reopened, "B000000009") == nil {
				t.Fatal("a fresh database restarted a later stop")
			}
			reopened.Close()
		}
	})
}

// TC-028 (the schedule part is in internal/service)
func TestAmazonHumanChecks_TC028_EveryOwnersItemsAreRestarted(t *testing.T) {
	dir := t.TempDir()
	legacyDatabase(t, dir, true, 5, 1, 2)
	database := openTestStore(t, dir)
	owners := map[int64]bool{}
	for _, id := range legacyAmazonIDs {
		listing, err := database.Listing(context.Background(), id)
		must(t, err)
		owners[listing.OwnerID] = true
		if at := stoppedAt(t, database, id); at != nil {
			t.Errorf("item %s of owner %d still stopped", id, listing.OwnerID)
		}
	}
	if !owners[1] || !owners[2] {
		t.Fatalf("the fixture does not cover two owners: %v", owners)
	}
}
