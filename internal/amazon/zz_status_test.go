package amazon

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"pricefollower.local/internal/amazon/amazontest"
)

func TestBrowserCollector_EmptyBodyStatusKeepsTheAnswerRules(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		unavailable bool
	}{
		{"HTTP 404", http.StatusNotFound, true},
		{"HTTP 410", http.StatusGone, true},
		{"HTTP 503", http.StatusServiceUnavailable, false},
		{"HTTP 429", http.StatusTooManyRequests, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := amazontest.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
			}))
			collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
			output := captureOutput(t)
			got := browserCheck(t, collector, fixtureItem("B0GONE0001"))
			logged := output()
			if test.unavailable {
				if got.Result != "unavailable" || got.Message != unavailableMessage || got.HTTPStatus != test.status {
					t.Fatalf("not an unavailable listing: %+v", got)
				}
				return
			}
			if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" || got.HTTPStatus != test.status {
				t.Fatalf("not a failure with the HTTP status: %+v", got)
			}
			if want := fmt.Sprintf("Amazon returned HTTP %d", test.status); !strings.Contains(logged, want) {
				t.Fatalf("log misses the HTTP status reason: %s", logged)
			}
		})
	}
}
