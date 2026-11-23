package amazon

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// Browser test of issue #185: a product page answered with an HTTP error status and an empty body
// is classified by that status (404 and 410 unavailable, any other a failure naming the status).

// TC-003 (issue #185)
func TestBrowserEmptyErrorStatus_Issue185_TC003_ClassifiedByStatus(t *testing.T) {
	cases := []struct {
		status      int
		unavailable bool
	}{
		{http.StatusNotFound, true},
		{http.StatusGone, true},
		{http.StatusServiceUnavailable, false},
		{http.StatusTooManyRequests, false},
	}
	for _, row := range cases {
		status := row.status
		t.Run("HTTP "+strconv.Itoa(status), func(t *testing.T) {
			server := errorPageServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				w.WriteHeader(status)
			}, amazonUsedOffer(t))
			collector := NewBrowserCollector(testBrowserOptions(t, server.URL))

			output := captureOutput(t)
			got := browserCheck(t, collector, fixtureItem("B0PRICE001"))
			logged := output()
			if server.Count("/dp/B0PRICE001") == 0 {
				t.Fatalf("the browser did not request the product page: %+v", server.Requests())
			}
			if got.HTTPStatus != status {
				t.Fatalf("HTTP status %d recorded for an empty HTTP %d answer: %+v", got.HTTPStatus, status, got)
			}
			if row.unavailable {
				if got.Result != "unavailable" || got.Message != unavailableMessage {
					t.Fatalf("HTTP %d with an empty body, want unavailable: %+v (requests %+v)", status, got, server.Requests())
				}
				return
			}
			if got.Result != "request_error" || got.Message != amazonCheckFailedMessage || got.SecondHandStatus != "check_error" {
				t.Fatalf("HTTP %d with an empty body, want a failure: %+v (requests %+v)", status, got, server.Requests())
			}
			cause := fmt.Sprintf("Amazon returned HTTP %d", status)
			if got.FailureCause != cause {
				t.Fatalf("failure cause %q, want %q: %+v", got.FailureCause, cause, got)
			}
			if line := "Amazon collection failed for amazon.fr/B0PRICE001: " + cause; !strings.Contains(logged, line) {
				t.Fatalf("output has no line %q:\n%s", line, logged)
			}
		})
	}
}
