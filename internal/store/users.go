package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrUsernameTaken reports that a username already exists, ignoring case.
var ErrUsernameTaken = errors.New("username taken")

// User is a login account; Role is "admin" or "user".
type User struct {
	ID           int64
	Username     string
	Role         string
	PasswordHash string
	LastLoginAt  *time.Time
}

func (s *Store) createUserSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_login_at TEXT
);
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_user ON sessions(user_id);
`)
	if err != nil {
		return fmt.Errorf("create user schema: %w", err)
	}
	return nil
}

// CreateAdminOrResetPassword creates the admin account, or replaces its password
// and ends its sessions. It reports whether the account was created.
func (s *Store) CreateAdminOrResetPassword(ctx context.Context, passwordHash string, now time.Time) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE role = 'admin'", passwordHash)
	if err != nil {
		return false, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if updated > 0 {
		_, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE role = 'admin')")
	} else {
		_, err = tx.ExecContext(ctx, "INSERT INTO users (username, role, password_hash, created_at) VALUES ('admin', 'admin', ?, ?)",
			passwordHash, now.UTC().Format(timestampLayout))
	}
	if err != nil {
		return false, err
	}
	return updated == 0, tx.Commit()
}

// CreateUser adds a regular user. The first regular user receives the open-mode (owner 0) data.
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string, now time.Time) (User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	var others int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = 'user'").Scan(&others); err != nil {
		return User{}, err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO users (username, role, password_hash, created_at) VALUES (?, 'user', ?, ?)",
		username, passwordHash, now.UTC().Format(timestampLayout))
	if IsUniqueConstraint(err) {
		return User{}, ErrUsernameTaken
	}
	if err != nil {
		return User{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}
	if others == 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE items SET owner_id = ? WHERE owner_id = 0", id); err != nil {
			return User{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return User{ID: id, Username: username, Role: "user", PasswordHash: passwordHash}, nil
}

const userColumns = "users.id, users.username, users.role, users.password_hash, users.last_login_at"

func scanUser(row scanner) (User, error) {
	var user User
	var lastLogin sql.NullString
	if err := row.Scan(&user.ID, &user.Username, &user.Role, &user.PasswordHash, &lastLogin); err != nil {
		return User{}, err
	}
	user.LastLoginAt = optionalTimestamp(lastLogin)
	return user, nil
}

// UserByUsername finds a user ignoring case; a missing user is sql.ErrNoRows.
func (s *Store) UserByUsername(ctx context.Context, username string) (User, error) {
	return scanUser(s.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE username = ?", username))
}

// ListUsers returns the regular users ordered by username.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+userColumns+" FROM users WHERE role = 'user' ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// HasRegularUser reports whether protected mode is on.
func (s *Store) HasRegularUser(ctx context.Context) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM users WHERE role = 'user' LIMIT 1").Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// CreateSession stores a login session by the hash of its token.
func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, now, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		tokenHash, userID, now.UTC().Format(timestampLayout), expiresAt.UTC().Format(timestampLayout))
	return err
}

// SessionUser returns the user of an unexpired session; otherwise sql.ErrNoRows.
func (s *Store) SessionUser(ctx context.Context, tokenHash string, now time.Time) (User, error) {
	return scanUser(s.db.QueryRowContext(ctx, "SELECT "+userColumns+` FROM sessions JOIN users ON users.id = sessions.user_id
WHERE sessions.token_hash = ? AND sessions.expires_at > ?`, tokenHash, now.UTC().Format(timestampLayout)))
}

// DeleteSession ends a session; an unknown token is not an error.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", tokenHash)
	return err
}

// DeleteExpiredSessions removes sessions that can no longer be used.
func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at <= ?", now.UTC().Format(timestampLayout))
	return err
}

// RecordLogin saves the user's last login time.
func (s *Store) RecordLogin(ctx context.Context, userID int64, now time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE users SET last_login_at = ? WHERE id = ?", now.UTC().Format(timestampLayout), userID)
	return err
}
