package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

const purchaseGoalsPath = "/api/v1/purchase-goals"

func insertGoalItem(t *testing.T, database *store.Store, id, goal string, owner int64) {
	t.Helper()
	url := "https://www.leboncoin.fr/ad/goal/" + id
	listing := model.Listing{ID: id, Platform: "leboncoin", ListingID: id, Marketplace: "leboncoin.fr", URL: url, PurchaseGoal: goal, OwnerID: owner}
	if err := database.Insert(context.Background(), listing, url, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func assertGoalsBody(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	if recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != want {
		t.Fatalf("purchase goals %d %s, want 200 %s", recorder.Code, recorder.Body.String(), want)
	}
}

// TC-002: the goals endpoint in protected mode.
func TestPurchaseGoalsEndpoint_TC002_ProtectedMode(t *testing.T) {
	server, database := newSettingsServer(t)
	ctx := context.Background()
	adminPassword, _, err := server.service.ResetAdminPassword(ctx)
	if err != nil {
		t.Fatal(err)
	}
	admin := login(t, server, "admin", adminPassword)
	for _, name := range []string{"alice", "bob", "carol"} {
		if recorder := request(t, server, http.MethodPost, "/api/v1/admin/users", `{"username":"`+name+`","password":"long enough pw"}`, admin); recorder.Code != http.StatusCreated {
			t.Fatalf("create %s: %s", name, recorder.Body.String())
		}
	}
	owners := map[string]int64{}
	for _, name := range []string{"alice", "bob", "carol"} {
		user, err := database.UserByUsername(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		owners[name] = user.ID
	}
	insertGoalItem(t, database, "alice-1", "Gaming", owners["alice"])
	insertGoalItem(t, database, "alice-2", "browsing", owners["alice"])
	insertGoalItem(t, database, "bob-1", "Office", owners["bob"])
	insertGoalItem(t, database, "carol-1", "", owners["carol"])
	alice := login(t, server, "alice", "long enough pw")
	carol := login(t, server, "carol", "long enough pw")

	recorder := request(t, server, http.MethodGet, purchaseGoalsPath, "", alice)
	assertGoalsBody(t, recorder, `{"purchaseGoals":["browsing","Gaming"]}`)
	if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("Content-Type %q", recorder.Header().Get("Content-Type"))
	}
	recorder = request(t, server, http.MethodGet, purchaseGoalsPath, "", carol)
	assertGoalsBody(t, recorder, `{"purchaseGoals":[]}`)

	assertError(t, request(t, server, http.MethodGet, purchaseGoalsPath, "", ""), http.StatusUnauthorized, "AUTH_REQUIRED", "")
	assertError(t, request(t, server, http.MethodGet, purchaseGoalsPath, "", admin), http.StatusForbidden, "FORBIDDEN", "")
	recorder = request(t, server, http.MethodPost, purchaseGoalsPath, "", alice)
	assertError(t, recorder, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "")
	if recorder.Header().Get("Allow") != "GET" {
		t.Fatalf("Allow %q", recorder.Header().Get("Allow"))
	}
}

// TC-002: the goals endpoint in open mode answers without login.
func TestPurchaseGoalsEndpoint_TC002_OpenModeWithoutLogin(t *testing.T) {
	server, database := newSettingsServer(t)
	insertGoalItem(t, database, "a", "Gaming", 0)
	insertGoalItem(t, database, "b", "browsing", 0)
	recorder := callPath(server, http.MethodGet, purchaseGoalsPath, "")
	assertGoalsBody(t, recorder, `{"purchaseGoals":["browsing","Gaming"]}`)
}

// TC-005: HTTP flow; the list reflects an add, a goal change and a delete.
func TestPurchaseGoalsEndpoint_TC005_ListReflectsAddChangeDelete(t *testing.T) {
	withClaudeStatus(t, http.StatusServiceUnavailable) // the listing fetch never leaves the test
	server, _ := newSettingsServer(t)
	readGoals := func(want string) {
		t.Helper()
		recorder := callPath(server, http.MethodGet, purchaseGoalsPath, "")
		assertGoalsBody(t, recorder, want)
	}
	readGoals(`{"purchaseGoals":[]}`)
	assertError(t, callPath(server, http.MethodPost, "/api/v1/items", `{"url":"not a url","purchaseGoal":"Gaming"}`), http.StatusBadRequest, "INVALID_URL", "")
	readGoals(`{"purchaseGoals":[]}`)

	added := callPath(server, http.MethodPost, "/api/v1/items", `{"url":"https://www.leboncoin.fr/ad/test/305","purchaseGoal":"  Gaming  "}`)
	var item model.Item
	if added.Code != http.StatusCreated || json.Unmarshal(added.Body.Bytes(), &item) != nil || item.ID == "" {
		t.Fatalf("add %d %s", added.Code, added.Body.String())
	}
	readGoals(`{"purchaseGoals":["Gaming"]}`)

	if saved := callPath(server, http.MethodPut, "/api/v1/items/"+item.ID+"/purchase-goal", `{"purchaseGoal":"Office"}`); saved.Code != http.StatusOK {
		t.Fatalf("save %d %s", saved.Code, saved.Body.String())
	}
	readGoals(`{"purchaseGoals":["Office"]}`)

	if deleted := callPath(server, http.MethodDelete, "/api/v1/items/"+item.ID, ""); deleted.Code/100 != 2 {
		t.Fatalf("delete %d %s", deleted.Code, deleted.Body.String())
	}
	readGoals(`{"purchaseGoals":[]}`)
}
