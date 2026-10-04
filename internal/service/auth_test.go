package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func init() { passwordIterations = 1000 } // keep tests fast; the stored format carries the cost

func errorCode(err error) string {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return ""
}

func TestAdminPasswordResetEndsSessionsAndReplacesPassword(t *testing.T) {
	service, _ := newSessionService(t)
	ctx := context.Background()
	first, created, err := service.ResetAdminPassword(ctx)
	if err != nil || !created || len(first) != 24 {
		t.Fatalf("create admin %q %v %v", first, created, err)
	}
	token, user, err := service.Login(ctx, "ADMIN", first)
	if err != nil || user.Role != "admin" {
		t.Fatalf("admin login %+v %v", user, err)
	}
	second, created, err := service.ResetAdminPassword(ctx)
	if err != nil || created || second == first {
		t.Fatalf("reset %v %v", created, err)
	}
	if _, ok, _ := service.SessionUser(ctx, token); ok {
		t.Fatal("admin session survived the reset")
	}
	if _, _, err := service.Login(ctx, "admin", first); errorCode(err) != "INVALID_CREDENTIALS" {
		t.Fatalf("old password: %v", err)
	}
	if _, _, err := service.Login(ctx, "admin", second); err != nil {
		t.Fatalf("new password: %v", err)
	}
}

func TestCreateUserValidation(t *testing.T) {
	service, _ := newSessionService(t)
	ctx := context.Background()
	for _, test := range []struct{ username, password, code string }{
		{"al", "long enough password", "INVALID_USERNAME"},
		{"al ice", "long enough password", "INVALID_USERNAME"},
		{"alice", "elevenchars", "PASSWORD_TOO_SHORT"},
		{"alice", string(make([]rune, 257)), "PASSWORD_TOO_LONG"},
	} {
		if _, err := service.CreateUser(ctx, test.username, test.password); errorCode(err) != test.code {
			t.Fatalf("%q/%d chars: %v, want %s", test.username, len(test.password), err, test.code)
		}
	}
	if protected, _ := service.ProtectedMode(ctx); protected {
		t.Fatal("invalid users started protected mode")
	}
	if _, err := service.CreateUser(ctx, "Alice", "twelve chars"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateUser(ctx, "alice", "twelve chars"); errorCode(err) != "USERNAME_TAKEN" {
		t.Fatalf("duplicate: %v", err)
	}
	if protected, _ := service.ProtectedMode(ctx); !protected {
		t.Fatal("first user did not start protected mode")
	}
}

func TestLoginLockoutAndSessionExpiry(t *testing.T) {
	service, _ := newSessionService(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	if _, err := service.CreateUser(ctx, "alice", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 5; attempt++ {
		if _, _, err := service.Login(ctx, "Alice", "wrong password"); errorCode(err) != "INVALID_CREDENTIALS" {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
	}
	if _, _, err := service.Login(ctx, "alice", "correct horse battery"); errorCode(err) != "LOGIN_LOCKED" {
		t.Fatalf("6th attempt: %v", err)
	}
	if _, _, err := service.Login(ctx, "nobody", "x"); errorCode(err) != "INVALID_CREDENTIALS" {
		t.Fatalf("other username: %v", err)
	}
	now = now.Add(time.Minute)
	token, user, err := service.Login(ctx, "alice", "correct horse battery")
	if err != nil || user.Username != "alice" {
		t.Fatalf("after the lock: %+v %v", user, err)
	}
	now = now.Add(SessionDuration - time.Second)
	if _, ok, err := service.SessionUser(ctx, token); !ok || err != nil {
		t.Fatalf("session before expiry: %v %v", ok, err)
	}
	now = now.Add(time.Second)
	if _, ok, _ := service.SessionUser(ctx, token); ok {
		t.Fatal("session valid after 30 days")
	}
	token, _, _ = service.Login(ctx, "alice", "correct horse battery")
	if err := service.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := service.SessionUser(ctx, token); ok {
		t.Fatal("session valid after logout")
	}
}

func TestFailureDuringLockKeepsItAndOldEntriesArePruned(t *testing.T) {
	service, _ := newSessionService(t)
	now := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	for attempt := 0; attempt < 5; attempt++ {
		service.recordLoginFailure("alice", now)
	}
	lockedUntil := service.loginFailures["alice"].lockedUntil
	// A request that passed the lock check before the lock was set fails afterwards.
	service.recordLoginFailure("alice", now.Add(10*time.Second))
	if entry := service.loginFailures["alice"]; !entry.lockedUntil.Equal(lockedUntil) {
		t.Fatalf("lock changed by a failure during the lock: %+v", entry)
	}
	service.now = func() time.Time { return now.Add(30 * time.Second) }
	if _, _, err := service.Login(context.Background(), "alice", "x"); errorCode(err) != "LOGIN_LOCKED" {
		t.Fatalf("still locked: %v", err)
	}
	service.recordLoginFailure("bob", now)
	service.recordLoginFailure("carol", now.Add(2*time.Hour))
	if _, kept := service.loginFailures["alice"]; kept {
		t.Fatal("expired lock not pruned")
	}
	if _, kept := service.loginFailures["bob"]; kept {
		t.Fatal("old failure not pruned")
	}
	if entry := service.loginFailures["carol"]; entry.count != 1 {
		t.Fatalf("carol %+v", entry)
	}
}
