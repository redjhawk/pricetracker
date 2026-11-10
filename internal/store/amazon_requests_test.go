package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

var searchTime = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func insertPlatformItem(t *testing.T, database *Store, id, platform string) {
	t.Helper()
	listing := model.Listing{ID: id, Platform: platform, ListingID: id, Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/x/" + id}
	if platform == "amazon" {
		listing = model.Listing{ID: id, Platform: platform, ListingID: id, ASIN: id, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + id}
	}
	must(t, database.Insert(context.Background(), listing, listing.URL, searchTime))
}

func stoppedAt(t *testing.T, database *Store, id string) *time.Time {
	t.Helper()
	item, err := database.Get(context.Background(), id)
	must(t, err)
	return item.AmazonStoppedAt
}

func TestAmazonFailureCountPersists(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	if state, err := openTestStore(t, dir).AmazonRequestState(ctx); err != nil || state.ConsecutiveFailures != 0 {
		t.Fatalf("unexpected initial state %+v %v", state, err)
	}
	database := openTestStore(t, dir)
	must(t, database.SaveAmazonFailures(ctx, 3))
	database.Close()
	if state, err := openTestStore(t, dir).AmazonRequestState(ctx); err != nil || state.ConsecutiveFailures != 3 {
		t.Fatalf("unexpected reopened state %+v %v", state, err)
	}
}

func TestAmazonItemStopsPersistAndRestartOneByOne(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	database := openTestStore(t, dir)
	for _, id := range []string{"B000000001", "B000000002"} {
		insertPlatformItem(t, database, id, "amazon")
	}
	insertPlatformItem(t, database, "123", "leboncoin")
	must(t, database.StopAmazonItems(ctx, searchTime))
	database.Close()

	reopened := openTestStore(t, dir)
	if at := stoppedAt(t, reopened, "B000000001"); at == nil || !at.Equal(searchTime) || stoppedAt(t, reopened, "123") != nil {
		t.Fatalf("stops not persisted: %v", at)
	}
	if due, err := reopened.DueIDs(ctx, searchTime.Add(time.Hour)); err != nil || len(due) != 1 || due[0] != "123" {
		t.Fatalf("due items %v %v", due, err)
	}
	must(t, reopened.ClearAmazonStop(ctx, "B000000001"))
	first, err := reopened.AmazonStopped(ctx, "B000000001")
	must(t, err)
	second, err := reopened.AmazonStopped(ctx, "B000000002")
	must(t, err)
	if first || !second {
		t.Fatalf("restart of one item: first=%v second=%v", first, second)
	}
}

// The former global stop becomes a stop of every Amazon item with the former stop time (issue #111).
func TestUpgradeConvertsGlobalAmazonStop(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	database := openTestStore(t, dir)
	insertPlatformItem(t, database, "B000000001", "amazon")
	insertPlatformItem(t, database, "123", "leboncoin")
	database.Close()
	legacy, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	must(t, err)
	_, err = legacy.Exec("UPDATE amazon_request_state SET consecutive_failures = 5, blocked = 1, stopped_at = ?", searchTime.Format(timestampLayout))
	must(t, err)
	legacy.Close()

	for range 2 { // the conversion runs once
		upgraded := openTestStore(t, dir)
		if at := stoppedAt(t, upgraded, "B000000001"); at == nil || !at.Equal(searchTime) || stoppedAt(t, upgraded, "123") != nil {
			t.Fatalf("Amazon item stop after upgrade: %v", at)
		}
		if state, err := upgraded.AmazonRequestState(ctx); err != nil || state.ConsecutiveFailures != 0 {
			t.Fatalf("state after upgrade %+v %v", state, err)
		}
		upgraded.Close()
	}
}
