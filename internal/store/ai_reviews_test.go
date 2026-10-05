package store

import (
	"context"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

func insertTestItem(t *testing.T, database *Store, id string) {
	t.Helper()
	listing := model.Listing{ID: id, Platform: "leboncoin", ListingID: "123", Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/x/" + id}
	if err := database.Insert(context.Background(), listing, listing.URL, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestAIReviewsHistoryRestartAndCascade(t *testing.T) {
	dir := t.TempDir()
	database := openTestStore(t, dir)
	ctx := context.Background()
	insertTestItem(t, database, "a")
	base := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	if latest, history, err := database.AIReviews(ctx, "a"); err != nil || latest != nil || history == nil || len(history) != 0 {
		t.Fatalf("empty reviews: %v %v %v", latest, history, err)
	}
	price := int64(1500)
	good := "good"
	content := &model.AIReviewContent{Price: model.AIRating{Rating: "fair", Explanation: "ok"}, Condition: model.AIConditionRating{Rating: &good, Explanation: "ok"}}
	first, _ := database.StartAIReview(ctx, "a", base)
	if err := database.FinishAIReview(ctx, first, "succeeded", &price, content, "", base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	second, _ := database.StartAIReview(ctx, "a", base.Add(time.Hour))
	database.FinishAIReview(ctx, second, "succeeded", &price, content, "", base.Add(time.Hour))
	failed, _ := database.StartAIReview(ctx, "a", base.Add(2*time.Hour))
	database.FinishAIReview(ctx, failed, "failed", nil, nil, "Claude could not be reached. Try again later.", base.Add(2*time.Hour))
	latest, history, err := database.AIReviews(ctx, "a")
	if err != nil || latest.ID != failed || latest.Status != "failed" || latest.Review != nil || *latest.ErrorMessage == "" {
		t.Fatalf("latest %+v %v", latest, err)
	}
	if len(history) != 2 || history[0].ID != second || history[1].ID != first || *history[0].PriceCents != 1500 ||
		*history[0].Review.Condition.Rating != "good" || history[0].ErrorMessage != nil || !history[1].CompletedAt.Equal(base.Add(time.Minute)) {
		t.Fatalf("history %+v", history)
	}

	pending, _ := database.StartAIReview(ctx, "a", base.Add(3*time.Hour))
	database.Close()
	reopened := openTestStore(t, dir)
	latest, _, err = reopened.AIReviews(ctx, "a")
	if err != nil || latest.ID != pending || latest.Status != "failed" || *latest.ErrorMessage != InterruptedReviewMessage || latest.CompletedAt == nil {
		t.Fatalf("pending review not failed at startup: %+v %v", latest, err)
	}
	if _, err := reopened.Delete(ctx, 0, "a"); err != nil {
		t.Fatal(err)
	}
	var count int
	reopened.db.QueryRow("SELECT COUNT(*) FROM ai_reviews").Scan(&count)
	if count != 0 {
		t.Fatalf("%d reviews left after item deletion", count)
	}
	if err := reopened.FinishAIReview(ctx, pending, "succeeded", &price, content, "", base); err != nil {
		t.Fatalf("finishing a review of a deleted item: %v", err)
	}
}

func TestLatestPriceAndPriceHistory(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	insertTestItem(t, database, "a")
	if latest, err := database.LatestPriceCents(ctx, "a"); err != nil || latest != nil {
		t.Fatalf("latest without prices: %v %v", latest, err)
	}
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for index, amount := range []int64{300, 200, 100} {
		if err := database.RecordCollection(ctx, "a", model.CollectionResult{Result: "success", AmountCents: amount}, base.Add(time.Duration(index)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if latest, err := database.LatestPriceCents(ctx, "a"); err != nil || *latest != 100 {
		t.Fatalf("latest %v %v", latest, err)
	}
	history, err := database.PriceHistory(ctx, "a", 2)
	if err != nil || len(history) != 2 || history[0].AmountCents != 200 || history[1].AmountCents != 100 {
		t.Fatalf("history %+v %v", history, err)
	}
}
