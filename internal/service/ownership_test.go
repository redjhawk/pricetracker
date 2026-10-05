package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestUsersReachOnlyTheirOwnItems(t *testing.T) {
	service, database := newSessionService(t)
	ctx := context.Background()
	url := "https://www.leboncoin.fr/ad/x/123"
	insertListing(t, database, "open-item", "leboncoin", url)
	alice, err := service.CreateUser(ctx, "alice", "long enough pw")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := service.CreateUser(ctx, "bob", "long enough pw")
	if err != nil {
		t.Fatal(err)
	}
	aliceCtx := WithPrincipal(ctx, Principal{UserID: alice.ID, Role: "user"})
	bobCtx := WithPrincipal(ctx, Principal{UserID: bob.ID, Role: "user"})

	if items, err := service.List(aliceCtx); err != nil || len(items) != 1 || items[0].ID != "open-item" {
		t.Fatalf("alice items %+v %v", items, err)
	}
	if items, err := service.List(bobCtx); err != nil || len(items) != 0 {
		t.Fatalf("bob items %+v %v", items, err)
	}
	if _, err := service.Get(bobCtx, "open-item"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("bob get: %v", err)
	}
	if _, err := service.RefreshItem(bobCtx, "open-item"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("bob refresh: %v", err)
	}
	if _, _, _, err := service.SetPurchaseGoal(bobCtx, "open-item", "x"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("bob goal: %v", err)
	}
	if _, _, err := service.RequestAIReview(bobCtx, "open-item"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("bob review: %v", err)
	}
	if deleted, err := service.Remove(bobCtx, "open-item"); err != nil || deleted {
		t.Fatalf("bob delete %v %v", deleted, err)
	}
	if _, count, err := service.RefreshAll(bobCtx); err != nil || count != 0 {
		t.Fatalf("bob refresh all %d %v", count, err)
	}
	if exists, err := database.IsCanonicalTracked(ctx, bob.ID, url); err != nil || exists {
		t.Fatalf("bob duplicate check %v %v", exists, err)
	}
	if exists, _ := database.IsCanonicalTracked(ctx, alice.ID, url); !exists {
		t.Fatal("alice duplicate check")
	}
	if _, err := service.Get(aliceCtx, "open-item"); err != nil {
		t.Fatalf("alice get: %v", err)
	}
}
