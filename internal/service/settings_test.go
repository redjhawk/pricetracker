package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pricefollower.local/internal/claude"
)

// fakeClaude records verified tokens and returns verifyErr.
type fakeClaude struct {
	verified  []string
	verifyErr error
}

func (f *fakeClaude) Verify(_ context.Context, token string) error {
	f.verified = append(f.verified, token)
	return f.verifyErr
}

func stringPointer(value string) *string { return &value }

func assertServiceError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var typed *Error
	if !errors.As(err, &typed) || typed.Status != status || typed.Code != code {
		t.Fatalf("got %v, want %d %s", err, status, code)
	}
}

func TestSaveSettingsTokenValidationVerificationAndClear(t *testing.T) {
	service, _ := newSessionService(t)
	fake := &fakeClaude{}
	service.claude = fake
	ctx := context.Background()
	for _, raw := range []string{"sk ant", "sk\nant", "sk-ant-é", strings.Repeat("a", 1025)} {
		_, _, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: stringPointer(raw)})
		assertServiceError(t, err, 400, "INVALID_CLAUDE_TOKEN")
	}
	_, token, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: stringPointer("  sk-ant-oat01-a \n")})
	if err != nil || token.Value == nil || *token.Value != "sk-ant-oat01-a" {
		t.Fatalf("token not saved trimmed: %+v %v", token, err)
	}
	if _, _, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: stringPointer("sk-ant-oat01-a")}); err != nil {
		t.Fatal(err)
	}
	if len(fake.verified) != 1 {
		t.Fatalf("unchanged token re-verified: %v", fake.verified)
	}
	_, token, err = service.SaveSettings(ctx, SettingsInput{ClaudeToken: stringPointer("  ")})
	if err != nil || token.Value != nil || len(fake.verified) != 1 {
		t.Fatalf("clear failed or verified: %+v %v %v", token, err, fake.verified)
	}
}

func TestSaveSettingsVerificationFailureSavesNothing(t *testing.T) {
	for _, test := range []struct {
		verifyErr error
		status    int
		code      string
	}{
		{claude.ErrRejected, 422, "CLAUDE_TOKEN_REJECTED"},
		{claude.ErrUsageLimit, 502, "CLAUDE_UNREACHABLE"},
		{claude.ErrUnreachable, 502, "CLAUDE_UNREACHABLE"},
	} {
		service, database := newSessionService(t)
		service.claude = &fakeClaude{verifyErr: test.verifyErr}
		_, _, err := service.SaveSettings(context.Background(), SettingsInput{
			LeboncoinSession: &SessionInput{Value: "datadome=abc", Revision: 0},
			ClaudeToken:      stringPointer("sk-ant-oat01-a"),
		})
		assertServiceError(t, err, test.status, test.code)
		session, _ := database.LeboncoinSession(context.Background(), 0)
		token, _ := database.ClaudeToken(context.Background(), 0)
		if session.Value != nil || token.Value != nil {
			t.Fatalf("partial save after %v: %+v %+v", test.verifyErr, session, token)
		}
	}
}

func TestSaveSettingsOrderAndSessionConflict(t *testing.T) {
	service, _ := newSessionService(t)
	fake := &fakeClaude{}
	service.claude = fake
	ctx := context.Background()
	_, _, err := service.SaveSettings(ctx, SettingsInput{
		LeboncoinSession: &SessionInput{Value: "no cookie here"},
		ClaudeToken:      stringPointer("sk ant"),
	})
	assertServiceError(t, err, 400, "INVALID_SESSION")
	_, _, err = service.SaveSettings(ctx, SettingsInput{
		LeboncoinSession: &SessionInput{Value: "datadome=abc", Revision: 3},
		ClaudeToken:      stringPointer("sk-ant-oat01-a"),
	})
	assertServiceError(t, err, 409, "SESSION_CHANGED")
	if token, _ := service.ClaudeToken(ctx); token.Value != nil {
		t.Fatal("token saved despite the session conflict")
	}
	session, token, err := service.SaveSettings(ctx, SettingsInput{
		LeboncoinSession: &SessionInput{Value: "datadome=abc", Revision: 0},
		ClaudeToken:      stringPointer("sk-ant-oat01-a"),
	})
	if err != nil || session.Value == nil || token.Value == nil {
		t.Fatalf("combined save failed: %+v %+v %v", session, token, err)
	}
}
