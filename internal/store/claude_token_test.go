package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClaudeTokenFreshSaveReopenAndClear(t *testing.T) {
	dir := t.TempDir()
	database := openTestStore(t, dir)
	ctx := context.Background()
	token, err := database.ClaudeToken(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if token.Value != nil || token.UpdatedAt != nil || token.LastRejectedAt != nil || token.Revision != 0 {
		t.Fatalf("unexpected fresh token %+v", token)
	}
	now := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	_, saved, err := database.SaveSettings(ctx, 0, nil, &TokenChange{Value: stringPointer("sk-ant-oat01-a")}, now)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Value == nil || *saved.Value != "sk-ant-oat01-a" || saved.Revision != 1 || !saved.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected saved token %+v", saved)
	}
	database.Close()
	reopened := openTestStore(t, dir)
	token, err = reopened.ClaudeToken(ctx, 0)
	if err != nil || token.Value == nil || *token.Value != "sk-ant-oat01-a" {
		t.Fatalf("token not persisted: %+v %v", token, err)
	}
	_, cleared, err := reopened.SaveSettings(ctx, 0, nil, &TokenChange{}, now.Add(time.Hour))
	if err != nil || cleared.Value != nil || cleared.Revision != 2 {
		t.Fatalf("unexpected cleared token %+v %v", cleared, err)
	}
	_, again, err := reopened.SaveSettings(ctx, 0, nil, &TokenChange{}, now.Add(2*time.Hour))
	if err != nil || again.Revision != 2 || !again.UpdatedAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("clearing an empty token changed it: %+v %v", again, err)
	}
}

func TestSaveSettingsSessionConflictSavesNothing(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	_, _, err := database.SaveSettings(ctx, 0, &SessionChange{Value: stringPointer("datadome=abc"), ExpectedRevision: 5},
		&TokenChange{Value: stringPointer("sk-ant-oat01-a")}, time.Now())
	if !errors.Is(err, ErrSessionChanged) {
		t.Fatalf("got %v, want ErrSessionChanged", err)
	}
	token, _ := database.ClaudeToken(ctx, 0)
	session, _ := database.LeboncoinSession(ctx, 0)
	if token.Value != nil || session.Value != nil {
		t.Fatalf("partial save: token %+v session %+v", token, session)
	}
	saved, savedToken, err := database.SaveSettings(ctx, 0, &SessionChange{Value: stringPointer("datadome=abc")},
		&TokenChange{Value: stringPointer("sk-ant-oat01-a")}, time.Now())
	if err != nil || saved.Value == nil || saved.Revision != 1 || savedToken.Value == nil {
		t.Fatalf("both entries not saved: %+v %+v %v", saved, savedToken, err)
	}
}

func TestClaudeTokenRejectionFollowsRevision(t *testing.T) {
	database := openTestStore(t, t.TempDir())
	ctx := context.Background()
	now := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	_, token, err := database.SaveSettings(ctx, 0, nil, &TokenChange{Value: stringPointer("sk-ant-oat01-a")}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.MarkClaudeTokenRejected(ctx, 0, token.Revision+1, now); err != nil {
		t.Fatal(err)
	}
	if current, _ := database.ClaudeToken(ctx, 0); current.LastRejectedAt != nil {
		t.Fatal("stale revision marked the token rejected")
	}
	if err := database.MarkClaudeTokenRejected(ctx, 0, token.Revision, now); err != nil {
		t.Fatal(err)
	}
	if current, _ := database.ClaudeToken(ctx, 0); current.LastRejectedAt == nil || !current.LastRejectedAt.Equal(now) {
		t.Fatalf("rejection not recorded: %+v", current)
	}
	if err := database.ClearClaudeTokenRejected(ctx, 0, token.Revision); err != nil {
		t.Fatal(err)
	}
	if current, _ := database.ClaudeToken(ctx, 0); current.LastRejectedAt != nil {
		t.Fatal("rejection not cleared after a successful review")
	}
	database.MarkClaudeTokenRejected(ctx, 0, token.Revision, now)
	_, saved, err := database.SaveSettings(ctx, 0, nil, &TokenChange{Value: stringPointer("sk-ant-oat01-b")}, now)
	if err != nil || saved.LastRejectedAt != nil {
		t.Fatalf("save did not clear the rejection: %+v %v", saved, err)
	}
}
