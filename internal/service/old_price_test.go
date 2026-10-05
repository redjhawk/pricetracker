package service

import (
	"context"
	"testing"

	"pricefollower.local/internal/leboncoin"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

func oldPriceCount(t *testing.T, database *store.Store, id string) int {
	t.Helper()
	item, err := database.GetWithHistory(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, observation := range item.PriceHistory {
		if observation.OldPrice {
			count++
		}
	}
	return count
}

func TestOldPriceRecordedOnlyOnAdd(t *testing.T) {
	service, database := newSessionService(t)
	result := "success"
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		oldPrice := int64(1500)
		return model.CollectionResult{Result: result, AmountCents: 1200, OldPriceCents: &oldPrice}, leboncoin.SessionOutcome{}
	})
	item, err := service.Add(context.Background(), "https://www.leboncoin.fr/ad/test/123", "")
	if err != nil {
		t.Fatal(err)
	}
	waitReviews(t, service, item.ID, true)
	service.Collect(context.Background(), item.ID)
	waitReviews(t, service, item.ID, true)
	if count := oldPriceCount(t, database, item.ID); count != 1 {
		t.Fatalf("got %d old prices after add and refresh, want 1", count)
	}

	insertListing(t, database, "existing", "leboncoin", "https://www.leboncoin.fr/ad/test/456")
	service.Collect(context.Background(), "existing")
	waitReviews(t, service, "existing", true)
	if count := oldPriceCount(t, database, "existing"); count != 0 {
		t.Fatalf("scheduled collection recorded %d old prices", count)
	}
}

func TestOldPriceNotRecordedWhenFirstCollectionFails(t *testing.T) {
	service, database := newSessionService(t)
	result := "request_error"
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		oldPrice := int64(1500)
		return model.CollectionResult{Result: result, AmountCents: 1200, OldPriceCents: &oldPrice}, leboncoin.SessionOutcome{}
	})
	item, err := service.Add(context.Background(), "https://www.leboncoin.fr/ad/test/123", "")
	if err != nil {
		t.Fatal(err)
	}
	waitReviews(t, service, item.ID, true)
	result = "success"
	service.Collect(context.Background(), item.ID)
	waitReviews(t, service, item.ID, true)
	if count := oldPriceCount(t, database, item.ID); count != 0 {
		t.Fatalf("got %d old prices after a failed first collection", count)
	}
}
