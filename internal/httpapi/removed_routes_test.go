package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The Amazon searches routes and the global Amazon request state were removed by issue #111.
func TestAmazonSearchRoutesAreGone(t *testing.T) {
	server, _ := newSettingsServer(t)
	for _, path := range []string{"/api/v1/amazon-searches", "/api/v1/amazon-searches/s1", "/api/v1/amazon-searches/s1/refresh", "/api/v1/amazon/requests"} {
		assertError(t, callPath(server, http.MethodGet, path, ""), 404, "ROUTE_NOT_FOUND", "API route was not found.")
	}
}

func TestItemsCarryTheirAmazonStop(t *testing.T) {
	server, database := newSettingsServer(t)
	insertItem(t, database, "lbc", "leboncoin")
	insertItem(t, database, "amz", "amazon")
	if err := database.StopAmazonItems(context.Background(), time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	body := callPath(server, http.MethodGet, "/api/v1/items", "").Body.String()
	if !strings.Contains(body, `"amazonStoppedAt":"2026-11-10T08:00:00Z"`) || !strings.Contains(body, `"amazonStoppedAt":null`) {
		t.Fatalf("items %s", body)
	}
}
