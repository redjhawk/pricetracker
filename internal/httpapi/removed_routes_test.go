package httpapi

import (
	"net/http"
	"testing"
)

// The Amazon searches routes were removed by issue #111.
func TestAmazonSearchRoutesAreGone(t *testing.T) {
	server, _ := newSettingsServer(t)
	for _, path := range []string{"/api/v1/amazon-searches", "/api/v1/amazon-searches/s1", "/api/v1/amazon-searches/s1/refresh"} {
		assertError(t, callPath(server, http.MethodGet, path, ""), 404, "ROUTE_NOT_FOUND", "API route was not found.")
	}
}
