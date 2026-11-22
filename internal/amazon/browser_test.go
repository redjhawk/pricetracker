package amazon

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
)

// Browser tests of issue #168: a real Chromium checks Amazon-like fixture pages served locally.

const pixelPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

const amazonCheckFailed = "Amazon could not be reached for a price check."

// fixturePages maps the ASIN of each fixture product page to its file in testdata.
var fixturePages = map[string]string{
	"B0PRICE001": "price-found.html",
	"B0USEDAMZ1": "used-amazon.html",
	"B0USEDNOP1": "used-no-price.html",
	"B0USEDOTH1": "used-other-seller.html",
	"B0NOPRICE1": "price-not-found.html",
	"B0UNAVAIL1": "unavailable.html",
	"B0SCRIPT01": "script-price.html",
}

// fixtureHandler serves the fixture product pages at /dp/<ASIN>, the used-offer responses at
// /gp/aod/ajax (an offer by another seller for B0USEDOTH1, no offer otherwise), a stylesheet and an
// image. /dp/B0GONE0001 answers 404.
func fixtureHandler(t *testing.T) http.Handler {
	t.Helper()
	pages := map[string]string{}
	for asin, name := range fixturePages {
		pages[asin] = amazontest.Fixture(t, name)
	}
	otherSeller, noOffer := amazontest.Fixture(t, "aod-other-seller.html"), amazontest.Fixture(t, "aod-none.html")
	pixel, err := base64.StdEncoding.DecodeString(pixelPNG)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/dp/", func(w http.ResponseWriter, r *http.Request) {
		page, ok := pages[strings.TrimPrefix(r.URL.Path, "/dp/")]
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "<html><body><h1>Page introuvable</h1></body></html>")
			return
		}
		io.WriteString(w, page)
	})
	usedOffers := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Query().Get("asin") == "B0USEDOTH1" {
			io.WriteString(w, otherSeller)
			return
		}
		io.WriteString(w, noOffer)
	}
	mux.HandleFunc("/gp/aod/ajax", usedOffers)
	mux.HandleFunc("/gp/aod/ajax/", usedOffers)
	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		io.WriteString(w, "body { font-family: sans-serif; }")
	})
	mux.HandleFunc("/static/pixel.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pixel)
	})
	return mux
}

func fixtureItem(asin string) model.Listing {
	return model.Listing{ID: "item-" + asin, Platform: "amazon", ListingID: asin, ASIN: asin, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + asin}
}

// testBrowserOptions returns the options of a browser test: the found Chromium, no sandbox, no
// reading pause, Amazon URLs sent to origin and a profile directory of the test.
func testBrowserOptions(t *testing.T, origin string) BrowserOptions {
	t.Helper()
	return BrowserOptions{ExecPath: amazontest.Browser(t), ProfileDir: filepath.Join(t.TempDir(), "profile"), NoSandbox: true,
		NavigationTimeout: 30 * time.Second, Pause: func() time.Duration { return 0 }, Origin: origin}
}

func browserCheck(t *testing.T, collector *BrowserCollector, item model.Listing) model.CollectionResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return collector.Collect(ctx, item)
}

func pointerValue(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}

// browserProcesses lists the running processes whose command line contains profile.
func browserProcesses(profile string) []string {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var found []string
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || !strings.Contains(string(cmdline), profile) {
			continue
		}
		found = append(found, entry.Name()+": "+strings.ReplaceAll(string(cmdline), "\x00", " "))
	}
	return found
}

// assertNothingLeft checks that no browser process of profile and no extra goroutine survive a check.
func assertNothingLeft(t *testing.T, profile string, goroutines int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(browserProcesses(profile)) > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if left := browserProcesses(profile); len(left) > 0 {
		t.Errorf("browser processes left after the check: %v", left)
	}
	for runtime.NumGoroutine() > goroutines && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if current := runtime.NumGoroutine(); current > goroutines {
		t.Errorf("goroutines left after the check: %d before, %d after", goroutines, current)
	}
}

// captureOutput collects what the code under test logs or prints until the returned function is called.
func captureOutput(t *testing.T) func() string {
	t.Helper()
	var logs lockedBuffer
	previousLog := log.Writer()
	log.SetOutput(&logs)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = writer
	var printed bytes.Buffer
	copied := make(chan struct{})
	go func() { io.Copy(&printed, reader); close(copied) }()
	var once sync.Once
	stop := func() string {
		once.Do(func() {
			os.Stdout = previousStdout
			writer.Close()
			<-copied
			reader.Close()
			log.SetOutput(previousLog)
		})
		return logs.String() + printed.String()
	}
	t.Cleanup(func() { stop() })
	return stop
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// The browser runs the page's scripts and loads its resources; a plain download finds no price.
// TC-002
func TestBrowserCollector_TC002_RunsScriptsAndLoadsResources(t *testing.T) {
	server := amazontest.NewServer(t, fixtureHandler(t))
	collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
	result := browserCheck(t,collector, fixtureItem("B0SCRIPT01"))
	if result.Result != "success" || result.AmountCents != 4999 || pointerValue(result.Title) != "Script product" {
		t.Fatalf("price inserted by the page script not detected: %+v (title %s)", result, pointerValue(result.Title))
	}
	for _, path := range []string{"/dp/B0SCRIPT01", "/static/style.css", "/static/pixel.png"} {
		if server.Count(path) == 0 {
			t.Errorf("the browser did not request %s; requests: %+v", path, server.Requests())
		}
	}

	response, err := http.Get(server.URL + "/dp/B0SCRIPT01")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if amount, found := productPrice(string(body)); found {
		t.Fatalf("a plain download of the fixture already shows a price (%d): the fixture does not need the browser", amount)
	}
}

// The browser keeps the cookies set by the page, runs without a display and sends no credential.
// TC-003
func TestBrowserCollector_TC003_KeepsCookiesHeadlessWithoutCredentials(t *testing.T) {
	t.Setenv("DISPLAY", "")
	os.Unsetenv("DISPLAY")
	t.Setenv("WAYLAND_DISPLAY", "")
	os.Unsetenv("WAYLAND_DISPLAY")
	pixel, err := base64.StdEncoding.DecodeString(pixelPNG)
	if err != nil {
		t.Fatal(err)
	}
	page := amazontest.Fixture(t, "price-found.html")
	noOffer := amazontest.Fixture(t, "aod-none.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/dp/B0PRICE001", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session-id", Value: "fixture-session-168", Path: "/", MaxAge: 3600})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	})
	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		io.WriteString(w, "body { margin: 0; }")
	})
	mux.HandleFunc("/static/pixel.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pixel)
	})
	mux.HandleFunc("/gp/aod/ajax", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, noOffer) })
	mux.HandleFunc("/gp/aod/ajax/", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, noOffer) })
	server := amazontest.NewServer(t, mux)
	collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
	item := fixtureItem("B0PRICE001")

	first := browserCheck(t,collector, item)
	firstRequests := server.Requests()
	second := browserCheck(t,collector, item)
	if first.Result != "success" || second.Result != "success" {
		t.Fatalf("checks without a display: first %+v, second %+v", first, second)
	}
	cookieSent := false
	for _, request := range firstRequests {
		if request.Path == "/static/style.css" && strings.Contains(request.Header.Get("Cookie"), "session-id=fixture-session-168") {
			cookieSent = true
		}
	}
	if !cookieSent {
		t.Fatalf("the resource request of the first check does not carry the cookie set by the page: %+v", firstRequests)
	}
	for _, request := range server.Requests() {
		if request.Header.Get("Authorization") != "" {
			t.Errorf("credential sent to %s: %q", request.Path, request.Header.Get("Authorization"))
		}
		for _, cookie := range strings.Split(request.Header.Get("Cookie"), ";") {
			if name, _, _ := strings.Cut(strings.TrimSpace(cookie), "="); name != "" && name != "session-id" {
				t.Errorf("cookie %q not set by the page sent to %s", name, request.Path)
			}
		}
	}
}

// The browser gives the same results as the parser for the existing kinds of Amazon pages.
// TC-004
func TestBrowserCollector_TC004_ResultsMatchTheParser(t *testing.T) {
	server := amazontest.NewServer(t, fixtureHandler(t))
	collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
	title := func(value string) *string { return &value }
	image := "https://m.media-amazon.com/images/I/price-found.jpg"
	tests := []struct {
		name string
		asin string
		want model.CollectionResult
	}{
		{"price found, no second-hand offer", "B0PRICE001", model.CollectionResult{Result: "success", AmountCents: 12999, Title: title("Price found product"), ThumbnailURL: &image, SecondHandStatus: "not_found"}},
		{"second-hand offer sold by Amazon", "B0USEDAMZ1", model.CollectionResult{Result: "success", AmountCents: 11900, Title: title("Used Amazon product"),
			SecondHandStatus: "available", SecondHandAmountCents: 8950, SecondHandCondition: "very_good", SecondHandConditionLabel: "Très bon état"}},
		{"second-hand offer and no new price", "B0USEDNOP1", model.CollectionResult{Result: "price_not_found", Message: "Amazon did not show a detectable euro price.", Title: title("Used only product"),
			SecondHandStatus: "available", SecondHandAmountCents: 4500, SecondHandCondition: "good", SecondHandConditionLabel: "Bon état"}},
		{"second-hand offer by another seller", "B0USEDOTH1", model.CollectionResult{Result: "success", AmountCents: 7500, Title: title("Other seller product"), SecondHandStatus: "not_found"}},
		{"price not found", "B0NOPRICE1", model.CollectionResult{Result: "price_not_found", Message: "Amazon did not show a detectable euro price.", Title: title("No price product"), SecondHandStatus: "not_found"}},
		{"listing unavailable", "B0UNAVAIL1", model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available.", SecondHandStatus: "not_found"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if fixture := amazontest.Fixture(t, fixturePages[test.asin]); test.want.Result == "success" {
				if amount, found := productPrice(fixture); !found || amount != test.want.AmountCents {
					t.Fatalf("fixture check: the parser reads %d, %v", amount, found)
				}
			}
			got := browserCheck(t,collector, fixtureItem(test.asin))
			if got.Result != test.want.Result || got.AmountCents != test.want.AmountCents || got.Message != test.want.Message ||
				pointerValue(got.Title) != pointerValue(test.want.Title) || pointerValue(got.ThumbnailURL) != pointerValue(test.want.ThumbnailURL) ||
				got.SecondHandStatus != test.want.SecondHandStatus || got.SecondHandAmountCents != test.want.SecondHandAmountCents ||
				got.SecondHandCondition != test.want.SecondHandCondition || got.SecondHandConditionLabel != test.want.SecondHandConditionLabel {
				t.Fatalf("got %+v (title %s, image %s)\nwant %+v (title %s, image %s)", got, pointerValue(got.Title), pointerValue(got.ThumbnailURL),
					test.want, pointerValue(test.want.Title), pointerValue(test.want.ThumbnailURL))
			}
		})
	}
	// As today, a product page answering 404 (or 410) is an unavailable listing, not a failure.
	t.Run("product page answering 404", func(t *testing.T) {
		got := browserCheck(t,collector, fixtureItem("B0GONE0001"))
		if got.Result != "unavailable" || got.Message != "The listing is no longer available." {
			t.Fatalf("404 product page: %+v", got)
		}
	})
}

// A non-success answer or a robot check page is a failure, with the diagnostics for the log.
// HTTP 404/410 stay an unavailable listing as today (technical specification, TS-AMZ-HCHK-003;
// covered by TC-004), so the other non-success case here is 403.
// TC-006
func TestBrowserCollector_TC006_NonSuccessAndRobotCheckAreFailures(t *testing.T) {
	const secret = "fixture-secret-token-168"
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"HTTP 503", http.StatusServiceUnavailable, "<html><body><h1>Service Unavailable fixture</h1></body></html>"},
		{"HTTP 403", http.StatusForbidden, "<html><body><h1>Forbidden fixture</h1></body></html>"},
		{"robot check", http.StatusOK, `<html><head><title>Amazon.fr</title></head><body><h4>Enter the characters you see below</h4>` +
			`<p>Sorry, we just need to make sure you're not a robot.</p><form action="/errors/validateCaptcha"><input name="field-keywords"></form></body></html>`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := amazontest.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.SetCookie(w, &http.Cookie{Name: "session-token", Value: secret, Path: "/"})
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(test.status)
				io.WriteString(w, test.body)
			}))
			collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
			output := captureOutput(t)
			item := fixtureItem("B0PRICE001")
			got := browserCheck(t,collector, item)
			logged := output()
			if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
				t.Fatalf("not a failure: %+v", got)
			}
			if !strings.HasSuffix(got.RequestURL, "/dp/B0PRICE001") || got.HTTPStatus != test.status || got.ResponseExcerpt == "" {
				t.Fatalf("missing diagnostics: url=%q status=%d excerpt=%q", got.RequestURL, got.HTTPStatus, got.ResponseExcerpt)
			}
			if strings.Contains(got.ResponseExcerpt+got.Message+got.RequestURL+logged, secret) {
				t.Fatalf("a cookie value reached the diagnostics or the log: %+v\n%s", got, logged)
			}
		})
	}
}

// Network errors, a page that never finishes, a browser that cannot start and a cancelled check
// end the check with a failure, leaving no browser process and no goroutine behind.
// TC-007
func TestBrowserCollector_TC007_ErrorsEndTheCheck(t *testing.T) {
	failure := func(t *testing.T, got model.CollectionResult) {
		t.Helper()
		if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
			t.Fatalf("not a failure: %+v", got)
		}
	}
	neverLoads := func(t *testing.T) *amazontest.Server {
		release := make(chan struct{})
		server := amazontest.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "<html><body><p>Loading")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			select {
			case <-r.Context().Done():
			case <-release:
			}
		}))
		t.Cleanup(func() { close(release) }) // runs before the server is closed
		return server
	}

	t.Run("browser cannot start", func(t *testing.T) {
		exitsAtOnce := filepath.Join(t.TempDir(), "exits-at-once")
		if err := os.WriteFile(exitsAtOnce, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, executable := range []string{filepath.Join(t.TempDir(), "missing-chromium"), exitsAtOnce} {
			goroutines := runtime.NumGoroutine()
			collector := NewBrowserCollector(BrowserOptions{ExecPath: executable, ProfileDir: filepath.Join(t.TempDir(), "profile"), NoSandbox: true,
				NavigationTimeout: 5 * time.Second, Pause: func() time.Duration { return 0 }, Origin: "http://127.0.0.1:9"})
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			started := time.Now()
			got := collector.Collect(ctx, fixtureItem("B0PRICE001"))
			cancel()
			failure(t, got)
			if elapsed := time.Since(started); elapsed > 20*time.Second {
				t.Fatalf("%s: the failed start took %s", executable, elapsed)
			}
			assertNothingLeft(t, executable, goroutines)
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		closedOrigin := "http://" + listener.Addr().String()
		listener.Close()
		options := testBrowserOptions(t, closedOrigin)
		goroutines := runtime.NumGoroutine()
		got := browserCheck(t,NewBrowserCollector(options), fixtureItem("B0PRICE001"))
		failure(t, got)
		if got.HTTPStatus != 0 || !strings.HasSuffix(got.RequestURL, "/dp/B0PRICE001") {
			t.Fatalf("diagnostics without a response: %+v", got)
		}
		assertNothingLeft(t, options.ProfileDir, goroutines)
	})

	t.Run("page never finishes loading", func(t *testing.T) {
		server := neverLoads(t)
		options := testBrowserOptions(t, server.URL)
		options.NavigationTimeout = 2 * time.Second
		goroutines := runtime.NumGoroutine()
		started := time.Now()
		got := browserCheck(t,NewBrowserCollector(options), fixtureItem("B0PRICE001"))
		failure(t, got)
		if elapsed := time.Since(started); elapsed > 30*time.Second {
			t.Fatalf("the navigation timeout did not end the check: %s", elapsed)
		}
		assertNothingLeft(t, options.ProfileDir, goroutines)
	})

	t.Run("check cancelled during the page load", func(t *testing.T) {
		server := neverLoads(t)
		options := testBrowserOptions(t, server.URL)
		options.NavigationTimeout = time.Minute
		goroutines := runtime.NumGoroutine()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		time.AfterFunc(time.Second, cancel)
		started := time.Now()
		got := NewBrowserCollector(options).Collect(ctx, fixtureItem("B0PRICE001"))
		if elapsed := time.Since(started); elapsed > 15*time.Second {
			t.Fatalf("the cancelled check took %s", elapsed)
		}
		if got.Result == "success" || got.Result == "" {
			t.Fatalf("cancelled check result %+v", got)
		}
		assertNothingLeft(t, options.ProfileDir, goroutines)
	})
}
