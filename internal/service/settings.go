package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/leboncoin"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// claudeClient lets tests replace the Claude API client.
type claudeClient interface {
	Verify(ctx context.Context, token string) error
	Review(ctx context.Context, token string, input claude.ReviewInput) (model.AIReviewContent, error)
}

// SettingsInput holds the Settings entries to save; a nil part is left unchanged.
type SettingsInput struct {
	LeboncoinSession *SessionInput
	ClaudeToken      *TokenInput
}

// SessionInput replaces the LeBoncoin session with Value, or removes it, if the
// stored revision is still Revision. A blank Value leaves the session unchanged.
type SessionInput struct {
	Value    string
	Remove   bool
	Revision int64
}

// TokenInput replaces the Claude token with Value, or removes it. A blank Value
// leaves the token unchanged.
type TokenInput struct {
	Value  string
	Remove bool
}

// ClaudeToken returns the saved Claude token settings.
func (s *Service) ClaudeToken(ctx context.Context) (model.ClaudeToken, error) {
	return s.store.ClaudeToken(ctx, ownerFrom(ctx))
}

// SaveSettings validates the entries, verifies a replacement Claude token, then
// saves them in one transaction. Untouched entries are neither parsed, verified
// nor revision-checked. The first error wins and nothing is saved. It never
// starts a price check or an AI review.
func (s *Service) SaveSettings(ctx context.Context, input SettingsInput) (model.LeboncoinSession, model.ClaudeToken, error) {
	sessionChange, err := sessionChangeFor(input.LeboncoinSession)
	if err != nil {
		return model.LeboncoinSession{}, model.ClaudeToken{}, err
	}
	tokenChange, err := s.tokenChangeFor(ctx, input.ClaudeToken)
	if err != nil {
		return model.LeboncoinSession{}, model.ClaudeToken{}, err
	}
	session, token, err := s.store.SaveSettings(ctx, ownerFrom(ctx), sessionChange, tokenChange, time.Now().UTC())
	if errors.Is(err, store.ErrSessionChanged) {
		return model.LeboncoinSession{}, model.ClaudeToken{}, &Error{Status: 409, Code: "SESSION_CHANGED", Message: "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving."}
	}
	return session, token, err
}

// sessionChangeFor returns the session change to save, or nil when the session
// is untouched (no input or a blank value).
func sessionChangeFor(input *SessionInput) (*store.SessionChange, error) {
	if input == nil {
		return nil, nil
	}
	if input.Remove {
		return &store.SessionChange{ExpectedRevision: input.Revision}, nil
	}
	value, err := leboncoin.ParseSessionInput(input.Value)
	var inputError *leboncoin.SessionInputError
	if errors.As(err, &inputError) {
		return nil, &Error{Status: 400, Code: "INVALID_SESSION", Message: inputError.Message}
	}
	if err != nil || value == "" {
		return nil, err
	}
	return &store.SessionChange{Value: &value, ExpectedRevision: input.Revision}, nil
}

// tokenChangeFor returns the token change to save, or nil when the token is
// untouched. A replacement is always verified with Claude; a removal never is.
func (s *Service) tokenChangeFor(ctx context.Context, input *TokenInput) (*store.TokenChange, error) {
	if input == nil {
		return nil, nil
	}
	if input.Remove {
		return &store.TokenChange{}, nil
	}
	token := strings.TrimSpace(input.Value)
	if token == "" {
		return nil, nil
	}
	if err := validateClaudeToken(token); err != nil {
		return nil, err
	}
	if err := s.verifyClaudeToken(ctx, token); err != nil {
		return nil, err
	}
	return &store.TokenChange{Value: &token}, nil
}

func validateClaudeToken(token string) error {
	if len(token) > 1024 {
		return &Error{Status: 400, Code: "INVALID_CLAUDE_TOKEN", Message: "The Claude token is too long (maximum 1024 characters)."}
	}
	for index := 0; index < len(token); index++ {
		if token[index] <= 0x20 || token[index] >= 0x7F {
			return &Error{Status: 400, Code: "INVALID_CLAUDE_TOKEN", Message: "The Claude token must not contain spaces, line breaks or other control characters."}
		}
	}
	return nil
}

func (s *Service) verifyClaudeToken(ctx context.Context, token string) error {
	err := s.claude.Verify(ctx, token)
	if errors.Is(err, claude.ErrRejected) {
		return &Error{Status: 422, Code: "CLAUDE_TOKEN_REJECTED", Message: "Claude refused this token. Create a new one with claude setup-token."}
	}
	if err != nil {
		return &Error{Status: 502, Code: "CLAUDE_UNREACHABLE", Message: "The token could not be verified because Claude could not be reached. Try again."}
	}
	return nil
}
