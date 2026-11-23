// Package amazontest supports the Amazon browser tests: it finds a Chromium, reads the fixture
// pages of internal/amazon/testdata and serves them from a local server that records every request.
// No test contacts Amazon.
package amazontest

import (
	"bytes"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
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

// paddingParagraph is neutral product-description markup: no price, product block, robot-check or
// availability text.
const paddingParagraph = `<p class="a-spacing-small"><span class="a-text-bold">Caractéristiques :</span> brosse latérale, réservoir de 450 ml, batterie 5200 mAh, autonomie de 180 minutes.</p>` + "\n"

// Expand replaces the <!--PADDING--> mark of page with about 210 kB of neutral product-description
// markup, so that what follows the mark lies beyond the first 200 kB of the page.
func Expand(page string) string {
	padding := strings.Repeat(paddingParagraph, 210_000/len(paddingParagraph)+1)
	return strings.Replace(page, "<!--PADDING-->", padding, 1)
}

// ExpandedFixture returns Fixture(t, name) with its <!--PADDING--> mark expanded (Expand).
func ExpandedFixture(t testing.TB, name string) string {
	t.Helper()
	page := Fixture(t, name)
	if !strings.Contains(page, "<!--PADDING-->") {
		t.Fatalf("fixture %s has no <!--PADDING--> mark", name)
	}
	return Expand(page)
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

// DownloadMarker is added to the bodies offered as downloads; normal pages never contain it.
const DownloadMarker = "<!-- download-marker-217 -->"

// IsolatedHome sets HOME and XDG_CONFIG_HOME (t.Setenv) to new directories of the test and returns the home,
// so the browser's default Downloads folder is <home>/Downloads. Call it after Browser(t), which may look
// for Playwright's Chromium under the real home.
func IsolatedHome(t testing.TB) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	config := filepath.Join(home, ".config")
	if err := os.MkdirAll(config, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", config)
	return home
}

// partialDownloadPrefixes and partialDownloadSuffix name the files of a download in progress.
var partialDownloadPrefixes = []string{"Unconfirmed ", ".com.google.Chrome.", ".org.chromium.Chromium."}

const partialDownloadSuffix = ".crdownload"

// downloadNamePattern matches name and the names Chromium derives from it: uniquified ("page (1).html")
// and, for a name without extension, with an added extension ("B0PRICE001.html").
func downloadNamePattern(name string) *regexp.Regexp {
	extension := filepath.Ext(name)
	if extension == "" {
		return regexp.MustCompile(`^` + regexp.QuoteMeta(name) + `( \(\d+\))?(\.[^.]+)?$`)
	}
	stem := strings.TrimSuffix(name, extension)
	return regexp.MustCompile(`^` + regexp.QuoteMeta(stem) + `( \(\d+\))?` + regexp.QuoteMeta(extension) + `$`)
}

// AssertNoDownload fails t when a root holds a file left by a download, after the browser has closed:
// a regular file whose base name is one of names, or that name uniquified by Chromium ("page (1).html"),
// or that starts with "download"; a partial download (suffix ".crdownload", prefix "Unconfirmed ",
// ".com.google.Chrome." or ".org.chromium.Chromium."); or a regular file containing DownloadMarker.
// Missing roots are ignored; symbolic links are not followed.
func AssertNoDownload(t testing.TB, names []string, roots ...string) {
	t.Helper()
	patterns := make([]*regexp.Regexp, 0, len(names))
	for _, name := range names {
		patterns = append(patterns, downloadNamePattern(name))
	}
	var found []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			if reason, _ := downloadFile(path, entry.Name(), patterns); reason != "" {
				found = append(found, path+" ("+reason+")")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("list the files of %s: %v", root, err)
		}
	}
	if len(found) > 0 {
		t.Fatalf("files left by a download:\n%s", strings.Join(found, "\n"))
	}
}

// downloadFile returns why the regular file at path with base name looks left by a download, or "".
// Stub (issue #232): the read error is not returned yet.
func downloadFile(path, name string, patterns []*regexp.Regexp) (string, error) {
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return "named after the attachment", nil
		}
	}
	if strings.HasPrefix(name, "download") {
		return "default download name", nil
	}
	if strings.HasSuffix(name, partialDownloadSuffix) {
		return "partial download", nil
	}
	for _, prefix := range partialDownloadPrefixes {
		if strings.HasPrefix(name, prefix) {
			return "partial download", nil
		}
	}
	data, err := os.ReadFile(path) // #nosec G304 -- file of a test directory
	if err == nil && bytes.Contains(data, []byte(DownloadMarker)) {
		return "holds the downloaded body", nil
	}
	return "", nil
}

// AssertDownloadsFolder fails t unless <home>/Downloads is absent (existed false) or an empty directory (existed true).
func AssertDownloadsFolder(t testing.TB, home string, existed bool) {
	t.Helper()
	downloads := filepath.Join(home, "Downloads")
	entries, err := os.ReadDir(downloads)
	switch {
	case !existed && errors.Is(err, fs.ErrNotExist):
	case !existed && err == nil:
		t.Fatalf("the Downloads folder %s was created, holding %d entries", downloads, len(entries))
	case err != nil:
		t.Fatalf("read the Downloads folder %s: %v", downloads, err)
	case len(entries) > 0:
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("the Downloads folder %s holds %s", downloads, strings.Join(names, ", "))
	}
}
