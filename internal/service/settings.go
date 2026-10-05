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
	ClaudeToken      *string
}

// SessionInput is the pasted LeBoncoin session and the revision shown in Settings.
type SessionInput struct {
	Value    string
	Revision int64
}

// ClaudeToken returns the saved Claude token settings.
func (s *Service) ClaudeToken(ctx context.Context) (model.ClaudeToken, error) {
	return s.store.ClaudeToken(ctx, ownerFrom(ctx))
}

// SaveSettings validates both entries, verifies a new Claude token, then saves
// both in one transaction. The first error wins and nothing is saved. It never
// starts a price check or an AI review.
func (s *Service) SaveSettings(ctx context.Context, input SettingsInput) (model.LeboncoinSession, model.ClaudeToken, error) {
	var sessionChange *store.SessionChange
	if input.LeboncoinSession != nil {
		value, err := leboncoin.ParseSessionInput(input.LeboncoinSession.Value)
		var inputError *leboncoin.SessionInputError
		if errors.As(err, &inputError) {
			return model.LeboncoinSession{}, model.ClaudeToken{}, &Error{Status: 400, Code: "INVALID_SESSION", Message: inputError.Message}
		}
		if err != nil {
			return model.LeboncoinSession{}, model.ClaudeToken{}, err
		}
		sessionChange = &store.SessionChange{ExpectedRevision: input.LeboncoinSession.Revision}
		if value != "" {
			sessionChange.Value = &value
		}
	}
	var tokenChange *store.TokenChange
	if input.ClaudeToken != nil {
		token := strings.TrimSpace(*input.ClaudeToken)
		if err := validateClaudeToken(token); err != nil {
			return model.LeboncoinSession{}, model.ClaudeToken{}, err
		}
		tokenChange = &store.TokenChange{}
		if token != "" {
			tokenChange.Value = &token
			stored, err := s.store.ClaudeToken(ctx, ownerFrom(ctx))
			if err != nil {
				return model.LeboncoinSession{}, model.ClaudeToken{}, err
			}
			if stored.Value != nil && *stored.Value == token {
				tokenChange = nil // unchanged: no verification and no change
			} else if err := s.verifyClaudeToken(ctx, token); err != nil {
				return model.LeboncoinSession{}, model.ClaudeToken{}, err
			}
		}
	}
	session, token, err := s.store.SaveSettings(ctx, ownerFrom(ctx), sessionChange, tokenChange, time.Now().UTC())
	if errors.Is(err, store.ErrSessionChanged) {
		return model.LeboncoinSession{}, model.ClaudeToken{}, &Error{Status: 409, Code: "SESSION_CHANGED", Message: "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving."}
	}
	return session, token, err
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
