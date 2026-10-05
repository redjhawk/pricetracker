package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// request sends a request with an optional session cookie and returns the recorder.
func request(t *testing.T, server *Server, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)
	return recorder
}

func login(t *testing.T, server *Server, username, password string) string {
	t.Helper()
	recorder := request(t, server, http.MethodPost, "/api/v1/auth/login", `{"username":"`+username+`","password":"`+password+`"}`, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", username, recorder.Code, recorder.Body.String())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == sessionCookie && cookie.HttpOnly && cookie.SameSite == http.SameSiteLaxMode && cookie.MaxAge == 30*24*3600 {
			return cookie.Value
		}
	}
	t.Fatalf("no session cookie: %v", recorder.Header())
	return ""
}

func TestAccessMatrixOpenAndProtectedModes(t *testing.T) {
	server, _ := newSettingsServer(t)
	adminPassword, _, err := server.service.ResetAdminPassword(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expect := func(method, path, body, token string, status int, code string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := request(t, server, method, path, body, token)
		if code != "" {
			assertError(t, recorder, status, code, "")
		} else if recorder.Code != status {
			t.Fatalf("%s %s: %d, want %d: %s", method, path, recorder.Code, status, recorder.Body.String())
		}
		return recorder
	}
	// Open mode: no login needed, even with the admin account present.
	expect(http.MethodGet, "/api/v1/items", "", "", http.StatusOK, "")
	session := expect(http.MethodGet, "/api/v1/auth/session", "", "", http.StatusOK, "")
	if strings.TrimSpace(session.Body.String()) != `{"mode":"open","user":null}` {
		t.Fatalf("open session %s", session.Body.String())
	}
	expect(http.MethodGet, "/api/v1/admin/users", "", "", http.StatusUnauthorized, "AUTH_REQUIRED")
	expect(http.MethodPost, "/api/v1/auth/login", `{"username":"admin","password":"wrong"}`, "", http.StatusUnauthorized, "INVALID_CREDENTIALS")
	expect(http.MethodPost, "/api/v1/auth/login", `{"username":1}`, "", http.StatusBadRequest, "INVALID_JSON")
	admin := login(t, server, "admin", adminPassword)
	expect(http.MethodGet, "/api/v1/items", "", admin, http.StatusForbidden, "FORBIDDEN")
	expect(http.MethodGet, "/api/v1/settings/claude-token", "", admin, http.StatusForbidden, "FORBIDDEN")
	expect(http.MethodPost, "/api/v1/admin/users", `{"username":"al","password":"long enough pw"}`, admin, http.StatusBadRequest, "INVALID_USERNAME")
	expect(http.MethodPost, "/api/v1/admin/users", `{"username":"alice","password":"short"}`, admin, http.StatusBadRequest, "PASSWORD_TOO_SHORT")
	created := expect(http.MethodPost, "/api/v1/admin/users", `{"username":"alice","password":"long enough pw"}`, admin, http.StatusCreated, "")
	if strings.TrimSpace(created.Body.String()) != `{"user":{"username":"alice","lastLoginAt":null}}` {
		t.Fatalf("created %s", created.Body.String())
	}
	expect(http.MethodPost, "/api/v1/admin/users", `{"username":"ALICE","password":"long enough pw"}`, admin, http.StatusConflict, "USERNAME_TAKEN")
	// Protected mode from the first user on.
	expect(http.MethodGet, "/api/v1/items", "", "", http.StatusUnauthorized, "AUTH_REQUIRED")
	expect(http.MethodGet, "/api/v1/items", "", "not-a-session", http.StatusUnauthorized, "AUTH_REQUIRED")
	user := login(t, server, "Alice", "long enough pw")
	expect(http.MethodGet, "/api/v1/items", "", user, http.StatusOK, "")
	expect(http.MethodGet, "/api/v1/admin/users", "", user, http.StatusForbidden, "FORBIDDEN")
	session = expect(http.MethodGet, "/api/v1/auth/session", "", user, http.StatusOK, "")
	if strings.TrimSpace(session.Body.String()) != `{"mode":"protected","user":{"username":"alice","role":"user"}}` {
		t.Fatalf("user session %s", session.Body.String())
	}
	var list struct {
		Users []struct {
			Username    string
			LastLoginAt *string
		}
	}
	listed := expect(http.MethodGet, "/api/v1/admin/users", "", admin, http.StatusOK, "")
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil || len(list.Users) != 1 || list.Users[0].LastLoginAt == nil {
		t.Fatalf("users %s %v", listed.Body.String(), err)
	}
	logout := expect(http.MethodPost, "/api/v1/auth/logout", "", user, http.StatusNoContent, "")
	if cookies := logout.Result().Cookies(); len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Fatalf("logout cookie %v", cookies)
	}
	expect(http.MethodGet, "/api/v1/items", "", user, http.StatusUnauthorized, "AUTH_REQUIRED")
}
