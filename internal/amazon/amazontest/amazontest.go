// Package amazontest supports the Amazon browser tests: it finds a Chromium, reads the fixture
// pages of internal/amazon/testdata and serves them from a local server that records every request.
// No test contacts Amazon.
package amazontest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
)

// Browser returns the Chromium or Chrome executable for browser tests: PRICEFOLLOWER_BROWSER_PATH,
// then PATH, then a Chromium installed by Playwright. Without one the test is skipped, or fails when
// CI is set.
func Browser(t testing.TB) string {
	t.Helper()
	if path := os.Getenv("PRICEFOLLOWER_BROWSER_PATH"); path != "" {
		return path
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	var roots []string
	if root := os.Getenv("PLAYWRIGHT_BROWSERS_PATH"); root != "" && root != "0" {
		roots = append(roots, root)
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".cache", "ms-playwright"))
	}
	roots = append(roots, "/ms-playwright")
	for _, root := range roots {
		matches, _ := filepath.Glob(filepath.Join(root, "chromium-*", "chrome-linux*", "chrome"))
		sort.Sort(sort.Reverse(sort.StringSlice(matches))) // newest revision first
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && !info.IsDir() {
				return match
			}
		}
	}
	if os.Getenv("CI") != "" {
		t.Fatal("no Chromium found for the Amazon browser tests: install Chromium or set PRICEFOLLOWER_BROWSER_PATH")
	}
	t.Skip("no Chromium found; install Chromium or set PRICEFOLLOWER_BROWSER_PATH to run the Amazon browser tests")
	return ""
}

// Fixture returns the content of internal/amazon/testdata/<name>.
func Fixture(t testing.TB, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate the Amazon fixture pages")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "testdata", filepath.Base(name))) // #nosec G304 -- fixture file of the repository
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Request is one request received by a Server.
type Request struct {
	Method string
	Path   string // URL path
	Query  string // raw query
	Header http.Header
}

// Server is a local HTTP server standing in for Amazon. It records every request (with its headers)
// before passing it to the handler.
type Server struct {
	*httptest.Server
	mu       sync.Mutex
	requests []Request
}

// NewServer starts a Server closed at the end of the test.
func NewServer(t testing.TB, handler http.Handler) *Server {
	t.Helper()
	server := &Server{}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.mu.Lock()
		server.requests = append(server.requests, Request{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Header: r.Header.Clone()})
		server.mu.Unlock()
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// Requests returns the requests received so far, in order.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Count returns the number of requests received for path.
func (s *Server) Count(path string) int {
	count := 0
	for _, request := range s.Requests() {
		if request.Path == path {
			count++
		}
	}
	return count
}

// Reset forgets the recorded requests.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}
