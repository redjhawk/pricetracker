package httpapi

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

const settingsPath = "/api/v1/settings/leboncoin-session"

func newSettingsServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	cfg := config.Config{DataDirectory: t.TempDir(), Location: time.UTC, CheckTimes: []string{"08:00"}, StaleAfter: 36 * time.Hour}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	items := service.New(cfg, database)
	t.Cleanup(items.Close)
	return New(cfg, items), database
}

func call(t *testing.T, server *Server, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, settingsPath, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func decodeSession(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope map[string]map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("body %q: %v", recorder.Body.String(), err)
	}
	session, ok := envelope["session"]
	if !ok || len(envelope) != 1 {
		t.Fatalf("missing session envelope: %s", recorder.Body.String())
	}
	return session
}

func assertError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status %d, want %d: %s", recorder.Code, status, recorder.Body.String())
	}
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Error.Code != code || (message != "" && body.Error.Message != message) {
		t.Fatalf("error body %s, want %s %q", recorder.Body.String(), code, message)
	}
}

func TestGetLeboncoinSessionEmpty(t *testing.T) {
	server, _ := newSettingsServer(t)
	recorder := call(t, server, http.MethodGet, "")
	if recorder.Code != 200 || recorder.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status=%d headers=%v", recorder.Code, recorder.Header())
	}
	want := `{"session":{"value":null,"revision":0,"updatedAt":null,"status":"none","expiresAt":null,"revokedAt":null,"lastAttempt":null}}`
	if strings.TrimSpace(recorder.Body.String()) != want {
		t.Fatalf("body %s", recorder.Body.String())
	}
}

func TestPutLeboncoinSessionSaveConflictAndClear(t *testing.T) {
	server, _ := newSettingsServer(t)
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	saved := call(t, server, http.MethodPut, `{"value":"Cookie: a=1; datadome=synthetic-http; b=2","revision":0}`)
	if saved.Code != 200 || saved.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("save %d %s", saved.Code, saved.Body.String())
	}
	session := decodeSession(t, saved)
	if session["value"] != "synthetic-http" || session["revision"] != float64(1) || session["status"] != "active" || session["expiresAt"] != nil || session["revokedAt"] != nil || session["lastAttempt"] != nil || session["updatedAt"] == nil {
		t.Fatalf("saved %v", session)
	}
	if got := decodeSession(t, call(t, server, http.MethodGet, "")); got["value"] != "synthetic-http" || got["revision"] != float64(1) {
		t.Fatalf("get after save %v", got)
	}
	assertError(t, call(t, server, http.MethodPut, `{"value":"synthetic-other","revision":0}`), 409, "SESSION_CHANGED", "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving.")
	assertError(t, call(t, server, http.MethodPut, `{"value":"datadome=","revision":1}`), 400, "INVALID_SESSION", "The datadome cookie value is empty.")
	cleared := decodeSession(t, call(t, server, http.MethodPut, `{"value":"  ","revision":1}`))
	if cleared["value"] != nil || cleared["status"] != "none" || cleared["revision"] != float64(2) {
		t.Fatalf("cleared %v", cleared)
	}
	unchanged := decodeSession(t, call(t, server, http.MethodPut, `{"value":"","revision":2}`))
	if unchanged["revision"] != float64(2) || unchanged["status"] != "none" {
		t.Fatalf("no-op clear %v", unchanged)
	}
	if strings.Contains(logs.String(), "synthetic-") {
		t.Fatal("session value logged")
	}
}

func TestPutLeboncoinSessionRequestErrors(t *testing.T) {
	server, _ := newSettingsServer(t)
	const invalidRequest = "Request must include a session value (use an empty string to remove it) and the revision from Settings."
	for _, body := range []string{`{}`, `{"value":"x"}`, `{"revision":0}`, `{"value":null,"revision":0}`, `{"value":1,"revision":0}`, `{"value":"x","revision":"0"}`, `{"value":"x","revision":1.5}`, `{"value":"x","revision":-1}`, `null`, `[]`} {
		assertError(t, call(t, server, http.MethodPut, body), 400, "INVALID_REQUEST", invalidRequest)
	}
	assertError(t, call(t, server, http.MethodPut, `{"value":`), 400, "INVALID_JSON", "")
	assertError(t, call(t, server, http.MethodPut, `{"value":"x","revision":0} {}`), 400, "INVALID_JSON", "")
	assertError(t, call(t, server, http.MethodPut, `{"value":"`+strings.Repeat("v", 32_768)+`","revision":0}`), 413, "REQUEST_TOO_LARGE", "Request body is too large.")
	assertError(t, call(t, server, http.MethodPut, `{"value":"`+strings.Repeat("v", 8193)+`","revision":0}`), 400, "INVALID_SESSION", "The pasted text is too long (maximum 8192 characters).")
	for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPatch} {
		recorder := call(t, server, method, "")
		assertError(t, recorder, 405, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
		if recorder.Header().Get("Allow") != "GET, PUT" {
			t.Fatalf("Allow %q", recorder.Header().Get("Allow"))
		}
	}
	if got := decodeSession(t, call(t, server, http.MethodGet, "")); got["revision"] != float64(0) || got["value"] != nil {
		t.Fatalf("rejected requests changed the session %v", got)
	}
}

func TestGetLeboncoinSessionStorageFailure(t *testing.T) {
	server, database := newSettingsServer(t)
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	database.Close()
	assertError(t, call(t, server, http.MethodGet, ""), 500, "INTERNAL_ERROR", "The server could not complete the request.")
	assertError(t, call(t, server, http.MethodPut, `{"value":"synthetic-unsaved","revision":0}`), 500, "INTERNAL_ERROR", "The server could not complete the request.")
	if strings.Contains(logs.String(), "synthetic-") {
		t.Fatal("session value logged")
	}
}

func TestReviewOversizedTrailingDataIs413(t *testing.T) {
	server, _ := newSettingsServer(t)
	assertError(t, call(t, server, http.MethodPut, `{"value":"synthetic-x","revision":0}`+strings.Repeat(" ", 40_000)+"x"), 413, "REQUEST_TOO_LARGE", "Request body is too large.")
	if got := decodeSession(t, call(t, server, http.MethodGet, "")); got["revision"] != float64(0) {
		t.Fatalf("session changed %v", got)
	}
}
