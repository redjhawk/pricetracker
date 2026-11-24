package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// Issue #273: the application's HTTP server limits the time to receive a whole request (15 s) and to
// send a response (60 s), and the size of the request headers (64 KiB). Live cases serve the real
// handler through NewHTTPServer on a loopback listener, with LeBoncoin and Claude faked.
//
// TC-013 is the existing suites (go test ./..., npx playwright test), which run unchanged.

const (
	limitsGoal = "A quiet lamp for a desk"
	limitsPage = `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":%s,"status":"active","subject":"Desk lamp","price":[12]}}}}</script>`
	// limitsReview is a valid review answer of the fake Claude.
	limitsReview = `{"price":{"rating":"good_deal","explanation":"Cheap."},"condition":{"rating":null,"explanation":"No photos."},` +
		`"recommendation":{"rating":"buy","explanation":"Go."},"fairPrice":{"minCents":1000,"maxCents":1600,"suggestedOfferCents":1200},` +
		`"risks":[],"missingInformation":[],"sellerQuestions":["Receipt?"],"descriptionVsPhotos":{"matches":true,"explanation":"Same.","mismatches":[]}}`
	limitsResponseSize = 256 << 20 // TC-005: far larger than the loopback buffers
)

// TC-001
func TestHTTPServerLimits_TC001_TheServerConstructorSetsEveryLimit(t *testing.T) {
	for _, development := range []bool{false, true} {
		handler := http.NewServeMux()
		server := NewHTTPServer(config.Config{Host: "127.0.0.1", Port: 8081, Development: development}, handler)
		if server == nil {
			t.Fatalf("development=%v: no server", development)
		}
		for _, check := range []struct {
			name      string
			got, want time.Duration
		}{
			{"ReadTimeout", server.ReadTimeout, 15 * time.Second},
			{"WriteTimeout", server.WriteTimeout, 60 * time.Second},
			{"ReadHeaderTimeout", server.ReadHeaderTimeout, 5 * time.Second},
			{"IdleTimeout", server.IdleTimeout, 60 * time.Second},
		} {
			if check.got != check.want {
				t.Errorf("development=%v: %s %v, want %v", development, check.name, check.got, check.want)
			}
		}
		if server.MaxHeaderBytes != 65_536 {
			t.Errorf("development=%v: MaxHeaderBytes %d, want 65536 (0 would be Go's 1 MiB default)", development, server.MaxHeaderBytes)
		}
		if server.Addr != "127.0.0.1:8081" {
			t.Errorf("development=%v: Addr %q, want 127.0.0.1:8081", development, server.Addr)
		}
		if server.Handler != http.Handler(handler) {
			t.Errorf("development=%v: Handler is not the given handler", development)
		}
	}
}

// TC-001: cmd/ builds its server only through the constructor.
func TestHTTPServerLimits_TC001_MainBuildsItsServerWithTheConstructor(t *testing.T) {
	usesConstructor := false
	err := filepath.WalkDir("../../cmd", func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			return err
		}
		source, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if strings.Contains(string(source), "http.Server{") {
			t.Errorf("%s builds an http.Server literal; use httpapi.NewHTTPServer", file)
		}
		usesConstructor = usesConstructor || strings.Contains(string(source), "httpapi.NewHTTPServer(")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !usesConstructor {
		t.Fatal("cmd/ does not build its server with httpapi.NewHTTPServer")
	}
}

// TestHTTPServerLimits_Live runs every live case at the same time with one fake network: each
// case uses its own LeBoncoin listing IDs and Claude tokens.
func TestHTTPServerLimits_Live(t *testing.T) {
	fake := withLimitsNet(t)
	runConcurrently(t, []namedTest{
		{"TC-002 add item body one byte per second", func(t *testing.T) {
			testSlowBodyIsDropped(t, fake, "2730021", http.MethodPost, "/api/v1/items", `{"url":"`+limitsListingURL("2730022")+`"}`, "2730022")
		}},
		{"TC-002 settings body one byte per second", func(t *testing.T) {
			testSlowBodyIsDropped(t, fake, "2730023", http.MethodPut, "/api/v1/settings", `{"leboncoinSession":{"value":"datadome=synthetic-limits-session","revision":0}}`, "")
		}},
		{"TC-002 purchase goal body one byte per second", func(t *testing.T) {
			testSlowBodyIsDropped(t, fake, "2730024", http.MethodPut, "/api/v1/items/I1/purchase-goal", `{"purchaseGoal":"`+limitsGoal+`"}`, "")
		}},
		{"TC-003 kept-alive connection", func(t *testing.T) { testKeptAliveSlowBody(t, fake) }},
		{"TC-004 1 MiB body at normal speed", func(t *testing.T) { testLargeBodyAtNormalSpeed(t, fake) }},
		{"TC-005 client that does not read", func(t *testing.T) { testUnreadResponseIsCutOff(t) }},
		{"TC-006 response not started within 60 s", func(t *testing.T) { testLateResponseIsNeverReceived(t) }},
		{"TC-007 headers up to 64 KiB", func(t *testing.T) { testHeadersUpTo64KiB(t, fake) }},
		{"TC-008 headers over 68 KiB", func(t *testing.T) { testHeadersOver68KiB(t, fake) }},
		{"TC-009 Claude accepts after 19 s", func(t *testing.T) { testClaudeAcceptsAfter19s(t, fake) }},
		{"TC-009 Claude never answers", func(t *testing.T) { testClaudeNeverAnswers(t, fake) }},
		{"TC-009 Claude refuses at once", func(t *testing.T) { testClaudeRefuses(t, fake) }},
		{"TC-011 work released after a few seconds", func(t *testing.T) {
			testBackgroundWorkAnswersAtOnce(t, fake, 3*time.Second, "2730111", "2730112", "sk-ant-oat01-limits-review-short-a111")
		}},
		{"TC-011 work blocked for 70 s", func(t *testing.T) {
			testBackgroundWorkAnswersAtOnce(t, fake, 70*time.Second, "2730113", "2730114", "sk-ant-oat01-limits-review-long-b222")
		}},
		{"TC-012 silent connection", func(t *testing.T) { testSilentConnection(t) }},
		{"TC-012 headers one byte per second", func(t *testing.T) { testSlowHeaders(t) }},
		{"TC-012 idle kept-alive connection", func(t *testing.T) { testIdleConnection(t) }},
	})
}

type namedTest struct {
	name string
	run  func(*testing.T)
}

// runConcurrently runs the subtests at the same time (t.Run may be called from several goroutines).
// t.Parallel is not used: its limit (GOMAXPROCS) would queue the long waits one after the other.
func runConcurrently(t *testing.T, tests []namedTest) {
	var group sync.WaitGroup
	for _, test := range tests {
		group.Add(1)
		go func() {
			defer group.Done()
			t.Run(test.name, test.run)
		}()
	}
	group.Wait()
}

// TC-002
func testSlowBodyIsDropped(t *testing.T, fake *limitsNet, itemListing, method, target, body, addedListing string) {
	fake.addListing(itemListing, nil)
	if addedListing != "" {
		fake.addListing(addedListing, nil)
	}
	s := openLimitsServer(t, limitsItem{"I1", itemListing})
	start := time.Now()
	conn := dialLimits(t, s.addr)
	request := rawRequest(method, target, s.session, nil, body)
	writeAll(t, conn, request[:len(request)-len(body)]) // the headers arrive at once, well within 5 s
	writeSlowly(t, conn, body)                          // the last byte would arrive after 20 s
	at, received, err := readUntilClosed(conn, conn, start.Add(25*time.Second))
	assertClosedBetween(t, "request with a slow body", start, at, err, 15*time.Second, 20*time.Second)
	if received != 0 {
		t.Fatalf("received %d response bytes, want none", received)
	}
	if ids := s.itemIDs(t); len(ids) != 1 || ids[0] != "I1" {
		t.Fatalf("items %v, want only I1", ids)
	}
	if session := s.getObject(t, "/api/v1/settings/leboncoin-session", "session"); session["saved"] != false {
		t.Fatalf("LeBoncoin session saved by a dropped request: %v", session)
	}
	if goal := s.details(t, "I1")["purchaseGoal"]; goal != "" {
		t.Fatalf("goal %q saved by a dropped request", goal)
	}
	for _, listing := range []string{itemListing, addedListing} {
		if listing != "" && fake.calls(listing) != 0 {
			t.Fatalf("LeBoncoin listing %s requested %d times, want 0", listing, fake.calls(listing))
		}
	}
}

// TC-003
func testKeptAliveSlowBody(t *testing.T, fake *limitsNet) {
	fake.addListing("2730031", nil)
	s := openLimitsServer(t, limitsItem{"I1", "2730031"})
	conn := dialLimits(t, s.addr)
	reader := bufio.NewReader(conn)
	writeAll(t, conn, rawRequest(http.MethodGet, "/api/v1/items", s.session, nil, ""))
	status, list := readResponse(t, conn, reader)
	if status != http.StatusOK || !strings.Contains(list, `"id":"I1"`) {
		t.Fatalf("first request: %d %s", status, list)
	}
	start := time.Now()
	time.Sleep(3 * time.Second)
	body := `{"purchaseGoal":"` + limitsGoal + `"}`
	request := rawRequest(http.MethodPut, "/api/v1/items/I1/purchase-goal", s.session, nil, body)
	writeAll(t, conn, request[:len(request)-len(body)])
	writeSlowly(t, conn, body)
	at, received, err := readUntilClosed(conn, reader, start.Add(25*time.Second))
	assertClosedBetween(t, "second request with a slow body", start, at, err, 15*time.Second, 20*time.Second)
	if received != 0 {
		t.Fatalf("received %d bytes of a second response, want none", received)
	}
	if goal := s.details(t, "I1")["purchaseGoal"]; goal != "" {
		t.Fatalf("goal %q saved by a dropped request", goal)
	}
}

// TC-004
func testLargeBodyAtNormalSpeed(t *testing.T, fake *limitsNet) {
	fake.addListing("2730041", nil)
	s := openLimitsServer(t, limitsItem{"I1", "2730041"})
	start := time.Now()
	padded := func(size int) string {
		head := `{"purchaseGoal":"` + limitsGoal + `"`
		return head + strings.Repeat(" ", size-len(head)-1) + "}"
	}
	status, data := s.call(t, http.MethodPut, "/api/v1/items/I1/purchase-goal", padded(1_048_000))
	want := `{"changed":true,"purchaseGoal":"` + limitsGoal + `","reviewStarted":false}`
	if status != http.StatusOK || strings.TrimSpace(string(data)) != want {
		t.Fatalf("1,048,000-byte body: %d %s", status, data)
	}
	if goal := s.details(t, "I1")["purchaseGoal"]; goal != limitsGoal {
		t.Fatalf("goal %q, want %q", goal, limitsGoal)
	}
	status, data = s.call(t, http.MethodPut, "/api/v1/items/I1/purchase-goal", padded(1_048_577))
	if code := errorCode(data); status != http.StatusRequestEntityTooLarge || code != "REQUEST_TOO_LARGE" {
		t.Fatalf("1 MiB + 1 byte body: %d %s, want 413 REQUEST_TOO_LARGE", status, data)
	}
	if elapsed := time.Since(start); elapsed >= 15*time.Second {
		t.Fatalf("the exchanges took %v, want well under 15 s", elapsed)
	}
}

// TC-005
func testUnreadResponseIsCutOff(t *testing.T) {
	type writes struct{ received, failed time.Time }
	result := make(chan writes, 1)
	addr := serveTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/large" {
			http.NotFound(w, r)
			return
		}
		received := time.Now()
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprint(limitsResponseSize))
		chunk := make([]byte, 64<<10)
		for written := 0; written < limitsResponseSize; written += len(chunk) {
			if _, err := w.Write(chunk); err != nil {
				result <- writes{received, time.Now()} // stop at the first failed write
				return
			}
		}
		result <- writes{received: received}
	}))
	// The loopback buffers (a few MiB) are far smaller than the response, so the server is blocked by
	// the client. They are not shrunk to 4 KiB: the queued data would then trickle in through TCP
	// zero-window probes long after the client starts reading.
	conn := dialLimits(t, addr)
	writeAll(t, conn, "GET /large HTTP/1.1\r\nHost: 127.0.0.1\r\n\r\n")
	time.Sleep(62 * time.Second)
	_, received, err := readUntilClosed(conn, conn, time.Now().Add(30*time.Second))
	if !connectionClosed(err) {
		t.Fatalf("the connection did not end after %d bytes: %v", received, err)
	}
	if received >= limitsResponseSize {
		t.Fatalf("received %d bytes, want fewer than the %d-byte body", received, limitsResponseSize)
	}
	select {
	case got := <-result:
		if got.failed.IsZero() {
			t.Fatal("every write succeeded, want the server to stop sending")
		}
		if elapsed := got.failed.Sub(got.received); elapsed < 59*time.Second || elapsed > 65*time.Second {
			t.Fatalf("first failed write %v after the request, want between 60 s and 65 s", elapsed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the handler did not receive the request or did not stop")
	}
}

// TC-006
func testLateResponseIsNeverReceived(t *testing.T) {
	var changed atomic.Bool
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	addr := serveTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ping":
			w.WriteHeader(http.StatusOK)
		case "/slow":
			if _, err := io.ReadAll(r.Body); err != nil {
				return
			}
			changed.Store(true)
			select {
			case <-time.After(61 * time.Second):
			case <-done:
				return
			}
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "done")
		default:
			http.NotFound(w, r)
		}
	}))
	conn := dialLimits(t, addr)
	writeAll(t, conn, "PUT /slow HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nContent-Length: 9\r\n\r\n{\"a\":\"b\"}")
	sent := time.Now()
	ping := make(chan error, 1)
	go func() {
		time.Sleep(30 * time.Second)
		client := &http.Client{Transport: &http.Transport{}, Timeout: 5 * time.Second}
		defer client.CloseIdleConnections()
		begin := time.Now()
		response, err := client.Get("http://" + addr + "/ping")
		if err != nil {
			ping <- err
			return
		}
		response.Body.Close()
		if elapsed := time.Since(begin); response.StatusCode != http.StatusOK || elapsed > 2*time.Second {
			ping <- fmt.Errorf("status %d after %v, want 200 at once", response.StatusCode, elapsed)
			return
		}
		ping <- nil
	}()
	at, received, err := readUntilClosed(conn, conn, sent.Add(70*time.Second))
	assertClosedBetween(t, "slow handler", sent, at, err, 60*time.Second, 65*time.Second)
	if received != 0 {
		t.Fatalf("received %d response bytes, want none", received)
	}
	if !changed.Load() {
		t.Fatal("the change recorded by the handler is lost")
	}
	select {
	case err := <-ping:
		if err != nil {
			t.Fatalf("second connection at 30 s: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("second connection at 30 s did not answer")
	}
}

// TC-007
func testHeadersUpTo64KiB(t *testing.T, fake *limitsNet) {
	fake.addListing("2730071", nil)
	s := openLimitsServer(t, limitsItem{"I1", "2730071"})
	listed := func(what, request string) {
		t.Helper()
		status, body := rawExchange(t, s.addr, request)
		if status != http.StatusOK || !strings.Contains(body, `"id":"I1"`) {
			t.Fatalf("%s: %d %.200s, want 200 with the item list", what, status, body)
		}
	}
	listed("60 KiB header", rawRequest(http.MethodGet, "/api/v1/items", s.session, []string{padding(61_440)}, ""))
	listed("65,536 bytes of headers", sizedRequest(t, http.MethodGet, "/api/v1/items", s.session, 65_536))
	cookie := "Cookie: " + sessionCookie + "=" + s.session + "; pricefollower_padding=" + strings.Repeat("a", 61_440)
	listed("60 KiB cookie", rawRequest(http.MethodGet, "/api/v1/items", "", []string{cookie}, ""))
	usual := request(t, s.api, http.MethodGet, "/", "", "").Code
	if status, body := rawExchange(t, s.addr, rawRequest(http.MethodGet, "/", "", []string{padding(61_440)}, "")); status != usual || status == http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("page with a 60 KiB header: %d %.200s, want its usual %d", status, body, usual)
	}
}

// TC-008
func testHeadersOver68KiB(t *testing.T, fake *limitsNet) {
	fake.addListing("2730081", nil)
	fake.addListing("2730082", nil)
	s := openLimitsServer(t, limitsItem{"I1", "2730081"})
	large := []string{padding(102_400)}
	refused := func(what, request string) {
		t.Helper()
		status, body, closed := refusedExchange(t, s.addr, request)
		if status != http.StatusRequestHeaderFieldsTooLarge || strings.Contains(body, "I1") || strings.Contains(body, `"items"`) {
			t.Fatalf("%s: %d %.200s, want 431 without the item list", what, status, body)
		}
		if !closed {
			t.Fatalf("%s: the connection stayed open after the 431 answer", what)
		}
	}
	refused("100 KiB header", rawRequest(http.MethodGet, "/api/v1/items", s.session, large, ""))
	refused("69,633 bytes of headers", sizedRequest(t, http.MethodGet, "/api/v1/items", s.session, 69_633))
	refused("100 KiB header without a session", rawRequest(http.MethodGet, "/api/v1/items", "", large, ""))
	cookie := "Cookie: " + sessionCookie + "=" + s.session + "; pricefollower_padding=" + strings.Repeat("a", 102_400)
	refused("100 KiB cookie", rawRequest(http.MethodGet, "/api/v1/items", "", []string{cookie}, ""))
	refused("add item with a 100 KiB header", rawRequest(http.MethodPost, "/api/v1/items", s.session, large, `{"url":"`+limitsListingURL("2730082")+`"}`))
	refused("page with a 100 KiB header", rawRequest(http.MethodGet, "/", "", large, ""))
	for attempt := 1; attempt <= 3; attempt++ {
		refused(fmt.Sprintf("repeated 100 KiB header %d", attempt), rawRequest(http.MethodGet, "/api/v1/items", s.session, large, ""))
	}
	status, body := rawExchange(t, s.addr, rawRequest(http.MethodGet, "/api/v1/items", s.session, nil, ""))
	if status != http.StatusOK || !strings.Contains(body, `"id":"I1"`) {
		t.Fatalf("normal request afterwards: %d %s", status, body)
	}
	if ids := s.itemIDs(t); len(ids) != 1 || ids[0] != "I1" {
		t.Fatalf("items %v, want only I1", ids)
	}
	if calls := fake.calls("2730082"); calls != 0 {
		t.Fatalf("the refused add requested LeBoncoin %d times", calls)
	}
}

// TC-009
func testClaudeAcceptsAfter19s(t *testing.T, fake *limitsNet) {
	const token = "sk-ant-oat01-limits-accept-19s-a1b2"
	s, status, answer, elapsed := saveClaudeToken(t, fake, token, verifyAfter19s)
	if status != http.StatusOK || elapsed < 19*time.Second || elapsed > 21*time.Second {
		t.Fatalf("status %d after %v, want 200 between 19 s and 21 s: %v", status, elapsed, answer)
	}
	saved, _ := answer["claudeToken"].(map[string]any)
	if saved["saved"] != true || saved["hint"] != lastFour(token) {
		t.Fatalf("answer token %v, want saved with hint %q", saved, lastFour(token))
	}
	if loaded := s.getObject(t, claudeTokenPath, "claudeToken"); loaded["saved"] != true || loaded["hint"] != lastFour(token) {
		t.Fatalf("loaded token %v, want saved with hint %q", loaded, lastFour(token))
	}
}

// TC-009
func testClaudeNeverAnswers(t *testing.T, fake *limitsNet) {
	s, status, answer, elapsed := saveClaudeToken(t, fake, "sk-ant-oat01-limits-silent-c3d4", verifyNever)
	if status != http.StatusBadGateway || elapsed < 19500*time.Millisecond || elapsed > 25*time.Second {
		t.Fatalf("status %d after %v, want 502 between 19.5 s and 25 s: %v", status, elapsed, answer)
	}
	failure, _ := answer["error"].(map[string]any)
	if failure["code"] != "CLAUDE_UNREACHABLE" || failure["message"] != "The token could not be verified because Claude could not be reached. Try again." {
		t.Fatalf("error %v, want CLAUDE_UNREACHABLE", answer)
	}
	if loaded := s.getObject(t, claudeTokenPath, "claudeToken"); loaded["saved"] != false {
		t.Fatalf("token saved after a failed verification: %v", loaded)
	}
}

// TC-009
func testClaudeRefuses(t *testing.T, fake *limitsNet) {
	s, status, answer, elapsed := saveClaudeToken(t, fake, "sk-ant-oat01-limits-reject-e5f6", verifyReject)
	failure, _ := answer["error"].(map[string]any)
	if status != http.StatusUnprocessableEntity || failure["code"] != "CLAUDE_TOKEN_REJECTED" || elapsed > 2*time.Second {
		t.Fatalf("status %d after %v: %v, want 422 CLAUDE_TOKEN_REJECTED at once", status, elapsed, answer)
	}
	if loaded := s.getObject(t, claudeTokenPath, "claudeToken"); loaded["saved"] != false {
		t.Fatalf("token saved after a refusal: %v", loaded)
	}
}

// saveClaudeToken sends PUT /api/v1/settings with a new token through a client without time limit; the
// answer must be complete JSON.
func saveClaudeToken(t *testing.T, fake *limitsNet, token string, verify limitsVerify) (*limitsServer, int, map[string]any, time.Duration) {
	t.Helper()
	fake.addToken(token, limitsToken{verify: verify})
	s := openLimitsServer(t)
	begin := time.Now()
	status, data := s.call(t, http.MethodPut, settingsSavePath, settingsBody(tokenPart(token)))
	elapsed := time.Since(begin)
	var answer map[string]any
	if err := json.Unmarshal(data, &answer); err != nil {
		t.Fatalf("answer after %v is not complete JSON: %v: %s", elapsed, err, data)
	}
	return s, status, answer, elapsed
}

// TC-011
func testBackgroundWorkAnswersAtOnce(t *testing.T, fake *limitsNet, hold time.Duration, itemListing, addedListing, token string) {
	release := make(chan struct{})
	var once sync.Once
	releaseWork := func() { once.Do(func() { close(release) }) }
	t.Cleanup(releaseWork)
	fake.addListing(itemListing, release)
	fake.addListing(addedListing, release)
	fake.addToken(token, limitsToken{review: release})
	s := openLimitsServer(t, limitsItem{"I1", itemListing})
	storeSecrets(t, s.database, s.aliceID, nil, text(token))
	started := time.Now()
	quick := func(what, method, target, body string, want int) map[string]any {
		t.Helper()
		begin := time.Now()
		status, data := s.call(t, method, target, body)
		if elapsed := time.Since(begin); elapsed > 5*time.Second {
			t.Fatalf("%s answered after %v, want within 5 s", what, elapsed)
		}
		if status != want {
			t.Fatalf("%s: status %d, want %d: %s", what, status, want, data)
		}
		return decodeLimits(t, what, data)
	}
	added := quick("add item", http.MethodPost, "/api/v1/items", `{"url":"`+limitsListingURL(addedListing)+`"}`, http.StatusCreated)
	addedID, _ := added["id"].(string)
	quick("Refresh price", http.MethodPost, "/api/v1/items/I1/refresh", "", http.StatusAccepted)
	if again := quick("second Refresh price", http.MethodPost, "/api/v1/items/I1/refresh", "", http.StatusAccepted); again["itemsQueued"] != float64(1) {
		t.Fatalf("second Refresh price: %v", again)
	}
	if all := quick("Refresh all", http.MethodPost, "/api/v1/items/refresh", "", http.StatusAccepted); all["itemsQueued"] != float64(2) {
		t.Fatalf("Refresh all: %v, want 2 items queued", all)
	}
	if review := quick("AI review", http.MethodPost, "/api/v1/items/I1/ai-review", "", http.StatusAccepted); review["alreadyRunning"] != false {
		t.Fatalf("AI review: %v", review)
	}
	if details := s.details(t, "I1"); attemptResult(details) != "pending" || !reviewRunning(details) {
		t.Fatalf("I1 while the work is blocked: check %q, review running %v", attemptResult(details), reviewRunning(details))
	}
	if details := s.details(t, addedID); attemptResult(details) != "pending" {
		t.Fatalf("added item while its check is blocked: check %q", attemptResult(details))
	}
	time.Sleep(time.Until(started.Add(hold)))
	releaseWork()
	for _, id := range []string{"I1", addedID} {
		details := waitWorkFinished(t, s, id)
		if hold < 15*time.Second && (attemptResult(details) != "success" || reviewStatus(details) != "succeeded") {
			t.Fatalf("item %s: check %q, review %q, want success and succeeded", id, attemptResult(details), reviewStatus(details))
		}
	}
}

// waitWorkFinished waits until the item's check and AI review are no longer pending.
func waitWorkFinished(t *testing.T, s *limitsServer, id string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		details := s.details(t, id)
		if attemptResult(details) != "pending" && attemptResult(details) != "" && !reviewRunning(details) && reviewStatus(details) != "pending" && reviewStatus(details) != "" {
			return details
		}
		if time.Now().After(deadline) {
			t.Fatalf("item %s still has work pending: check %q, review %q running %v", id, attemptResult(details), reviewStatus(details), reviewRunning(details))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func attemptResult(details map[string]any) string {
	attempt, _ := details["lastAttempt"].(map[string]any)
	result, _ := attempt["result"].(string)
	return result
}

func reviewRunning(details map[string]any) bool {
	review, _ := details["aiReview"].(map[string]any)
	return review["running"] == true
}

func reviewStatus(details map[string]any) string {
	review, _ := details["aiReview"].(map[string]any)
	attempt, _ := review["lastAttempt"].(map[string]any)
	status, _ := attempt["status"].(string)
	return status
}

// TC-012
func testSilentConnection(t *testing.T) {
	addr := openLimitsServer(t).addr
	start := time.Now()
	conn := dialLimits(t, addr)
	at, received, err := readUntilClosed(conn, conn, start.Add(15*time.Second))
	assertClosedBetween(t, "silent connection", start, at, err, 5*time.Second, 7*time.Second)
	if received != 0 {
		t.Fatalf("received %d bytes, want none", received)
	}
}

// TC-012
func testSlowHeaders(t *testing.T) {
	s := openLimitsServer(t)
	start := time.Now()
	conn := dialLimits(t, s.addr)
	writeSlowly(t, conn, rawRequest(http.MethodGet, "/api/v1/items", s.session, nil, ""))
	if err := conn.SetReadDeadline(start.Add(15 * time.Second)); err != nil {
		t.Fatal(err)
	}
	answer, err := io.ReadAll(conn)
	at := time.Now()
	if err == nil {
		err = io.EOF // ReadAll ends without error at the end of the connection
	}
	assertClosedBetween(t, "headers one byte per second", start, at, err, 5*time.Second, 7*time.Second)
	// The request is never handled. Go answers the request line cut by the 5 s limit with its plain
	// "400 Bad Request" (Go's behaviour before this change too), so that is the only answer allowed.
	if len(answer) != 0 && !strings.HasPrefix(string(answer), "HTTP/1.1 400 Bad Request\r\n") {
		t.Fatalf("received %q, want no answer or Go's plain 400 Bad Request", answer)
	}
}

// TC-012
func testIdleConnection(t *testing.T) {
	s := openLimitsServer(t)
	conn := dialLimits(t, s.addr)
	reader := bufio.NewReader(conn)
	// The request takes milliseconds, so its start stands for the end of the response.
	start := time.Now()
	writeAll(t, conn, rawRequest(http.MethodGet, "/api/v1/items", s.session, nil, ""))
	if status, body := readResponse(t, conn, reader); status != http.StatusOK || !strings.Contains(body, `"items"`) {
		t.Fatalf("request: %d %s", status, body)
	}
	if err := conn.SetReadDeadline(start.Add(16 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Peek(1); !isTimeout(err) {
		t.Fatalf("the idle connection ended before 16 s: %v", err)
	}
	at, received, err := readUntilClosed(conn, reader, start.Add(70*time.Second))
	assertClosedBetween(t, "idle connection", start, at, err, 60*time.Second, 65*time.Second)
	if received != 0 {
		t.Fatalf("received %d bytes on the idle connection", received)
	}
	if status, body := rawExchange(t, s.addr, rawRequest(http.MethodGet, "/api/v1/items", s.session, nil, "")); status != http.StatusOK {
		t.Fatalf("new connection after the idle close: %d %s", status, body)
	}
}

// limitsVerify is how the fake Claude answers the verification of a token.
type limitsVerify int

const (
	verifyAccept limitsVerify = iota
	verifyAfter19s
	verifyNever // until the request's context ends
	verifyReject
)

type limitsToken struct {
	verify limitsVerify
	review <-chan struct{} // reviews answer once it is closed; nil: at once
}

type limitsListing struct {
	calls   int
	release <-chan struct{} // the page is answered once it is closed; nil: at once
}

// limitsNet answers every outgoing request of the live cases: LeBoncoin by the listing ID of the
// URL, Claude by the token.
type limitsNet struct {
	mu       sync.Mutex
	listings map[string]*limitsListing
	tokens   map[string]limitsToken
}

func withLimitsNet(t *testing.T) *limitsNet {
	t.Helper()
	fake := &limitsNet{listings: map[string]*limitsListing{}, tokens: map[string]limitsToken{}}
	original := http.DefaultTransport
	http.DefaultTransport = claudeTransport(fake.roundTrip)
	t.Cleanup(func() { http.DefaultTransport = original })
	return fake
}

func (n *limitsNet) addListing(id string, release <-chan struct{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.listings[id] = &limitsListing{release: release}
}

func (n *limitsNet) addToken(token string, fake limitsToken) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.tokens[token] = fake
}

func (n *limitsNet) calls(id string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	if listing := n.listings[id]; listing != nil {
		return listing.calls
	}
	return 0
}

func (n *limitsNet) roundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Hostname() == "api.anthropic.com" {
		return n.claude(r)
	}
	id := path.Base(r.URL.Path)
	n.mu.Lock()
	listing := n.listings[id]
	if listing != nil {
		listing.calls++
	}
	n.mu.Unlock()
	if listing == nil {
		return nil, fmt.Errorf("unexpected request to %s", r.URL)
	}
	if err := waitRelease(r.Context(), listing.release); err != nil {
		return nil, err
	}
	return limitsAnswer(r, http.StatusOK, "text/html", fmt.Sprintf(limitsPage, id)), nil
}

func (n *limitsNet) claude(r *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(r.Body)
	n.mu.Lock()
	fake, ok := n.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	n.mu.Unlock()
	if !ok {
		return nil, errors.New("unexpected Claude token")
	}
	if !strings.Contains(string(body), `"max_tokens":1,`) {
		if err := waitRelease(r.Context(), fake.review); err != nil {
			return nil, err
		}
		answer, _ := json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": limitsReview}}, "stop_reason": "end_turn"})
		return limitsAnswer(r, http.StatusOK, "application/json", string(answer)), nil
	}
	switch fake.verify {
	case verifyAfter19s:
		select {
		case <-time.After(19 * time.Second):
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	case verifyNever:
		<-r.Context().Done()
		return nil, r.Context().Err()
	case verifyReject:
		return limitsAnswer(r, http.StatusUnauthorized, "application/json", `{"type":"error"}`), nil
	}
	return limitsAnswer(r, http.StatusOK, "application/json", `{"content":[]}`), nil
}

func waitRelease(ctx context.Context, release <-chan struct{}) error {
	if release == nil {
		return nil
	}
	select {
	case <-release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func limitsAnswer(r *http.Request, status int, contentType, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func limitsListingURL(listingID string) string {
	return "https://www.leboncoin.fr/ad/informatique/" + listingID
}

type limitsItem struct{ id, listingID string }

// limitsLogins serializes the logins of the live servers started at the same time.
var limitsLogins sync.Mutex

// limitsServer is a live server in protected mode: alice owns the items and session is her cookie.
type limitsServer struct {
	addr     string
	api      *Server
	database *store.Store
	aliceID  int64
	session  string
	client   *http.Client
}

// openLimitsServer serves the real handler on a real store through NewHTTPServer; the scheduler is not started.
func openLimitsServer(t *testing.T, items ...limitsItem) *limitsServer {
	t.Helper()
	api, database, _ := openSecretServer(t, t.TempDir())
	aliceID, alice := func() (int64, string) {
		limitsLogins.Lock() // every service shares one password-check slot; a second waiting login is refused busy
		defer limitsLogins.Unlock()
		aliceID, _, alice, _ := protectedUsers(t, api, database)
		return aliceID, alice
	}()
	for _, item := range items {
		url := limitsListingURL(item.listingID)
		listing := model.Listing{ID: item.id, Platform: "leboncoin", ListingID: item.listingID, Marketplace: "leboncoin.fr", URL: url, OwnerID: aliceID}
		if err := database.Insert(context.Background(), listing, url, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	client := &http.Client{Transport: &http.Transport{}} // never http.DefaultTransport, which is the fake
	t.Cleanup(client.CloseIdleConnections)
	return &limitsServer{addr: serveTestHandler(t, api.Handler()), api: api, database: database, aliceID: aliceID, session: alice, client: client}
}

// serveTestHandler serves handler with the application's server on a loopback port and returns its address.
func serveTestHandler(t *testing.T, handler http.Handler) string {
	t.Helper()
	server := NewHTTPServer(config.Config{Host: "127.0.0.1"}, handler)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	return listener.Addr().String()
}

// call sends a request with alice's session through a standard client and returns the whole body.
func (s *limitsServer) call(t *testing.T, method, target, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, "http://"+s.addr+target, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: s.session})
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("%s %s: body cut off: %v", method, target, err)
	}
	return response.StatusCode, data
}

func (s *limitsServer) getObject(t *testing.T, target, key string) map[string]any {
	t.Helper()
	status, data := s.call(t, http.MethodGet, target, "")
	if status != http.StatusOK {
		t.Fatalf("GET %s: %d %s", target, status, data)
	}
	object, _ := decodeLimits(t, target, data)[key].(map[string]any)
	if object == nil {
		t.Fatalf("GET %s: no %q in %s", target, key, data)
	}
	return object
}

func (s *limitsServer) details(t *testing.T, id string) map[string]any {
	t.Helper()
	status, data := s.call(t, http.MethodGet, "/api/v1/items/"+id, "")
	if status != http.StatusOK {
		t.Fatalf("details of %s: %d %s", id, status, data)
	}
	return decodeLimits(t, "details of "+id, data)
}

func (s *limitsServer) itemIDs(t *testing.T) []string {
	t.Helper()
	status, data := s.call(t, http.MethodGet, "/api/v1/items", "")
	var list struct{ Items []struct{ ID string } }
	if err := json.Unmarshal(data, &list); err != nil || status != http.StatusOK {
		t.Fatalf("item list: %d %s %v", status, data, err)
	}
	ids := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func decodeLimits(t *testing.T, what string, data []byte) map[string]any {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatalf("%s: body is not complete JSON: %v: %s", what, err, data)
	}
	return object
}

func errorCode(data []byte) string {
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(data, &body)
	return body.Error.Code
}

// rawRequest builds an HTTP/1.1 request with the session cookie ("" : none), the extra header lines
// and a JSON body.
func rawRequest(method, target, session string, extra []string, body string) string {
	var request strings.Builder
	request.WriteString(method + " " + target + " HTTP/1.1\r\nHost: 127.0.0.1\r\n")
	if session != "" {
		request.WriteString("Cookie: " + sessionCookie + "=" + session + "\r\n")
	}
	for _, line := range extra {
		request.WriteString(line + "\r\n")
	}
	if body != "" {
		request.WriteString(fmt.Sprintf("Content-Type: application/json\r\nContent-Length: %d\r\n", len(body)))
	}
	request.WriteString("\r\n")
	return request.String() + body
}

func padding(size int) string { return "X-Padding: " + strings.Repeat("a", size) }

// sizedRequest is a request without body whose request line and headers, final empty line included, are size bytes.
func sizedRequest(t *testing.T, method, target, session string, size int) string {
	t.Helper()
	base := len(rawRequest(method, target, session, []string{padding(0)}, ""))
	request := rawRequest(method, target, session, []string{padding(size - base)}, "")
	if len(request) != size {
		t.Fatalf("request of %d bytes, want %d", len(request), size)
	}
	return request
}

func dialLimits(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func writeAll(t *testing.T, conn net.Conn, data string) {
	t.Helper()
	if _, err := io.WriteString(conn, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// writeSlowly writes data one byte per second, from now on, until it is sent, a write fails or the test ends.
func writeSlowly(t *testing.T, conn net.Conn, data string) {
	stop := make(chan struct{})
	stopped := make(chan struct{})
	t.Cleanup(func() { close(stop); <-stopped })
	go func() {
		defer close(stopped)
		for index := 0; index < len(data); index++ {
			if _, err := conn.Write([]byte{data[index]}); err != nil {
				return
			}
			select {
			case <-stop:
				return
			case <-time.After(time.Second):
			}
		}
	}()
}

// readUntilClosed reads until the connection ends or the deadline passes, and returns when the
// last read ended, the bytes read and the error that ended it.
func readUntilClosed(conn net.Conn, reader io.Reader, deadline time.Time) (time.Time, int, error) {
	if err := conn.SetReadDeadline(deadline); err != nil {
		return time.Now(), 0, err
	}
	buffer := make([]byte, 64<<10)
	total := 0
	for {
		n, err := reader.Read(buffer)
		total += n
		if err != nil {
			return time.Now(), total, err
		}
	}
}

func connectionClosed(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET)
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// assertClosedBetween checks that the server closed the connection between min and max after from.
func assertClosedBetween(t *testing.T, what string, from, at time.Time, err error, min, max time.Duration) {
	t.Helper()
	elapsed := at.Sub(from)
	if !connectionClosed(err) {
		t.Fatalf("%s: the server did not close the connection within %v: %v", what, elapsed, err)
	}
	if elapsed < min || elapsed > max {
		t.Fatalf("%s: closed after %v, want between %v and %v", what, elapsed, min, max)
	}
}

// readResponse reads one whole response from a kept-alive connection.
func readResponse(t *testing.T, conn net.Conn, reader *bufio.Reader) (int, string) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return response.StatusCode, string(body)
}

// rawExchange sends request on a new connection and returns the status and body of the answer.
func rawExchange(t *testing.T, addr, request string) (int, string) {
	t.Helper()
	conn := dialLimits(t, addr)
	writeAll(t, conn, request)
	return readResponse(t, conn, bufio.NewReader(conn))
}

// refusedExchange is rawExchange for an answer that closes the connection: the body ends with the
// connection, and closed reports that nothing follows it.
func refusedExchange(t *testing.T, addr, request string) (status int, body string, closed bool) {
	t.Helper()
	conn := dialLimits(t, addr)
	writeAll(t, conn, request)
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	data, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil && !connectionClosed(err) {
		t.Fatalf("read response body: %v", err)
	}
	_, _, end := readUntilClosed(conn, reader, time.Now().Add(3*time.Second))
	return response.StatusCode, string(data), connectionClosed(end)
}
