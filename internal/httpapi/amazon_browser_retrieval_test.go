package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

// Amazon browser retrieval of issue #204 through the HTTP API: a real browser collector checks
// product pages that mention "captcha" or "robot", as Amazon's do, served by a local fixture server.

const (
	retrievalTitle = "Robot aspirateur laveur X200"
	retrievalImage = "https://m.media-amazon.com/images/I/mentions-captcha.jpg"
)

// retrievalFixture serves the product page of each ASIN (page by default), setting a session cookie,
// the used-offer answer used and the page resources. Its pages can be switched during a test.
type retrievalFixture struct {
	mu    sync.Mutex
	page  string
	pages map[string]string // pages by ASIN, before page
	used  string
}

func (f *retrievalFixture) setPage(page string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.page = page
}

func (f *retrievalFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	page, used := f.page, f.used
	if byASIN, ok := f.pages[strings.TrimPrefix(r.URL.Path, "/dp/")]; ok {
		page = byASIN
	}
	f.mu.Unlock()
	switch {
	case strings.HasPrefix(r.URL.Path, "/dp/"):
		http.SetCookie(w, &http.Cookie{Name: "session-id", Value: "fixture-session-204", Path: "/", MaxAge: 3600})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	case strings.HasPrefix(r.URL.Path, "/gp/aod/ajax"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, used)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func retrievalCollector(t *testing.T, origin string) *amazon.BrowserCollector {
	t.Helper()
	return amazon.NewBrowserCollector(amazon.BrowserOptions{ExecPath: amazontest.Browser(t), ProfileDir: filepath.Join(t.TempDir(), "profile"), NoSandbox: true,
		NavigationTimeout: 30 * time.Second, Pause: func() time.Duration { return 0 }, Origin: origin})
}

// captureRetrievalLog collects the standard logger output until the end of the test.
func captureRetrievalLog(t *testing.T) *syncBuffer {
	t.Helper()
	logs := &syncBuffer{}
	previous := log.Writer()
	log.SetOutput(logs)
	t.Cleanup(func() { log.SetOutput(previous) })
	return logs
}

// runScheduler runs the scheduler until the check of id at the time at is finished. Its first run is at once.
func runScheduler(t *testing.T, server *Server, items *service.Service, id string, at time.Time) model.Item {
	t.Helper()
	scheduler, stop := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() { items.RunScheduler(scheduler); close(finished) }()
	item := waitBrowserCheck(t, server, id, at)
	stop()
	<-finished
	return item
}

// waitStoredCheck waits until the check of id at the time at is recorded (protected mode: no anonymous API access).
func waitStoredCheck(t *testing.T, database *store.Store, id string, at time.Time) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Minute); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		item, err := database.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if item.LastAttempt != nil && item.LastAttempt.Result != "pending" && item.LastAttempt.Timestamp.Equal(at) {
			return
		}
	}
	t.Fatalf("the check of %s did not finish", id)
}

// assertRetrievalListed checks that item shows the fixture price, title and image and the Amazon
// second-hand offer of aod-amazon-used.html, checked at the time at, without a retrieval issue.
func assertRetrievalListed(t *testing.T, item model.Item, at time.Time) {
	t.Helper()
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 || !equalTime(item.LatestPrice.Timestamp, at) {
		t.Fatalf("item %s: latest price %+v, want 129.99 checked at %s", item.ID, item.LatestPrice, at)
	}
	if item.Title == nil || *item.Title != retrievalTitle || item.ThumbnailURL == nil || *item.ThumbnailURL != retrievalImage {
		t.Fatalf("item %s: title %v, image %v", item.ID, item.Title, item.ThumbnailURL)
	}
	if item.Status != "active" || item.LastAttempt == nil || item.LastAttempt.Result != "success" || !item.LastAttempt.Timestamp.Equal(at) || item.LastAttempt.Message != nil {
		t.Fatalf("item %s: status %s, last attempt %+v", item.ID, item.Status, item.LastAttempt)
	}
	offer := item.SecondHandOffer
	if offer == nil || offer.Status != "available" || offer.LatestDetection == nil || offer.LatestDetection.AmountCents != 9550 ||
		offer.LatestDetection.Condition != "like_new" || !offer.LatestDetection.Timestamp.Equal(at) {
		t.Fatalf("item %s: second-hand offer %+v", item.ID, offer)
	}
}

// assertNoRetrievalFailure checks that no Amazon failure is counted or logged.
func assertNoRetrievalFailure(t *testing.T, database *store.Store, logs *syncBuffer) {
	t.Helper()
	if got := storedFailures(t, database); got != 0 {
		t.Fatalf("consecutive failure count %d, want 0", got)
	}
	if strings.Contains(logs.String(), "Amazon request failed") {
		t.Fatalf("a failure was logged:\n%s", logs.String())
	}
}

// TC-009
func TestAmazonBrowserRetrieval_TC009_EveryAmazonCheckRecordsThePrice(t *testing.T) {
	newFixture := func(t *testing.T) *amazontest.Server {
		t.Helper()
		return amazontest.NewServer(t, &retrievalFixture{page: amazontest.ExpandedFixture(t, "consent-banner.html"), used: amazontest.Fixture(t, "aod-amazon-used.html"),
			pages: map[string]string{"B0BRETINST": amazontest.Fixture(t, "price-not-found.html"), "B0BRETUNAV": amazontest.Fixture(t, "unavailable.html")}})
	}

	t.Run("every entry point", func(t *testing.T) {
		fixture := newFixture(t)
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		clock := &humanClock{now: humanTime(14, 0)}
		logs := captureRetrievalLog(t)
		server, items := openBrowserProfileAPI(t, dir, database, clock, retrievalCollector(t, fixture.URL))

		added := addItem(t, server, "https://www.amazon.fr/dp/B0BRET0091")
		waitBrowserCheck(t, server, added.ID, humanTime(14, 0))
		listed := listItems(t, server)[added.ID]
		assertRetrievalListed(t, listed, humanTime(14, 0))
		assertNoRetrievalFailure(t, database, logs)

		if listed.NextCheckAt == nil {
			t.Fatal("no scheduled check after the first check")
		}
		scheduled := listed.NextCheckAt.Add(time.Second)
		clock.Set(scheduled)
		runScheduler(t, server, items, added.ID, scheduled)
		assertRetrievalListed(t, listItems(t, server)[added.ID], scheduled)
		assertNoRetrievalFailure(t, database, logs)

		all := scheduled.Add(10 * time.Minute)
		clock.Set(all)
		if recorder := callPath(server, http.MethodPost, "/api/v1/items/refresh", ""); recorder.Code != http.StatusAccepted {
			t.Fatalf("Refresh all prices: %d %s", recorder.Code, recorder.Body.String())
		}
		waitBrowserCheck(t, server, added.ID, all)
		assertRetrievalListed(t, listItems(t, server)[added.ID], all)
		assertNoRetrievalFailure(t, database, logs)

		for _, at := range []time.Time{all.Add(10 * time.Minute), all.Add(20 * time.Minute)} {
			clock.Set(at)
			refresh(t, server, added.ID)
			waitBrowserCheck(t, server, added.ID, at)
			assertRetrievalListed(t, listItems(t, server)[added.ID], at)
			assertNoRetrievalFailure(t, database, logs)
		}
	})

	t.Run("instalment-only page and unavailable listing", func(t *testing.T) {
		fixture := newFixture(t)
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		insertHumanItem(t, database, "B0BRETINST", "amazon", humanTime(20, 0))
		insertHumanItem(t, database, "B0BRETUNAV", "amazon", humanTime(20, 0))
		clock := &humanClock{now: humanTime(14, 0)}
		server, _ := openBrowserProfileAPI(t, dir, database, clock, retrievalCollector(t, fixture.URL))
		refresh(t, server, "B0BRETINST")
		if item := waitBrowserCheck(t, server, "B0BRETINST", humanTime(14, 0)); item.LastAttempt.Result != "price_not_found" {
			t.Fatalf("instalment-only page: status %s, attempt %+v", item.Status, item.LastAttempt)
		}
		clock.Set(humanTime(14, 5))
		refresh(t, server, "B0BRETUNAV")
		if item := waitBrowserCheck(t, server, "B0BRETUNAV", humanTime(14, 5)); item.Status != "unavailable" || item.LastAttempt.Result != "unavailable" {
			t.Fatalf("unavailable listing: status %s, attempt %+v", item.Status, item.LastAttempt)
		}
		if got := storedFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d, want 0", got)
		}
	})

	t.Run("protected mode with two owners", func(t *testing.T) {
		fixture := newFixture(t)
		dir := t.TempDir()
		database := openHumanStore(t, dir)
		clock := &humanClock{now: humanTime(14, 0)}
		logs := captureRetrievalLog(t)
		server, items := openBrowserProfileAPI(t, dir, database, clock, retrievalCollector(t, fixture.URL))
		ctx := context.Background()
		adminPassword, _, err := items.ResetAdminPassword(ctx)
		if err != nil {
			t.Fatal(err)
		}
		admin := login(t, server, "admin", adminPassword)
		owners := map[string]string{"alice": "B0BRETALI1", "bob": "B0BRETBOB1"}
		tokens := map[string]string{}
		for name, id := range owners {
			if recorder := request(t, server, http.MethodPost, "/api/v1/admin/users", `{"username":"`+name+`","password":"long enough pw"}`, admin); recorder.Code != http.StatusCreated {
				t.Fatalf("create %s: %s", name, recorder.Body.String())
			}
			user, err := database.UserByUsername(ctx, name)
			if err != nil {
				t.Fatal(err)
			}
			listing := model.Listing{ID: id, Platform: "amazon", ListingID: id, ASIN: id, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + id, OwnerID: user.ID}
			if err := database.Insert(ctx, listing, listing.URL, humanTime(20, 0)); err != nil {
				t.Fatal(err)
			}
			tokens[name] = login(t, server, name, "long enough pw")
		}
		index := 0
		for name, id := range owners {
			at := humanTime(14, 10+index)
			index++
			clock.Set(at)
			if recorder := request(t, server, http.MethodPost, "/api/v1/items/"+id+"/refresh", "", tokens[name]); recorder.Code != http.StatusAccepted {
				t.Fatalf("Refresh price of %s by %s: %d %s", id, name, recorder.Code, recorder.Body.String())
			}
			waitStoredCheck(t, database, id, at)
			var body struct {
				Items []model.Item `json:"items"`
			}
			recorder := request(t, server, http.MethodGet, "/api/v1/items", "", tokens[name])
			if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &body) != nil || len(body.Items) != 1 || body.Items[0].ID != id {
				t.Fatalf("items of %s: %d %s", name, recorder.Code, recorder.Body.String())
			}
			assertRetrievalListed(t, body.Items[0], at)
		}
		assertNoRetrievalFailure(t, database, logs)
	})
}

// TC-010
func TestAmazonBrowserRetrieval_TC010_RefreshPriceRestartsAnItemStoppedByRetrievalErrors(t *testing.T) {
	robot, mention := amazontest.Fixture(t, "robot-check-en.html"), amazontest.ExpandedFixture(t, "mentions-captcha.html")
	pages := &retrievalFixture{page: robot, used: amazontest.Fixture(t, "aod-none.html")}
	fixture := amazontest.NewServer(t, pages)
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	ids := []string{"B0BRET0101", "B0BRET0102", "B0BRET0103"}
	for _, id := range ids {
		insertCheckedAmazonItem(t, database, id, humanTime(7, 0))
	}
	clock := &humanClock{now: humanTime(6, 0)}
	logs := captureRetrievalLog(t)
	server, items := openBrowserProfileAPI(t, dir, database, clock, retrievalCollector(t, fixture.URL))

	// The issue #204 situation: five robot-check failures stop every Amazon item.
	for index := range 5 {
		at := humanTime(6, index+1)
		clock.Set(at)
		refresh(t, server, ids[index%len(ids)])
		waitBrowserCheck(t, server, ids[index%len(ids)], at)
	}
	for _, id := range ids {
		if item := listItems(t, server)[id]; item.AmazonStoppedAt == nil || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
			t.Fatalf("precondition: item %s after five robot checks: stopped %v, latest price %+v", id, item.AmazonStoppedAt, item.LatestPrice)
		}
	}

	t.Run("Refresh price of a stopped item on a robot page", func(t *testing.T) {
		at := humanTime(6, 30)
		clock.Set(at)
		refresh(t, server, ids[1])
		item := waitBrowserCheck(t, server, ids[1], at)
		if item.AmazonStoppedAt == nil || item.LastAttempt.Result != "request_error" || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
			t.Fatalf("item %s: stopped %v, attempt %+v, latest price %+v", ids[1], item.AmazonStoppedAt, item.LastAttempt, item.LatestPrice)
		}
		if got := storedFailures(t, database); got != 1 {
			t.Fatalf("consecutive failure count %d, want 1", got)
		}
		cause := ""
		for _, line := range strings.Split(logs.String(), "\n") {
			if strings.Contains(line, "Amazon request failed url=https://www.amazon.fr/dp/"+ids[1]+" ") && strings.Contains(line, "at="+at.UTC().Format(time.RFC3339)) {
				if _, after, found := strings.Cut(line, " cause="); found {
					cause, _, _ = strings.Cut(after, " response=")
				}
			}
		}
		if !strings.Contains(strings.ToLower(cause), "robot check") {
			t.Fatalf("the failure is not logged with a robot-check cause: %q", cause)
		}
	})

	pages.setPage(mention)
	fixture.Reset()
	clock.Set(humanTime(7, 30)) // the stopped items are due
	scheduler, stopScheduler := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() { items.RunScheduler(scheduler); close(finished) }()
	time.Sleep(time.Second)
	stopScheduler()
	<-finished
	for _, request := range fixture.Requests() {
		if strings.HasPrefix(request.Path, "/dp/") {
			t.Fatalf("the scheduler sent a request for a stopped item: %s", request.Path)
		}
	}

	if recorder := callPath(server, http.MethodPost, "/api/v1/items/"+ids[0]+"/refresh", ""); recorder.Code != http.StatusAccepted {
		t.Fatalf("Refresh price of the stopped item: %d %s", recorder.Code, recorder.Body.String())
	}
	waitBrowserCheck(t, server, ids[0], humanTime(7, 30))
	listed := listItems(t, server)
	assertBrowserPriceListed(t, listed[ids[0]], humanTime(7, 30))
	if body := callPath(server, http.MethodGet, "/api/v1/items", "").Body.String(); !strings.Contains(body, `"id":"`+ids[0]+`"`) || strings.Count(body, `"amazonStoppedAt":null`) != 1 {
		t.Fatalf("GET /api/v1/items does not show only the restarted item with amazonStoppedAt null: %s", body)
	}
	for _, id := range ids[1:] {
		if listed[id].AmazonStoppedAt == nil {
			t.Fatalf("item %s was restarted by the Refresh price of %s", id, ids[0])
		}
	}
	if got := storedFailures(t, database); got != 0 {
		t.Fatalf("consecutive failure count %d after the check, want 0", got)
	}

	t.Run("Refresh price repeated on the restarted item", func(t *testing.T) {
		clock.Set(humanTime(7, 40))
		refresh(t, server, ids[0])
		waitBrowserCheck(t, server, ids[0], humanTime(7, 40))
		assertBrowserPriceListed(t, listItems(t, server)[ids[0]], humanTime(7, 40))
	})

	t.Run("the restarted item gets its next scheduled check", func(t *testing.T) {
		next := listItems(t, server)[ids[0]].NextCheckAt
		if next == nil || !next.After(humanTime(7, 40)) {
			t.Fatalf("next check of the restarted item: %v", next)
		}
		at := next.Add(time.Second)
		clock.Set(at)
		runScheduler(t, server, items, ids[0], at)
		assertBrowserPriceListed(t, listItems(t, server)[ids[0]], at)
		for _, id := range ids[1:] {
			if item := listItems(t, server)[id]; item.AmazonStoppedAt == nil || item.LastAttempt.Timestamp.Equal(at) {
				t.Fatalf("stopped item %s was checked by the scheduler: stopped %v, attempt %+v", id, item.AmazonStoppedAt, item.LastAttempt)
			}
		}
	})
}
