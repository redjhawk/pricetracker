package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

func TestUsersSeeOnlyTheirOwnItemsAndSettings(t *testing.T) {
	server, database := newSettingsServer(t)
	ctx := context.Background()
	insertItem(t, database, "open-item", "leboncoin")
	adminPassword, _, err := server.service.ResetAdminPassword(ctx)
	if err != nil {
		t.Fatal(err)
	}
	admin := login(t, server, "admin", adminPassword)
	for _, name := range []string{"alice", "bob"} {
		if recorder := request(t, server, http.MethodPost, "/api/v1/admin/users", `{"username":"`+name+`","password":"long enough pw"}`, admin); recorder.Code != http.StatusCreated {
			t.Fatalf("create %s: %s", name, recorder.Body.String())
		}
	}
	bobUser, _ := database.UserByUsername(ctx, "bob")
	// Bob tracks the same URL as the item Alice inherited, as an independent item.
	listing := model.Listing{ID: "bob-item", Platform: "leboncoin", ListingID: "123", Marketplace: "leboncoin.fr", URL: "https://www.leboncoin.fr/ad/x/123", OwnerID: bobUser.ID}
	if err := database.Insert(ctx, listing, listing.URL, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	alice := login(t, server, "alice", "long enough pw")
	bob := login(t, server, "bob", "long enough pw")

	list := request(t, server, http.MethodGet, "/api/v1/items", "", alice).Body.String()
	if !strings.Contains(list, `"open-item"`) || strings.Contains(list, `"bob-item"`) {
		t.Fatalf("alice list %s", list)
	}
	list = request(t, server, http.MethodGet, "/api/v1/items", "", bob).Body.String()
	if !strings.Contains(list, `"bob-item"`) || strings.Contains(list, `"open-item"`) {
		t.Fatalf("bob list %s", list)
	}
	for _, call := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/items/open-item", ""},
		{http.MethodPost, "/api/v1/items/open-item/refresh", ""},
		{http.MethodPut, "/api/v1/items/open-item/purchase-goal", `{"purchaseGoal":"x"}`},
		{http.MethodPost, "/api/v1/items/open-item/ai-review", ""},
		{http.MethodDelete, "/api/v1/items/open-item", ""},
	} {
		assertError(t, request(t, server, call.method, call.path, call.body, bob), http.StatusNotFound, "ITEM_NOT_FOUND", "")
	}
	if recorder := request(t, server, http.MethodGet, "/api/v1/items/open-item", "", alice); recorder.Code != http.StatusOK {
		t.Fatalf("alice lost her item: %s", recorder.Body.String())
	}
	if recorder := request(t, server, http.MethodPut, settingsPath, `{"value":"","revision":0}`, bob); recorder.Code != http.StatusOK {
		t.Fatalf("bob settings: %s", recorder.Body.String())
	}
	bobSession := "bob-session"
	if _, err := database.SaveLeboncoinSession(ctx, bobUser.ID, &bobSession, 0, time.Now()); err != nil {
		t.Fatal(err)
	}
	if body := request(t, server, http.MethodGet, settingsPath, "", alice).Body.String(); strings.Contains(body, "bob-session") {
		t.Fatalf("alice sees bob's session: %s", body)
	}
}
