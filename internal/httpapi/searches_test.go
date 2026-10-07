package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

const issue56URL = "https://www.amazon.fr/joursprime/?_encoding=UTF8&promotionsSearchStartIndex=0&promotionsSearchPageSize=60"

// addSearch adds a search through the API and captures two products for it.
func addSearch(t *testing.T, server *Server, database *store.Store, token string, ownerID int64) string {
	t.Helper()
	recorder := request(t, server, http.MethodPost, "/api/v1/amazon-searches", `{"url":" `+issue56URL+` "}`, token)
	var search struct{ ID, URL, Label, State string }
	if recorder.Code != http.StatusCreated || json.Unmarshal(recorder.Body.Bytes(), &search) != nil ||
		search.URL != issue56URL || search.Label != "amazon.fr/joursprime/" || search.State != "waiting" {
		t.Fatalf("add search: %d %s", recorder.Code, recorder.Body.String())
	}
	items := []store.CapturedItem{}
	for _, asin := range []string{"B000000001", "B000000002"} {
		url := "https://www.amazon.fr/dp/" + asin
		price := int64(1000)
		items = append(items, store.CapturedItem{ID: search.ID + asin, ASIN: asin, Marketplace: "amazon.fr", CanonicalURL: "https://amazon.fr/dp/" + asin, URL: url, PriceCents: &price})
	}
	if err := database.CaptureSearch(context.Background(), search.ID, ownerID, items, time.Now()); err != nil {
		t.Fatal(err)
	}
	return search.ID
}

func TestAmazonSearchEndpoints(t *testing.T) {
	server, database := newSettingsServer(t)
	id := addSearch(t, server, database, "", 0)
	base := "/api/v1/amazon-searches/" + id

	assertError(t, request(t, server, http.MethodPost, "/api/v1/amazon-searches", `{"url":"`+issue56URL+`"}`, ""), http.StatusConflict, "SEARCH_ALREADY_ADDED", "This search is already added: amazon.fr/joursprime/.")
	assertError(t, request(t, server, http.MethodPost, "/api/v1/amazon-searches", `{"url":"lego"}`, ""), http.StatusBadRequest, "INVALID_URL", "")
	assertError(t, request(t, server, http.MethodPost, "/api/v1/amazon-searches", `{"url":"https://www.leboncoin.fr/recherche"}`, ""), http.StatusUnprocessableEntity, "UNSUPPORTED_SEARCH", "")
	assertError(t, request(t, server, http.MethodPost, "/api/v1/amazon-searches", `{"url":1}`, ""), http.StatusBadRequest, "INVALID_JSON", "")
	assertError(t, request(t, server, http.MethodPost, "/api/v1/amazon-searches", `{} {}`, ""), http.StatusBadRequest, "INVALID_JSON", "")
	assertError(t, request(t, server, http.MethodPut, "/api/v1/amazon-searches", ``, ""), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "")

	list := request(t, server, http.MethodGet, "/api/v1/amazon-searches", "", "").Body.String()
	if !strings.Contains(list, `"itemCount":2`) || !strings.Contains(list, `"amazonRequests":{"stopped":false,"stoppedAt":null,"consecutiveFailures":0}`) {
		t.Fatalf("unexpected list %s", list)
	}
	details := request(t, server, http.MethodGet, base, "", "").Body.String()
	if !strings.Contains(details, `"position":2`) || !strings.Contains(details, `"tracked":false`) ||
		!strings.Contains(details, `"aiReviewSummary":{"status":"no_token","priceRating":null,"priceCents":null}`) {
		t.Fatalf("unexpected details %s", details)
	}
	// A search item has an AI review state; refresh is refused for it.
	if body := request(t, server, http.MethodGet, "/api/v1/items/"+id+"B000000001", "", "").Body.String(); !strings.Contains(body, `"aiReview":{"tokenConfigured":false`) {
		t.Fatalf("unexpected search item details %s", body)
	}
	assertError(t, request(t, server, http.MethodPost, "/api/v1/items/"+id+"B000000001/refresh", "", ""), http.StatusConflict, "ITEM_NOT_TRACKED", "")

	recorder := request(t, server, http.MethodPost, base+"/items/"+id+"B000000001/track", "", "")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"tracked":true`) {
		t.Fatalf("track: %d %s", recorder.Code, recorder.Body.String())
	}
	assertError(t, request(t, server, http.MethodPost, base+"/items/"+id+"B000000001/track", "", ""), http.StatusConflict, "ITEM_ALREADY_TRACKED", "")
	assertError(t, request(t, server, http.MethodPost, base+"/items/unknown/track", "", ""), http.StatusNotFound, "ITEM_NOT_FOUND", "")
	assertError(t, request(t, server, http.MethodPost, base+"/refresh", "", ""), http.StatusConflict, "AMAZON_NOT_STOPPED", "")
	server = restartStopped(t, server, database)
	if body := request(t, server, http.MethodGet, "/api/v1/amazon/requests", "", "").Body.String(); !strings.Contains(body, `"stopped":true`) {
		t.Fatalf("unexpected requests %s", body)
	}
	if body := request(t, server, http.MethodGet, base, "", "").Body.String(); !strings.Contains(body, `"state":"stopped"`) {
		t.Fatalf("search not stopped %s", body)
	}
	recorder = request(t, server, http.MethodPost, base+"/refresh", "", "")
	if recorder.Code != http.StatusAccepted || !strings.Contains(recorder.Body.String(), `"requestedAt"`) || !strings.Contains(recorder.Body.String(), `"state":"waiting"`) {
		t.Fatalf("refresh: %d %s", recorder.Code, recorder.Body.String())
	}
	if body := request(t, server, http.MethodGet, "/api/v1/amazon/requests", "", "").Body.String(); !strings.Contains(body, `"stopped":false,"stoppedAt":"0001-01-01T00:00:00Z"`) {
		t.Fatalf("stoppedAt must stay until a success: %s", body)
	}
	assertError(t, request(t, server, http.MethodPost, "/api/v1/amazon/requests", "", ""), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "")

	if recorder := request(t, server, http.MethodDelete, base, "", ""); recorder.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", recorder.Code, recorder.Body.String())
	}
	assertError(t, request(t, server, http.MethodGet, base, "", ""), http.StatusNotFound, "SEARCH_NOT_FOUND", "Amazon search was not found.")
	assertError(t, request(t, server, http.MethodDelete, base, "", ""), http.StatusNotFound, "SEARCH_NOT_FOUND", "")
	// The moved product survives the deletion of the search; the other one is deleted.
	if body := request(t, server, http.MethodGet, "/api/v1/items/"+id+"B000000001", "", "").Body.String(); !strings.Contains(body, `"tracked":true`) {
		t.Fatalf("tracked item lost %s", body)
	}
	assertError(t, request(t, server, http.MethodGet, "/api/v1/items/"+id+"B000000002", "", ""), http.StatusNotFound, "ITEM_NOT_FOUND", "")
}

// restartStopped stores a stop of Amazon requests and restarts the server process on the same database.
func restartStopped(t *testing.T, server *Server, database *store.Store) *Server {
	t.Helper()
	server.service.Close()
	if err := database.SaveAmazonRequestState(context.Background(), store.AmazonRequestState{ConsecutiveFailures: 5, Blocked: true, StoppedAt: &time.Time{}}); err != nil {
		t.Fatal(err)
	}
	items := service.New(server.config, database)
	t.Cleanup(items.Close)
	return New(server.config, items)
}

func TestAmazonSearchesBelongToTheirOwner(t *testing.T) {
	server, database := newSettingsServer(t)
	adminPassword, _, err := server.service.ResetAdminPassword(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	admin := login(t, server, "admin", adminPassword)
	for _, name := range []string{"alice", "bob"} {
		request(t, server, http.MethodPost, "/api/v1/admin/users", `{"username":"`+name+`","password":"long enough pw"}`, admin)
	}
	alice, bob := login(t, server, "alice", "long enough pw"), login(t, server, "bob", "long enough pw")
	aliceUser, _ := database.UserByUsername(context.Background(), "alice")
	id := addSearch(t, server, database, alice, aliceUser.ID)
	if recorder := request(t, server, http.MethodPost, "/api/v1/amazon-searches", `{"url":"`+issue56URL+`"}`, bob); recorder.Code != http.StatusCreated {
		t.Fatalf("bob cannot add the same URL: %s", recorder.Body.String())
	}
	if body := request(t, server, http.MethodGet, "/api/v1/amazon-searches", "", bob).Body.String(); strings.Contains(body, id) {
		t.Fatalf("bob sees alice's search %s", body)
	}
	base := "/api/v1/amazon-searches/" + id
	for _, call := range []struct{ method, path string }{
		{http.MethodGet, base}, {http.MethodDelete, base}, {http.MethodPost, base + "/refresh"}, {http.MethodPost, base + "/items/" + id + "B000000001/track"},
	} {
		assertError(t, request(t, server, call.method, call.path, "", bob), http.StatusNotFound, "SEARCH_NOT_FOUND", "")
	}
	if recorder := request(t, server, http.MethodGet, base, "", alice); recorder.Code != http.StatusOK {
		t.Fatalf("alice lost her search: %s", recorder.Body.String())
	}
}
