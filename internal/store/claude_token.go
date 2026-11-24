package store

import (
	"context"
	"database/sql"
	"time"

	"pricefollower.local/internal/model"
)

// SessionChange is a LeBoncoin session save applied by SaveSettings.
type SessionChange struct {
	Value            *string // nil clears the session
	ExpectedRevision int64
}

// TokenChange is a Claude token save applied by SaveSettings.
type TokenChange struct {
	Value *string // nil clears the token
}

func readClaudeToken(ctx context.Context, q rowQuerier, ownerID int64) (model.ClaudeToken, error) {
	var value, updatedAt, lastRejectedAt sql.NullString
	token := model.ClaudeToken{OwnerID: ownerID}
	err := q.QueryRowContext(ctx, "SELECT value, revision, updated_at, last_rejected_at FROM claude_token WHERE owner_id = ?", ownerID).
		Scan(&value, &token.Revision, &updatedAt, &lastRejectedAt)
	if err != nil {
		return model.ClaudeToken{}, err
	}
	token.Value = nullString(value)
	token.UpdatedAt = optionalTimestamp(updatedAt)
	token.LastRejectedAt = optionalTimestamp(lastRejectedAt)
	return token, nil
}

// ClaudeToken returns the saved Claude token.
func (s *Store) ClaudeToken(ctx context.Context, ownerID int64) (model.ClaudeToken, error) {
	if err := s.ensureSettings(ctx, ownerID); err != nil {
		return model.ClaudeToken{}, err
	}
	return readClaudeToken(ctx, s.db, ownerID)
}

// SaveSettings applies both Settings entries in one transaction; either may be nil.
// The session follows the SaveLeboncoinSession rules. Saving a token replaces it,
// increments its revision and clears the rejection time; clearing an empty token
// changes nothing.
func (s *Store) SaveSettings(ctx context.Context, ownerID int64, session *SessionChange, token *TokenChange, now time.Time) (model.LeboncoinSession, model.ClaudeToken, error) {
	if err := s.ensureSettings(ctx, ownerID); err != nil {
		return model.LeboncoinSession{}, model.ClaudeToken{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.LeboncoinSession{}, model.ClaudeToken{}, err
	}
	defer tx.Rollback()
	var savedSession model.LeboncoinSession
	if session != nil {
		savedSession, err = saveLeboncoinSessionTx(ctx, tx, ownerID, session.Value, session.ExpectedRevision, now)
	} else {
		var row leboncoinSessionRow
		row, err = readLeboncoinSession(ctx, tx, ownerID)
		savedSession = row.model(now)
	}
	if err != nil {
		return model.LeboncoinSession{}, model.ClaudeToken{}, err
	}
	current, err := readClaudeToken(ctx, tx, ownerID)
	if err != nil {
		return model.LeboncoinSession{}, model.ClaudeToken{}, err
	}
	if token != nil && (token.Value != nil || current.Value != nil) {
		if _, err := tx.ExecContext(ctx, `UPDATE claude_token SET value = ?, revision = revision + 1, updated_at = ?,
last_rejected_at = NULL WHERE owner_id = ?`, token.Value, now.UTC().Format(timestampLayout), ownerID); err != nil {
			return model.LeboncoinSession{}, model.ClaudeToken{}, err
		}
		if current, err = readClaudeToken(ctx, tx, ownerID); err != nil {
			return model.LeboncoinSession{}, model.ClaudeToken{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return model.LeboncoinSession{}, model.ClaudeToken{}, err
	}
	return savedSession, current, nil
}

// MarkClaudeTokenRejected records a review rejection, only if the token revision is unchanged.
func (s *Store) MarkClaudeTokenRejected(ctx context.Context, ownerID, revision int64, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE claude_token SET last_rejected_at = ? WHERE owner_id = ? AND revision = ?", now.UTC().Format(timestampLayout), ownerID, revision)
	return err
}

// ClearClaudeTokenRejected clears the rejection time after a successful review, only if the token revision is unchanged.
func (s *Store) ClearClaudeTokenRejected(ctx context.Context, ownerID, revision int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE claude_token SET last_rejected_at = NULL WHERE owner_id = ? AND revision = ?", ownerID, revision)
	return err
}
