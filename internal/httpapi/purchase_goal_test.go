package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestPurchaseGoalStatuses(t *testing.T) {
	server, database := newSettingsServer(t)
	insertItem(t, database, "lbc", "leboncoin")
	insertItem(t, database, "amz", "amazon")
	if details := callPath(server, http.MethodGet, "/api/v1/items/lbc", ""); !strings.Contains(details.Body.String(), `"purchaseGoal":""`) {
		t.Fatalf("existing item goal %s", details.Body.String())
	}
	long := strings.Repeat("Light Linux distro. ", 500)
	saved := callPath(server, http.MethodPut, "/api/v1/items/lbc/purchase-goal", `{"purchaseGoal":"`+long+`"}`)
	want := `{"changed":true,"purchaseGoal":"` + strings.TrimSpace(long) + `","reviewStarted":false}`
	if saved.Code != 200 || strings.TrimSpace(saved.Body.String()) != want {
		t.Fatalf("save %d %s", saved.Code, saved.Body.String())
	}
	if details := callPath(server, http.MethodGet, "/api/v1/items/lbc", ""); !strings.Contains(details.Body.String(), `"purchaseGoal":"`+strings.TrimSpace(long)+`"`) {
		t.Fatal("long goal not returned in details")
	}
	assertError(t, callPath(server, http.MethodPut, "/api/v1/items/lbc/purchase-goal", `{"purchaseGoal":3}`), 400, "INVALID_JSON", "")
	assertError(t, callPath(server, http.MethodPut, "/api/v1/items/lbc/purchase-goal", `{}`), 400, "INVALID_JSON", "")
	assertError(t, callPath(server, http.MethodPut, "/api/v1/items/missing/purchase-goal", `{"purchaseGoal":""}`), 404, "ITEM_NOT_FOUND", "Tracked item was not found.")
	assertError(t, callPath(server, http.MethodPut, "/api/v1/items/amz/purchase-goal", `{"purchaseGoal":""}`), 422, "PURCHASE_GOAL_UNSUPPORTED", "Purchase goals are available for LeBoncoin items only.")
	recorder := callPath(server, http.MethodPost, "/api/v1/items/lbc/purchase-goal", "")
	assertError(t, recorder, 405, "METHOD_NOT_ALLOWED", "")
	if recorder.Header().Get("Allow") != "PUT" {
		t.Fatalf("Allow %q", recorder.Header().Get("Allow"))
	}
}
