package service

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"pricefollower.local/internal/store"
)

// SessionDuration is how long a login lasts; it is not extended by use.
const SessionDuration = 30 * 24 * time.Hour

const (
	maxFailedLogins = 5
	loginLockTime   = time.Minute
	// failureMemory is how long an unlocked pair's failures are remembered.
	failureMemory = time.Hour
)

// passwordIterations is the PBKDF2 cost for new hashes (about 1 s on ARMv6); tests lower it.
// Stored hashes carry their own count, so it can change without a migration.
var passwordIterations = 100_000

// hashSlot allows one PBKDF2 computation at a time, so password checks cannot saturate the CPU.
var hashSlot = make(chan struct{}, 1)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,32}$`)

// maxWaitingLoginChecks is the most logins that may wait for the password-check slot.
const maxWaitingLoginChecks = 4

// waitingLoginChecks counts the logins waiting for the password-check slot.
var waitingLoginChecks atomic.Int32

// WaitingLoginChecks returns the number of logins waiting for the password-check slot (tests).
func WaitingLoginChecks() int {
	return int(waitingLoginChecks.Load())
}

// loginKey identifies the failures of one client address and lowercased username.
type loginKey struct{ client, username string }

// loginFailures counts consecutive failed logins of one client address and username.
type loginFailures struct {
	count       int
	lockedUntil time.Time
	lastFailure time.Time
}

func loginBusyError() error {
	return &Error{Status: 429, Code: "LOGIN_BUSY", Message: "Too many login attempts are being checked. Try again in a moment."}
}

// dummyPasswordHash is checked for unknown usernames, so they cost the same hash check and timing
// does not reveal them. It is built without a computation; no password matches it in practice.
func dummyPasswordHash() string {
	encoding := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordIterations, encoding.EncodeToString(make([]byte, 16)), encoding.EncodeToString(make([]byte, 32)))
}

func pbkdf2Key(ctx context.Context, password string, salt []byte, iterations, length int) ([]byte, error) {
	select {
	case hashSlot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-hashSlot }()
	return pbkdf2.Key(sha256.New, password, salt, iterations, length)
}

// Principal is the logged-in user of a request; without one, requests use the open-mode owner 0.
type Principal struct {
	UserID int64
	Role   string
}

type principalKey struct{}

// WithPrincipal returns a context carrying the request's logged-in user.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// ownerFrom returns the request's owner id: the logged-in user, or 0 in open mode.
func ownerFrom(ctx context.Context) int64 {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	if !ok || principal.Role != "user" {
		return 0
	}
	return principal.UserID
}

// HashPassword returns a salted PBKDF2-SHA256 hash in the stored text format.
func HashPassword(ctx context.Context, password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2Key(ctx, password, salt, passwordIterations, 32)
	if err != nil {
		return "", err
	}
	encoding := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordIterations, encoding.EncodeToString(salt), encoding.EncodeToString(key)), nil
}

// acquireLoginCheck takes the password-check slot for a login. When maxWaitingLoginChecks logins
// already wait for it, the login is refused busy at once instead of waiting.
func acquireLoginCheck(ctx context.Context) error {
	select {
	case hashSlot <- struct{}{}:
		return nil
	default:
	}
	if waitingLoginChecks.Add(1) > maxWaitingLoginChecks {
		waitingLoginChecks.Add(-1)
		return loginBusyError()
	}
	defer waitingLoginChecks.Add(-1)
	select {
	case hashSlot <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// checkPassword reports whether password matches the stored hash of a login; err is only a
// cancelled wait or the busy refusal.
func (s *Service) checkPassword(ctx context.Context, stored, password string) (bool, error) {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false, nil
	}
	iterations, err := strconv.Atoi(parts[1])
	salt, saltErr := base64.RawStdEncoding.DecodeString(parts[2])
	want, keyErr := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || saltErr != nil || keyErr != nil || iterations < 1 {
		return false, nil
	}
	if err := acquireLoginCheck(ctx); err != nil {
		return false, err
	}
	defer func() { <-hashSlot }()
	if s.beforePasswordCheck != nil {
		s.beforePasswordCheck(ctx)
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1, nil
}

func randomText(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ResetAdminPassword creates the admin account or replaces its password with a new
// generated one, ending its sessions. It returns the password to show on the device.
func (s *Service) ResetAdminPassword(ctx context.Context) (string, bool, error) {
	password, err := randomText(18)
	if err != nil {
		return "", false, err
	}
	hash, err := HashPassword(ctx, password)
	if err != nil {
		return "", false, err
	}
	created, err := s.store.CreateAdminOrResetPassword(ctx, hash, s.now())
	return password, created, err
}

// CreateUser validates and adds a regular user; the first one starts protected mode.
func (s *Service) CreateUser(ctx context.Context, username, password string) (store.User, error) {
	if !usernamePattern.MatchString(username) {
		return store.User{}, &Error{Status: 400, Code: "INVALID_USERNAME", Message: "Use 3 to 32 letters, digits, dots, dashes or underscores."}
	}
	length := utf8.RuneCountInString(password)
	if length < 12 {
		return store.User{}, &Error{Status: 400, Code: "PASSWORD_TOO_SHORT", Message: "The password must have at least 12 characters."}
	}
	if length > 256 {
		return store.User{}, &Error{Status: 400, Code: "PASSWORD_TOO_LONG", Message: "The password must have at most 256 characters."}
	}
	hash, err := HashPassword(ctx, password)
	if err != nil {
		return store.User{}, err
	}
	user, err := s.store.CreateUser(ctx, username, hash, s.now())
	if errors.Is(err, store.ErrUsernameTaken) {
		return store.User{}, &Error{Status: 409, Code: "USERNAME_TAKEN", Message: "This username is already used."}
	}
	return user, err
}

// ListUsers returns the regular users for the administrator page.
func (s *Service) ListUsers(ctx context.Context) ([]store.User, error) {
	return s.store.ListUsers(ctx)
}

// ProtectedMode reports whether a regular user exists, so login is required.
func (s *Service) ProtectedMode(ctx context.Context) (bool, error) {
	return s.store.HasRegularUser(ctx)
}

// Login checks the credentials and returns a new session token. After 5 consecutive failures of
// a client address for a username, that pair is refused for 1 minute without a password check.
func (s *Service) Login(ctx context.Context, client, username, password string) (string, store.User, error) {
	key := loginKey{client: client, username: strings.ToLower(username)}
	now := s.now()
	s.mu.Lock()
	locked := now.Before(s.loginFailures[key].lockedUntil)
	s.mu.Unlock()
	if locked {
		return "", store.User{}, &Error{Status: 429, Code: "LOGIN_LOCKED", Message: "Too many failed attempts. Wait one minute and try again."}
	}
	user, err := s.store.UserByUsername(ctx, username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", store.User{}, err
	}
	stored := user.PasswordHash
	if err != nil {
		stored = dummyPasswordHash()
	}
	matches, checkErr := s.checkPassword(ctx, stored, password)
	if checkErr != nil {
		// Cancelled and busy logins are not counted.
		return "", store.User{}, checkErr
	}
	if err != nil || !matches {
		// Time after the check, so the lock lasts 1 minute from the failure even when the check waited.
		s.recordLoginFailure(key, s.now())
		return "", store.User{}, &Error{Status: 401, Code: "INVALID_CREDENTIALS", Message: "Incorrect username or password."}
	}
	s.mu.Lock()
	delete(s.loginFailures, key)
	s.mu.Unlock()
	token, err := randomText(32)
	if err != nil {
		return "", store.User{}, err
	}
	if err := s.store.DeleteExpiredSessions(ctx, now); err != nil {
		return "", store.User{}, err
	}
	if err := s.store.CreateSession(ctx, tokenHash(token), user.ID, now, now.Add(SessionDuration)); err != nil {
		return "", store.User{}, err
	}
	if err := s.store.RecordLogin(ctx, user.ID, now); err != nil {
		return "", store.User{}, err
	}
	return token, user, nil
}

func (s *Service) recordLoginFailure(key loginKey, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Forget expired locks and old failures, so the map cannot grow without bound.
	for other, entry := range s.loginFailures {
		expired := !entry.lockedUntil.IsZero() && !now.Before(entry.lockedUntil)
		if expired || (entry.lockedUntil.IsZero() && now.Sub(entry.lastFailure) > failureMemory) {
			delete(s.loginFailures, other)
		}
	}
	// A failure while still locked (a request that started before the lock) keeps the lock as is.
	failures := s.loginFailures[key]
	failures.count++
	failures.lastFailure = now
	if failures.lockedUntil.IsZero() && failures.count >= maxFailedLogins {
		failures.lockedUntil = now.Add(loginLockTime)
	}
	s.loginFailures[key] = failures
}

// SessionUser returns the user of a valid session token; ok is false when there is none.
func (s *Service) SessionUser(ctx context.Context, token string) (store.User, bool, error) {
	if token == "" {
		return store.User{}, false, nil
	}
	user, err := s.store.SessionUser(ctx, tokenHash(token), s.now())
	if errors.Is(err, sql.ErrNoRows) {
		return store.User{}, false, nil
	}
	return user, err == nil, err
}

// Logout ends the session of the token, if any.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, tokenHash(token))
}
