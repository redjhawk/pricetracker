package service

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Synthetic secrets of the issue #265 test cases; never real tokens or cookies.
const (
	maskingTokenT   = "sk-ant-oat01-synthetic-token-wxyz"
	maskingTokenU   = "sk-ant-oat01-synthetic-new-uvst"
	maskingSessionS = "synthetic-session-value-9f3a"
	maskingSessionN = "synthetic-renewed-or-new-n0p1"
)

// savedSecrets stores T (with its rejected-token warning) and S (with a rejected
// attempt) through the store and returns the stored state.
func savedSecrets(t *testing.T, database *store.Store) (model.LeboncoinSession, model.ClaudeToken) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	session, token := maskingSessionS, maskingTokenT
	if _, _, err := database.SaveSettings(ctx, 0, &store.SessionChange{Value: &session}, &store.TokenChange{Value: &token}, now); err != nil {
		t.Fatal(err)
	}
	stored, _ := database.ClaudeToken(ctx, 0)
	if err := database.MarkClaudeTokenRejected(ctx, 0, stored.Revision, now); err != nil {
		t.Fatal(err)
	}
	current, _ := database.LeboncoinSession(ctx, 0)
	if _, err := database.FinishLeboncoinSessionAttempt(ctx, 0, current.Revision, store.LeboncoinSessionOutcome{Attempt: "rejected"}, now); err != nil {
		t.Fatal(err)
	}
	return storedMaskingSecrets(t, database)
}

func storedMaskingSecrets(t *testing.T, database *store.Store) (model.LeboncoinSession, model.ClaudeToken) {
	t.Helper()
	session, err := database.LeboncoinSession(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	token, err := database.ClaudeToken(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	return session, token
}

func assertMaskingSecretsUnchanged(t *testing.T, database *store.Store, session model.LeboncoinSession, token model.ClaudeToken) {
	t.Helper()
	gotSession, gotToken := storedMaskingSecrets(t, database)
	if !reflect.DeepEqual(gotSession, session) || !reflect.DeepEqual(gotToken, token) {
		t.Fatalf("stored secrets changed:\n got %+v %+v\nwant %+v %+v", gotSession, gotToken, session, token)
	}
}

// TC-006
func TestSecretMasking_TC006_SameValuesAreReplacements(t *testing.T) {
	service, database := newSessionService(t)
	fake := &fakeClaude{}
	service.claude = fake
	beforeSession, beforeToken := savedSecrets(t, database)
	if beforeToken.LastRejectedAt == nil || beforeSession.LastAttempt == nil {
		t.Fatalf("fixture warnings missing: %+v %+v", beforeSession, beforeToken)
	}

	session, token, err := service.SaveSettings(context.Background(), SettingsInput{
		LeboncoinSession: &SessionInput{Value: maskingSessionS, Revision: beforeSession.Revision},
		ClaudeToken:      &TokenInput{Value: maskingTokenT},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fake.verified, []string{maskingTokenT}) {
		t.Fatalf("verified %q, want T once", fake.verified)
	}
	if token.Value == nil || *token.Value != maskingTokenT || token.LastRejectedAt != nil || token.Revision != beforeToken.Revision+1 {
		t.Fatalf("token not replaced: %+v", token)
	}
	if session.Value == nil || *session.Value != maskingSessionS || session.Revision != beforeSession.Revision+1 || session.LastAttempt != nil || session.Status != "active" {
		t.Fatalf("session not replaced: %+v", session)
	}
}

// TC-006
func TestSecretMasking_TC006_TokenTrimmedAndInvalidTokensSaveNothing(t *testing.T) {
	service, database := newSessionService(t)
	fake := &fakeClaude{}
	service.claude = fake
	ctx := context.Background()
	beforeSession, beforeToken := savedSecrets(t, database)
	for _, raw := range []string{"sk ant", "sk-ant\noat01", "sk-ant-oat01-é", strings.Repeat("a", 1025)} {
		_, _, err := service.SaveSettings(ctx, SettingsInput{
			LeboncoinSession: &SessionInput{Value: "datadome=" + maskingSessionN, Revision: beforeSession.Revision},
			ClaudeToken:      &TokenInput{Value: raw},
		})
		assertServiceError(t, err, 400, "INVALID_CLAUDE_TOKEN")
		assertMaskingSecretsUnchanged(t, database, beforeSession, beforeToken)
	}
	if len(fake.verified) != 0 {
		t.Fatalf("invalid tokens verified: %q", fake.verified)
	}
	_, token, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: &TokenInput{Value: "  " + maskingTokenU + "  "}})
	if err != nil || token.Value == nil || *token.Value != maskingTokenU || !reflect.DeepEqual(fake.verified, []string{maskingTokenU}) {
		t.Fatalf("token not trimmed: %+v %v %q", token, err, fake.verified)
	}
}

// TC-006
func TestSecretMasking_TC006_RefusedTokenSavesNeitherEntry(t *testing.T) {
	for _, verifyErr := range []error{claude.ErrRejected, claude.ErrUnreachable} {
		service, database := newSessionService(t)
		service.claude = &fakeClaude{verifyErr: verifyErr}
		beforeSession, beforeToken := savedSecrets(t, database)
		_, _, err := service.SaveSettings(context.Background(), SettingsInput{
			LeboncoinSession: &SessionInput{Value: "datadome=" + maskingSessionN, Revision: beforeSession.Revision},
			ClaudeToken:      &TokenInput{Value: maskingTokenU},
		})
		if err == nil {
			t.Fatalf("%v: save accepted", verifyErr)
		}
		assertMaskingSecretsUnchanged(t, database, beforeSession, beforeToken)
	}
}

// TC-006
func TestSecretMasking_TC006_CookieStringsAndRevokedSession(t *testing.T) {
	service, database := newSessionService(t)
	service.claude = &fakeClaude{}
	ctx := context.Background()
	beforeSession, beforeToken := savedSecrets(t, database)
	_, _, err := service.SaveSettings(ctx, SettingsInput{LeboncoinSession: &SessionInput{Value: "a=1; b=2", Revision: beforeSession.Revision}})
	assertServiceError(t, err, 400, "INVALID_SESSION")
	assertMaskingSecretsUnchanged(t, database, beforeSession, beforeToken)

	if _, err := database.FinishLeboncoinSessionAttempt(ctx, 0, beforeSession.Revision, store.LeboncoinSessionOutcome{Attempt: "rejected", Revoked: true}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	revoked, _ := storedMaskingSecrets(t, database)
	if revoked.Status != "revoked" {
		t.Fatalf("fixture status %q", revoked.Status)
	}
	session, token, err := service.SaveSettings(ctx, SettingsInput{LeboncoinSession: &SessionInput{Value: "Cookie: a=1; datadome=" + maskingSessionN + "; b=2", Revision: revoked.Revision}})
	if err != nil {
		t.Fatal(err)
	}
	if session.Value == nil || *session.Value != maskingSessionN || session.Status != "active" || session.RevokedAt != nil || session.Revision != revoked.Revision+1 {
		t.Fatalf("revoked session not replaced with N: %+v", session)
	}
	if !reflect.DeepEqual(token, beforeToken) {
		t.Fatalf("token changed by a session-only save: %+v, was %+v", token, beforeToken)
	}
}

// TC-007 (service rules, also TC-004/TC-005): untouched entries are neither parsed, checked
// against the revision nor verified; a removal is never verified.
func TestSecretMasking_TC007_UntouchedAndRemovedEntriesAreNotVerified(t *testing.T) {
	service, database := newSessionService(t)
	fake := &fakeClaude{verifyErr: claude.ErrUnreachable}
	service.claude = fake
	ctx := context.Background()
	beforeSession, beforeToken := savedSecrets(t, database)
	for _, input := range []SettingsInput{
		{},
		{LeboncoinSession: &SessionInput{Value: "  ", Revision: beforeSession.Revision + 5}, ClaudeToken: &TokenInput{Value: " \t "}},
		{ClaudeToken: &TokenInput{Value: ""}},
	} {
		session, token, err := service.SaveSettings(ctx, input)
		if err != nil {
			t.Fatalf("untouched save %+v: %v", input, err)
		}
		if !reflect.DeepEqual(session, beforeSession) || !reflect.DeepEqual(token, beforeToken) {
			t.Fatalf("untouched save %+v returned %+v %+v", input, session, token)
		}
		assertMaskingSecretsUnchanged(t, database, beforeSession, beforeToken)
	}
	session, token, err := service.SaveSettings(ctx, SettingsInput{ClaudeToken: &TokenInput{Remove: true}})
	if err != nil || token.Value != nil || !reflect.DeepEqual(session, beforeSession) {
		t.Fatalf("token removal: %+v %+v %v", session, token, err)
	}
	_, _, err = service.SaveSettings(ctx, SettingsInput{LeboncoinSession: &SessionInput{Remove: true, Revision: beforeSession.Revision + 1}})
	assertServiceError(t, err, 409, "SESSION_CHANGED")
	session, _, err = service.SaveSettings(ctx, SettingsInput{LeboncoinSession: &SessionInput{Remove: true, Revision: beforeSession.Revision}})
	if err != nil || session.Value != nil || session.Status != "none" {
		t.Fatalf("session removal: %+v %v", session, err)
	}
	if len(fake.verified) != 0 {
		t.Fatalf("verification calls %q", fake.verified)
	}
}
