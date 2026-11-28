package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Issue #267: same-origin and JSON content-type checks on state-changing API requests (FR-API-REQ-001..009).

const (
	checkHost        = "prices.example"
	foreignOrigin    = "https://evil.example"
	ownOrigin        = "https://prices.example"
	originRefusal    = "Requests from another site are not allowed."
	mediaTypeRefusal = "Request body must be JSON (Content-Type: application/json)."
	jsonType         = "application/json"
	newUserBody      = `{"username":"bob","password":"long enough pw"}`
)

// checkHeaders lists the only headers of a checked request. "Host" sets the request host
// (default prices.example) and "RemoteAddr" the client address; an empty value is sent as an empty header.
type checkHeaders map[string]string

func sendChecked(t *testing.T, server *Server, method, path, body string, headers checkHeaders) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = checkHost
	for name, value := range headers {
		switch name {
		case "Host":
			req.Host = value
		case "RemoteAddr":
			req.RemoteAddr = value
		default:
			req.Header[http.CanonicalHeaderKey(name)] = []string{value}
		}
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)
	return recorder
}

// withCookie adds alice's or the administrator's session cookie to the headers.
func withCookie(headers checkHeaders, token string) checkHeaders {
	copied := checkHeaders{"Cookie": sessionCookie + "=" + token}
	for name, value := range headers {
		copied[name] = value
	}
	return copied
}

// expectRefused checks a 403 ORIGIN_NOT_ALLOWED or 415 UNSUPPORTED_MEDIA_TYPE answer in the standard error shape.
func expectRefused(t *testing.T, recorder *httptest.ResponseRecorder, status int, what string) {
	t.Helper()
	code, message := "ORIGIN_NOT_ALLOWED", originRefusal
	if status == http.StatusUnsupportedMediaType {
		code, message = "UNSUPPORTED_MEDIA_TYPE", mediaTypeRefusal
	}
	if recorder.Code != status {
		t.Fatalf("%s: status %d, want %d %s: %s", what, recorder.Code, status, code, recorder.Body.String())
	}
	assertError(t, recorder, status, code, message)
}

func expectStatus(t *testing.T, recorder *httptest.ResponseRecorder, status int, what string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("%s: status %d, want %d: %s", what, recorder.Code, status, recorder.Body.String())
	}
}

func addBody(n int) string {
	return fmt.Sprintf(`{"url":"https://www.leboncoin.fr/ad/test/%d"}`, n)
}

// itemCount answers the length of GET /api/v1/items (token "" in open mode).
func itemCount(t *testing.T, server *Server, token string) int {
	t.Helper()
	recorder := request(t, server, http.MethodGet, "/api/v1/items", "", token)
	var body struct{ Items []json.RawMessage }
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &body) != nil {
		t.Fatalf("list items: %d %s", recorder.Code, recorder.Body.String())
	}
	return len(body.Items)
}

// openChecksServer serves open mode; every outgoing request (listing fetch, Claude) answers status.
func openChecksServer(t *testing.T, claudeStatus int) (*Server, *store.Store) {
	t.Helper()
	withClaudeStatus(t, claudeStatus)
	return newSettingsServer(t)
}

// seedPricedItem stores the LeBoncoin item lbc with a current price and purchase goal Office, and a Claude token.
func seedPricedItem(t *testing.T, database *store.Store) {
	t.Helper()
	insertGoalItem(t, database, "lbc", "Office", 0)
	recordCheck(t, database, "lbc", model.CollectionResult{Result: "success", AmountCents: 4500}, reviewCheckAt)
	saveClaudeToken(t, database, syntheticToken())
}

// stateSnapshot reads the item list, the details of lbc (status, next check, goal, AI review) and both Settings objects.
func stateSnapshot(t *testing.T, server *Server, withItem bool) string {
	t.Helper()
	paths := []string{"/api/v1/items", settingsPath, claudeTokenPath}
	if withItem {
		paths = append(paths, "/api/v1/items/lbc")
	}
	var snapshot strings.Builder
	for _, path := range paths {
		recorder := callPath(server, http.MethodGet, path, "")
		expectStatus(t, recorder, http.StatusOK, "GET "+path)
		snapshot.WriteString(path + " " + recorder.Body.String())
	}
	return snapshot.String()
}

func noCORSHeaders(t *testing.T, recorder *httptest.ResponseRecorder, what string) {
	t.Helper()
	for name := range recorder.Header() {
		if strings.HasPrefix(strings.ToLower(name), "access-control-") {
			t.Fatalf("%s: CORS header %s: %v", what, name, recorder.Header())
		}
	}
}

// TC-001: adding an item from another site is refused and adds nothing.
func TestAPIRequestChecks_TC001_AddFromAnotherSiteIsRefused(t *testing.T) {
	server, _ := openChecksServer(t, http.StatusServiceUnavailable)
	foreign := checkHeaders{"Origin": foreignOrigin, "Content-Type": jsonType}
	for attempt := 1; attempt <= 2; attempt++ {
		recorder := sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), foreign)
		expectRefused(t, recorder, http.StatusForbidden, fmt.Sprintf("foreign add %d", attempt))
		noCORSHeaders(t, recorder, "refusal")
		if count := itemCount(t, server, ""); count != 0 {
			t.Fatalf("after foreign add %d: %d items, want none", attempt, count)
		}
	}
	own := sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), checkHeaders{"Origin": ownOrigin, "Content-Type": jsonType})
	expectStatus(t, own, http.StatusCreated, "same body from the own origin")
	if count := itemCount(t, server, ""); count != 1 {
		t.Fatalf("%d items, want 1", count)
	}
}

// TC-002: Origin values that do not name the own origin.
func TestAPIRequestChecks_TC002_OriginsThatAreNotTheOwnOrigin(t *testing.T) {
	server, _ := openChecksServer(t, http.StatusServiceUnavailable)
	rows := []struct{ host, origin string }{
		{checkHost, "https://evil.example"},
		{checkHost, "http://evil.example"},
		{checkHost, "https://sub.prices.example"},
		{checkHost, "https://prices.example.evil.example"},
		{checkHost, "null"},
		{checkHost, ""},
		{checkHost, "prices.example"},
		{checkHost, "not an origin"},
		{checkHost, "https://prices.example:443"},
		{checkHost, "http://prices.example:8080"},
		{"prices.example:8443", "https://prices.example"},
		{"localhost:5173", "http://localhost:3001"},
		{"localhost:5173", "http://localhost"},
		{"127.0.0.1:4173", "http://localhost:4173"},
	}
	for n, row := range rows {
		what := fmt.Sprintf("Host %q Origin %q", row.host, row.origin)
		recorder := sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(n), checkHeaders{"Host": row.host, "Origin": row.origin, "Content-Type": jsonType})
		expectRefused(t, recorder, http.StatusForbidden, what)
		if count := itemCount(t, server, ""); count != 0 {
			t.Fatalf("%s: %d items, want none", what, count)
		}
	}
}

// TC-003: own origin and missing Origin pass the check.
func TestAPIRequestChecks_TC003_OwnOriginAndMissingOriginPass(t *testing.T) {
	server, _ := openChecksServer(t, http.StatusServiceUnavailable)
	rows := []struct {
		host, origin string
		sendOrigin   bool
	}{
		{checkHost, "https://prices.example", true},
		{checkHost, "http://prices.example", true},
		{checkHost, "http://PRICES.example", true},
		{checkHost, "HTTPS://Prices.Example", true},
		{checkHost, "", false},
		{"PRICES.EXAMPLE", "https://prices.example", true},
		{"localhost:5173", "http://localhost:5173", true},
		{"127.0.0.1:4173", "http://127.0.0.1:4173", true},
		{"[::1]:5173", "http://[::1]:5173", true},
	}
	for n, row := range rows {
		headers := checkHeaders{"Host": row.host, "Content-Type": jsonType}
		what := fmt.Sprintf("Host %q without Origin", row.host)
		if row.sendOrigin {
			headers["Origin"] = row.origin
			what = fmt.Sprintf("Host %q Origin %q", row.host, row.origin)
		}
		recorder := sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(n), headers)
		var item model.Item
		if recorder.Code != http.StatusCreated || json.Unmarshal(recorder.Body.Bytes(), &item) != nil || item.ID == "" {
			t.Fatalf("%s: %d %s, want 201 with the item", what, recorder.Code, recorder.Body.String())
		}
	}
	if count := itemCount(t, server, ""); count != len(rows) {
		t.Fatalf("%d items, want %d", count, len(rows))
	}
}

// TC-004: every state-changing route refuses another site and changes nothing.
func TestAPIRequestChecks_TC004_EveryStateChangingRouteRefusesAnotherSite(t *testing.T) {
	server, database := openChecksServer(t, http.StatusServiceUnavailable)
	seedPricedItem(t, database)
	before := stateSnapshot(t, server, true)
	for _, row := range []struct{ method, path, body string }{
		{http.MethodDelete, "/api/v1/items/lbc", ""},
		{http.MethodPost, "/api/v1/items/refresh", ""},
		{http.MethodPost, "/api/v1/items/lbc/refresh", ""},
		{http.MethodPost, "/api/v1/items/lbc/ai-review", ""},
		{http.MethodPut, "/api/v1/items/lbc/purchase-goal", `{"purchaseGoal":"Gaming"}`},
		{http.MethodPut, "/api/v1/settings", `{"claudeToken":{"remove":true}}`},
		{http.MethodPatch, "/api/v1/items/lbc", `{}`},
	} {
		headers := checkHeaders{"Origin": foreignOrigin}
		if row.body != "" {
			headers["Content-Type"] = jsonType
		}
		expectRefused(t, sendChecked(t, server, row.method, row.path, row.body, headers), http.StatusForbidden, row.method+" "+row.path)
	}
	if after := stateSnapshot(t, server, true); after != before {
		t.Fatalf("state changed by refused requests:\nbefore %s\nafter  %s", before, after)
	}
}

// TC-005: read requests and non-API paths are unchanged.
func TestAPIRequestChecks_TC005_ReadRequestsAndNonAPIPathsAreUnchanged(t *testing.T) {
	server, database := openChecksServer(t, http.StatusServiceUnavailable)
	insertItem(t, database, "lbc", "leboncoin")
	foreign := checkHeaders{"Origin": foreignOrigin, "Content-Type": "text/plain"}
	for _, row := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/items"},
		{http.MethodGet, "/api/v1/items/lbc"},
		{http.MethodGet, "/api/v1/settings"},
		{http.MethodGet, settingsPath},
		{http.MethodGet, claudeTokenPath},
		{http.MethodGet, sessionPath},
		{http.MethodHead, "/api/v1/items"},
		{http.MethodOptions, "/api/v1/items"},
	} {
		what := row.method + " " + row.path
		plain := sendChecked(t, server, row.method, row.path, "", checkHeaders{})
		withHeaders := sendChecked(t, server, row.method, row.path, "", foreign)
		if plain.Code != withHeaders.Code || plain.Body.String() != withHeaders.Body.String() {
			t.Fatalf("%s: %d %s without the headers, %d %s with them", what, plain.Code, plain.Body.String(), withHeaders.Code, withHeaders.Body.String())
		}
		if withHeaders.Code == http.StatusForbidden || withHeaders.Code == http.StatusUnsupportedMediaType {
			t.Fatalf("%s: refused %d %s", what, withHeaders.Code, withHeaders.Body.String())
		}
		noCORSHeaders(t, plain, what)
		noCORSHeaders(t, withHeaders, what+" with Origin")
	}
	plain := sendChecked(t, server, http.MethodPost, "/", "x", checkHeaders{})
	withHeaders := sendChecked(t, server, http.MethodPost, "/", "x", foreign)
	if plain.Code != withHeaders.Code || plain.Body.String() != withHeaders.Body.String() || withHeaders.Code == http.StatusForbidden || withHeaders.Code == http.StatusUnsupportedMediaType {
		t.Fatalf("POST /: %d %s without the headers, %d %s with them", plain.Code, plain.Body.String(), withHeaders.Code, withHeaders.Body.String())
	}
	noCORSHeaders(t, withHeaders, "POST /")
	refused := sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), checkHeaders{"Origin": foreignOrigin, "Content-Type": jsonType})
	expectRefused(t, refused, http.StatusForbidden, "foreign add")
	noCORSHeaders(t, refused, "foreign add")
}

// sessionRevision reads the LeBoncoin session object of Settings: saved and revision.
func sessionRevision(t *testing.T, server *Server) (bool, float64) {
	t.Helper()
	session := decodeSession(t, call(t, server, http.MethodGet, ""))
	saved, _ := session["saved"].(bool)
	revision, _ := session["revision"].(float64)
	return saved, revision
}

// TC-006: saving settings with a body not declared as JSON is refused.
func TestAPIRequestChecks_TC006_SettingsBodyNotDeclaredAsJSONIsRefused(t *testing.T) {
	server, _ := openChecksServer(t, http.StatusServiceUnavailable)
	const body = `{"leboncoinSession":{"value":"Cookie: a=1; datadome=synthetic; b=2","revision":0}}`
	for _, contentType := range []string{"text/plain", "text/plain; charset=UTF-8", "application/x-www-form-urlencoded", "multipart/form-data; boundary=x",
		"application/merge-patch+json", "text/json", "application/jsonx", "", "<none>"} {
		headers := checkHeaders{"Content-Type": contentType}
		if contentType == "<none>" {
			headers = checkHeaders{}
		}
		what := fmt.Sprintf("Content-Type %q", contentType)
		expectRefused(t, sendChecked(t, server, http.MethodPut, "/api/v1/settings", body, headers), http.StatusUnsupportedMediaType, what)
		if saved, revision := sessionRevision(t, server); saved || revision != 0 {
			t.Fatalf("%s: session saved=%v revision=%v, want none at revision 0", what, saved, revision)
		}
	}
	saved := sendChecked(t, server, http.MethodPut, "/api/v1/settings", body, checkHeaders{"Content-Type": jsonType})
	expectStatus(t, saved, http.StatusOK, "JSON save")
	if isSaved, revision := sessionRevision(t, server); !isSaved || revision != 1 {
		t.Fatalf("after the JSON save: saved=%v revision=%v, want saved at revision 1", isSaved, revision)
	}
}

// TC-007: a non-empty body on any route needs the JSON content type.
func TestAPIRequestChecks_TC007_NonEmptyBodyOnAnyRouteNeedsJSON(t *testing.T) {
	server, database := openChecksServer(t, http.StatusServiceUnavailable)
	seedPricedItem(t, database)
	before := stateSnapshot(t, server, true)
	for _, row := range []struct {
		method, path, body string
		headers            checkHeaders
	}{
		{http.MethodPost, "/api/v1/items/refresh", "x", checkHeaders{"Content-Type": "text/plain"}},
		{http.MethodPost, "/api/v1/items/lbc/refresh", "{}", checkHeaders{}},
		{http.MethodPost, "/api/v1/items/lbc/ai-review", "x", checkHeaders{"Content-Type": "text/plain"}},
		{http.MethodDelete, "/api/v1/items/lbc", "a=1", checkHeaders{"Content-Type": "application/x-www-form-urlencoded"}},
	} {
		expectRefused(t, sendChecked(t, server, row.method, row.path, row.body, row.headers), http.StatusUnsupportedMediaType, row.method+" "+row.path)
	}
	// A body of unknown length (chunked).
	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", io.MultiReader(strings.NewReader(addBody(1))))
	req.Host = checkHost
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	req.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)
	expectRefused(t, recorder, http.StatusUnsupportedMediaType, "add with a body of unknown length")
	if after := stateSnapshot(t, server, true); after != before {
		t.Fatalf("state changed by refused requests:\nbefore %s\nafter  %s", before, after)
	}
	if count := itemCount(t, server, ""); count != 1 {
		t.Fatalf("%d items, want only lbc", count)
	}
}

// TC-008: a JSON body is handled by the route as before.
func TestAPIRequestChecks_TC008_JSONBodyIsHandledAsBefore(t *testing.T) {
	server, _ := openChecksServer(t, http.StatusServiceUnavailable)
	n := 0
	var added model.Item
	for _, origin := range []string{ownOrigin, ""} {
		for _, contentType := range []string{"application/json", "application/json; charset=utf-8", "Application/JSON; charset=UTF-8", "APPLICATION/JSON"} {
			headers := checkHeaders{"Content-Type": contentType}
			if origin != "" {
				headers["Origin"] = origin
			}
			n++
			recorder := sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(n), headers)
			if recorder.Code != http.StatusCreated || json.Unmarshal(recorder.Body.Bytes(), &added) != nil || added.ID == "" {
				t.Fatalf("Origin %q Content-Type %q: %d %s, want 201 with the item", origin, contentType, recorder.Code, recorder.Body.String())
			}
		}
	}
	asJSON := checkHeaders{"Content-Type": jsonType}
	assertError(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", `{"url":`, asJSON), http.StatusBadRequest, "INVALID_JSON", "")
	assertError(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", `{}`, asJSON), http.StatusBadRequest, "INVALID_URL", "")
	tooLarge := `{"url":"https://www.leboncoin.fr/ad/test/` + strings.Repeat("9", maxGoalBodyBytes) + `"}`
	assertError(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", tooLarge, asJSON), http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "")
	expectStatus(t, sendChecked(t, server, http.MethodPut, "/api/v1/settings", `{"leboncoinSession":{"value":"Cookie: a=1; datadome=synthetic; b=2","revision":0}}`, asJSON), http.StatusOK, "settings save")
	assertError(t, sendChecked(t, server, http.MethodPut, "/api/v1/settings", `{"leboncoinSession":`, asJSON), http.StatusBadRequest, "INVALID_JSON", "")
	goal := sendChecked(t, server, http.MethodPut, "/api/v1/items/"+added.ID+"/purchase-goal", `{"purchaseGoal":"Gaming"}`, asJSON)
	var saved struct{ PurchaseGoal string }
	if goal.Code != http.StatusOK || json.Unmarshal(goal.Body.Bytes(), &saved) != nil || saved.PurchaseGoal != "Gaming" {
		t.Fatalf("purchase goal: %d %s", goal.Code, goal.Body.String())
	}
	// The content-type check comes before the body is read.
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", `{"url":`, checkHeaders{"Content-Type": "text/plain"}), http.StatusUnsupportedMediaType, "invalid JSON as text/plain")
}

// TC-009: requests without a body need no content type.
func TestAPIRequestChecks_TC009_RequestsWithoutABodyNeedNoContentType(t *testing.T) {
	for _, variant := range []struct {
		name    string
		headers checkHeaders
	}{
		{"no body and no Content-Type", checkHeaders{}},
		{"empty body as text/plain", checkHeaders{"Content-Type": "text/plain"}},
		{"explicit Content-Length 0", checkHeaders{"Content-Length": "0"}},
	} {
		t.Run(variant.name, func(t *testing.T) {
			server, database := openChecksServer(t, http.StatusServiceUnavailable)
			seedPricedItem(t, database)
			expectStatus(t, sendChecked(t, server, http.MethodPost, "/api/v1/items/refresh", "", variant.headers), http.StatusAccepted, "refresh all")
			expectStatus(t, sendChecked(t, server, http.MethodPost, "/api/v1/items/lbc/refresh", "", variant.headers), http.StatusAccepted, "refresh lbc")
			expectStatus(t, sendChecked(t, server, http.MethodPost, "/api/v1/items/lbc/ai-review", "", variant.headers), http.StatusAccepted, "AI review")
			waitReviewDone(t, server, "lbc")
			expectStatus(t, sendChecked(t, server, http.MethodDelete, "/api/v1/items/lbc", "", variant.headers), http.StatusNoContent, "delete")
			assertError(t, sendChecked(t, server, http.MethodDelete, "/api/v1/items/lbc", "", variant.headers), http.StatusNotFound, "ITEM_NOT_FOUND", "")

			protected := newCookieServer(t)
			token := login(t, protected, cookieUsername, cookiePassword)
			recorder := sendChecked(t, protected, http.MethodPost, logoutPath, "", withCookie(variant.headers, token))
			expectStatus(t, recorder, http.StatusNoContent, "logout")
			if cookie, _ := sessionSetCookie(t, recorder.Header(), "logout"); cookie.Value != "" || cookie.MaxAge >= 0 {
				t.Fatalf("logout cookie %+v, want the clearing cookie", cookie)
			}
			session := request(t, protected, http.MethodGet, sessionPath, "", token)
			if !strings.Contains(session.Body.String(), `"user":null`) {
				t.Fatalf("session after logout: %s", session.Body.String())
			}
		})
	}
}

// TC-010: when both checks fail, the origin refusal is given.
func TestAPIRequestChecks_TC010_OriginRefusalComesFirst(t *testing.T) {
	server, _ := openChecksServer(t, http.StatusServiceUnavailable)
	before := stateSnapshot(t, server, false)
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), checkHeaders{"Origin": foreignOrigin, "Content-Type": "text/plain"}), http.StatusForbidden, "add as text/plain")
	expectRefused(t, sendChecked(t, server, http.MethodPut, "/api/v1/settings", "a=1", checkHeaders{"Origin": foreignOrigin, "Content-Type": "application/x-www-form-urlencoded"}), http.StatusForbidden, "settings as a form")
	if after := stateSnapshot(t, server, false); after != before {
		t.Fatalf("state changed:\nbefore %s\nafter  %s", before, after)
	}
	if count := itemCount(t, server, ""); count != 0 {
		t.Fatalf("%d items, want none", count)
	}

	protected := newCookieServer(t)
	recorder := sendChecked(t, protected, http.MethodPost, loginPath, cookieLogin, checkHeaders{"Origin": foreignOrigin})
	expectRefused(t, recorder, http.StatusForbidden, "login without Content-Type")
	noSessionSetCookie(t, recorder.Header(), "refused login")
}

// sessionUsername answers the username of GET /api/v1/auth/session with the cookie, "" for user null.
func sessionUsername(t *testing.T, server *Server, token string) string {
	t.Helper()
	recorder := request(t, server, http.MethodGet, sessionPath, "", token)
	var body struct{ User *userJSON }
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &body) != nil {
		t.Fatalf("session: %d %s", recorder.Code, recorder.Body.String())
	}
	if body.User == nil {
		return ""
	}
	return body.User.Username
}

// adminUsers answers the users of GET /api/v1/admin/users by username.
func adminUsers(t *testing.T, server *Server, admin string) map[string]adminUserJSON {
	t.Helper()
	recorder := request(t, server, http.MethodGet, "/api/v1/admin/users", "", admin)
	var body struct{ Users []adminUserJSON }
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &body) != nil {
		t.Fatalf("admin users: %d %s", recorder.Code, recorder.Body.String())
	}
	users := map[string]adminUserJSON{}
	for _, user := range body.Users {
		users[user.Username] = user
	}
	return users
}

// TC-011: refusals come before authentication, routing and method handling.
func TestAPIRequestChecks_TC011_RefusalsComeBeforeAuthenticationAndRouting(t *testing.T) {
	withClaudeStatus(t, http.StatusServiceUnavailable)
	server := newCookieServer(t)
	adminPassword, _, err := server.service.ResetAdminPassword(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	alice := login(t, server, cookieUsername, cookiePassword)
	admin := login(t, server, "admin", adminPassword)
	foreignJSON := checkHeaders{"Origin": foreignOrigin, "Content-Type": jsonType}
	plainText := checkHeaders{"Content-Type": "text/plain"}

	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), foreignJSON), http.StatusForbidden, "logged out, foreign add")
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), plainText), http.StatusUnsupportedMediaType, "logged out, add as text/plain")
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), withCookie(foreignJSON, alice)), http.StatusForbidden, "alice, foreign add")
	if count := itemCount(t, server, alice); count != 0 {
		t.Fatalf("alice has %d items, want none", count)
	}
	expectRefused(t, sendChecked(t, server, http.MethodPost, logoutPath, "", withCookie(checkHeaders{"Origin": foreignOrigin}, alice)), http.StatusForbidden, "alice, foreign logout")
	if username := sessionUsername(t, server, alice); username != cookieUsername {
		t.Fatalf("session after the refused logout: %q, want alice", username)
	}
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/admin/users", newUserBody, foreignJSON), http.StatusForbidden, "logged out, foreign user creation")
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/admin/users", newUserBody, withCookie(foreignJSON, admin)), http.StatusForbidden, "administrator, foreign user creation")
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/admin/users", newUserBody, withCookie(plainText, admin)), http.StatusUnsupportedMediaType, "administrator, user creation as text/plain")
	if _, found := adminUsers(t, server, admin)["bob"]; found {
		t.Fatal("bob was created by a refused request")
	}
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/admin/users", newUserBody, withCookie(foreignJSON, alice)), http.StatusForbidden, "alice, foreign user creation")
	expectRefused(t, sendChecked(t, server, http.MethodPost, "/api/v1/unknown", "", checkHeaders{"Origin": foreignOrigin}), http.StatusForbidden, "unknown path")
	expectRefused(t, sendChecked(t, server, http.MethodPatch, "/api/v1/items/refresh", "x", plainText), http.StatusUnsupportedMediaType, "PATCH refresh as text/plain")
}

// TC-012: a refused login neither logs in nor counts as a failed login.
func TestAPIRequestChecks_TC012_RefusedLoginNeitherLogsInNorCounts(t *testing.T) {
	const remote = "198.51.100.20:4000"
	wrong := `{"username":"alice","password":"wrong password"}`
	right := `{"username":"alice","password":"` + limitPassword + `"}`
	foreignJSON := checkHeaders{"RemoteAddr": remote, "Origin": foreignOrigin, "Content-Type": jsonType}
	plainText := checkHeaders{"RemoteAddr": remote, "Content-Type": "text/plain"}
	ownJSON := checkHeaders{"RemoteAddr": remote, "Origin": ownOrigin, "Content-Type": jsonType}
	refuseTwelve := func(t *testing.T, server *Server) {
		t.Helper()
		for attempt := 1; attempt <= 6; attempt++ {
			recorder := sendChecked(t, server, http.MethodPost, loginPath, wrong, foreignJSON)
			expectRefused(t, recorder, http.StatusForbidden, fmt.Sprintf("foreign wrong login %d", attempt))
		}
		for attempt := 1; attempt <= 6; attempt++ {
			recorder := sendChecked(t, server, http.MethodPost, loginPath, wrong, plainText)
			expectRefused(t, recorder, http.StatusUnsupportedMediaType, fmt.Sprintf("text/plain wrong login %d", attempt))
		}
	}

	t.Run("refused logins", func(t *testing.T) {
		server := newLimitServer(t)
		adminPassword, _, err := server.service.ResetAdminPassword(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		admin := login(t, server, "admin", adminPassword)
		recorder := sendChecked(t, server, http.MethodPost, loginPath, right, foreignJSON)
		expectRefused(t, recorder, http.StatusForbidden, "foreign login with the right password")
		if lines := recorder.Header().Values("Set-Cookie"); len(lines) != 0 {
			t.Fatalf("refused login sets cookies %q", lines)
		}
		if user, found := adminUsers(t, server, admin)["alice"]; !found || user.LastLoginAt != nil {
			t.Fatalf("alice %+v found=%v, want lastLoginAt null", user, found)
		}
		refuseTwelve(t, server)
		expectHTTPLogin(t, sendChecked(t, server, http.MethodPost, loginPath, right, ownJSON), "same-origin login after the refused ones")
	})

	t.Run("refused logins are not counted", func(t *testing.T) {
		server := newLimitServer(t)
		for attempt := 1; attempt <= 4; attempt++ {
			expectHTTPInvalid(t, sendChecked(t, server, http.MethodPost, loginPath, wrong, ownJSON), fmt.Sprintf("real failure %d", attempt))
		}
		refuseTwelve(t, server)
		expectHTTPInvalid(t, sendChecked(t, server, http.MethodPost, loginPath, wrong, ownJSON), "fifth real failure")
		expectHTTPLocked(t, sendChecked(t, server, http.MethodPost, loginPath, right, ownJSON), "after five real failures")
	})
}

// flowClient sends the requests the frontend sends: the JSON content type only with a body,
// and the page's Origin only on state-changing requests.
type flowClient struct {
	t                  *testing.T
	client             *http.Client
	base, host, origin string
	forwardedProto     string
	sendOrigin         bool
}

func (c *flowClient) do(method, path, body string, want int, keys ...string) map[string]json.RawMessage {
	c.t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Host = c.host
	req.Header.Set("Accept", jsonType)
	if body != "" {
		req.Header.Set("Content-Type", jsonType)
	}
	if c.sendOrigin && method != http.MethodGet {
		req.Header.Set("Origin", c.origin)
	}
	if c.forwardedProto != "" {
		req.Header.Set("X-Forwarded-Proto", c.forwardedProto)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	if resp.StatusCode != want {
		c.t.Fatalf("%s %s: %d %s, want %d", method, path, resp.StatusCode, data, want)
	}
	var object map[string]json.RawMessage
	if want != http.StatusNoContent {
		if err := json.Unmarshal(data, &object); err != nil {
			c.t.Fatalf("%s %s: body %s: %v", method, path, data, err)
		}
	}
	for _, key := range keys {
		if _, ok := object[key]; !ok {
			c.t.Fatalf("%s %s: no %q in %s", method, path, key, data)
		}
	}
	return object
}

// TC-013: the application's requests work behind the proxy and in development.
func TestAPIRequestChecks_TC013_ApplicationRequestsWorkBehindTheProxyAndInDevelopment(t *testing.T) {
	for _, setup := range []string{"proxy", "development"} {
		for _, protected := range []bool{false, true} {
			for _, sendOrigin := range []bool{true, false} {
				name := fmt.Sprintf("%s protected=%v origin=%v", setup, protected, sendOrigin)
				t.Run(name, func(t *testing.T) { runApplicationFlow(t, setup, protected, sendOrigin) })
			}
		}
	}
}

func runApplicationFlow(t *testing.T, setup string, protected, sendOrigin bool) {
	withClaudeStatus(t, http.StatusOK) // Claude accepts the token; the listing fetch never leaves the test
	server, _ := newSettingsServer(t)
	ctx := context.Background()
	if protected {
		if _, err := server.service.CreateUser(ctx, cookieUsername, cookiePassword); err != nil {
			t.Fatal(err)
		}
	}
	adminPassword, _, err := server.service.ResetAdminPassword(ctx)
	if err != nil {
		t.Fatal(err)
	}
	backend := httptest.NewServer(server.Handler())
	t.Cleanup(backend.Close)
	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	// A reverse proxy in front keeps the browser's Host, as Cloudflare (HTTPS ends there)
	// and the Vite dev server proxy without changeOrigin do.
	frontProxy := httputil.NewSingleHostReverseProxy(target)
	proxyTransport := &http.Transport{}
	t.Cleanup(proxyTransport.CloseIdleConnections)
	frontProxy.Transport = proxyTransport
	var front *httptest.Server
	host, origin, proto := checkHost, ownOrigin, "https"
	if setup == "development" {
		front = httptest.NewServer(frontProxy)
		host, origin, proto = "localhost:5173", "http://localhost:5173", ""
	} else {
		front = httptest.NewTLSServer(frontProxy)
	}
	t.Cleanup(front.Close)
	newClient := func() *flowClient {
		jar, err := cookiejar.New(nil)
		if err != nil {
			t.Fatal(err)
		}
		transport := front.Client().Transport.(*http.Transport).Clone()
		t.Cleanup(transport.CloseIdleConnections)
		return &flowClient{t: t, client: &http.Client{Transport: transport, Jar: jar}, base: front.URL, host: host, origin: origin, forwardedProto: proto, sendOrigin: sendOrigin}
	}

	user := newClient()
	if protected {
		user.do(http.MethodPost, loginPath, cookieLogin, http.StatusOK, "user")
	}
	added := user.do(http.MethodPost, "/api/v1/items", addBody(1), http.StatusCreated, "id")
	var id string
	if err := json.Unmarshal(added["id"], &id); err != nil || id == "" {
		t.Fatalf("added item id %s", added["id"])
	}
	user.do(http.MethodPut, "/api/v1/items/"+id+"/purchase-goal", `{"purchaseGoal":"Gaming"}`, http.StatusOK, "purchaseGoal")
	user.do(http.MethodPut, "/api/v1/settings", `{"claudeToken":{"value":"sk-ant-oat01-synthetic"}}`, http.StatusOK, "session", "claudeToken")
	user.do(http.MethodPost, "/api/v1/items/"+id+"/refresh", "", http.StatusAccepted, "requestedAt")
	user.do(http.MethodPost, "/api/v1/items/refresh", "", http.StatusAccepted, "requestedAt", "itemsQueued")
	user.do(http.MethodPost, "/api/v1/items/"+id+"/ai-review", "", http.StatusAccepted, "requestedAt", "alreadyRunning")
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		var review struct{ Running bool }
		if err := json.Unmarshal(user.do(http.MethodGet, "/api/v1/items/"+id, "", http.StatusOK, "aiReview")["aiReview"], &review); err != nil {
			t.Fatal(err)
		}
		if !review.Running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the AI review did not finish")
		}
	}
	user.do(http.MethodDelete, "/api/v1/items/"+id, "", http.StatusNoContent)
	if protected {
		user.do(http.MethodPost, logoutPath, "", http.StatusNoContent)
	}

	admin := newClient()
	admin.do(http.MethodPost, loginPath, `{"username":"admin","password":"`+adminPassword+`"}`, http.StatusOK, "user")
	admin.do(http.MethodPost, "/api/v1/admin/users", newUserBody, http.StatusCreated, "user")
}

// TC-015: API_SPECIFICATION.md documents both rules.
func TestAPIRequestChecks_TC015_APISpecificationDocumentsBothRules(t *testing.T) {
	data, err := os.ReadFile("../../API_SPECIFICATION.md")
	if err != nil {
		t.Fatal(err)
	}
	const heading = "\n## Conventions\n"
	text := string(data)
	start := strings.Index(text, heading)
	if start < 0 {
		t.Fatal("API_SPECIFICATION.md has no Conventions section")
	}
	conventions := text[start+len(heading):]
	if end := strings.Index(conventions, "\n## "); end >= 0 {
		conventions = conventions[:end]
	}
	for _, token := range []string{"ORIGIN_NOT_ALLOWED", "403", originRefusal, "UNSUPPORTED_MEDIA_TYPE", "415", mediaTypeRefusal,
		"application/json", "Origin", "Host", "`GET`", "`HEAD`", "`OPTIONS`"} {
		if !strings.Contains(conventions, token) {
			t.Fatalf("the Conventions section does not contain %q", token)
		}
	}
	if strings.Index(conventions, "ORIGIN_NOT_ALLOWED") > strings.Index(conventions, "UNSUPPORTED_MEDIA_TYPE") {
		t.Fatal("the origin rule is not stated before the content-type rule")
	}

	// The documented codes and messages are those the server answers (TC-001 and TC-006 requests).
	server, _ := openChecksServer(t, http.StatusServiceUnavailable)
	for _, recorder := range []*httptest.ResponseRecorder{
		sendChecked(t, server, http.MethodPost, "/api/v1/items", addBody(1), checkHeaders{"Origin": foreignOrigin, "Content-Type": jsonType}),
		sendChecked(t, server, http.MethodPut, "/api/v1/settings", `{"leboncoinSession":{"value":"Cookie: a=1; datadome=synthetic; b=2","revision":0}}`, checkHeaders{"Content-Type": "text/plain"}),
	} {
		var body struct {
			Error struct{ Code, Message string } `json:"error"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Error.Code == "" ||
			!strings.Contains(conventions, fmt.Sprintf("%d", recorder.Code)) || !strings.Contains(conventions, body.Error.Code) || !strings.Contains(conventions, `"`+body.Error.Message+`"`) {
			t.Fatalf("server answer %d %s is not the one documented in Conventions", recorder.Code, recorder.Body.String())
		}
		if recorder.Code != http.StatusForbidden && recorder.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("server answer %d %s, want a documented refusal", recorder.Code, recorder.Body.String())
		}
	}
}
