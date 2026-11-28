package httpapi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Issue #268: the session cookie is Secure over direct TLS or when the first X-Forwarded-Proto entry is https.

const (
	cookieUsername = "alice"
	cookiePassword = "correct horse battery"
	cookieLogin    = `{"username":"alice","password":"correct horse battery"}`
	loginPath      = "/api/v1/auth/login"
	logoutPath     = "/api/v1/auth/logout"
	sessionPath    = "/api/v1/auth/session"
)

// newCookieServer returns a server in protected mode with the regular user alice.
func newCookieServer(t *testing.T) *Server {
	t.Helper()
	server, _ := newSettingsServer(t)
	if _, err := server.service.CreateUser(context.Background(), cookieUsername, cookiePassword); err != nil {
		t.Fatal(err)
	}
	return server
}

// sendAuth posts with an optional session cookie, over direct TLS when secure is set,
// and with one X-Forwarded-Proto header per entry of protos (nil sends none).
func sendAuth(t *testing.T, server *Server, path, body, token string, secure bool, protos []string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if secure {
		req.TLS = &tls.ConnectionState{}
	}
	for _, proto := range protos {
		req.Header.Add("X-Forwarded-Proto", proto)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)
	return recorder
}

// sessionSetCookie returns the only session cookie of a response, parsed and as its raw Set-Cookie line.
func sessionSetCookie(t *testing.T, header http.Header, what string) (*http.Cookie, string) {
	t.Helper()
	var raws []string
	for _, line := range header.Values("Set-Cookie") {
		if strings.HasPrefix(line, sessionCookie+"=") {
			raws = append(raws, line)
		}
	}
	if len(raws) != 1 {
		t.Fatalf("%s: session Set-Cookie lines %q, want one", what, raws)
	}
	cookie, err := http.ParseSetCookie(raws[0])
	if err != nil {
		t.Fatalf("%s: parse %q: %v", what, raws[0], err)
	}
	return cookie, raws[0]
}

func noSessionSetCookie(t *testing.T, header http.Header, what string) {
	t.Helper()
	for _, line := range header.Values("Set-Cookie") {
		if strings.HasPrefix(line, sessionCookie+"=") {
			t.Fatalf("%s: unexpected session Set-Cookie %q", what, line)
		}
	}
}

func assertSecure(t *testing.T, cookie *http.Cookie, raw string, want bool, what string) {
	t.Helper()
	if cookie.Secure != want || strings.Contains(raw, "; Secure") != want {
		t.Fatalf("%s: Secure parsed %v, raw %q, want Secure %v", what, cookie.Secure, raw, want)
	}
}

// loginCookie logs alice in and returns the session cookie, which must have a value.
func loginCookie(t *testing.T, server *Server, secure bool, protos []string, what string) (*http.Cookie, string) {
	t.Helper()
	recorder := sendAuth(t, server, loginPath, cookieLogin, "", secure, protos)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s: login %d %s", what, recorder.Code, recorder.Body.String())
	}
	cookie, raw := sessionSetCookie(t, recorder.Result().Header, what)
	if cookie.Value == "" {
		t.Fatalf("%s: empty session value: %q", what, raw)
	}
	return cookie, raw
}

// TC-001: login over plain HTTP with X-Forwarded-Proto: https sets a Secure cookie.
func TestSessionCookie_TC001_ForwardedHTTPSLoginSetsSecure(t *testing.T) {
	server := newCookieServer(t)
	cookie, raw := loginCookie(t, server, false, []string{"https"}, "X-Forwarded-Proto: https")
	assertSecure(t, cookie, raw, true, "X-Forwarded-Proto: https")
}

// TC-002: header values that state HTTPS.
func TestSessionCookie_TC002_HeaderValuesStatingHTTPS(t *testing.T) {
	server := newCookieServer(t)
	for _, value := range []string{"HTTPS", "Https", " https ", "https, http", "https,http", " HTTPS , http"} {
		what := "X-Forwarded-Proto: " + value
		cookie, raw := loginCookie(t, server, false, []string{value}, what)
		assertSecure(t, cookie, raw, true, what)
	}
}

// TC-003: header values that do not state HTTPS, and no header.
func TestSessionCookie_TC003_HeaderValuesNotStatingHTTPS(t *testing.T) {
	server := newCookieServer(t)
	rows := []struct {
		name   string
		protos []string
	}{
		{"no header", nil},
		{"http", []string{"http"}},
		{"HTTP", []string{"HTTP"}},
		{"empty value", []string{""}},
		{"only spaces", []string{"   "}},
		{"ftp", []string{"ftp"}},
		{"http, https", []string{"http, https"}},
		{"httpsx", []string{"httpsx"}},
		{"https;", []string{"https;"}},
		{"xhttps", []string{"xhttps"}},
	}
	for _, row := range rows {
		cookie, raw := loginCookie(t, server, false, row.protos, row.name)
		assertSecure(t, cookie, raw, false, row.name)
	}
}

// TC-004: direct TLS gives a Secure cookie whatever the header says.
func TestSessionCookie_TC004_DirectTLSIsAlwaysSecure(t *testing.T) {
	server := newCookieServer(t)
	rows := []struct {
		name   string
		protos []string
	}{
		{"TLS, no header", nil},
		{"TLS, http", []string{"http"}},
		{"TLS, https", []string{"https"}},
		{"TLS, empty value", []string{""}},
	}
	for _, row := range rows {
		cookie, raw := loginCookie(t, server, true, row.protos, row.name)
		assertSecure(t, cookie, raw, true, row.name)
	}
}

// TC-005: only Secure differs between plain HTTP and proxied HTTPS login.
func TestSessionCookie_TC005_OnlySecureDiffers(t *testing.T) {
	server := newCookieServer(t)
	plain, plainRaw := loginCookie(t, server, false, nil, "plain HTTP")
	proxied, proxiedRaw := loginCookie(t, server, false, []string{"https"}, "X-Forwarded-Proto: https")
	for _, row := range []struct {
		what   string
		cookie *http.Cookie
		raw    string
		secure bool
	}{{"plain HTTP", plain, plainRaw, false}, {"proxied HTTPS", proxied, proxiedRaw, true}} {
		c := row.cookie
		if c.Name != sessionCookie || c.Value == "" || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 2592000 {
			t.Fatalf("%s: cookie attributes %q", row.what, row.raw)
		}
		assertSecure(t, c, row.raw, row.secure, row.what)
	}
	strip := func(raw, value string) string {
		return strings.Replace(strings.Replace(raw, value, "", 1), "; Secure", "", 1)
	}
	if strip(plainRaw, plain.Value) != strip(proxiedRaw, proxied.Value) {
		t.Fatalf("Set-Cookie differs beyond value and Secure: %q vs %q", plainRaw, proxiedRaw)
	}
}

// assertClearingCookie checks a 204 logout answer with its clearing session cookie and returns the raw line.
func assertClearingCookie(t *testing.T, recorder *httptest.ResponseRecorder, secure bool, what string) string {
	t.Helper()
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("%s: logout %d %s", what, recorder.Code, recorder.Body.String())
	}
	cookie, raw := sessionSetCookie(t, recorder.Result().Header, what)
	if cookie.Value != "" || cookie.MaxAge >= 0 || !strings.Contains(raw, "Max-Age=0") || !cookie.HttpOnly ||
		cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("%s: clearing cookie %q", what, raw)
	}
	assertSecure(t, cookie, raw, secure, what)
	return raw
}

// TC-006: the logout clearing cookie follows the same Secure rule.
func TestSessionCookie_TC006_LogoutClearingCookieFollowsTheRule(t *testing.T) {
	server := newCookieServer(t)
	rows := []struct {
		name   string
		tls    bool
		protos []string
		secure bool
	}{
		{"plain, https", false, []string{"https"}, true},
		{"plain, HTTPS , http", false, []string{"HTTPS , http"}, true},
		{"plain, no header", false, nil, false},
		{"plain, http, https", false, []string{"http, https"}, false},
		{"plain, empty value", false, []string{""}, false},
		{"TLS, http", true, []string{"http"}, true},
	}
	for _, row := range rows {
		session, _ := loginCookie(t, server, false, nil, row.name+" login")
		assertClearingCookie(t, sendAuth(t, server, logoutPath, "", session.Value, row.tls, row.protos), row.secure, row.name)
	}

	// Logout without a session cookie still clears it.
	assertClearingCookie(t, sendAuth(t, server, logoutPath, "", "", false, []string{"https"}), true, "logout without cookie")

	// Repeated logout of the same session answers the same clearing cookie.
	session, _ := loginCookie(t, server, false, nil, "repeat login")
	first := assertClearingCookie(t, sendAuth(t, server, logoutPath, "", session.Value, false, []string{"https"}), true, "first logout")
	second := assertClearingCookie(t, sendAuth(t, server, logoutPath, "", session.Value, false, []string{"https"}), true, "second logout")
	if first != second {
		t.Fatalf("repeated logout cookie %q, first %q", second, first)
	}
}

// TC-007: with several X-Forwarded-Proto headers only the first is judged.
func TestSessionCookie_TC007_OnlyTheFirstHeaderCounts(t *testing.T) {
	server := newCookieServer(t)
	cookie, raw := loginCookie(t, server, false, []string{"https", "http"}, "https then http")
	assertSecure(t, cookie, raw, true, "https then http")
	cookie, raw = loginCookie(t, server, false, []string{"http", "https"}, "http then https")
	assertSecure(t, cookie, raw, false, "http then https")
}

// browserClient talks to a real plain HTTP test server, with a cookie jar like a browser when jar is set.
type browserClient struct {
	t      *testing.T
	base   string
	client *http.Client
	jar    http.CookieJar
}

// browserServer is a real plain HTTP test server reached under a non-loopback host name:
// cookie jars, like browsers, treat loopback hosts as secure origins and would send Secure cookies to them.
type browserServer struct {
	base      string
	transport *http.Transport
}

func newBrowserServer(t *testing.T) browserServer {
	t.Helper()
	ts := httptest.NewServer(newCookieServer(t).Handler())
	t.Cleanup(ts.Close)
	addr := ts.Listener.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	return browserServer{base: "http://pricefollower.test:" + port, transport: transport}
}

func newBrowserClient(t *testing.T, server browserServer) *browserClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &browserClient{t: t, base: server.base, client: &http.Client{Transport: server.transport, Jar: jar}, jar: jar}
}

// do sends a request with one X-Forwarded-Proto header per entry of protos and an optional replayed cookie,
// and returns the response with its body read.
func (b *browserClient) do(method, path, body string, protos []string, cookie *http.Cookie) (*http.Response, []byte) {
	b.t.Helper()
	req, err := http.NewRequest(method, b.base+path, strings.NewReader(body))
	if err != nil {
		b.t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, proto := range protos {
		req.Header.Add("X-Forwarded-Proto", proto)
	}
	if cookie != nil {
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
	}
	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		b.t.Fatal(err)
	}
	return resp, data
}

// reload answers the logged-in username of GET /api/v1/auth/session, "" for user null (login screen).
func (b *browserClient) reload(protos []string, cookie *http.Cookie) string {
	b.t.Helper()
	resp, data := b.do(http.MethodGet, sessionPath, "", protos, cookie)
	var body struct {
		Mode string `json:"mode"`
		User *struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if resp.StatusCode != http.StatusOK || json.Unmarshal(data, &body) != nil || body.Mode != "protected" {
		b.t.Fatalf("session %d %s", resp.StatusCode, data)
	}
	if body.User == nil {
		return ""
	}
	return body.User.Username
}

func (b *browserClient) login(protos []string, password string) (*http.Response, []byte) {
	b.t.Helper()
	return b.do(http.MethodPost, loginPath, `{"username":"alice","password":"`+password+`"}`, protos, nil)
}

// jarHasSession reports whether the jar would send a session cookie to the server.
func (b *browserClient) jarHasSession() bool {
	u, err := url.Parse(b.base)
	if err != nil {
		b.t.Fatal(err)
	}
	for _, cookie := range b.jar.Cookies(u) {
		if cookie.Name == sessionCookie {
			return true
		}
	}
	return false
}

// TC-008: development login over plain HTTP stays logged in after reload.
func TestSessionCookie_TC008_DevelopmentLoginSurvivesReload(t *testing.T) {
	site := newBrowserServer(t)
	for _, row := range []struct {
		name   string
		protos []string
	}{{"no header", nil}, {"X-Forwarded-Proto: http", []string{"http"}}} {
		browser := newBrowserClient(t, site)
		if user := browser.reload(row.protos, nil); user != "" {
			t.Fatalf("%s: before login user %q", row.name, user)
		}
		resp, data := browser.login(row.protos, cookiePassword)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: login %d %s", row.name, resp.StatusCode, data)
		}
		cookie, raw := sessionSetCookie(t, resp.Header, row.name)
		assertSecure(t, cookie, raw, false, row.name)
		for reload := 1; reload <= 2; reload++ {
			if user := browser.reload(row.protos, nil); user != cookieUsername {
				t.Fatalf("%s: reload %d user %q, want alice", row.name, reload, user)
			}
		}
	}

	browser := newBrowserClient(t, site)
	resp, data := browser.login(nil, "wrong password")
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	if resp.StatusCode != http.StatusUnauthorized || json.Unmarshal(data, &body) != nil || body.Error.Code != "INVALID_CREDENTIALS" {
		t.Fatalf("wrong password: %d %s", resp.StatusCode, data)
	}
	noSessionSetCookie(t, resp.Header, "wrong password")
	if user := browser.reload(nil, nil); user != "" {
		t.Fatalf("after wrong password user %q", user)
	}
}

// TC-009: a header stating HTTPS sent directly over plain HTTP harms only that client.
func TestSessionCookie_TC009_ForwardedHTTPSOverPlainHTTPHarmsOnlyThatClient(t *testing.T) {
	site := newBrowserServer(t)
	first := newBrowserClient(t, site)
	resp, data := first.login([]string{"https"}, cookiePassword)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login %d %s", resp.StatusCode, data)
	}
	cookie, raw := sessionSetCookie(t, resp.Header, "X-Forwarded-Proto: https over plain HTTP")
	assertSecure(t, cookie, raw, true, "X-Forwarded-Proto: https over plain HTTP")
	if first.jarHasSession() {
		t.Fatal("jar sends a Secure cookie over plain HTTP")
	}
	if user := first.reload(nil, nil); user != "" {
		t.Fatalf("first client reload user %q, want null", user)
	}

	second := newBrowserClient(t, site)
	if resp, data := second.login(nil, cookiePassword); resp.StatusCode != http.StatusOK {
		t.Fatalf("second login %d %s", resp.StatusCode, data)
	}
	if user := second.reload(nil, nil); user != cookieUsername {
		t.Fatalf("second client reload user %q, want alice", user)
	}
	if user := first.reload(nil, nil); user != "" {
		t.Fatalf("first client after second login user %q, want null", user)
	}
}

// TC-010: the logout flow removes the cookie behind the proxy and over plain HTTP.
func TestSessionCookie_TC010_LogoutFlowRemovesTheCookie(t *testing.T) {
	site := newBrowserServer(t)

	// Development: plain HTTP, no header, cookie jar.
	dev := newBrowserClient(t, site)
	if resp, data := dev.login(nil, cookiePassword); resp.StatusCode != http.StatusOK {
		t.Fatalf("development login %d %s", resp.StatusCode, data)
	}
	if user := dev.reload(nil, nil); user != cookieUsername {
		t.Fatalf("development reload user %q, want alice", user)
	}
	resp, data := dev.do(http.MethodPost, logoutPath, "", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("development logout %d %s", resp.StatusCode, data)
	}
	cleared, raw := sessionSetCookie(t, resp.Header, "development logout")
	if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Fatalf("development logout cookie %q", raw)
	}
	assertSecure(t, cleared, raw, false, "development logout")
	if dev.jarHasSession() {
		t.Fatal("development jar still holds the session cookie")
	}
	if user := dev.reload(nil, nil); user != "" {
		t.Fatalf("development reload after logout user %q", user)
	}

	// Proxy: every request carries X-Forwarded-Proto: https; the cookie is replayed as a browser on HTTPS would.
	https := []string{"https"}
	proxy := &browserClient{t: t, base: site.base, client: &http.Client{Transport: site.transport}}
	resp, data = proxy.login(https, cookiePassword)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("proxy login %d %s", resp.StatusCode, data)
	}
	session, raw := sessionSetCookie(t, resp.Header, "proxy login")
	assertSecure(t, session, raw, true, "proxy login")
	if user := proxy.reload(https, session); user != cookieUsername {
		t.Fatalf("proxy reload user %q, want alice", user)
	}
	resp, data = proxy.do(http.MethodPost, logoutPath, "", https, session)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("proxy logout %d %s", resp.StatusCode, data)
	}
	cleared, raw = sessionSetCookie(t, resp.Header, "proxy logout")
	if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Fatalf("proxy logout did not clear the cookie: %q", raw)
	}
	assertSecure(t, cleared, raw, true, "proxy logout")
	// The browser dropped the cleared cookie: the reload sends none.
	if user := proxy.reload(https, nil); user != "" {
		t.Fatalf("proxy reload after logout user %q", user)
	}
	if user := proxy.reload(https, session); user != "" {
		t.Fatalf("old session token after logout user %q", user)
	}
}

// TC-011: a refused login sets no cookie, whatever the header.
func TestSessionCookie_TC011_RefusedLoginSetsNoCookie(t *testing.T) {
	server := newCookieServer(t)
	https := []string{"https"}
	wrong := sendAuth(t, server, loginPath, `{"username":"alice","password":"wrong password"}`, "", false, https)
	assertError(t, wrong, http.StatusUnauthorized, "INVALID_CREDENTIALS", "")
	noSessionSetCookie(t, wrong.Result().Header, "wrong password")
	invalid := sendAuth(t, server, loginPath, `{"username":`, "", false, https)
	assertError(t, invalid, http.StatusBadRequest, "INVALID_JSON", "")
	noSessionSetCookie(t, invalid.Result().Header, "invalid JSON")
}
