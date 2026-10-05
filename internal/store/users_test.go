package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

func TestItemsMigrationKeepsDataAsOpenModeOwner(t *testing.T) {
	dir := t.TempDir()
	legacy, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`CREATE TABLE items (id TEXT PRIMARY KEY, asin TEXT NOT NULL, platform TEXT NOT NULL DEFAULT 'amazon', listing_id TEXT NOT NULL DEFAULT '', marketplace TEXT NOT NULL, canonical_url TEXT NOT NULL UNIQUE, url TEXT NOT NULL, title TEXT, thumbnail_url TEXT, next_check_at TEXT, added_at TEXT NOT NULL, purchase_goal TEXT NOT NULL DEFAULT '');
CREATE TABLE price_observations (id INTEGER PRIMARY KEY AUTOINCREMENT, item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE, amount_cents INTEGER NOT NULL, currency TEXT NOT NULL DEFAULT 'EUR', observed_at TEXT NOT NULL);
INSERT INTO items (id, asin, marketplace, canonical_url, url, added_at, purchase_goal) VALUES ('kept', 'B012345678', 'amazon.fr', 'https://www.amazon.fr/dp/B012345678', 'https://www.amazon.fr/dp/B012345678', '2026-01-01T00:00:00.000Z', 'goal');
INSERT INTO price_observations (item_id, amount_cents, observed_at) VALUES ('kept', 1999, '2026-01-01T00:00:00.000Z');`); err != nil {
		t.Fatal(err)
	}
	legacy.Close()
	database := openTestStore(t, dir)
	ctx := context.Background()
	listing, err := database.Listing(ctx, "kept")
	if err != nil || listing.OwnerID != 0 || listing.PurchaseGoal != "goal" {
		t.Fatalf("migrated listing %+v %v", listing, err)
	}
	item, err := database.Get(ctx, "kept")
	if err != nil || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
		t.Fatalf("price history lost: %+v %v", item, err)
	}
	// Deleting still cascades to child tables after the rebuild.
	if _, err := database.Delete(ctx, "kept"); err != nil {
		t.Fatal(err)
	}
	var children int
	if err := database.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM price_observations").Scan(&children); err != nil || children != 0 {
		t.Fatalf("children %d %v", children, err)
	}
}

func TestSameURLForTwoOwners(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	url := "https://www.leboncoin.fr/ad/x/1"
	for _, listing := range []model.Listing{{ID: "a", OwnerID: 1}, {ID: "b", OwnerID: 2}} {
		listing.Platform, listing.Marketplace, listing.URL = "leboncoin", "leboncoin.fr", url
		if err := database.Insert(ctx, listing, url, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	err := database.Insert(ctx, model.Listing{ID: "c", OwnerID: 1, Platform: "leboncoin", URL: url}, url, time.Now())
	if !IsUniqueConstraint(err) {
		t.Fatalf("duplicate for the same owner: %v", err)
	}
}

func TestCreateUserInheritsOpenModeItemsOnlyOnce(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	now := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	insertTestItem(t, database, "open")
	if protected, err := database.HasRegularUser(ctx); err != nil || protected {
		t.Fatalf("protected %v %v", protected, err)
	}
	if _, err := database.CreateAdminOrResetPassword(ctx, "hash", now); err != nil {
		t.Fatal(err)
	}
	if protected, _ := database.HasRegularUser(ctx); protected {
		t.Fatal("admin alone must not start protected mode")
	}
	first, err := database.CreateUser(ctx, "Alice", "hash-a", now)
	if err != nil {
		t.Fatal(err)
	}
	insertTestItem(t, database, "later-open")
	if _, err := database.CreateUser(ctx, "bob", "hash-b", now); err != nil {
		t.Fatal(err)
	}
	if listing, _ := database.Listing(ctx, "open"); listing.OwnerID != first.ID {
		t.Fatalf("first user did not inherit: %+v", listing)
	}
	if listing, _ := database.Listing(ctx, "later-open"); listing.OwnerID != 0 {
		t.Fatalf("second user inherited: %+v", listing)
	}
	if _, err := database.CreateUser(ctx, "ALICE", "x", now); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate username: %v", err)
	}
	if _, err := database.CreateUser(ctx, "Admin", "x", now); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("admin username: %v", err)
	}
	users, err := database.ListUsers(ctx)
	if err != nil || len(users) != 2 || users[0].Username != "Alice" || users[1].Username != "bob" {
		t.Fatalf("users %+v %v", users, err)
	}
	if user, err := database.UserByUsername(ctx, "alice"); err != nil || user.ID != first.ID || user.PasswordHash != "hash-a" {
		t.Fatalf("lookup %+v %v", user, err)
	}
}

func TestSessionsExpireAndAdminResetEndsThem(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	now := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	if created, err := database.CreateAdminOrResetPassword(ctx, "old", now); err != nil || !created {
		t.Fatalf("created %v %v", created, err)
	}
	admin, err := database.UserByUsername(ctx, "admin")
	if err != nil || admin.Role != "admin" {
		t.Fatalf("admin %+v %v", admin, err)
	}
	if err := database.CreateSession(ctx, "token", admin.ID, now, now.Add(30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if user, err := database.SessionUser(ctx, "token", now.Add(29*24*time.Hour)); err != nil || user.ID != admin.ID {
		t.Fatalf("session within 30 days %+v %v", user, err)
	}
	if _, err := database.SessionUser(ctx, "token", now.Add(31*24*time.Hour)); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("session after 31 days: %v", err)
	}
	if err := database.RecordLogin(ctx, admin.ID, now); err != nil {
		t.Fatal(err)
	}
	if created, err := database.CreateAdminOrResetPassword(ctx, "new", now); err != nil || created {
		t.Fatalf("reset %v %v", created, err)
	}
	if _, err := database.SessionUser(ctx, "token", now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("session survived the reset: %v", err)
	}
	admin, _ = database.UserByUsername(ctx, "admin")
	if admin.PasswordHash != "new" || admin.LastLoginAt == nil || !admin.LastLoginAt.Equal(now) {
		t.Fatalf("admin after reset %+v", admin)
	}
	database.CreateSession(ctx, "old", admin.ID, now, now.Add(time.Hour))
	if err := database.DeleteExpiredSessions(ctx, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	database.CreateSession(ctx, "kept", admin.ID, now, now.Add(2*time.Hour))
	database.DeleteSession(ctx, "kept")
	var count int
	database.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&count)
	if count != 0 {
		t.Fatalf("%d sessions left", count)
	}
}
