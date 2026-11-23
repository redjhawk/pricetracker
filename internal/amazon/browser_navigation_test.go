package amazon

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
)

// Browser tests of issue #186: an Amazon answer with a success status that the browser does not open
// as a page (a download, no content) is a failed check, never "price not found"; a non-success answer
// with an empty body is still classified by its status.

// navigationServer serves the fixture pages of fixtureHandler, with /dp/B0PRICE001 answered by product.
func navigationServer(t *testing.T, product http.HandlerFunc) *amazontest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle("/", fixtureHandler(t))
	mux.HandleFunc("/dp/B0PRICE001", product)
	return amazontest.NewServer(t, mux)
}

// assertNavigationFailure checks that got is the failure of a check whose page the browser did not
// open, logged with the request URL and the browser's error on one line.
func assertNavigationFailure(t *testing.T, got model.CollectionResult, item model.Listing, logged string) {
	t.Helper()
	if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
		t.Fatalf("not a failure: %+v", got)
	}
	if !strings.HasSuffix(got.RequestURL, "/dp/"+item.ASIN) || got.HTTPStatus != 0 {
		t.Fatalf("diagnostics of a page not opened: url=%q status=%d", got.RequestURL, got.HTTPStatus)
	}
	for _, line := range strings.Split(logged, "\n") {
		if strings.Contains(line, "Amazon page could not be opened: net::") && strings.Contains(line, item.URL) {
			return
		}
	}
	t.Fatalf("no log line with the request URL %s and the browser's error:\n%s", item.URL, logged)
}

// A 200 page offered as a download is a failure, not "price not found"; an inline page is opened.
// The download is refused: no file of it is written in the profile or under HOME (issue #217).
// TC-001
// TC-001, TC-002 (issue #217)
func TestBrowserNavigation_TC001_AttachmentIsAFailure(t *testing.T) {
	const (
		attachment int32 = iota
		normal
		inline
	)
	page := amazontest.Fixture(t, "price-found.html")
	if amount, found := productPrice(page); !found || amount != 12999 {
		t.Fatalf("fixture check: the parser reads %d, %v", amount, found)
	}
	if strings.Contains(page, amazontest.DownloadMarker) {
		t.Fatal("fixture check: the normal page holds the download marker")
	}
	newServer := func(t *testing.T, mode *atomic.Int32) *amazontest.Server {
		return navigationServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			switch mode.Load() {
			case attachment:
				w.Header().Set("Content-Disposition", `attachment; filename="page.html"`)
				io.WriteString(w, page+amazontest.DownloadMarker)
				return
			case inline:
				w.Header().Set("Content-Disposition", "inline")
			}
			io.WriteString(w, page)
		})
	}
	item := fixtureItem("B0PRICE001")

	t.Run("attachment then a normal page", func(t *testing.T) {
		var mode atomic.Int32
		mode.Store(attachment)
		server := newServer(t, &mode)
		options := testBrowserOptions(t, server.URL)
		home := amazontest.IsolatedHome(t)
		collector := NewBrowserCollector(options)
		assertNoFileWritten := func() {
			t.Helper()
			amazontest.AssertNoDownload(t, []string{"page.html"}, options.ProfileDir, home)
			amazontest.AssertDownloadsFolder(t, home, false)
		}

		output := captureOutput(t)
		got := browserCheck(t, collector, item)
		logged := output()
		if server.Count("/dp/B0PRICE001") == 0 {
			t.Fatalf("the browser did not request the product page: %+v", server.Requests())
		}
		if got.Result == "price_not_found" || got.Result == "success" || got.Result == "unavailable" {
			t.Fatalf("an attachment gave the Amazon answer %q: %+v", got.Result, got)
		}
		assertNavigationFailure(t, got, item, logged)
		assertNoFileWritten()

		// TC-001 (issue #217): the same check answered again with a download adds no file.
		output = captureOutput(t)
		again := browserCheck(t, collector, item)
		logged = output()
		if count := server.Count("/dp/B0PRICE001"); count != 2 {
			t.Fatalf("%d product page requests after the repeated check, want 2: %+v", count, server.Requests())
		}
		assertNavigationFailure(t, again, item, logged)
		assertNoFileWritten()

		mode.Store(normal)
		if next := browserCheck(t, collector, item); next.Result != "success" || next.AmountCents != 12999 {
			t.Fatalf("check after the failed downloads: %+v", next)
		}
		assertNoFileWritten()
	})

	// TC-002 (issue #217): an existing Downloads folder stays empty, whatever the attachment's file name.
	for _, answer := range []struct {
		name, contentType, disposition string
	}{
		{"attachment named page.html", "text/html; charset=utf-8", `attachment; filename="page.html"`},
		{"attachment without a file name", "text/html; charset=utf-8", "attachment"},
		{"octet-stream attachment", "application/octet-stream", `attachment; filename="B0PRICE001.bin"`},
	} {
		t.Run("TC-002 existing Downloads folder, "+answer.name, func(t *testing.T) {
			server := navigationServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", answer.contentType)
				w.Header().Set("Content-Disposition", answer.disposition)
				io.WriteString(w, page+amazontest.DownloadMarker)
			})
			options := testBrowserOptions(t, server.URL)
			home := amazontest.IsolatedHome(t)
			if err := os.Mkdir(filepath.Join(home, "Downloads"), 0o700); err != nil {
				t.Fatal(err)
			}
			output := captureOutput(t)
			got := browserCheck(t, NewBrowserCollector(options), item)
			logged := output()
			if got.Result == "price_not_found" || got.Result == "success" || got.Result == "unavailable" {
				t.Fatalf("%s gave the Amazon answer %q: %+v", answer.name, got.Result, got)
			}
			assertNavigationFailure(t, got, item, logged)
			amazontest.AssertDownloadsFolder(t, home, true)
			amazontest.AssertNoDownload(t, []string{"page.html", "B0PRICE001", "B0PRICE001.bin"}, options.ProfileDir, home)
		})
	}

	t.Run("inline page", func(t *testing.T) {
		var mode atomic.Int32
		mode.Store(inline)
		server := newServer(t, &mode)
		got := browserCheck(t, NewBrowserCollector(testBrowserOptions(t, server.URL)), item)
		if got.Result != "success" || got.AmountCents != 12999 {
			t.Fatalf("inline page: %+v", got)
		}
	})
}

// A 204 or 205 response without content is a failure.
// TC-002
func TestBrowserNavigation_TC002_NoContentIsAFailure(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusResetContent} {
		t.Run("HTTP "+strconv.Itoa(status), func(t *testing.T) {
			server := navigationServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			})
			collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
			item := fixtureItem("B0PRICE001")
			output := captureOutput(t)
			got := browserCheck(t, collector, item)
			logged := output()
			if got.Result == "price_not_found" || got.Result == "unavailable" || got.Result == "success" {
				t.Fatalf("HTTP %d gave the Amazon answer %q: %+v", status, got.Result, got)
			}
			assertNavigationFailure(t, got, item, logged)
		})
	}
}

// A non-success response with an empty body is still classified by its status.
// TC-003
func TestBrowserNavigation_TC003_EmptyNonSuccessKeepsItsStatus(t *testing.T) {
	tests := []struct {
		status      int
		unavailable bool
	}{
		{http.StatusNotFound, true},
		{http.StatusGone, true},
		{http.StatusServiceUnavailable, false},
		{http.StatusForbidden, false},
	}
	for _, test := range tests {
		t.Run("HTTP "+strconv.Itoa(test.status), func(t *testing.T) {
			server := navigationServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(test.status)
			})
			got := browserCheck(t, NewBrowserCollector(testBrowserOptions(t, server.URL)), fixtureItem("B0PRICE001"))
			if test.unavailable {
				if got.Result != "unavailable" || got.Message != unavailableMessage {
					t.Fatalf("HTTP %d with an empty body: %+v", test.status, got)
				}
				return
			}
			if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" || got.HTTPStatus != test.status {
				t.Fatalf("HTTP %d with an empty body: %+v", test.status, got)
			}
			if !strings.HasSuffix(got.RequestURL, "/dp/B0PRICE001") {
				t.Fatalf("request URL %q", got.RequestURL)
			}
		})
	}
}
