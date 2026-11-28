package amazon

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
)

// Browser tests of issue #187: a 404/410 product page with an empty body leaves the tab on Chromium's
// error page, so the check sends no used-offer request; a 404/410 page the browser opens still does.

const usedOfferFailedLine = "Amazon used-offer check failed for"

// errorPageServer serves the fixture pages of fixtureHandler, with /dp/B0PRICE001 answered by product
// and both used-offer paths by usedOffers.
func errorPageServer(t *testing.T, product, usedOffers http.HandlerFunc) *amazontest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle("/", fixtureHandler(t))
	mux.HandleFunc("/dp/B0PRICE001", product)
	mux.HandleFunc("/gp/aod/ajax", usedOffers)
	mux.HandleFunc("/gp/aod/ajax/", usedOffers)
	return amazontest.NewServer(t, mux)
}

// amazonUsedOffer answers the used-offer requests with the Amazon Seconde main offer at 95.50, "Comme neuf".
func amazonUsedOffer(t *testing.T) http.HandlerFunc {
	t.Helper()
	page := amazontest.Fixture(t, "aod-amazon-used.html")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	}
}

// usedOfferCount counts the requests received on both used-offer paths, whatever their method
// (a CORS preflight counts too).
func usedOfferCount(server *amazontest.Server) int {
	return server.Count("/gp/aod/ajax/ref=tmm_pap_used_aod_0") + server.Count("/gp/aod/ajax")
}

// assertAmazonUsedOffer checks that got carries the used offer of aod-amazon-used.html.
func assertAmazonUsedOffer(t *testing.T, got model.CollectionResult) {
	t.Helper()
	if got.SecondHandStatus != "available" || got.SecondHandAmountCents != 9550 || got.SecondHandCondition != "like_new" {
		t.Fatalf("second-hand result %q %d %q, want the used offer at 95.50 like new: %+v", got.SecondHandStatus, got.SecondHandAmountCents, got.SecondHandCondition, got)
	}
}

// An empty-body 404 or 410 product page is unavailable and sends no used-offer request; the next
// check on the same collector sends them again.
// TC-001
func TestBrowserErrorPage_TC001_EmptyUnavailablePageSendsNoUsedOfferRequest(t *testing.T) {
	page := amazontest.Fixture(t, "price-found.html")
	for _, status := range []int{http.StatusNotFound, http.StatusGone} {
		t.Run("HTTP "+strconv.Itoa(status), func(t *testing.T) {
			var empty atomic.Bool
			empty.Store(true)
			server := errorPageServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store") // Chromium keeps a 410 without cache headers forever
				if empty.Load() {
					w.WriteHeader(status)
					return
				}
				io.WriteString(w, page)
			}, amazonUsedOffer(t))
			collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
			item := fixtureItem("B0PRICE001")

			output := amazontest.CaptureOutput(t)
			got := browserCheck(t, collector, item)
			logged := output()
			if server.Count("/dp/B0PRICE001") == 0 {
				t.Fatalf("the browser did not request the product page: %+v", server.Requests())
			}
			if got.Result != "unavailable" || got.Message != unavailableMessage || got.HTTPStatus != status {
				t.Fatalf("HTTP %d with an empty body: %+v", status, got)
			}
			if got.SecondHandStatus != "check_error" || got.SecondHandAmountCents != 0 || got.SecondHandCondition != "" || got.SecondHandConditionLabel != "" {
				t.Fatalf("second-hand result of the error page: %+v", got)
			}
			if count := usedOfferCount(server); count != 0 {
				t.Fatalf("%d used-offer requests sent from the browser's error page: %+v", count, server.Requests())
			}
			if strings.Contains(logged, usedOfferFailedLine) {
				t.Fatalf("used-offer failure logged without a used-offer request:\n%s", logged)
			}

			empty.Store(false)
			pageRequests := server.Count("/dp/B0PRICE001")
			again := browserCheck(t, collector, item)
			if server.Count("/dp/B0PRICE001") == pageRequests {
				t.Fatal("the check after the error page did not request the product page")
			}
			if again.Result != "success" || again.AmountCents != 12999 {
				t.Fatalf("check after the error page: %+v", again)
			}
			if count := usedOfferCount(server); count != 2 {
				t.Fatalf("%d used-offer requests in the check after the error page, want 2: %+v", count, server.Requests())
			}
			assertAmazonUsedOffer(t, again)
		})
	}
}

// A 404 or 410 product page with content is opened by the browser and still gets its used-offer requests.
// TC-002
func TestBrowserErrorPage_TC002_UnavailablePageWithContentSendsUsedOfferRequests(t *testing.T) {
	notFoundPage := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			io.WriteString(w, "<html><body><h1>Page introuvable</h1></body></html>")
		}
	}
	for _, status := range []int{http.StatusNotFound, http.StatusGone} {
		t.Run("HTTP "+strconv.Itoa(status), func(t *testing.T) {
			server := errorPageServer(t, notFoundPage(status), amazonUsedOffer(t))
			got := browserCheck(t, NewBrowserCollector(testBrowserOptions(t, server.URL)), fixtureItem("B0PRICE001"))
			if got.Result != "unavailable" || got.Message != unavailableMessage || got.HTTPStatus != status {
				t.Fatalf("HTTP %d page with content: %+v", status, got)
			}
			if count := usedOfferCount(server); count == 0 {
				t.Fatalf("no used-offer request sent from the opened HTTP %d page: %+v", status, server.Requests())
			}
			assertAmazonUsedOffer(t, got)
		})

		t.Run("HTTP "+strconv.Itoa(status)+" and used offers answering 503", func(t *testing.T) {
			server := errorPageServer(t, notFoundPage(status), func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusServiceUnavailable)
				io.WriteString(w, "<html><body><h1>Service Unavailable fixture</h1></body></html>")
			})
			output := amazontest.CaptureOutput(t)
			got := browserCheck(t, NewBrowserCollector(testBrowserOptions(t, server.URL)), fixtureItem("B0PRICE001"))
			logged := output()
			if got.Result != "unavailable" || got.SecondHandStatus != "check_error" {
				t.Fatalf("HTTP %d page with failing used offers: %+v", status, got)
			}
			if count := usedOfferCount(server); count == 0 {
				t.Fatalf("no used-offer request sent from the opened HTTP %d page: %+v", status, server.Requests())
			}
			if !strings.Contains(logged, usedOfferFailedLine+" amazon.fr/B0PRICE001") {
				t.Fatalf("no used-offer failure logged after failed used-offer requests:\n%s", logged)
			}
		})
	}
}
