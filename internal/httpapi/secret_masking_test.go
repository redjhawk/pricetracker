package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

// Issue #265: the saved Claude token and LeBoncoin session never leave the server;
// Settings objects carry "saved" and "hint" instead.

// Synthetic secrets of the test cases; never real tokens or cookies.
const (
	secretT     = "sk-ant-oat01-synthetic-token-wxyz" // saved Claude token
	secretU     = "sk-ant-oat01-synthetic-new-uvst"   // new Claude token
	secretS     = "synthetic-session-value-9f3a"      // saved LeBoncoin session
	secretN     = "synthetic-renewed-or-new-n0p1"     // new LeBoncoin session
	secretY     = "synthetic-renewed-value-y7k2"      // session stored by a renewal
	secretShort = "Qx7Zk9"                            // shorter than model.SecretHintMinLength
)

const (
	settingsSavePath       = "/api/v1/settings"
	claudeTokenPath        = "/api/v1/settings/claude-token"
	invalidSettingsRequest = "Each Settings entry sent must have either a value or \"remove\": true, and the LeBoncoin session needs the revision from Settings."
	sessionChangedMessage  = "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving."
	maskedListingHTML      = `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","price":[12]}}}}</script>`
)

// secretNet answers every outgoing request: LeBoncoin with a listing page, Claude
// with claudeStatus (0: network error). It records the Cookie header of each
// LeBoncoin request and the token of each Claude verification and review.
type secretNet struct {
	mu           sync.Mutex
	claudeStatus int
	verified     []string
	reviewed     []string
	cookies      []string
}

func withSecretNet(t *testing.T, claudeStatus int) *secretNet {
	t.Helper()
	fake := &secretNet{claudeStatus: claudeStatus}
	original := http.DefaultTransport
	http.DefaultTransport = claudeTransport(fake.roundTrip)
	t.Cleanup(func() { http.DefaultTransport = original })
	return fake
}

func (n *secretNet) roundTrip(r *http.Request) (*http.Response, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if r.URL.Hostname() == "api.anthropic.com" {
		body, _ := io.ReadAll(r.Body)
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if strings.Contains(string(body), `"max_tokens":1,`) {
			n.verified = append(n.verified, token)
		} else {
			n.reviewed = append(n.reviewed, token)
		}
		if n.claudeStatus == 0 {
			return nil, errors.New("synthetic network error")
		}
		return &http.Response{StatusCode: n.claudeStatus, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"content":[]}`)), Request: r}, nil
	}
	n.cookies = append(n.cookies, r.Header.Get("Cookie"))
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/html"}}, Body: io.NopCloser(strings.NewReader(maskedListingHTML)), Request: r}, nil
}

func (n *secretNet) setClaude(status int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.claudeStatus = status
}

func (n *secretNet) verifications() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.verified...)
}

func (n *secretNet) reviews() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.reviewed...)
}

func (n *secretNet) sentCookies() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.cookies...)
}

func text(value string) *string { return &value }

func captureSecretLogs(t *testing.T) *syncBuffer {
	t.Helper()
	logs := &syncBuffer{}
	previous := log.Writer()
	log.SetOutput(logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	return logs
}

// assertNoSecret fails when output contains a secret or any 5-character part of it
// (3 characters for a secret shorter than the hint threshold). Parts made only of
// letters are ordinary words of the synthetic values ("token", "session") that
// responses contain, and parts of "setup-token" come from the fixed Claude
// messages, so neither is checked.
func assertNoSecret(t *testing.T, what, output string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if strings.Contains(output, secret) {
			t.Fatalf("%s contains the secret %q: %s", what, secret, output)
		}
		window := 5
		if len(secret) < model.SecretHintMinLength {
			window = 3
		}
		for start := 0; start+window <= len(secret); start++ {
			part := secret[start : start+window]
			if strings.IndexFunc(part, func(r rune) bool { return !unicode.IsLetter(r) }) < 0 || strings.Contains("setup-token", part) {
				continue
			}
			if strings.Contains(output, part) {
				t.Fatalf("%s contains %q, part of the secret %q: %s", what, part, secret, output)
			}
		}
	}
}

// responseText is the headers and body of a response, as checked for secrets.
func responseText(recorder *httptest.ResponseRecorder) string {
	return fmt.Sprintf("%v\n%s", recorder.Header(), recorder.Body.String())
}

// storeSecrets saves the session and the token (nil: unchanged) through the store, as
// earlier versions saved them.
func storeSecrets(t *testing.T, database *store.Store, owner int64, session, token *string) {
	t.Helper()
	ctx := context.Background()
	var sessionChange *store.SessionChange
	if session != nil {
		current, err := database.LeboncoinSession(ctx, owner)
		if err != nil {
			t.Fatal(err)
		}
		sessionChange = &store.SessionChange{Value: session, ExpectedRevision: current.Revision}
	}
	var tokenChange *store.TokenChange
	if token != nil {
		tokenChange = &store.TokenChange{Value: token}
	}
	if _, _, err := database.SaveSettings(ctx, owner, sessionChange, tokenChange, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

// rejectStoredToken sets lastRejectedAt on the stored token, as a rejected AI review does.
func rejectStoredToken(t *testing.T, database *store.Store, owner int64) {
	t.Helper()
	token, err := database.ClaudeToken(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.MarkClaudeTokenRejected(context.Background(), owner, token.Revision, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

// finishSessionAttempt records a collection outcome on the stored session, as the collector does.
func finishSessionAttempt(t *testing.T, database *store.Store, owner int64, outcome store.LeboncoinSessionOutcome) {
	t.Helper()
	current, err := database.LeboncoinSession(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := database.FinishLeboncoinSessionAttempt(context.Background(), owner, current.Revision, outcome, time.Now().UTC())
	if err != nil || !applied {
		t.Fatalf("attempt not recorded: %v %v", applied, err)
	}
}

func storedSecrets(t *testing.T, database *store.Store, owner int64) (model.LeboncoinSession, model.ClaudeToken) {
	t.Helper()
	session, err := database.LeboncoinSession(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	token, err := database.ClaudeToken(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	return session, token
}

func assertStoredSession(t *testing.T, database *store.Store, owner int64, want model.LeboncoinSession) {
	t.Helper()
	got, _ := storedSecrets(t, database, owner)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stored session changed:\n got %+v\nwant %+v", got, want)
	}
}

func assertStoredToken(t *testing.T, database *store.Store, owner int64, want model.ClaudeToken) {
	t.Helper()
	_, got := storedSecrets(t, database, owner)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stored token changed:\n got %+v\nwant %+v", got, want)
	}
}

func storedValue(value *string) string {
	if value == nil {
		return "<none>"
	}
	return *value
}

func decodeObject(t *testing.T, recorder *httptest.ResponseRecorder, key string) map[string]any {
	t.Helper()
	var envelope map[string]map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("body %q: %v", recorder.Body.String(), err)
	}
	object, ok := envelope[key]
	if !ok {
		t.Fatalf("no %q in %s", key, recorder.Body.String())
	}
	return object
}

// loadSettings loads both Settings objects (auth "" in open mode) and checks the common response rules.
func loadSettings(t *testing.T, server *Server, auth string) (session, token map[string]any) {
	t.Helper()
	sessionRecorder := request(t, server, http.MethodGet, settingsPath, "", auth)
	tokenRecorder := request(t, server, http.MethodGet, claudeTokenPath, "", auth)
	for _, recorder := range []*httptest.ResponseRecorder{sessionRecorder, tokenRecorder} {
		if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("load: status %d, Cache-Control %q: %s", recorder.Code, recorder.Header().Get("Cache-Control"), recorder.Body.String())
		}
	}
	session = decodeObject(t, sessionRecorder, "session")
	token = decodeObject(t, tokenRecorder, "claudeToken")
	assertMaskedObjects(t, session, token)
	return session, token
}

// assertMaskedObjects checks that the objects have exactly the fields of the contract (no "value").
func assertMaskedObjects(t *testing.T, session, token map[string]any) {
	t.Helper()
	for _, check := range []struct {
		name   string
		object map[string]any
		keys   []string
	}{
		{"session", session, []string{"saved", "hint", "revision", "updatedAt", "status", "expiresAt", "revokedAt", "lastAttempt"}},
		{"claudeToken", token, []string{"saved", "hint", "updatedAt", "lastRejectedAt"}},
	} {
		if len(check.object) != len(check.keys) {
			t.Fatalf("%s has fields %v, want %v", check.name, check.object, check.keys)
		}
		for _, key := range check.keys {
			if _, ok := check.object[key]; !ok {
				t.Fatalf("%s has no %q: %v", check.name, key, check.object)
			}
		}
	}
}

// assertHint checks the saved and hint fields of a Settings object (hint "": null).
func assertHint(t *testing.T, what string, object map[string]any, saved bool, hint string) {
	t.Helper()
	var want any
	if hint != "" {
		want = hint
	}
	if object["saved"] != saved || object["hint"] != want {
		t.Fatalf("%s: saved=%v hint=%v, want saved=%v hint=%v (%v)", what, object["saved"], object["hint"], saved, want, object)
	}
}

func putSettings(t *testing.T, server *Server, body, auth string) *httptest.ResponseRecorder {
	t.Helper()
	return request(t, server, http.MethodPut, settingsSavePath, body, auth)
}

// saveSettingsOK sends a save that must succeed and returns the masked objects of the answer.
func saveSettingsOK(t *testing.T, server *Server, body, auth string) (session, token map[string]any) {
	t.Helper()
	recorder := putSettings(t, server, body, auth)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("save %s: status %d: %s", body, recorder.Code, recorder.Body.String())
	}
	session = decodeObject(t, recorder, "session")
	token = decodeObject(t, recorder, "claudeToken")
	assertMaskedObjects(t, session, token)
	return session, token
}

func sessionPart(value string, revision int64) string {
	encoded, _ := json.Marshal(value)
	return fmt.Sprintf(`"leboncoinSession":{"value":%s,"revision":%d}`, encoded, revision)
}

func sessionRemoval(revision int64) string {
	return fmt.Sprintf(`"leboncoinSession":{"remove":true,"revision":%d}`, revision)
}

func tokenPart(value string) string {
	encoded, _ := json.Marshal(value)
	return fmt.Sprintf(`"claudeToken":{"value":%s}`, encoded)
}

const tokenRemoval = `"claudeToken":{"remove":true}`

func settingsBody(parts ...string) string { return "{" + strings.Join(parts, ",") + "}" }

// runAIReview requests an AI review of item id and waits until it has finished.
func runAIReview(t *testing.T, server *Server, id, auth string) string {
	t.Helper()
	if recorder := request(t, server, http.MethodPost, "/api/v1/items/"+id+"/ai-review", "", auth); recorder.Code != http.StatusAccepted {
		t.Fatalf("AI review not started: %d %s", recorder.Code, recorder.Body.String())
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		body := request(t, server, http.MethodGet, "/api/v1/items/"+id, "", auth).Body.String()
		if strings.Contains(body, `"running":false`) && strings.Contains(body, `"status":"failed"`) {
			return body
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("AI review did not finish")
	return ""
}

func assertLastCookie(t *testing.T, fake *secretNet, want string) {
	t.Helper()
	cookies := fake.sentCookies()
	if len(cookies) == 0 || cookies[len(cookies)-1] != want {
		t.Fatalf("LeBoncoin cookies %q, want the last to be %q", cookies, want)
	}
}

func assertLastReviewToken(t *testing.T, fake *secretNet, want string) {
	t.Helper()
	reviews := fake.reviews()
	if len(reviews) == 0 || reviews[len(reviews)-1] != want {
		t.Fatalf("tokens sent for AI reviews %q, want the last to be %q", reviews, want)
	}
}

// protectedUsers creates alice and bob (protected mode) and returns their IDs and session cookies.
func protectedUsers(t *testing.T, server *Server, database *store.Store) (aliceID, bobID int64, alice, bob string) {
	t.Helper()
	ctx := context.Background()
	adminPassword, _, err := server.service.ResetAdminPassword(ctx)
	if err != nil {
		t.Fatal(err)
	}
	admin := login(t, server, "admin", adminPassword)
	for _, name := range []string{"alice", "bob"} {
		if recorder := request(t, server, http.MethodPost, "/api/v1/admin/users", `{"username":"`+name+`","password":"long enough pw"}`, admin); recorder.Code != http.StatusCreated {
			t.Fatalf("create %s: %s", name, recorder.Body.String())
		}
	}
	aliceUser, err := database.UserByUsername(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	bobUser, err := database.UserByUsername(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}
	return aliceUser.ID, bobUser.ID, login(t, server, "alice", "long enough pw"), login(t, server, "bob", "long enough pw")
}

// openSecretServer opens the store and the server on dir; stop closes both (also at cleanup).
func openSecretServer(t *testing.T, dir string) (*Server, *store.Store, func()) {
	t.Helper()
	cfg := config.Config{DataDirectory: dir, Location: time.UTC, CheckTimes: []string{"08:00"}, StaleAfter: 36 * time.Hour}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	items := service.New(cfg, database)
	var once sync.Once
	stop := func() {
		once.Do(func() {
			items.Close()
			database.Close()
		})
	}
	t.Cleanup(stop)
	return New(cfg, items), database, stop
}

// TC-001
func TestSecretMasking_TC001_LoadReturnsSavedAndHintNeverTheSecret(t *testing.T) {
	server, database := newSettingsServer(t)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	rejectStoredToken(t, database, 0)
	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "rejected"})
	storedSession, _ := storedSecrets(t, database, 0)

	sessionRecorder := request(t, server, http.MethodGet, settingsPath, "", "")
	tokenRecorder := request(t, server, http.MethodGet, claudeTokenPath, "", "")
	assertNoSecret(t, "session load", responseText(sessionRecorder), secretS, secretT)
	assertNoSecret(t, "token load", responseText(tokenRecorder), secretS, secretT)
	session, token := loadSettings(t, server, "")
	assertHint(t, "session", session, true, "9f3a")
	assertHint(t, "token", token, true, "wxyz")
	if session["revision"] != float64(storedSession.Revision) || session["status"] != "active" || session["updatedAt"] == nil ||
		session["expiresAt"] != nil || session["revokedAt"] != nil {
		t.Fatalf("session fields %v", session)
	}
	if attempt, ok := session["lastAttempt"].(map[string]any); !ok || attempt["outcome"] != "rejected" || attempt["attemptedAt"] == nil {
		t.Fatalf("session lastAttempt %v", session["lastAttempt"])
	}
	if token["updatedAt"] == nil || token["lastRejectedAt"] == nil {
		t.Fatalf("token fields %v", token)
	}
}

// TC-001
func TestSecretMasking_TC001_NothingSaved(t *testing.T) {
	server, _ := newSettingsServer(t)
	for _, check := range []struct{ path, want string }{
		{settingsPath, `{"session":{"saved":false,"hint":null,"revision":0,"updatedAt":null,"status":"none","expiresAt":null,"revokedAt":null,"lastAttempt":null}}`},
		{claudeTokenPath, `{"claudeToken":{"saved":false,"hint":null,"updatedAt":null,"lastRejectedAt":null}}`},
	} {
		recorder := request(t, server, http.MethodGet, check.path, "", "")
		if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" || strings.TrimSpace(recorder.Body.String()) != check.want {
			t.Fatalf("%s: status %d body %s", check.path, recorder.Code, recorder.Body.String())
		}
	}
}

// TC-001
func TestSecretMasking_TC001_ShortSecretIsSavedWithoutCharacters(t *testing.T) {
	server, database := newSettingsServer(t)
	shortToken := "sk-Zq8Vw3"
	storeSecrets(t, database, 0, text(secretShort), text(shortToken))
	session, token := loadSettings(t, server, "")
	assertHint(t, "short session", session, true, "")
	assertHint(t, "short token", token, true, "")
	for _, path := range []string{settingsPath, claudeTokenPath} {
		assertNoSecret(t, path, responseText(request(t, server, http.MethodGet, path, "", "")), secretShort, shortToken)
	}
}

// TC-001
func TestSecretMasking_TC001_HintThresholdBoundary(t *testing.T) {
	for _, test := range []struct{ value, hint string }{
		{"Thr3shold-15-ch", ""},
		{"Thr3shold-16-chr", "-chr"},
	} {
		t.Run(fmt.Sprint(len(test.value)), func(t *testing.T) {
			server, database := newSettingsServer(t)
			storeSecrets(t, database, 0, text(test.value), text(test.value))
			session, token := loadSettings(t, server, "")
			assertHint(t, "session", session, true, test.hint)
			assertHint(t, "token", token, true, test.hint)
			for _, path := range []string{settingsPath, claudeTokenPath} {
				if body := request(t, server, http.MethodGet, path, "", "").Body.String(); strings.Contains(body, test.value) {
					t.Fatalf("%s returned the secret: %s", path, body)
				}
			}
		})
	}
}

// TC-001
func TestSecretMasking_TC001_CookieStringShowsTheDatadomeHint(t *testing.T) {
	server, _ := newSettingsServer(t)
	saved, _ := saveSettingsOK(t, server, settingsBody(sessionPart("Cookie: a=1; datadome="+secretS+"; b=2", 0)), "")
	assertHint(t, "save answer", saved, true, "9f3a")
	session, _ := loadSettings(t, server, "")
	assertHint(t, "load", session, true, "9f3a")
	for _, path := range []string{settingsPath, claudeTokenPath} {
		body := responseText(request(t, server, http.MethodGet, path, "", ""))
		if strings.Contains(body, "a=1") || strings.Contains(body, "b=2") {
			t.Fatalf("%s returned other cookies: %s", path, body)
		}
		assertNoSecret(t, path, body, secretS)
	}
}

// TC-001
func TestSecretMasking_TC001_ProtectedModeUsersSeeOnlyTheirOwnHints(t *testing.T) {
	server, database := newSettingsServer(t)
	aliceID, bobID, alice, bob := protectedUsers(t, server, database)
	storeSecrets(t, database, aliceID, text(secretS), text(secretT))
	storeSecrets(t, database, bobID, text(secretN), text(secretU))
	session, token := loadSettings(t, server, alice)
	assertHint(t, "alice session", session, true, "9f3a")
	assertHint(t, "alice token", token, true, "wxyz")
	session, token = loadSettings(t, server, bob)
	assertHint(t, "bob session", session, true, "n0p1")
	assertHint(t, "bob token", token, true, "uvst")
	for _, path := range []string{settingsPath, claudeTokenPath} {
		assertNoSecret(t, "alice "+path, responseText(request(t, server, http.MethodGet, path, "", alice)), secretS, secretT, secretN, secretU)
		assertNoSecret(t, "bob "+path, responseText(request(t, server, http.MethodGet, path, "", bob)), secretS, secretT, secretN, secretU)
	}
}

// TC-001
func TestSecretMasking_TC001_RepeatedLoadsAreIdenticalAndChangeNothing(t *testing.T) {
	server, database := newSettingsServer(t)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	beforeSession, beforeToken := storedSecrets(t, database, 0)
	var first []string
	for round := 0; round < 3; round++ {
		var bodies []string
		for _, path := range []string{settingsPath, claudeTokenPath} {
			bodies = append(bodies, request(t, server, http.MethodGet, path, "", "").Body.String())
		}
		if round == 0 {
			first = bodies
		} else if !reflect.DeepEqual(bodies, first) {
			t.Fatalf("load %d differs: %q, first %q", round, bodies, first)
		}
	}
	session, token := loadSettings(t, server, "")
	assertHint(t, "session", session, true, "9f3a")
	assertHint(t, "token", token, true, "wxyz")
	assertStoredSession(t, database, 0, beforeSession)
	assertStoredToken(t, database, 0, beforeToken)
}

// TC-002
func TestSecretMasking_TC002_NoResponseOrLogCarriesASecret(t *testing.T) {
	server, database := newSettingsServer(t)
	fake := withSecretNet(t, 200)
	logs := captureSecretLogs(t)
	ctx := context.Background()
	insertItem(t, database, "lbc", "leboncoin")
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	server.service.Collect(ctx, "lbc") // a session-assisted attempt
	runAIReview(t, server, "lbc", "")  // an AI review
	assertLastCookie(t, fake, "datadome="+secretS)
	assertLastReviewToken(t, fake, secretT)

	var output strings.Builder
	send := func(method, path, body string, status int, code string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := request(t, server, method, path, body, "")
		output.WriteString(responseText(recorder) + "\n")
		if status != 0 && recorder.Code != status {
			t.Fatalf("%s %s %.80s: status %d, want %d: %s", method, path, body, recorder.Code, status, recorder.Body.String())
		}
		if code != "" && !strings.Contains(recorder.Body.String(), `"code":"`+code+`"`) {
			t.Fatalf("%s %s %.80s: want %s: %s", method, path, body, code, recorder.Body.String())
		}
		return recorder
	}
	stored := func() model.LeboncoinSession {
		session, _ := storedSecrets(t, database, 0)
		return session
	}

	// Loads, items, application information and auth routes.
	send(http.MethodGet, settingsPath, "", 200, "")
	send(http.MethodGet, claudeTokenPath, "", 200, "")
	send(http.MethodGet, "/api/v1/items", "", 200, "")
	send(http.MethodGet, "/api/v1/items/lbc", "", 200, "")
	send(http.MethodGet, "/api/v1/auth/session", "", 200, "")
	send(http.MethodPost, "/api/v1/auth/login", `{"username":"nobody","password":"wrong password"}`, 0, "")
	send(http.MethodPost, "/api/v1/auth/logout", "", 0, "")

	// Every error path of the save, with the typed secrets U and N.
	revision := stored().Revision
	send(http.MethodPut, settingsSavePath, `{"claudeToken":{"value":"`+secretU, 400, "INVALID_JSON")
	send(http.MethodPut, settingsSavePath, `{"claudeToken":{"value":"`+secretU+`","remove":true}}`, 400, "INVALID_REQUEST")
	send(http.MethodPut, settingsSavePath, settingsBody(sessionPart("datadome="+secretN+",x", revision)), 400, "INVALID_SESSION")
	send(http.MethodPut, settingsSavePath, settingsBody(tokenPart(secretU+" x")), 400, "INVALID_CLAUDE_TOKEN")
	fake.setClaude(401)
	send(http.MethodPut, settingsSavePath, settingsBody(tokenPart(secretU)), 422, "CLAUDE_TOKEN_REJECTED")
	fake.setClaude(0)
	send(http.MethodPut, settingsSavePath, settingsBody(tokenPart(secretU)), 502, "CLAUDE_UNREACHABLE")
	fake.setClaude(200)
	send(http.MethodPut, settingsSavePath, settingsBody(tokenPart(secretU+strings.Repeat("u", 33_000))), 413, "REQUEST_TOO_LARGE")
	send(http.MethodGet, settingsSavePath, "", 405, "METHOD_NOT_ALLOWED")
	send(http.MethodPut, claudeTokenPath, settingsBody(tokenPart(secretU)), 405, "METHOD_NOT_ALLOWED")
	send(http.MethodPut, settingsPath, `{"value":"`+secretN+`","revision":`+fmt.Sprint(revision)+`}`, 405, "METHOD_NOT_ALLOWED")
	send(http.MethodPost, settingsPath, "", 405, "METHOD_NOT_ALLOWED")
	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: secretY})
	send(http.MethodPut, settingsSavePath, settingsBody(sessionPart("datadome="+secretN, revision)), 409, "SESSION_CHANGED")

	// Each successful save shape answers only the state and hints of the stored secrets.
	save := func(body string) (session, token map[string]any) {
		t.Helper()
		recorder := send(http.MethodPut, settingsSavePath, body, 200, "")
		session, token = decodeObject(t, recorder, "session"), decodeObject(t, recorder, "claudeToken")
		assertMaskedObjects(t, session, token)
		return session, token
	}
	session, token := save(`{}`)
	assertHint(t, "untouched session", session, true, "y7k2")
	assertHint(t, "untouched token", token, true, "wxyz")
	session, token = save(settingsBody(sessionPart("", 0), tokenPart("")))
	assertHint(t, "empty session", session, true, "y7k2")
	assertHint(t, "empty token", token, true, "wxyz")
	session, _ = save(settingsBody(sessionPart("datadome="+secretN, stored().Revision)))
	assertHint(t, "replaced session", session, true, "n0p1")
	_, token = save(settingsBody(tokenPart(secretU)))
	assertHint(t, "replaced token", token, true, "uvst")
	_, token = save(settingsBody(tokenRemoval))
	assertHint(t, "removed token", token, false, "")
	session, _ = save(settingsBody(sessionRemoval(stored().Revision)))
	assertHint(t, "removed session", session, false, "")
	send(http.MethodGet, settingsPath, "", 200, "")
	send(http.MethodGet, claudeTokenPath, "", 200, "")

	assertNoSecret(t, "responses", output.String(), secretT, secretS, secretU, secretN, secretY)
	assertNoSecret(t, "server log", logs.String(), secretT, secretS, secretU, secretN, secretY)
}

// TC-002
func TestSecretMasking_TC002_StorageFailureOnLoadCarriesNoSecret(t *testing.T) {
	server, database := newSettingsServer(t)
	logs := captureSecretLogs(t)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	database.Close()
	for _, path := range []string{settingsPath, claudeTokenPath} {
		recorder := request(t, server, http.MethodGet, path, "", "")
		assertError(t, recorder, 500, "INTERNAL_ERROR", "The server could not complete the request.")
		assertNoSecret(t, path, responseText(recorder), secretS, secretT)
	}
	recorder := putSettings(t, server, settingsBody(sessionPart("datadome="+secretN, 1)), "")
	assertError(t, recorder, 500, "INTERNAL_ERROR", "The server could not complete the request.")
	assertNoSecret(t, "failed save", responseText(recorder)+logs.String(), secretS, secretT, secretN)
}

// TC-003
func TestSecretMasking_TC003_SavingOnlyTheSessionLeavesTheTokenExactly(t *testing.T) {
	for _, test := range []struct {
		name         string
		claudeStatus int
		tokenPart    []string
	}{
		{"claude unreachable", 0, nil},
		{"claude refuses", 401, nil},
		{"token sent as whitespace", 0, []string{tokenPart("   ")}},
		{"token sent empty", 401, []string{tokenPart("")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, database := newSettingsServer(t)
			fake := withSecretNet(t, test.claudeStatus)
			insertItem(t, database, "lbc", "leboncoin")
			storeSecrets(t, database, 0, text(secretS), text(secretT))
			rejectStoredToken(t, database, 0)
			beforeSession, beforeToken := storedSecrets(t, database, 0)

			parts := append([]string{sessionPart("datadome="+secretN, beforeSession.Revision)}, test.tokenPart...)
			session, token := saveSettingsOK(t, server, settingsBody(parts...), "")
			assertHint(t, "saved session", session, true, "n0p1")
			assertHint(t, "kept token", token, true, "wxyz")
			if got := fake.verifications(); len(got) != 0 {
				t.Fatalf("verification calls %q", got)
			}
			assertStoredToken(t, database, 0, beforeToken)
			afterSession, _ := storedSecrets(t, database, 0)
			if storedValue(afterSession.Value) != secretN || afterSession.Revision != beforeSession.Revision+1 {
				t.Fatalf("session not replaced: %+v", afterSession)
			}
			session, token = loadSettings(t, server, "")
			assertHint(t, "reloaded token", token, true, "wxyz")
			assertHint(t, "reloaded session", session, true, "n0p1")
			if token["lastRejectedAt"] == nil || token["updatedAt"] == nil {
				t.Fatalf("token warning lost: %v", token)
			}
			runAIReview(t, server, "lbc", "")
			assertLastReviewToken(t, fake, secretT)
		})
	}
}

// TC-004
func TestSecretMasking_TC004_SavingOnlyTheTokenLeavesTheSessionExactly(t *testing.T) {
	for _, test := range []struct {
		name    string
		outcome store.LeboncoinSessionOutcome
		status  string
		cookie  string
	}{
		{"rejected attempt", store.LeboncoinSessionOutcome{Attempt: "rejected"}, "active", "datadome=" + secretS},
		{"expired", store.LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: secretS, ExpiresAt: timeFromNow(-time.Minute)}, "expired", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, database := newSettingsServer(t)
			fake := withSecretNet(t, 200)
			insertItem(t, database, "lbc", "leboncoin")
			storeSecrets(t, database, 0, text(secretS), text(secretT))
			finishSessionAttempt(t, database, 0, test.outcome)
			modalB, _ := loadSettings(t, server, "") // a second client loads Settings at revision r
			before, _ := storedSecrets(t, database, 0)
			if before.Status != test.status {
				t.Fatalf("fixture status %q", before.Status)
			}

			session, token := saveSettingsOK(t, server, settingsBody(tokenPart(secretU)), "")
			assertHint(t, "kept session", session, true, "9f3a")
			assertHint(t, "saved token", token, true, "uvst")
			if session["status"] != test.status || session["revision"] != float64(before.Revision) || session["lastAttempt"] == nil {
				t.Fatalf("session state in the answer %v", session)
			}
			if got := fake.verifications(); !reflect.DeepEqual(got, []string{secretU}) {
				t.Fatalf("verification calls %q", got)
			}
			assertStoredSession(t, database, 0, before)

			server.service.Collect(context.Background(), "lbc")
			cookies := fake.sentCookies()
			if test.cookie == "" && len(cookies) != 0 {
				t.Fatalf("an unusable session was sent: %q", cookies)
			}
			if test.cookie != "" {
				assertLastCookie(t, fake, test.cookie)
			}

			// Modal B, loaded before the token save, replaces the session without SESSION_CHANGED.
			revision := int64(modalB["revision"].(float64))
			session, _ = saveSettingsOK(t, server, settingsBody(sessionPart("datadome="+secretN, revision)), "")
			assertHint(t, "modal B session", session, true, "n0p1")
		})
	}
}

// timeFromNow returns the current time moved by offset.
func timeFromNow(offset time.Duration) *time.Time {
	at := time.Now().Add(offset)
	return &at
}

// TC-004
func TestSecretMasking_TC004_FailedVerificationChangesNothing(t *testing.T) {
	server, database := newSettingsServer(t)
	withSecretNet(t, 0)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	beforeSession, beforeToken := storedSecrets(t, database, 0)
	assertError(t, putSettings(t, server, settingsBody(tokenPart(secretU)), ""), 502, "CLAUDE_UNREACHABLE", "")
	assertStoredSession(t, database, 0, beforeSession)
	assertStoredToken(t, database, 0, beforeToken)
}

// TC-005
func TestSecretMasking_TC005_UntouchedSavesNeverChangeASecret(t *testing.T) {
	for _, test := range []struct {
		name string
		body func(revision int64) string
	}{
		{"no part", func(int64) string { return `{}` }},
		{"empty values", func(revision int64) string { return settingsBody(sessionPart("", revision), tokenPart("")) }},
		{"whitespace values", func(revision int64) string { return settingsBody(sessionPart(" \n\t", revision), tokenPart("   ")) }},
		{"null parts", func(int64) string { return `{"leboncoinSession":null,"claudeToken":null}` }},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, database := newSettingsServer(t)
			fake := withSecretNet(t, 200)
			insertItem(t, database, "lbc", "leboncoin")
			storeSecrets(t, database, 0, text(secretS), text(secretT))
			rejectStoredToken(t, database, 0)
			finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "rejected"})
			beforeSession, beforeToken := storedSecrets(t, database, 0)
			for round := 0; round < 3; round++ {
				loaded, _ := loadSettings(t, server, "")
				session, token := saveSettingsOK(t, server, test.body(int64(loaded["revision"].(float64))), "")
				assertHint(t, "session", session, true, "9f3a")
				assertHint(t, "token", token, true, "wxyz")
				assertStoredSession(t, database, 0, beforeSession)
				assertStoredToken(t, database, 0, beforeToken)
			}
			if got := fake.verifications(); len(got) != 0 {
				t.Fatalf("verification calls %q", got)
			}
			session, token := loadSettings(t, server, "")
			assertHint(t, "reloaded session", session, true, "9f3a")
			assertHint(t, "reloaded token", token, true, "wxyz")
			server.service.Collect(context.Background(), "lbc")
			assertLastCookie(t, fake, "datadome="+secretS)
			runAIReview(t, server, "lbc", "")
			assertLastReviewToken(t, fake, secretT)
		})
	}
}

// TC-005
func TestSecretMasking_TC005_MaskedHintTextIsRejected(t *testing.T) {
	server, database := newSettingsServer(t)
	fake := withSecretNet(t, 200)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	beforeSession, beforeToken := storedSecrets(t, database, 0)
	assertError(t, putSettings(t, server, settingsBody(sessionPart("••••9f3a", beforeSession.Revision)), ""), 400, "INVALID_SESSION", "")
	assertError(t, putSettings(t, server, settingsBody(sessionPart("datadome=••••9f3a", beforeSession.Revision)), ""), 400, "INVALID_SESSION", "")
	assertError(t, putSettings(t, server, settingsBody(tokenPart("••••wxyz")), ""), 400, "INVALID_CLAUDE_TOKEN", "")
	assertStoredSession(t, database, 0, beforeSession)
	assertStoredToken(t, database, 0, beforeToken)
	if got := fake.verifications(); len(got) != 0 {
		t.Fatalf("verification calls %q", got)
	}
}

// TC-005
func TestSecretMasking_TC005_NothingSavedAndUntouchedCreatesNothing(t *testing.T) {
	server, database := newSettingsServer(t)
	withSecretNet(t, 200)
	beforeSession, beforeToken := storedSecrets(t, database, 0)
	session, token := saveSettingsOK(t, server, `{}`, "")
	assertHint(t, "session", session, false, "")
	assertHint(t, "token", token, false, "")
	session, token = saveSettingsOK(t, server, settingsBody(sessionPart("", 0), tokenPart("")), "")
	assertHint(t, "session", session, false, "")
	assertHint(t, "token", token, false, "")
	assertStoredSession(t, database, 0, beforeSession)
	assertStoredToken(t, database, 0, beforeToken)
}

// TC-006
func TestSecretMasking_TC006_SameValuesAreReplacementsOverHTTP(t *testing.T) {
	server, database := newSettingsServer(t)
	fake := withSecretNet(t, 200)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	rejectStoredToken(t, database, 0)
	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "rejected"})
	before, _ := storedSecrets(t, database, 0)

	session, token := saveSettingsOK(t, server, settingsBody(sessionPart(secretS, before.Revision), tokenPart(secretT)), "")
	if got := fake.verifications(); !reflect.DeepEqual(got, []string{secretT}) {
		t.Fatalf("verification calls %q, want one with T", got)
	}
	assertHint(t, "session", session, true, "9f3a")
	assertHint(t, "token", token, true, "wxyz")
	if token["lastRejectedAt"] != nil || session["lastAttempt"] != nil || session["status"] != "active" || session["revision"] != float64(before.Revision+1) {
		t.Fatalf("warnings not cleared: session %v token %v", session, token)
	}
	session, token = loadSettings(t, server, "")
	assertHint(t, "reloaded session", session, true, "9f3a")
	assertHint(t, "reloaded token", token, true, "wxyz")
	if token["lastRejectedAt"] != nil || session["lastAttempt"] != nil {
		t.Fatalf("warnings not cleared after reload: session %v token %v", session, token)
	}
}

// TC-006
func TestSecretMasking_TC006_CookieStringAndRevokedSessionOverHTTP(t *testing.T) {
	server, database := newSettingsServer(t)
	withSecretNet(t, 200)
	storeSecrets(t, database, 0, text(secretS), nil)
	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "rejected", Revoked: true})
	before, _ := storedSecrets(t, database, 0)
	if before.Status != "revoked" {
		t.Fatalf("fixture status %q", before.Status)
	}
	assertError(t, putSettings(t, server, settingsBody(sessionPart("a=1; b=2", before.Revision)), ""), 400, "INVALID_SESSION", "No datadome cookie was found in the pasted text.")
	assertStoredSession(t, database, 0, before)
	session, _ := saveSettingsOK(t, server, settingsBody(sessionPart("Cookie: a=1; datadome="+secretN+"; b=2", before.Revision)), "")
	assertHint(t, "session", session, true, "n0p1")
	if session["status"] != "active" || session["revokedAt"] != nil {
		t.Fatalf("revoked session not replaced: %v", session)
	}
	if after, _ := storedSecrets(t, database, 0); storedValue(after.Value) != secretN {
		t.Fatalf("stored %q, want N", storedValue(after.Value))
	}
}

// TC-007
func TestSecretMasking_TC007_ExplicitRemovalOfEachSecret(t *testing.T) {
	server, database := newSettingsServer(t)
	fake := withSecretNet(t, 0)
	insertItem(t, database, "lbc", "leboncoin")
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	beforeSession, _ := storedSecrets(t, database, 0)

	session, token := saveSettingsOK(t, server, settingsBody(tokenRemoval), "")
	assertHint(t, "removed token", token, false, "")
	assertHint(t, "kept session", session, true, "9f3a")
	if got := fake.verifications(); len(got) != 0 {
		t.Fatalf("removal verified the token: %q", got)
	}
	assertStoredSession(t, database, 0, beforeSession)

	session, token = saveSettingsOK(t, server, settingsBody(sessionRemoval(beforeSession.Revision)), "")
	assertHint(t, "removed session", session, false, "")
	assertHint(t, "token still removed", token, false, "")
	if session["status"] != "none" {
		t.Fatalf("removed session status %v", session["status"])
	}
	session, token = loadSettings(t, server, "")
	assertHint(t, "reloaded session", session, false, "")
	assertHint(t, "reloaded token", token, false, "")
	if after, afterToken := storedSecrets(t, database, 0); after.Value != nil || afterToken.Value != nil {
		t.Fatalf("secrets still stored: %+v %+v", after, afterToken)
	}

	server.service.Collect(context.Background(), "lbc")
	assertLastCookie(t, fake, "")
	assertError(t, request(t, server, http.MethodPost, "/api/v1/items/lbc/ai-review", "", ""), 409, "CLAUDE_TOKEN_MISSING", "Configure a Claude token in Settings.")
	if details := request(t, server, http.MethodGet, "/api/v1/items/lbc", "", "").Body.String(); !strings.Contains(details, `"tokenConfigured":false`) {
		t.Fatalf("item details %s", details)
	}
}

// TC-007
func TestSecretMasking_TC007_EmptyInputsNeverRemove(t *testing.T) {
	server, database := newSettingsServer(t)
	fake := withSecretNet(t, 0)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	beforeSession, beforeToken := storedSecrets(t, database, 0)
	for _, body := range []string{
		settingsBody(tokenPart("")),
		settingsBody(tokenPart("  \t ")),
		settingsBody(sessionPart("", beforeSession.Revision)),
		settingsBody(sessionPart(" \n ", beforeSession.Revision)),
		settingsBody(sessionPart("", beforeSession.Revision), tokenPart("")),
	} {
		session, token := saveSettingsOK(t, server, body, "")
		assertHint(t, body, session, true, "9f3a")
		assertHint(t, body, token, true, "wxyz")
		assertStoredSession(t, database, 0, beforeSession)
		assertStoredToken(t, database, 0, beforeToken)
	}
	if got := fake.verifications(); len(got) != 0 {
		t.Fatalf("verification calls %q", got)
	}
}

// TC-007
func TestSecretMasking_TC007_RemovingWhatIsNotSaved(t *testing.T) {
	server, database := newSettingsServer(t)
	withSecretNet(t, 0)
	storeSecrets(t, database, 0, text(secretS), nil)
	beforeSession, beforeToken := storedSecrets(t, database, 0)
	session, token := saveSettingsOK(t, server, settingsBody(tokenRemoval), "")
	assertHint(t, "no token", token, false, "")
	assertHint(t, "session", session, true, "9f3a")
	assertStoredSession(t, database, 0, beforeSession)
	assertStoredToken(t, database, 0, beforeToken)

	server2, database2 := newSettingsServer(t)
	storeSecrets(t, database2, 0, nil, text(secretT))
	beforeSession, beforeToken = storedSecrets(t, database2, 0)
	session, token = saveSettingsOK(t, server2, settingsBody(sessionRemoval(beforeSession.Revision)), "")
	assertHint(t, "no session", session, false, "")
	assertHint(t, "token", token, true, "wxyz")
	assertStoredSession(t, database2, 0, beforeSession)
	assertStoredToken(t, database2, 0, beforeToken)
}

// TC-007
func TestSecretMasking_TC007_RemovalCombinedWithAReplacement(t *testing.T) {
	t.Run("remove token, replace session", func(t *testing.T) {
		server, database := newSettingsServer(t)
		withSecretNet(t, 0)
		storeSecrets(t, database, 0, text(secretS), text(secretT))
		before, beforeToken := storedSecrets(t, database, 0)
		assertError(t, putSettings(t, server, settingsBody(tokenRemoval, sessionPart("a=1; b=2", before.Revision)), ""), 400, "INVALID_SESSION", "")
		assertStoredSession(t, database, 0, before)
		assertStoredToken(t, database, 0, beforeToken)
		session, token := saveSettingsOK(t, server, settingsBody(tokenRemoval, sessionPart("datadome="+secretN, before.Revision)), "")
		assertHint(t, "session", session, true, "n0p1")
		assertHint(t, "token", token, false, "")
	})
	t.Run("remove session, refused token", func(t *testing.T) {
		server, database := newSettingsServer(t)
		withSecretNet(t, 401)
		storeSecrets(t, database, 0, text(secretS), text(secretT))
		before, beforeToken := storedSecrets(t, database, 0)
		assertError(t, putSettings(t, server, settingsBody(sessionRemoval(before.Revision), tokenPart(secretU)), ""), 422, "CLAUDE_TOKEN_REJECTED", "")
		assertStoredSession(t, database, 0, before)
		assertStoredToken(t, database, 0, beforeToken)
	})
}

// TC-007: a part with both or neither of value and remove, or remove other than true, is refused.
func TestSecretMasking_TC007_AmbiguousPartsAreRefused(t *testing.T) {
	server, database := newSettingsServer(t)
	fake := withSecretNet(t, 200)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	beforeSession, beforeToken := storedSecrets(t, database, 0)
	revision := fmt.Sprint(beforeSession.Revision)
	for _, body := range []string{
		`{"claudeToken":{"value":"` + secretU + `","remove":true}}`,
		`{"claudeToken":{"value":"","remove":true}}`,
		`{"claudeToken":{}}`,
		`{"claudeToken":{"remove":false}}`,
		`{"claudeToken":{"remove":null}}`,
		`{"claudeToken":{"remove":"true"}}`,
		`{"claudeToken":{"value":1}}`,
		`{"claudeToken":"` + secretU + `"}`,
		`{"leboncoinSession":{"value":"datadome=` + secretN + `","remove":true,"revision":` + revision + `}}`,
		`{"leboncoinSession":{"revision":` + revision + `}}`,
		`{"leboncoinSession":{"remove":false,"revision":` + revision + `}}`,
		`{"leboncoinSession":{"remove":true}}`,
		`{"leboncoinSession":{"remove":true,"revision":-1}}`,
		`{"leboncoinSession":{"value":""}}`,
		`{"leboncoinSession":{"value":"","revision":1.5}}`,
		`{"leboncoinSession":[]}`,
		`{"leboncoinSession":{"remove":true,"revision":` + revision + `},"claudeToken":{"remove":false}}`,
		`[]`,
	} {
		recorder := putSettings(t, server, body, "")
		assertError(t, recorder, 400, "INVALID_REQUEST", invalidSettingsRequest)
		assertNoSecret(t, body, responseText(recorder), secretS, secretT, secretU, secretN)
	}
	assertStoredSession(t, database, 0, beforeSession)
	assertStoredToken(t, database, 0, beforeToken)
	if got := fake.verifications(); len(got) != 0 {
		t.Fatalf("verification calls %q", got)
	}
}

// TC-008
func TestSecretMasking_TC008_SessionChangedOnlyWhenReplacingOrRemoving(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(t *testing.T, server *Server, database *store.Store, revision int64)
	}{
		{"renewal", func(t *testing.T, _ *Server, database *store.Store, _ int64) {
			finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: secretY})
		}},
		{"revocation", func(t *testing.T, _ *Server, database *store.Store, _ int64) {
			finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "rejected", Revoked: true})
		}},
		{"save from another client", func(t *testing.T, server *Server, _ *store.Store, revision int64) {
			saveSettingsOK(t, server, settingsBody(sessionPart(secretY, revision)), "")
		}},
		{"removal from another client", func(t *testing.T, server *Server, _ *store.Store, revision int64) {
			saveSettingsOK(t, server, settingsBody(sessionRemoval(revision)), "")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, database := newSettingsServer(t)
			fake := withSecretNet(t, 200)
			storeSecrets(t, database, 0, text(secretS), text(secretT))
			loaded, _ := loadSettings(t, server, "")
			revision := int64(loaded["revision"].(float64))
			test.change(t, server, database, revision)
			changed, beforeToken := storedSecrets(t, database, 0)
			if changed.Revision == revision {
				t.Fatalf("the fixture did not change the revision")
			}

			for _, body := range []string{
				settingsBody(sessionPart("datadome="+secretN, revision), tokenPart(secretU)),
				settingsBody(sessionPart("datadome="+secretN, revision)),
				settingsBody(sessionRemoval(revision)),
				settingsBody(sessionRemoval(revision), tokenPart(secretU)),
			} {
				recorder := putSettings(t, server, body, "")
				assertError(t, recorder, 409, "SESSION_CHANGED", sessionChangedMessage)
				assertNoSecret(t, "refused save", responseText(recorder), secretS, secretT, secretU, secretN, secretY)
				assertStoredSession(t, database, 0, changed)
				assertStoredToken(t, database, 0, beforeToken)
			}

			// The token alone, with the session absent or empty at the old revision, is saved.
			session, token := saveSettingsOK(t, server, settingsBody(tokenPart(secretU)), "")
			assertHint(t, "token", token, true, "uvst")
			if session["revision"] != float64(changed.Revision) || session["status"] != changed.Status {
				t.Fatalf("session state in the answer %v, stored %+v", session, changed)
			}
			assertStoredSession(t, database, 0, changed)
			if _, storedToken := storedSecrets(t, database, 0); storedValue(storedToken.Value) != secretU {
				t.Fatalf("stored token %q, want U", storedValue(storedToken.Value))
			}
			saveSettingsOK(t, server, settingsBody(sessionPart("", revision), tokenPart(secretT)), "")
			assertStoredSession(t, database, 0, changed)
			if _, storedToken := storedSecrets(t, database, 0); storedValue(storedToken.Value) != secretT {
				t.Fatalf("stored token %q, want T", storedValue(storedToken.Value))
			}
			if got := fake.verifications(); len(got) < 2 || !reflect.DeepEqual(got[len(got)-2:], []string{secretU, secretT}) {
				t.Fatalf("verification calls %q", got)
			}

			// After reloading, replacing and removing succeed.
			reloaded, _ := loadSettings(t, server, "")
			if reloaded["revision"] != float64(changed.Revision) {
				t.Fatalf("reloaded revision %v, stored %d", reloaded["revision"], changed.Revision)
			}
			if changed.Value != nil {
				assertHint(t, "reloaded session", reloaded, true, lastFour(*changed.Value))
			} else {
				assertHint(t, "reloaded session", reloaded, false, "")
			}
			session, _ = saveSettingsOK(t, server, settingsBody(sessionPart("datadome="+secretN, changed.Revision)), "")
			assertHint(t, "replaced", session, true, "n0p1")
			session, _ = saveSettingsOK(t, server, settingsBody(sessionRemoval(int64(session["revision"].(float64)))), "")
			assertHint(t, "removed", session, false, "")
		})
	}
}

func lastFour(value string) string { return value[len(value)-4:] }

// TC-008
func TestSecretMasking_TC008_ExpiryWithoutRevisionChangeIsNoConflict(t *testing.T) {
	server, database := newSettingsServer(t)
	withSecretNet(t, 200)
	storeSecrets(t, database, 0, text(secretS), text(secretT))
	loaded, _ := loadSettings(t, server, "")
	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: secretS, ExpiresAt: timeFromNow(-time.Minute)})
	if expired, _ := storedSecrets(t, database, 0); expired.Status != "expired" || float64(expired.Revision) != loaded["revision"] {
		t.Fatalf("fixture %+v", expired)
	}
	session, _ := saveSettingsOK(t, server, settingsBody(sessionPart("datadome="+secretN, int64(loaded["revision"].(float64)))), "")
	assertHint(t, "replaced", session, true, "n0p1")
	if session["status"] != "active" {
		t.Fatalf("status %v", session["status"])
	}
}

// TC-009
func TestSecretMasking_TC009_RenewedExpiredAndRevokedSessionsShowTheirHint(t *testing.T) {
	server, database := newSettingsServer(t)
	storeSecrets(t, database, 0, text(secretS), nil)
	var output strings.Builder

	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: secretY})
	session, _ := loadSettings(t, server, "")
	assertHint(t, "renewed", session, true, "y7k2")
	if session["status"] != "active" {
		t.Fatalf("renewed status %v", session["status"])
	}
	output.WriteString(responseText(request(t, server, http.MethodGet, settingsPath, "", "")))

	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: secretY, ExpiresAt: timeFromNow(-time.Minute)})
	session, _ = loadSettings(t, server, "")
	assertHint(t, "expired", session, true, "y7k2")
	if session["status"] != "expired" || session["expiresAt"] == nil {
		t.Fatalf("expired session %v", session)
	}
	output.WriteString(responseText(request(t, server, http.MethodGet, settingsPath, "", "")))

	finishSessionAttempt(t, database, 0, store.LeboncoinSessionOutcome{Attempt: "rejected", Revoked: true})
	session, _ = loadSettings(t, server, "")
	if revoked, _ := storedSecrets(t, database, 0); revoked.Value != nil {
		assertHint(t, "revoked", session, true, lastFour(*revoked.Value))
	} else {
		assertHint(t, "revoked", session, false, "")
	}
	if session["status"] != "revoked" || session["revokedAt"] == nil {
		t.Fatalf("revoked session %v", session)
	}
	output.WriteString(responseText(request(t, server, http.MethodGet, settingsPath, "", "")))
	assertNoSecret(t, "loads", output.String(), secretS, secretY)
}

// TC-009
func TestSecretMasking_TC009_TokenChangedFromAnotherClient(t *testing.T) {
	server, database := newSettingsServer(t)
	withSecretNet(t, 200)
	storeSecrets(t, database, 0, nil, text(secretT))
	_, token := loadSettings(t, server, "")
	assertHint(t, "loaded", token, true, "wxyz")
	saveSettingsOK(t, server, settingsBody(tokenPart(secretU)), "")
	_, token = loadSettings(t, server, "")
	assertHint(t, "replaced elsewhere", token, true, "uvst")
	saveSettingsOK(t, server, settingsBody(tokenRemoval), "")
	_, token = loadSettings(t, server, "")
	assertHint(t, "removed elsewhere", token, false, "")
}

// TC-010
func TestSecretMasking_TC010_PreviouslySavedSecretsKeepWorkingAfterRestarts(t *testing.T) {
	for _, test := range []struct {
		name, session, token, sessionHint, tokenHint string
	}{
		{"long secrets", secretS, secretT, "9f3a", "wxyz"},
		{"short secrets", secretShort, "sk-Zq8Vw3", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			fake := withSecretNet(t, 200)
			// Rows written as the previous version wrote them, before this server starts.
			previous, err := store.Open(config.Config{DataDirectory: dir, Location: time.UTC, CheckTimes: []string{"08:00"}, StaleAfter: 36 * time.Hour})
			if err != nil {
				t.Fatal(err)
			}
			insertItem(t, previous, "lbc", "leboncoin")
			storeSecrets(t, previous, 0, text(test.session), text(test.token))
			rejectStoredToken(t, previous, 0)
			finishSessionAttempt(t, previous, 0, store.LeboncoinSessionOutcome{Attempt: "rejected"})
			previous.Close()

			server, _, stop := openSecretServer(t, dir)
			session, token := loadSettings(t, server, "")
			assertHint(t, "session", session, true, test.sessionHint)
			assertHint(t, "token", token, true, test.tokenHint)
			if token["lastRejectedAt"] == nil || session["lastAttempt"] == nil {
				t.Fatalf("previous warnings lost: session %v token %v", session, token)
			}
			for _, path := range []string{settingsPath, claudeTokenPath} {
				assertNoSecret(t, path, responseText(request(t, server, http.MethodGet, path, "", "")), test.session, test.token)
			}
			server.service.Collect(context.Background(), "lbc")
			assertLastCookie(t, fake, "datadome="+test.session)
			runAIReview(t, server, "lbc", "")
			assertLastReviewToken(t, fake, test.token)
			stop()

			restarted, _, _ := openSecretServer(t, dir)
			session, token = loadSettings(t, restarted, "")
			assertHint(t, "session after restart", session, true, test.sessionHint)
			assertHint(t, "token after restart", token, true, test.tokenHint)
			if got := fake.verifications(); len(got) != 0 {
				t.Fatalf("verification calls %q", got)
			}
		})
	}
}
