package httpapi

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type claudeTransport func(*http.Request) (*http.Response, error)

func (f claudeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// withClaudeStatus answers every outgoing request (the Claude verification) with status.
func withClaudeStatus(t *testing.T, status int) *int {
	t.Helper()
	calls := 0
	original := http.DefaultTransport
	http.DefaultTransport = claudeTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"content":[]}`)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	return &calls
}

func callPath(server *Server, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}

func TestGetClaudeTokenEmptyAndMethod(t *testing.T) {
	server, _ := newSettingsServer(t)
	recorder := callPath(server, http.MethodGet, "/api/v1/settings/claude-token", "")
	want := `{"claudeToken":{"value":null,"updatedAt":null,"lastRejectedAt":null}}`
	if recorder.Code != 200 || recorder.Header().Get("Cache-Control") != "no-store" || strings.TrimSpace(recorder.Body.String()) != want {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder = callPath(server, http.MethodPut, "/api/v1/settings/claude-token", "{}")
	assertError(t, recorder, 405, "METHOD_NOT_ALLOWED", "")
	if recorder.Header().Get("Allow") != "GET" {
		t.Fatalf("Allow %q", recorder.Header().Get("Allow"))
	}
	recorder = callPath(server, http.MethodGet, "/api/v1/settings", "")
	assertError(t, recorder, 405, "METHOD_NOT_ALLOWED", "")
	if recorder.Header().Get("Allow") != "PUT" {
		t.Fatalf("Allow %q", recorder.Header().Get("Allow"))
	}
}

func TestPutSettingsSavesBothEntries(t *testing.T) {
	server, _ := newSettingsServer(t)
	calls := withClaudeStatus(t, 200)
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	body := `{"leboncoinSession":{"value":"datadome=synthetic-lbc","revision":0},"claudeToken":{"value":"sk-ant-oat01-synthetic"}}`
	recorder := callPath(server, http.MethodPut, "/api/v1/settings", body)
	if recorder.Code != 200 || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"value":"synthetic-lbc"`) || !strings.Contains(recorder.Body.String(), `"claudeToken":{"value":"sk-ant-oat01-synthetic"`) {
		t.Fatalf("body %s", recorder.Body.String())
	}
	if *calls != 1 {
		t.Fatalf("verification calls %d", *calls)
	}
	got := callPath(server, http.MethodGet, "/api/v1/settings/claude-token", "")
	if !strings.Contains(got.Body.String(), `"value":"sk-ant-oat01-synthetic"`) {
		t.Fatalf("token not persisted: %s", got.Body.String())
	}
	items := callPath(server, http.MethodGet, "/api/v1/items", "")
	if strings.Contains(items.Body.String(), "sk-ant-oat01-synthetic") || strings.Contains(logs.String(), "sk-ant-oat01-synthetic") {
		t.Fatal("token leaked into items or logs")
	}
}

func TestPutSettingsErrors(t *testing.T) {
	server, _ := newSettingsServer(t)
	const token = `{"claudeToken":{"value":"sk-ant-oat01-synthetic"}}`
	for _, test := range []struct {
		name, body   string
		claudeStatus int
		status       int
		code         string
	}{
		{"invalid json", `{`, 200, 400, "INVALID_JSON"},
		{"two values", `{} {}`, 200, 400, "INVALID_JSON"},
		{"no part", `{}`, 200, 400, "INVALID_REQUEST"},
		{"token without value", `{"claudeToken":{}}`, 200, 400, "INVALID_REQUEST"},
		{"token wrong type", `{"claudeToken":{"value":1}}`, 200, 400, "INVALID_REQUEST"},
		{"session without revision", `{"leboncoinSession":{"value":""}}`, 200, 400, "INVALID_REQUEST"},
		{"session negative revision", `{"leboncoinSession":{"value":"","revision":-1}}`, 200, 400, "INVALID_REQUEST"},
		{"invalid session", `{"leboncoinSession":{"value":"no cookie here","revision":0}}`, 200, 400, "INVALID_SESSION"},
		{"invalid token", `{"claudeToken":{"value":"sk ant"}}`, 200, 400, "INVALID_CLAUDE_TOKEN"},
		{"rejected", token, 401, 422, "CLAUDE_TOKEN_REJECTED"},
		{"usage limit", token, 429, 502, "CLAUDE_UNREACHABLE"},
		{"server error", token, 500, 502, "CLAUDE_UNREACHABLE"},
		{"stale session", `{"leboncoinSession":{"value":"datadome=x","revision":4}}`, 200, 409, "SESSION_CHANGED"},
		{"too large", `{"claudeToken":{"value":"` + strings.Repeat("a", 33_000) + `"}}`, 200, 413, "REQUEST_TOO_LARGE"},
	} {
		withClaudeStatus(t, test.claudeStatus)
		recorder := callPath(server, http.MethodPut, "/api/v1/settings", test.body)
		if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), `"code":"`+test.code+`"`) {
			t.Errorf("%s: status=%d body=%s", test.name, recorder.Code, recorder.Body.String())
		}
	}
	got := callPath(server, http.MethodGet, "/api/v1/settings/claude-token", "")
	if !strings.Contains(got.Body.String(), `"value":null`) {
		t.Fatalf("a failed save stored the token: %s", got.Body.String())
	}
}
