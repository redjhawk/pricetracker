package service

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestAddKeepsPurchaseGoalForLeboncoinOnly(t *testing.T) {
	service, _, _, _ := newReviewService(t, false)
	item, err := service.Add(context.Background(), "https://www.leboncoin.fr/ad/test/123", "  Light Linux distro \n")
	if err != nil {
		t.Fatal(err)
	}
	if item.PurchaseGoal != "Light Linux distro" {
		t.Fatalf("LeBoncoin goal %q", item.PurchaseGoal)
	}
	amazon, err := service.Add(context.Background(), "https://www.amazon.fr/dp/B012345678", "Ignored")
	if err != nil {
		t.Fatal(err)
	}
	if amazon.PurchaseGoal != "" {
		t.Fatalf("Amazon goal %q", amazon.PurchaseGoal)
	}
	waitReviews(t, service, item.ID, true)
}

func TestSetPurchaseGoalStartsReviewWithGoal(t *testing.T) {
	service, database, reviewer, _ := newReviewService(t, true)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	goal, changed, started, err := service.SetPurchaseGoal(context.Background(), "lbc", " Light distro ")
	if err != nil || goal != "Light distro" || !changed || !started {
		t.Fatalf("set: %q %v %v %v", goal, changed, started, err)
	}
	waitReviews(t, service, "lbc", false)
	if reviewer.calls() != 1 || reviewer.inputs[0].PurchaseGoal != "Light distro" {
		t.Fatalf("review inputs %+v", reviewer.inputs)
	}
	goal, changed, started, err = service.SetPurchaseGoal(context.Background(), "lbc", "Light distro  ")
	if err != nil || goal != "Light distro" || changed || started {
		t.Fatalf("unchanged: %q %v %v %v", goal, changed, started, err)
	}
	if _, changed, _, _ = service.SetPurchaseGoal(context.Background(), "lbc", ""); !changed {
		t.Fatal("clearing the goal is a change")
	}
	waitReviews(t, service, "lbc", false)
	if reviewer.calls() != 2 || reviewer.inputs[1].PurchaseGoal != "" {
		t.Fatalf("review after clearing %+v", reviewer.inputs)
	}
}

func TestSetPurchaseGoalWithoutTokenAndErrors(t *testing.T) {
	service, database, reviewer, _ := newReviewService(t, false)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	insertListing(t, database, "amz", "amazon", "https://www.amazon.fr/dp/B012345678")
	goal, changed, started, err := service.SetPurchaseGoal(context.Background(), "lbc", "Goal")
	if err != nil || goal != "Goal" || !changed || started || reviewer.calls() != 0 {
		t.Fatalf("no token: %q %v %v %v", goal, changed, started, err)
	}
	if item, _ := service.Get(context.Background(), "lbc"); item.PurchaseGoal != "Goal" {
		t.Fatalf("stored goal %q", item.PurchaseGoal)
	}
	_, _, _, err = service.SetPurchaseGoal(context.Background(), "amz", "Goal")
	assertServiceError(t, err, 422, "PURCHASE_GOAL_UNSUPPORTED")
	if _, _, _, err := service.SetPurchaseGoal(context.Background(), "missing", "Goal"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing item: %v", err)
	}
}

// REV-001: a reservation released without running still honours and clears the rerun flag.
func TestReleasedReservationRunsRequestedReview(t *testing.T) {
	service, database, reviewer, _ := newReviewService(t, true)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	if _, outcome, err := service.reserveReview("lbc"); err != nil || outcome != reviewStarted {
		t.Fatalf("reserve: %v %v", outcome, err)
	}
	if _, _, started, err := service.SetPurchaseGoal(context.Background(), "lbc", "Goal"); err != nil || !started {
		t.Fatalf("set: %v %v", started, err)
	}
	service.releaseReview("lbc") // failure path: the reserved review never ran
	waitReviews(t, service, "lbc", false)
	service.mu.Lock()
	flagged := service.reviewAgain["lbc"]
	service.mu.Unlock()
	if flagged || reviewer.calls() != 1 || reviewer.inputs[0].PurchaseGoal != "Goal" {
		t.Fatalf("flag %v, reviews %+v", flagged, reviewer.inputs)
	}
}

// REV-002: a review start failure after the goal is saved is not a save failure.
func TestReviewStartFailureAfterSaveIsNotAnError(t *testing.T) {
	service, database, _, _ := newReviewService(t, true)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	failReviewInserts(t, service)
	goal, changed, started, err := service.SetPurchaseGoal(context.Background(), "lbc", "Goal")
	if err != nil || goal != "Goal" || !changed || started {
		t.Fatalf("set: %q %v %v %v", goal, changed, started, err)
	}
	if item, _ := service.Get(context.Background(), "lbc"); item.PurchaseGoal != "Goal" {
		t.Fatalf("stored goal %q", item.PurchaseGoal)
	}
}

// failReviewInserts makes every new AI review row fail to save.
func failReviewInserts(t *testing.T, service *Service) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(service.config.DataDirectory, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TRIGGER fail_reviews BEFORE INSERT ON ai_reviews BEGIN SELECT RAISE(ABORT, 'storage failure'); END"); err != nil {
		t.Fatal(err)
	}
}

func TestGoalChangeDuringReviewRunsAnotherReview(t *testing.T) {
	service, database, reviewer, _ := newReviewService(t, true)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	reviewer.release = make(chan struct{})
	if _, running, err := service.RequestAIReview(context.Background(), "lbc"); err != nil || running {
		t.Fatalf("request: %v %v", running, err)
	}
	if _, _, started, err := service.SetPurchaseGoal(context.Background(), "lbc", "Newest goal"); err != nil || !started {
		t.Fatalf("set: %v %v", started, err)
	}
	close(reviewer.release)
	deadline := time.Now().Add(5 * time.Second)
	for reviewer.calls() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	waitReviews(t, service, "lbc", false)
	if reviewer.calls() != 2 || reviewer.inputs[1].PurchaseGoal != "Newest goal" {
		t.Fatalf("rerun inputs %+v", reviewer.inputs)
	}
}
