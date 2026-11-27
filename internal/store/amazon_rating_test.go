package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

// Customer rating storage and Amazon reviews of issue #305.

var ratingCheckAt = time.Date(2026, time.November, 2, 8, 0, 0, 0, time.UTC)

func ratingOf(tenths, count *int64) *model.CustomerRating {
	return &model.CustomerRating{StarsTenths: tenths, Count: count}
}

func number(value int64) *int64 { return &value }

func assertStoredRating(t *testing.T, database *Store, id, step string, tenths, count *int64) {
	t.Helper()
	stored, err := database.AmazonCustomerRating(context.Background(), id)
	if err != nil {
		t.Fatalf("%s: %v", step, err)
	}
	same := func(got, want *int64) bool {
		return got == nil && want == nil || got != nil && want != nil && *got == *want
	}
	if !same(stored.StarsTenths, tenths) || !same(stored.Count, count) {
		t.Fatalf("%s: rating %v/%v, want %v/%v", step, deref(stored.StarsTenths), deref(stored.Count), deref(tenths), deref(count))
	}
}

func deref(value *int64) any {
	if value == nil {
		return "absent"
	}
	return *value
}

func amazonSuccess(cents int64, rating *model.CustomerRating) model.CollectionResult {
	return model.CollectionResult{Result: "success", AmountCents: cents, SecondHandStatus: "not_found", CustomerRating: rating}
}

func TestAmazonRating_TC013_RecordCollectionKeepsOrReplacesTheRating(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	insertPlatformItem(t, database, "amz", "amazon")
	insertPlatformItem(t, database, "lbc", "leboncoin")
	assertStoredRating(t, database, "amz", "never checked", nil, nil)

	must(t, database.RecordCollection(ctx, "amz", amazonSuccess(11999, ratingOf(number(45), number(1234))), ratingCheckAt))
	assertStoredRating(t, database, "amz", "product page with a rating", number(45), number(1234))
	must(t, database.RecordFailure(ctx, "amz", "request_error", "Amazon could not be reached for a price check.", ratingCheckAt.Add(time.Hour)))
	assertStoredRating(t, database, "amz", "failed check", number(45), number(1234))
	must(t, database.RecordCollection(ctx, "amz", model.CollectionResult{Result: "unavailable", Message: "gone", SecondHandStatus: "check_error"}, ratingCheckAt.Add(2*time.Hour)))
	assertStoredRating(t, database, "amz", "answer without a product page", number(45), number(1234))
	must(t, database.RecordCollection(ctx, "amz", amazonSuccess(11999, ratingOf(nil, nil)), ratingCheckAt.Add(3*time.Hour)))
	assertStoredRating(t, database, "amz", "product page without a rating", nil, nil)
	must(t, database.RecordCollection(ctx, "amz", model.CollectionResult{Result: "price_not_found", Message: "no price", SecondHandStatus: "check_error",
		CustomerRating: ratingOf(number(50), nil)}, ratingCheckAt.Add(4*time.Hour)))
	assertStoredRating(t, database, "amz", "price not found with a rating only", number(50), nil)
	must(t, database.RecordCollection(ctx, "amz", amazonSuccess(11999, ratingOf(nil, number(7))), ratingCheckAt.Add(5*time.Hour)))
	assertStoredRating(t, database, "amz", "count only", nil, number(7))

	// LeBoncoin items have no customer rating.
	must(t, database.RecordCollection(ctx, "lbc", model.CollectionResult{Result: "success", AmountCents: 1200, CustomerRating: ratingOf(number(45), number(1234))}, ratingCheckAt))
	assertStoredRating(t, database, "lbc", "LeBoncoin item", nil, nil)

	// The rating is never part of an item.
	item, err := database.GetWithHistory(ctx, "amz")
	must(t, err)
	items, err := database.List(ctx, 0)
	must(t, err)
	for what, value := range map[string]any{"item": item, "list": items} {
		encoded, _ := json.Marshal(value)
		if strings.Contains(strings.ToLower(string(encoded)), "rating") {
			t.Fatalf("the rating is returned in the %s: %s", what, encoded)
		}
	}
}

func TestAmazonRating_TC015_UpgradeAddsAnUnknownRating(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	database := openTestStore(t, dir)
	insertPlatformItem(t, database, "amz", "amazon")
	must(t, database.RecordCollection(ctx, "amz", amazonSuccess(11999, nil), ratingCheckAt))
	insertPlatformItem(t, database, "lbc", "leboncoin")
	_, err := database.SetPurchaseGoal(ctx, "lbc", "Light distro")
	must(t, err)
	good := "good"
	price := int64(1200)
	reviewID, err := database.StartAIReview(ctx, "lbc", ratingCheckAt)
	must(t, err)
	must(t, database.FinishAIReview(ctx, reviewID, "succeeded", &price, &model.AIReviewContent{Price: model.AIRating{Rating: "fair", Explanation: "ok"},
		Condition: model.AIConditionRating{Rating: &good, Explanation: "ok"}}, "", ratingCheckAt))
	database.Close()

	// The database as written before issue #305: no rating columns.
	legacy, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	must(t, err)
	for _, column := range []string{"amazon_rating_tenths", "amazon_rating_count"} {
		var exists bool
		must(t, legacy.QueryRow("SELECT EXISTS (SELECT 1 FROM pragma_table_info('items') WHERE name = ?)", column).Scan(&exists))
		if exists {
			_, err := legacy.Exec("ALTER TABLE items DROP COLUMN " + column)
			must(t, err)
		}
	}
	legacy.Close()

	for _, step := range []string{"first open after the upgrade", "second open"} {
		upgraded := openTestStore(t, dir)
		for _, column := range []string{"amazon_rating_tenths", "amazon_rating_count"} {
			if exists, err := upgraded.hasColumn(ctx, "items", column); err != nil || !exists {
				t.Fatalf("%s: column %s missing (%v)", step, column, err)
			}
		}
		assertStoredRating(t, upgraded, "amz", step, nil, nil)
		amazon, err := upgraded.Listing(ctx, "amz")
		must(t, err)
		if amazon.PurchaseGoal != "" {
			t.Fatalf("%s: Amazon goal %q", step, amazon.PurchaseGoal)
		}
		if latest, history, err := upgraded.AIReviews(ctx, "amz"); err != nil || latest != nil || len(history) != 0 {
			t.Fatalf("%s: Amazon reviews %+v %v", step, latest, err)
		}
		lbc, err := upgraded.Listing(ctx, "lbc")
		must(t, err)
		latest, history, err := upgraded.AIReviews(ctx, "lbc")
		if err != nil || lbc.PurchaseGoal != "Light distro" || latest == nil || latest.ID != reviewID || len(history) != 1 {
			t.Fatalf("%s: LeBoncoin item changed: %q %+v %v", step, lbc.PurchaseGoal, latest, err)
		}
		if item, err := upgraded.Get(ctx, "amz"); err != nil || item.LatestPrice == nil || item.LatestPrice.AmountCents != 11999 {
			t.Fatalf("%s: Amazon price %+v %v", step, item.LatestPrice, err)
		}
		upgraded.Close()
	}

	reopened := openTestStore(t, dir)
	must(t, reopened.RecordCollection(ctx, "amz", amazonSuccess(11999, ratingOf(number(45), number(1234))), ratingCheckAt.Add(time.Hour)))
	reopened.Close()
	assertStoredRating(t, openTestStore(t, dir), "amz", "reopened after a rating", number(45), number(1234))
}

func TestAmazonRating_TC015_ColumnsRejectInvalidValues(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	insertPlatformItem(t, database, "amz", "amazon")
	for _, statement := range []string{
		"UPDATE items SET amazon_rating_tenths = 51 WHERE id = 'amz'",
		"UPDATE items SET amazon_rating_tenths = -1 WHERE id = 'amz'",
		"UPDATE items SET amazon_rating_count = -1 WHERE id = 'amz'",
	} {
		if _, err := database.db.Exec(statement); err == nil {
			t.Errorf("accepted: %s", statement)
		}
	}
	for _, statement := range []string{
		"UPDATE items SET amazon_rating_tenths = 0, amazon_rating_count = 0 WHERE id = 'amz'",
		"UPDATE items SET amazon_rating_tenths = 50, amazon_rating_count = 123456 WHERE id = 'amz'",
	} {
		if _, err := database.db.Exec(statement); err != nil {
			t.Errorf("rejected %s: %v", statement, err)
		}
	}
}

// TC-007, TC-016: Amazon reviews are stored in ai_reviews with their own shape and deleted with the item.
func TestAmazonAIReviews_TC007_DecodedByPlatform(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	insertPlatformItem(t, database, "amz", "amazon")
	insertPlatformItem(t, database, "lbc", "leboncoin")
	content := &model.AmazonAIReviewContent{
		Price:          model.AIRating{Rating: "good_deal", Explanation: "Below the usual price."},
		FairPrice:      model.AmazonFairPrice{MinCents: 11000, MaxCents: 13000},
		Recommendation: model.AIRating{Rating: "wait", Explanation: "The price often drops."},
	}
	price := int64(11999)
	reviewID, err := database.StartAIReview(ctx, "amz", ratingCheckAt)
	must(t, err)
	must(t, database.FinishAIReview(ctx, reviewID, "succeeded", &price, content, "", ratingCheckAt.Add(time.Minute)))
	good := "good"
	lbcPrice := int64(1200)
	lbcID, err := database.StartAIReview(ctx, "lbc", ratingCheckAt)
	must(t, err)
	must(t, database.FinishAIReview(ctx, lbcID, "succeeded", &lbcPrice, &model.AIReviewContent{Price: model.AIRating{Rating: "fair", Explanation: "ok"},
		Condition: model.AIConditionRating{Rating: &good, Explanation: "ok"}}, "", ratingCheckAt))

	latest, history, err := database.AIReviews(ctx, "amz")
	if err != nil || latest == nil || len(history) != 1 || *latest.PriceCents != 11999 {
		t.Fatalf("Amazon reviews %+v %v", latest, err)
	}
	stored, ok := history[0].Review.(*model.AmazonAIReviewContent)
	if !ok || *stored != *content {
		t.Fatalf("Amazon review decoded as %T %+v", history[0].Review, history[0].Review)
	}
	encoded, _ := json.Marshal(history[0])
	if strings.Contains(string(encoded), "condition") || strings.Contains(string(encoded), "suggestedOfferCents") || strings.Contains(string(encoded), "risks") {
		t.Fatalf("Amazon review has LeBoncoin parts: %s", encoded)
	}
	if _, history, _ := database.AIReviews(ctx, "lbc"); len(history) != 1 {
		t.Fatal("LeBoncoin review missing")
	} else if _, ok := history[0].Review.(*model.AIReviewContent); !ok {
		t.Fatalf("LeBoncoin review decoded as %T", history[0].Review)
	}

	if found, err := database.Delete(ctx, 0, "amz"); err != nil || !found {
		t.Fatalf("delete: %v %v", found, err)
	}
	var count int
	must(t, database.db.QueryRow("SELECT (SELECT COUNT(*) FROM ai_reviews WHERE item_id = 'amz') + (SELECT COUNT(*) FROM items WHERE id = 'amz')").Scan(&count))
	if count != 0 {
		t.Fatalf("%d rows of the deleted Amazon item kept", count)
	}
}
