package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

func TestOldPriceHistoryOrderingAndCascade(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	insertTestItem(t, database, "undated")
	insertTestItem(t, database, "dated")
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	oldPrice := int64(32000)
	if err := database.RecordCollection(ctx, "undated", model.CollectionResult{Result: "success", AmountCents: 30000, OldPriceCents: &oldPrice}, now); err != nil {
		t.Fatal(err)
	}
	if err := database.RecordCollection(ctx, "undated", model.CollectionResult{Result: "success", AmountCents: 29000}, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	item, err := database.GetWithHistory(ctx, "undated")
	if err != nil {
		t.Fatal(err)
	}
	if len(item.PriceHistory) != 3 || item.PriceHistory[2].AmountCents != 32000 || !item.PriceHistory[2].OldPrice || item.PriceHistory[2].Timestamp != nil ||
		item.PriceHistory[0].OldPrice || item.LatestPrice.AmountCents != 29000 || item.LatestPrice.OldPrice || len(item.LastThreeDetections) != 3 {
		t.Fatalf("unexpected undated history %+v", item)
	}
	encoded, _ := json.Marshal(item.PriceHistory[2])
	if !strings.Contains(string(encoded), `"timestamp":null,"oldPrice":true`) {
		t.Fatalf("unexpected JSON %s", encoded)
	}

	oldAt := now.Add(-48 * time.Hour)
	if err := database.RecordCollection(ctx, "dated", model.CollectionResult{Result: "success", AmountCents: 30000, OldPriceCents: &oldPrice, OldPriceAt: &oldAt}, now); err != nil {
		t.Fatal(err)
	}
	item, err = database.GetWithHistory(ctx, "dated")
	if err != nil {
		t.Fatal(err)
	}
	if len(item.PriceHistory) != 2 || !item.PriceHistory[1].OldPrice || !item.PriceHistory[1].Timestamp.Equal(oldAt) || item.LatestPrice.OldPrice {
		t.Fatalf("unexpected dated history %+v", item.PriceHistory)
	}

	if _, err := database.Delete(ctx, 0, "undated"); err != nil {
		t.Fatal(err)
	}
	var count int
	database.db.QueryRow("SELECT COUNT(*) FROM price_observations WHERE item_id = 'undated'").Scan(&count)
	if count != 0 {
		t.Fatalf("%d observations remain after delete", count)
	}
}

func TestOldPriceColumnMigration(t *testing.T) {
	dir := t.TempDir()
	database := openTestStore(t, dir)
	insertTestItem(t, database, "a")
	if _, err := database.db.Exec("ALTER TABLE price_observations DROP COLUMN is_old_price"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.db.Exec("INSERT INTO price_observations (item_id, amount_cents, observed_at) VALUES ('a', 100, '2026-10-01T00:00:00Z')"); err != nil {
		t.Fatal(err)
	}
	database.Close()
	openTestStore(t, dir).Close()
	reopened := openTestStore(t, dir)
	item, err := reopened.GetWithHistory(context.Background(), "a")
	if err != nil || len(item.PriceHistory) != 1 || item.PriceHistory[0].OldPrice || item.PriceHistory[0].Timestamp == nil {
		t.Fatalf("unexpected migrated history %+v %v", item.PriceHistory, err)
	}
}
