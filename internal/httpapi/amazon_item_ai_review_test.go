package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

// Amazon item AI review and purchase goal (issue #305) through the HTTP API: real SQLite, a fake
// Amazon collector counting every check and a fake Claude answering valid Amazon reviews.

const (
	amazonReviewGoal      = "gift for a 10-year-old"
	stoppedReviewMessage  = "Amazon checks are stopped: restart the item with Refresh price to request an AI review."
	noCurrentPriceMessage = "No current price: an AI review needs a current Amazon price."
	amazonReviewAnswer    = `{"price":{"rating":"good_deal","explanation":"Below the usual price."},"fairPrice":{"minCents":11000,"maxCents":13000},"recommendation":{"rating":"wait","explanation":"The price often drops."}}`
)

var reviewCheckAt = time.Date(2026, time.November, 2, 8, 0, 0, 0, time.UTC)

// claudeFake answers every outgoing request (only Claude in these tests) with a valid Amazon review
// and records the request bodies.
type claudeFake struct {
	mu     sync.Mutex
	bodies []string
}

func withClaudeReviews(t *testing.T) *claudeFake {
	t.Helper()
	fake := &claudeFake{}
	original := http.DefaultTransport
	http.DefaultTransport = claudeTransport(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		fake.mu.Lock()
		fake.bodies = append(fake.bodies, string(body))
		fake.mu.Unlock()
		answer, _ := json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": amazonReviewAnswer}}, "stop_reason": "end_turn"})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewReader(answer)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	return fake
}

func (f *claudeFake) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bodies)
}

// userText returns the user text of the index-th Claude request.
func (f *claudeFake) userText(t *testing.T, index int) string {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if index >= len(f.bodies) {
		t.Fatalf("Claude received %d requests, want more than %d", len(f.bodies), index)
	}
	var body struct {
		Messages []struct {
			Content []struct{ Text string }
		}
	}
	if err := json.Unmarshal([]byte(f.bodies[index]), &body); err != nil || len(body.Messages) == 0 || len(body.Messages[0].Content) == 0 {
		t.Fatalf("Claude request %d: %v %s", index, err, f.bodies[index])
	}
	return body.Messages[0].Content[0].Text
}

// openAmazonReviewAPI serves the API of a service using checks for Amazon.
func openAmazonReviewAPI(t *testing.T) (*Server, *store.Store, *amazonChecks) {
	t.Helper()
	cfg := config.Config{DataDirectory: t.TempDir(), Location: time.UTC, CheckTimes: []string{"08:00"}, StaleAfter: 36 * time.Hour}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	checks := &amazonChecks{}
	items := service.NewWithOptions(cfg, database, service.Options{Amazon: checks})
	t.Cleanup(items.Close)
	return New(cfg, items), database, checks
}

func saveClaudeToken(t *testing.T, database *store.Store, token *string) {
	t.Helper()
	if _, _, err := database.SaveSettings(context.Background(), 0, nil, &store.TokenChange{Value: token}, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func syntheticToken() *string {
	token := "sk-ant-oat01-synthetic"
	return &token
}

// insertReviewedAmazonItem inserts a priced Amazon item (asin) with one stored review and returns the review ID.
func insertReviewedAmazonItem(t *testing.T, database *store.Store, id, asin string, reviewed bool) int64 {
	t.Helper()
	listing := model.Listing{ID: id, Platform: "amazon", ListingID: asin, ASIN: asin, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + asin}
	if err := database.Insert(context.Background(), listing, listing.URL, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	recordCheck(t, database, id, model.CollectionResult{Result: "success", AmountCents: 11999, SecondHandStatus: "not_found"}, reviewCheckAt)
	if !reviewed {
		return 0
	}
	return storeReview(t, database, id)
}

func recordCheck(t *testing.T, database *store.Store, id string, result model.CollectionResult, at time.Time) {
	t.Helper()
	if err := database.RecordCollection(context.Background(), id, result, at); err != nil {
		t.Fatal(err)
	}
}

func storeReview(t *testing.T, database *store.Store, id string) int64 {
	t.Helper()
	ctx := context.Background()
	price := int64(11999)
	reviewID, err := database.StartAIReview(ctx, id, reviewCheckAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	content := &model.AmazonAIReviewContent{
		Price:          model.AIRating{Rating: "fair", Explanation: "Usual price."},
		FairPrice:      model.AmazonFairPrice{MinCents: 11000, MaxCents: 13000},
		Recommendation: model.AIRating{Rating: "buy", Explanation: "Good time to buy."},
	}
	if err := database.FinishAIReview(ctx, reviewID, "succeeded", &price, content, "", reviewCheckAt.Add(time.Hour+time.Minute)); err != nil {
		t.Fatal(err)
	}
	return reviewID
}

// reviewDetails returns the decoded details of the item and the raw body.
func reviewDetails(t *testing.T, server *Server, id string) (model.ItemDetails, string) {
	t.Helper()
	recorder := callPath(server, http.MethodGet, "/api/v1/items/"+id, "")
	var details model.ItemDetails
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &details) != nil {
		t.Fatalf("GET /api/v1/items/%s: %d %s", id, recorder.Code, recorder.Body.String())
	}
	if details.AIReview == nil {
		t.Fatalf("item %s has no aiReview object: %s", id, recorder.Body.String())
	}
	return details, recorder.Body.String()
}

func reason(details model.ItemDetails) string {
	if details.AIReview.UnavailableReason == nil {
		return "<null>"
	}
	return *details.AIReview.UnavailableReason
}

// waitReviewDone polls the details until no review is running.
func waitReviewDone(t *testing.T, server *Server, id string) model.ItemDetails {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if details, _ := reviewDetails(t, server, id); !details.AIReview.Running {
			return details
		}
	}
	t.Fatalf("the review of %s did not finish", id)
	return model.ItemDetails{}
}

func lastAttemptID(details model.ItemDetails) int64 {
	if details.AIReview.LastAttempt == nil {
		return 0
	}
	return details.AIReview.LastAttempt.ID
}

func TestAmazonItemAIReview_TC005_AddSavesGoalAndStartsNoReview(t *testing.T) {
	server, database, _ := openAmazonReviewAPI(t)
	claude := withClaudeReviews(t)
	saveClaudeToken(t, database, syntheticToken())
	added := callPath(server, http.MethodPost, "/api/v1/items", `{"url":"https://www.amazon.fr/dp/B012345678","purchaseGoal":"`+amazonReviewGoal+`"}`)
	var item model.Item
	if added.Code != http.StatusCreated || json.Unmarshal(added.Body.Bytes(), &item) != nil || item.PurchaseGoal != amazonReviewGoal {
		t.Fatalf("add: %d %s", added.Code, added.Body.String())
	}
	waitChecked(t, server, item.ID)
	time.Sleep(100 * time.Millisecond)
	details, body := reviewDetails(t, server, item.ID)
	if details.PurchaseGoal != amazonReviewGoal || details.LatestPrice == nil || details.LatestPrice.AmountCents != 12999 {
		t.Fatalf("details %s", body)
	}
	state := details.AIReview
	if !state.TokenConfigured || state.Running || state.Latest != nil || state.LastAttempt != nil || state.UnavailableReason != nil || !strings.Contains(body, `"history":[]`) {
		t.Fatalf("AI review state %s", body)
	}
	if claude.calls() != 0 {
		t.Fatalf("adding an Amazon item sent %d Claude requests", claude.calls())
	}

	paragraphs := make([]string, 250)
	for index := range paragraphs {
		paragraphs[index] = "A gift for my nephew, who loves music."
	}
	long := strings.Join(paragraphs, "\n\n") // about 10 000 characters
	encodedLong, _ := json.Marshal("  " + long + " \n")
	for _, test := range []struct{ asin, body, want string }{
		{"B012345679", `{"url":"https://www.amazon.fr/dp/B012345679","purchaseGoal":"   \n "}`, ""},
		{"B01234567A", `{"url":"https://www.amazon.fr/dp/B01234567A","purchaseGoal":` + string(encodedLong) + `}`, long},
		{"B01234567B", `{"url":"https://www.amazon.fr/dp/B01234567B"}`, ""},
	} {
		recorder := callPath(server, http.MethodPost, "/api/v1/items", test.body)
		var added model.Item
		if recorder.Code != http.StatusCreated || json.Unmarshal(recorder.Body.Bytes(), &added) != nil {
			t.Fatalf("add %s: %d %s", test.asin, recorder.Code, recorder.Body.String())
		}
		waitChecked(t, server, added.ID)
		if details, _ := reviewDetails(t, server, added.ID); details.PurchaseGoal != test.want {
			t.Fatalf("goal of %s: %d characters, want %d", test.asin, len(details.PurchaseGoal), len(test.want))
		}
	}
	before := len(listItems(t, server))
	assertError(t, callPath(server, http.MethodPost, "/api/v1/items", `{"url":"https://www.amazon.fr/not-a-product","purchaseGoal":"x"}`), 422, "UNSUPPORTED_LISTING", "")
	assertError(t, callPath(server, http.MethodPost, "/api/v1/items", `{"url":"not a url","purchaseGoal":"x"}`), 400, "INVALID_URL", "")
	if after := len(listItems(t, server)); after != before {
		t.Fatalf("a failed add stored an item: %d items, want %d", after, before)
	}
	if list := callPath(server, http.MethodGet, "/api/v1/items", ""); strings.Contains(list.Body.String(), "aiReview") {
		t.Fatalf("the items list includes aiReview: %s", list.Body.String())
	}
	if claude.calls() != 0 {
		t.Fatalf("%d Claude requests", claude.calls())
	}
}

func TestAmazonItemAIReview_TC006_SavingGoalStartsNoReview(t *testing.T) {
	server, database, _ := openAmazonReviewAPI(t)
	claude := withClaudeReviews(t)
	saveClaudeToken(t, database, syntheticToken())
	reviewID := insertReviewedAmazonItem(t, database, "A", "B012345678", true)
	if _, err := database.SetPurchaseGoal(context.Background(), "A", "Goal A"); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct{ body, want string }{
		{`{"purchaseGoal":"Goal B"}`, `{"changed":true,"purchaseGoal":"Goal B","reviewStarted":false}`},
		{`{"purchaseGoal":" Goal B "}`, `{"changed":false,"purchaseGoal":"Goal B","reviewStarted":false}`},
		{`{"purchaseGoal":""}`, `{"changed":true,"purchaseGoal":"","reviewStarted":false}`},
		{`{"purchaseGoal":"  \n "}`, `{"changed":false,"purchaseGoal":"","reviewStarted":false}`},
	} {
		saved := callPath(server, http.MethodPut, "/api/v1/items/A/purchase-goal", step.body)
		if saved.Code != http.StatusOK || strings.TrimSpace(saved.Body.String()) != step.want {
			t.Fatalf("save %s: %d %s", step.body, saved.Code, saved.Body.String())
		}
		var want struct{ PurchaseGoal string }
		json.Unmarshal([]byte(step.want), &want)
		details, body := reviewDetails(t, server, "A")
		if details.PurchaseGoal != want.PurchaseGoal || details.AIReview.Running || details.AIReview.Latest == nil || details.AIReview.Latest.ID != reviewID ||
			len(details.AIReview.History) != 1 || lastAttemptID(details) != reviewID {
			t.Fatalf("after saving %s: %s", step.body, body)
		}
	}
	time.Sleep(100 * time.Millisecond)
	if claude.calls() != 0 {
		t.Fatalf("saving a goal sent %d Claude requests", claude.calls())
	}

	if recorder := callPath(server, http.MethodPost, "/api/v1/items/A/ai-review", ""); recorder.Code != http.StatusAccepted {
		t.Fatalf("review request %d %s", recorder.Code, recorder.Body.String())
	}
	waitReviewDone(t, server, "A")
	if text := claude.userText(t, 0); strings.Contains(text, "purchase_goal") || strings.Contains(text, "Goal B") {
		t.Fatalf("a cleared goal was sent:\n%s", text)
	}
	callPath(server, http.MethodPut, "/api/v1/items/A/purchase-goal", `{"purchaseGoal":"Goal B for a teenager"}`)
	if recorder := callPath(server, http.MethodPost, "/api/v1/items/A/ai-review", ""); recorder.Code != http.StatusAccepted {
		t.Fatalf("review request %d %s", recorder.Code, recorder.Body.String())
	}
	waitReviewDone(t, server, "A")
	if text := claude.userText(t, 1); !strings.Contains(text, "<purchase_goal>") || !strings.Contains(text, "Goal B for a teenager") {
		t.Fatalf("the saved goal was not sent:\n%s", text)
	}
	if claude.calls() != 2 {
		t.Fatalf("%d Claude requests, want 2", claude.calls())
	}
	assertError(t, callPath(server, http.MethodPut, "/api/v1/items/missing/purchase-goal", `{"purchaseGoal":"x"}`), 404, "ITEM_NOT_FOUND", "Tracked item was not found.")

	t.Run("protected mode", func(t *testing.T) {
		ctx := context.Background()
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
		// The first user inherits the open-mode items: item A is Alice's.
		bob := login(t, server, "bob", "long enough pw")
		assertError(t, request(t, server, http.MethodPut, "/api/v1/items/A/purchase-goal", `{"purchaseGoal":"Bob's goal"}`, bob), 404, "ITEM_NOT_FOUND", "")
		alice := login(t, server, "alice", "long enough pw")
		var item model.Item
		recorder := request(t, server, http.MethodGet, "/api/v1/items/A", "", alice)
		if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &item) != nil || item.PurchaseGoal != "Goal B for a teenager" {
			t.Fatalf("Alice's goal changed: %d %s", recorder.Code, recorder.Body.String())
		}
	})
}

func TestAmazonItemAIReview_TC011_StoppedItemRefusesReviewsUntilRestarted(t *testing.T) {
	server, database, checks := openAmazonReviewAPI(t)
	claude := withClaudeReviews(t)
	ctx := context.Background()
	saveClaudeToken(t, database, syntheticToken())
	reviewID := insertReviewedAmazonItem(t, database, "A", "B012345678", true)
	insertReviewedAmazonItem(t, database, "F", "B012345679", true)
	neverPriced := model.Listing{ID: "N", Platform: "amazon", ListingID: "B01234567A", ASIN: "B01234567A", Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/B01234567A"}
	if err := database.Insert(ctx, neverPriced, neverPriced.URL, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	insertReviewedAmazonItem(t, database, "P", "B01234567B", true)
	if err := database.StopAmazonItems(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}

	details, body := reviewDetails(t, server, "A")
	if reason(details) != "stopped" || details.AIReview.Latest == nil || details.AIReview.Latest.ID != reviewID || len(details.AIReview.History) != 1 {
		t.Fatalf("stopped item details %s", body)
	}
	for _, token := range []*string{syntheticToken(), nil} {
		saveClaudeToken(t, database, token)
		assertError(t, callPath(server, http.MethodPost, "/api/v1/items/A/ai-review", ""), 409, "AI_REVIEW_ITEM_STOPPED", stoppedReviewMessage)
	}
	saveClaudeToken(t, database, syntheticToken())
	if details, _ := reviewDetails(t, server, "A"); lastAttemptID(details) != reviewID || details.AIReview.Running {
		t.Fatalf("a refused review stored an attempt: %+v", details.AIReview.LastAttempt)
	}
	if details, body := reviewDetails(t, server, "N"); reason(details) != "stopped" {
		t.Fatalf("stopped item without a price %s", body)
	}

	refresh(t, server, "A")
	waitChecked(t, server, "A")
	details, _ = reviewDetails(t, server, "A")
	if details.AmazonStoppedAt != nil || details.AIReview.UnavailableReason != nil || details.AIReview.Running || lastAttemptID(details) != reviewID {
		t.Fatalf("restarted item: stopped=%v reason=%s running=%v", details.AmazonStoppedAt, reason(details), details.AIReview.Running)
	}

	t.Run("failed Refresh price keeps the stop", func(t *testing.T) {
		checks.mu.Lock()
		checks.result = func(item model.Listing) model.CollectionResult {
			if item.ID == "F" {
				return failedCheck(item)
			}
			if item.ID == "P" {
				return model.CollectionResult{Result: "price_not_found", Message: "Amazon did not show a detectable euro price.", SecondHandStatus: "check_error"}
			}
			return model.CollectionResult{Result: "success", AmountCents: 12999, SecondHandStatus: "not_found"}
		}
		checks.mu.Unlock()
		refresh(t, server, "F")
		waitChecked(t, server, "F")
		details, body := reviewDetails(t, server, "F")
		if details.AmazonStoppedAt == nil || reason(details) != "stopped" {
			t.Fatalf("item after a failed restart %s", body)
		}
		assertError(t, callPath(server, http.MethodPost, "/api/v1/items/F/ai-review", ""), 409, "AI_REVIEW_ITEM_STOPPED", stoppedReviewMessage)
	})
	t.Run("restart that finds no price", func(t *testing.T) {
		refresh(t, server, "P")
		waitChecked(t, server, "P")
		details, body := reviewDetails(t, server, "P")
		if details.AmazonStoppedAt != nil || reason(details) != "no_current_price" {
			t.Fatalf("item restarted without a price %s", body)
		}
		assertError(t, callPath(server, http.MethodPost, "/api/v1/items/P/ai-review", ""), 409, "AI_REVIEW_NO_CURRENT_PRICE", noCurrentPriceMessage)
	})
	time.Sleep(100 * time.Millisecond)
	if claude.calls() != 0 {
		t.Fatalf("%d Claude requests", claude.calls())
	}
}

func TestAmazonItemAIReview_TC012_NoCurrentPriceRefusesReviewsUntilPriced(t *testing.T) {
	server, database, _ := openAmazonReviewAPI(t)
	claude := withClaudeReviews(t)
	ctx := context.Background()
	saveClaudeToken(t, database, syntheticToken())
	reviews := map[string]int64{}
	never := model.Listing{ID: "N", Platform: "amazon", ListingID: "B01234567A", ASIN: "B01234567A", Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/B01234567A"}
	if err := database.Insert(ctx, never, never.URL, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	reviews["N"] = storeReview(t, database, "N")
	reviews["P"] = insertReviewedAmazonItem(t, database, "P", "B012345678", true)
	recordCheck(t, database, "P", model.CollectionResult{Result: "price_not_found", Message: "Amazon did not show a detectable euro price.", SecondHandStatus: "check_error"}, reviewCheckAt.Add(2*time.Hour))
	reviews["U"] = insertReviewedAmazonItem(t, database, "U", "B012345679", true)
	recordCheck(t, database, "U", model.CollectionResult{Result: "unavailable", Message: "The listing is unavailable.", SecondHandStatus: "check_error"}, reviewCheckAt.Add(2*time.Hour))

	for _, id := range []string{"N", "P", "U"} {
		details, body := reviewDetails(t, server, id)
		if reason(details) != "no_current_price" || details.AIReview.Latest == nil || details.AIReview.Latest.ID != reviews[id] {
			t.Fatalf("details of %s: %s", id, body)
		}
		assertError(t, callPath(server, http.MethodPost, "/api/v1/items/"+id+"/ai-review", ""), 409, "AI_REVIEW_NO_CURRENT_PRICE", noCurrentPriceMessage)
		if details, _ := reviewDetails(t, server, id); lastAttemptID(details) != reviews[id] || details.AIReview.Running {
			t.Fatalf("a refused review of %s stored an attempt", id)
		}
	}
	saveClaudeToken(t, database, nil)
	assertError(t, callPath(server, http.MethodPost, "/api/v1/items/N/ai-review", ""), 409, "AI_REVIEW_NO_CURRENT_PRICE", noCurrentPriceMessage)
	saveClaudeToken(t, database, syntheticToken())

	for _, id := range []string{"N", "P", "U"} {
		refresh(t, server, id)
		waitChecked(t, server, id)
		details, body := reviewDetails(t, server, id)
		if details.AIReview.UnavailableReason != nil || details.AIReview.Running || details.AIReview.Latest == nil || details.AIReview.Latest.ID != reviews[id] || lastAttemptID(details) != reviews[id] {
			t.Fatalf("details of %s after a price: %s", id, body)
		}
	}
	time.Sleep(100 * time.Millisecond)
	if claude.calls() != 0 {
		t.Fatalf("%d Claude requests", claude.calls())
	}
}
