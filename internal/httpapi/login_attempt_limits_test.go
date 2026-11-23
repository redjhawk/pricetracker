package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

// Login attempt limits by client address (issue #266, FR-LOGIN-LIM-007..011).

const (
	limitPassword = "correct horse battery"
	lockedMessage = "Too many failed attempts. Wait one minute and try again."
	busyMessage   = "Too many login attempts are being checked. Try again in a moment."
	// maxWaitingChecks is the maximum number of waiting password checks of API_SPECIFICATION.md.
	maxWaitingChecks = 4
)

// postLoginBody posts a login body from remoteAddr; a cfConnectingIP value, even empty, sets the header.
func postLoginBody(server *Server, remoteAddr, body string, cfConnectingIP ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.RemoteAddr = remoteAddr
	if len(cfConnectingIP) > 0 {
		req.Header.Set("CF-Connecting-IP", cfConnectingIP[0])
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)
	return recorder
}

func postLogin(server *Server, remoteAddr, username, password string, cfConnectingIP ...string) *httptest.ResponseRecorder {
	return postLoginBody(server, remoteAddr, `{"username":"`+username+`","password":"`+password+`"}`, cfConnectingIP...)
}

func hasSessionCookie(recorder *httptest.ResponseRecorder) bool {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookie && cookie.Value != "" {
			return true
		}
	}
	return false
}

func expectHTTPLogin(t *testing.T, recorder *httptest.ResponseRecorder, what string) {
	t.Helper()
	if recorder.Code != http.StatusOK || !hasSessionCookie(recorder) {
		t.Fatalf("%s: status %d, cookie %v, want 200 with a session cookie: %s", what, recorder.Code, hasSessionCookie(recorder), recorder.Body.String())
	}
}

func expectHTTPError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code, message, what string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("%s: status %d, want %d %s: %s", what, recorder.Code, status, code, recorder.Body.String())
	}
	assertError(t, recorder, status, code, message)
	if hasSessionCookie(recorder) {
		t.Fatalf("%s: session cookie set on an error: %v", what, recorder.Header())
	}
}

func expectHTTPLocked(t *testing.T, recorder *httptest.ResponseRecorder, what string) {
	t.Helper()
	expectHTTPError(t, recorder, http.StatusTooManyRequests, "LOGIN_LOCKED", lockedMessage, what)
}

func expectHTTPInvalid(t *testing.T, recorder *httptest.ResponseRecorder, what string) {
	t.Helper()
	expectHTTPError(t, recorder, http.StatusUnauthorized, "INVALID_CREDENTIALS", "", what)
}

// failHTTPLogins posts count wrong passwords for alice, each answered 401.
func failHTTPLogins(t *testing.T, server *Server, count int, remoteAddr string, cfConnectingIP ...string) {
	t.Helper()
	for attempt := 1; attempt <= count; attempt++ {
		expectHTTPInvalid(t, postLogin(server, remoteAddr, "alice", "wrong password", cfConnectingIP...), fmt.Sprintf("%s %v wrong password %d", remoteAddr, cfConnectingIP, attempt))
	}
}

func newLimitServer(t *testing.T) *Server {
	t.Helper()
	server, _ := newSettingsServer(t)
	if _, err := server.service.CreateUser(context.Background(), "alice", limitPassword); err != nil {
		t.Fatal(err)
	}
	return server
}

// expectNoSessionUser checks that the session endpoint without a cookie reports no user.
func expectNoSessionUser(t *testing.T, server *Server) {
	t.Helper()
	recorder := request(t, server, http.MethodGet, "/api/v1/auth/session", "", "")
	var body struct {
		Mode string
		User *userJSON
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || recorder.Code != http.StatusOK || body.Mode != "protected" || body.User != nil {
		t.Fatalf("session %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginLimits_TC009_CFConnectingIPIsTheClientAddress(t *testing.T) {
	server := newLimitServer(t)
	failHTTPLogins(t, server, 5, "192.0.2.10:5000", "203.0.113.1")
	expectHTTPLocked(t, postLogin(server, "192.0.2.10:5000", "alice", limitPassword, "203.0.113.1"), "203.0.113.1 after 5 failures")
	expectHTTPLocked(t, postLogin(server, "192.0.2.11:6000", "alice", limitPassword, "203.0.113.1"), "203.0.113.1 from another connection")
	expectHTTPLogin(t, postLogin(server, "192.0.2.10:5000", "alice", limitPassword, "203.0.113.2"), "203.0.113.2 through the same proxy")
	expectHTTPLogin(t, postLogin(server, "192.0.2.10:5000", "alice", limitPassword), "the proxy connection without the header")

	failHTTPLogins(t, server, 5, "192.0.2.10:5000", "2001:db8::1")
	expectHTTPLocked(t, postLogin(server, "192.0.2.10:5000", "alice", limitPassword, "2001:db8::1"), "2001:db8::1 after 5 failures")
	expectHTTPLogin(t, postLogin(server, "192.0.2.10:5000", "alice", limitPassword, "2001:db8::2"), "2001:db8::2")
}

func TestLoginLimits_TC010_ConnectionAddressWithoutAValidHeader(t *testing.T) {
	server := newLimitServer(t)
	failHTTPLogins(t, server, 5, "198.51.100.7:40000")
	expectHTTPLocked(t, postLogin(server, "198.51.100.7:40000", "alice", limitPassword), "no header")
	for _, header := range []string{"not-an-ip", "", "203.0.113.1:443"} {
		expectHTTPLocked(t, postLogin(server, "198.51.100.7:40000", "alice", limitPassword, header), fmt.Sprintf("CF-Connecting-IP %q", header))
	}
	expectHTTPLocked(t, postLogin(server, "198.51.100.7:40001", "alice", limitPassword), "another port")
	expectHTTPLogin(t, postLogin(server, "198.51.100.8:40000", "alice", limitPassword), "another connection address")
	expectHTTPLogin(t, postLogin(server, "198.51.100.8:40000", "alice", limitPassword, "not-an-ip"), "not-an-ip from another connection address")
}

func TestLoginLimits_TC013_LockedAnswerOverHTTP(t *testing.T) {
	server := newLimitServer(t)
	const remote = "192.0.2.1:1234"
	failHTTPLogins(t, server, 4, remote)
	expectHTTPError(t, postLoginBody(server, remote, `{"username":1}`), http.StatusBadRequest, "INVALID_JSON", "", "invalid body")
	expectHTTPLogin(t, postLogin(server, remote, "alice", limitPassword), "after 4 failures and an invalid body")
	failHTTPLogins(t, server, 5, remote)
	expectHTTPLocked(t, postLogin(server, remote, "alice", limitPassword), "after 5 failures")
	expectNoSessionUser(t, server)
}

func TestLoginLimits_TC013_BusyAnswerOverHTTP(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var first atomic.Bool
	cfg := config.Config{DataDirectory: t.TempDir(), Location: time.UTC, CheckTimes: []string{"08:00"}, StaleAfter: 36 * time.Hour}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	items := service.NewWithOptions(cfg, database, service.Options{BeforePasswordCheck: func(context.Context) {
		if first.CompareAndSwap(false, true) {
			close(entered)
			<-release
		}
	}})
	t.Cleanup(items.Close)
	if _, err := items.CreateUser(context.Background(), "alice", limitPassword); err != nil {
		t.Fatal(err)
	}
	server := New(cfg, items)

	var released sync.Once
	var running sync.WaitGroup
	releaseAll := func() { released.Do(func() { close(release) }) }
	// Runs before the service and database are closed: every started request must end first.
	t.Cleanup(func() {
		releaseAll()
		running.Wait()
	})
	start := func(remoteAddr string) <-chan *httptest.ResponseRecorder {
		results := make(chan *httptest.ResponseRecorder, 1)
		running.Add(1)
		go func() {
			defer running.Done()
			results <- postLogin(server, remoteAddr, "alice", limitPassword)
		}()
		return results
	}

	held := start("192.0.2.20:1000")
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the password check was not held (BeforePasswordCheck not called)")
	}
	var waiting []<-chan *httptest.ResponseRecorder
	for index := 0; index < maxWaitingChecks; index++ {
		waiting = append(waiting, start(fmt.Sprintf("192.0.2.%d:1000", 21+index)))
	}
	deadline := time.Now().Add(5 * time.Second)
	for service.WaitingLoginChecks() != maxWaitingChecks {
		if time.Now().After(deadline) {
			t.Fatalf("%d logins waiting for a password check, want %d", service.WaitingLoginChecks(), maxWaitingChecks)
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case recorder := <-start("192.0.2.30:1000"):
		expectHTTPError(t, recorder, http.StatusTooManyRequests, "LOGIN_BUSY", busyMessage, "login with the maximum waiting")
	case <-time.After(2 * time.Second):
		t.Fatal("the busy login did not answer while the password checks are held")
	}
	expectNoSessionUser(t, server)

	releaseAll()
	for index, results := range append([]<-chan *httptest.ResponseRecorder{held}, waiting...) {
		select {
		case recorder := <-results:
			expectHTTPLogin(t, recorder, fmt.Sprintf("held or waiting login %d", index))
		case <-time.After(30 * time.Second):
			t.Fatalf("held or waiting login %d did not finish after the release", index)
		}
	}
}
