package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

func insertItem(t *testing.T, database *store.Store, id, platform string) {
	t.Helper()
	listing := model.Listing{ID: id, Platform: platform, ListingID: "123", Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/x/123"}
	if platform == "amazon" {
		listing = model.Listing{ID: id, Platform: platform, ListingID: "B012345678", ASIN: "B012345678", Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/B012345678"}
	}
	if err := database.Insert(context.Background(), listing, listing.URL, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func TestItemDetailsAIReviewField(t *testing.T) {
	server, database := newSettingsServer(t)
	insertItem(t, database, "lbc", "leboncoin")
	insertItem(t, database, "amz", "amazon")
	lbc := callPath(server, http.MethodGet, "/api/v1/items/lbc", "")
	want := `"aiReview":{"tokenConfigured":false,"running":false,"lastAttempt":null,"latest":null,"history":[]}`
	if lbc.Code != 200 || !strings.Contains(lbc.Body.String(), want) || !strings.Contains(lbc.Body.String(), `"id":"lbc"`) {
		t.Fatalf("LeBoncoin details %d %s", lbc.Code, lbc.Body.String())
	}
	if amz := callPath(server, http.MethodGet, "/api/v1/items/amz", ""); !strings.Contains(amz.Body.String(), `"aiReview":null`) {
		t.Fatalf("Amazon details %s", amz.Body.String())
	}
	if list := callPath(server, http.MethodGet, "/api/v1/items", ""); strings.Contains(list.Body.String(), "aiReview") {
		t.Fatalf("list includes aiReview: %s", list.Body.String())
	}
}

func TestRequestAIReviewStatuses(t *testing.T) {
	server, database := newSettingsServer(t)
	insertItem(t, database, "lbc", "leboncoin")
	insertItem(t, database, "amz", "amazon")
	assertError(t, callPath(server, http.MethodPost, "/api/v1/items/missing/ai-review", ""), 404, "ITEM_NOT_FOUND", "Tracked item was not found.")
	assertError(t, callPath(server, http.MethodPost, "/api/v1/items/amz/ai-review", ""), 422, "AI_REVIEW_UNSUPPORTED", "AI reviews are available for LeBoncoin items only.")
	assertError(t, callPath(server, http.MethodPost, "/api/v1/items/lbc/ai-review", ""), 409, "CLAUDE_TOKEN_MISSING", "Configure a Claude token in Settings.")
	recorder := callPath(server, http.MethodGet, "/api/v1/items/lbc/ai-review", "")
	assertError(t, recorder, 405, "METHOD_NOT_ALLOWED", "")
	if recorder.Header().Get("Allow") != "POST" {
		t.Fatalf("Allow %q", recorder.Header().Get("Allow"))
	}

	// Every outgoing request (LeBoncoin fetch, Claude) fails, so the review fails quickly.
	withClaudeStatus(t, 500)
	token := "sk-ant-oat01-synthetic"
	if _, _, err := database.SaveSettings(context.Background(), 0, nil, &store.TokenChange{Value: &token}, time.Now()); err != nil {
		t.Fatal(err)
	}
	accepted := callPath(server, http.MethodPost, "/api/v1/items/lbc/ai-review", "")
	if accepted.Code != 202 || !strings.Contains(accepted.Body.String(), `"requestedAt"`) || !strings.Contains(accepted.Body.String(), `"alreadyRunning":`) {
		t.Fatalf("accepted %d %s", accepted.Code, accepted.Body.String())
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		body := callPath(server, http.MethodGet, "/api/v1/items/lbc", "").Body.String()
		if strings.Contains(body, `"running":false`) && strings.Contains(body, `"status":"failed"`) {
			if strings.Contains(body, token) {
				t.Fatal("token leaked into the item details")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("review did not fail")
}
