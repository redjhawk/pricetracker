package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"pricefollower.local/config"
)

func openTestStore(t *testing.T, dir string) *Store {
	t.Helper()
	database, err := Open(config.Config{DataDirectory: dir, StaleAfter: 36 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func stringPointer(value string) *string { return &value }

func TestLeboncoinSessionFreshDatabaseHasNoSession(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	session, err := database.LeboncoinSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.Value != nil || session.Revision != 0 || session.UpdatedAt != nil || session.Status != "none" || session.ExpiresAt != nil || session.RevokedAt != nil || session.LastAttempt != nil {
		t.Fatalf("unexpected fresh session %+v", session)
	}
}

func TestLeboncoinSessionMigratesExistingDatabaseAndPersists(t *testing.T) {
	dir := t.TempDir()
	// A database created by the previous schema (no leboncoin_session table) keeps its items.
	legacy, err := sql.Open("sqlite", filepath.Join(dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`CREATE TABLE items (id TEXT PRIMARY KEY, asin TEXT NOT NULL, platform TEXT NOT NULL DEFAULT 'amazon', listing_id TEXT NOT NULL DEFAULT '', marketplace TEXT NOT NULL, canonical_url TEXT NOT NULL UNIQUE, url TEXT NOT NULL, title TEXT, thumbnail_url TEXT, next_check_at TEXT, added_at TEXT NOT NULL);
INSERT INTO items (id, asin, marketplace, canonical_url, url, added_at) VALUES ('kept', 'B012345678', 'amazon.fr', 'https://www.amazon.fr/dp/B012345678', 'https://www.amazon.fr/dp/B012345678', '2026-01-01T00:00:00.000Z');`); err != nil {
		t.Fatal(err)
	}
	legacy.Close()
	database, err := Open(config.Config{DataDirectory: dir, StaleAfter: 36 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Get(context.Background(), "kept"); err != nil {
		t.Fatal("existing item lost", err)
	}
	saved, err := database.SaveLeboncoinSession(context.Background(), stringPointer("synthetic-value"), 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	database.Close()
	reopened := openTestStore(t, dir)
	session, err := reopened.LeboncoinSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.Value == nil || *session.Value != "synthetic-value" || session.Revision != saved.Revision || session.Status != "active" {
		t.Fatalf("session not persisted: %+v", session)
	}
}

func TestSaveLeboncoinSession(t *testing.T) {
	ctx := context.Background()
	database := openTestStore(t, t.TempDir())
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	saved, err := database.SaveLeboncoinSession(ctx, stringPointer("first"), 0, now)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Value == nil || *saved.Value != "first" || saved.Revision != 1 || saved.Status != "active" || saved.UpdatedAt == nil || !saved.UpdatedAt.Equal(now) {
		t.Fatalf("save %+v", saved)
	}
	if _, err := database.SaveLeboncoinSession(ctx, stringPointer("stale"), 0, now); !errors.Is(err, ErrSessionChanged) {
		t.Fatalf("stale revision accepted: %v", err)
	}
	// Saving the same value again is a normal save that clears attempt hints.
	if _, err := database.FinishLeboncoinSessionAttempt(ctx, 1, LeboncoinSessionOutcome{Attempt: "rejected"}, now); err != nil {
		t.Fatal(err)
	}
	again, err := database.SaveLeboncoinSession(ctx, stringPointer("first"), 1, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if again.Revision != 2 || again.LastAttempt != nil {
		t.Fatalf("resave %+v", again)
	}
	cleared, err := database.SaveLeboncoinSession(ctx, nil, 2, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Value != nil || cleared.Status != "none" || cleared.Revision != 3 {
		t.Fatalf("clear %+v", cleared)
	}
	unchanged, err := database.SaveLeboncoinSession(ctx, nil, 3, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Revision != 3 || unchanged.UpdatedAt == nil || !unchanged.UpdatedAt.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("no-op clear changed the row: %+v", unchanged)
	}
	if _, err := database.SaveLeboncoinSession(ctx, nil, 2, now); !errors.Is(err, ErrSessionChanged) {
		t.Fatalf("stale clear accepted: %v", err)
	}
}

func TestLeboncoinSessionStatusDerivation(t *testing.T) {
	ctx := context.Background()
	database := openTestStore(t, t.TempDir())
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	if _, err := database.SaveLeboncoinSession(ctx, stringPointer("value"), 0, now); err != nil {
		t.Fatal(err)
	}
	expiry := now.Add(time.Hour)
	applied, err := database.FinishLeboncoinSessionAttempt(ctx, 1, LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: "renewed", ExpiresAt: &expiry}, now)
	if err != nil || !applied {
		t.Fatal(applied, err)
	}
	before, err := database.leboncoinSessionAt(ctx, expiry.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if before.Status != "active" || before.ExpiresAt == nil || !before.ExpiresAt.Equal(expiry) {
		t.Fatalf("before expiry %+v", before)
	}
	after, err := database.leboncoinSessionAt(ctx, expiry)
	if err != nil {
		t.Fatal(err)
	}
	// Natural expiry is not a stored change and does not alter the revision.
	if after.Status != "expired" || after.Revision != before.Revision {
		t.Fatalf("at expiry %+v", after)
	}
	applied, err = database.FinishLeboncoinSessionAttempt(ctx, before.Revision, LeboncoinSessionOutcome{Attempt: "rejected", Revoked: true}, now.Add(time.Minute))
	if err != nil || !applied {
		t.Fatal(applied, err)
	}
	revoked, err := database.leboncoinSessionAt(ctx, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Status != "revoked" || revoked.RevokedAt == nil || revoked.Value == nil || *revoked.Value != "renewed" || revoked.Revision != before.Revision+1 {
		t.Fatalf("revoked %+v", revoked)
	}
	if revoked.LastAttempt == nil || revoked.LastAttempt.Outcome != "rejected" || !revoked.LastAttempt.AttemptedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("last attempt %+v", revoked.LastAttempt)
	}
	resaved, err := database.SaveLeboncoinSession(ctx, stringPointer("fresh"), revoked.Revision, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if resaved.Status != "active" || resaved.RevokedAt != nil || resaved.ExpiresAt != nil || resaved.LastAttempt != nil {
		t.Fatalf("resave after revocation %+v", resaved)
	}
}

func TestFinishLeboncoinSessionAttempt(t *testing.T) {
	ctx := context.Background()
	database := openTestStore(t, t.TempDir())
	now := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	if _, err := database.SaveLeboncoinSession(ctx, stringPointer("value"), 0, now); err != nil {
		t.Fatal(err)
	}
	// An outcome alone does not change the revision.
	applied, err := database.FinishLeboncoinSessionAttempt(ctx, 1, LeboncoinSessionOutcome{Attempt: "failed"}, now)
	if err != nil || !applied {
		t.Fatal(applied, err)
	}
	session, _ := database.LeboncoinSession(ctx)
	if session.Revision != 1 || session.LastAttempt == nil || session.LastAttempt.Outcome != "failed" {
		t.Fatalf("outcome only %+v", session)
	}
	// A renewal with the same value only updates the expiry.
	expiry := now.Add(24 * time.Hour)
	if _, err := database.FinishLeboncoinSessionAttempt(ctx, 1, LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: "value", ExpiresAt: &expiry}, now); err != nil {
		t.Fatal(err)
	}
	session, _ = database.LeboncoinSession(ctx)
	if session.Revision != 1 || session.ExpiresAt == nil || !session.ExpiresAt.Equal(expiry) || session.LastAttempt.Outcome != "accepted" {
		t.Fatalf("expiry-only renewal %+v", session)
	}
	// A changed value increments the revision.
	if _, err := database.FinishLeboncoinSessionAttempt(ctx, 1, LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: "renewed"}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	session, _ = database.LeboncoinSession(ctx)
	if session.Revision != 2 || *session.Value != "renewed" || session.ExpiresAt != nil || !session.UpdatedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("renewal %+v", session)
	}
	// An attempt started at an older revision changes nothing.
	applied, err = database.FinishLeboncoinSessionAttempt(ctx, 1, LeboncoinSessionOutcome{Attempt: "rejected", Revoked: true}, now.Add(2*time.Minute))
	if err != nil || applied {
		t.Fatal("stale attempt applied", err)
	}
	stale, _ := database.LeboncoinSession(ctx)
	if stale.Revision != 2 || stale.Status != "active" || stale.LastAttempt.Outcome != "accepted" {
		t.Fatalf("stale attempt changed row %+v", stale)
	}
	// An invalid renewed value is rejected by the schema and leaves the row intact.
	if _, err := database.FinishLeboncoinSessionAttempt(ctx, 2, LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: ""}, now); err == nil {
		t.Fatal("empty renewal stored")
	}
	intact, _ := database.LeboncoinSession(ctx)
	if *intact.Value != "renewed" || intact.Revision != 2 {
		t.Fatalf("row damaged %+v", intact)
	}
}
