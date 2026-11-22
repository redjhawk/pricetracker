package httpapi

import (
	"context"
	"io"
	mathrand "math/rand/v2"
	"net/http"
	"os"
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

// Amazon browser profile recovery of issue #180 through the HTTP API: the data directory holds an
// amazon-browser profile left with the lock of another host, and the service uses a real browser
// collector on it, as in production.

// browserProfileServer serves the price-found fixture page at /dp/<ASIN> for every ASIN, setting the
// cookie fixture, with the used-offer responses and the page resources.
func browserProfileServer(t *testing.T) *amazontest.Server {
	t.Helper()
	page, noOffer := amazontest.Fixture(t, "price-found.html"), amazontest.Fixture(t, "aod-none.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/dp/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session-id", Value: "fixture-session-180", Path: "/", MaxAge: 3600})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	})
	usedOffers := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, noOffer)
	}
	mux.HandleFunc("/gp/aod/ajax", usedOffers)
	mux.HandleFunc("/gp/aod/ajax/", usedOffers)
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return amazontest.NewServer(t, mux)
}

func browserProfileOptions(t *testing.T, profile, origin string) amazon.BrowserOptions {
	t.Helper()
	return amazon.BrowserOptions{ExecPath: amazontest.Browser(t), ProfileDir: profile, NoSandbox: true, NavigationTimeout: 30 * time.Second,
		Pause: func() time.Duration { return 0 }, Origin: origin}
}

// leaveOtherHostLock leaves in profile the lock marks of a browser of another host.
func leaveOtherHostLock(t *testing.T, profile string) {
	t.Helper()
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{"SingletonLock": "otherhost-12345", "SingletonSocket": "/tmp/org.chromium.Chromium.gone/SingletonSocket", "SingletonCookie": "12345"}
	for name, target := range links {
		if err := os.RemoveAll(filepath.Join(profile, name)); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(profile, name)); err != nil {
			t.Fatal(err)
		}
	}
}

// openBrowserProfileAPI serves the API of a service on dir using collector for Amazon and the fake clock.
func openBrowserProfileAPI(t *testing.T, dir string, database *store.Store, clock *humanClock, collector service.AmazonCollector) (*Server, *service.Service) {
	t.Helper()
	var mu sync.Mutex
	source := mathrand.New(mathrand.NewPCG(180, 181))
	random := func(n int64) int64 {
		mu.Lock()
		defer mu.Unlock()
		return source.Int64N(n)
	}
	items := service.NewWithOptions(humanConfig(dir), database, service.Options{Amazon: collector, Now: clock.Now, Random: random,
		SchedulePause: func(context.Context) bool { return true }})
	t.Cleanup(items.Close)
	return New(humanConfig(dir), items), items
}

// waitBrowserCheck waits until the check of id at the time at is finished and returns the item.
func waitBrowserCheck(t *testing.T, server *Server, id string, at time.Time) model.Item {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Minute); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if item := itemDetails(t, server, id); item.LastAttempt != nil && item.LastAttempt.Result != "pending" && item.LastAttempt.Timestamp.Equal(at) {
			return item
		}
	}
	t.Fatalf("the check of %s did not finish", id)
	return model.Item{}
}

func insertCheckedAmazonItem(t *testing.T, database *store.Store, id string, next time.Time) {
	t.Helper()
	insertHumanItem(t, database, id, "amazon", next)
	if err := database.RecordCollection(context.Background(), id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "not_found"},
		humanTime(8, 0).AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
}

func assertBrowserPriceListed(t *testing.T, item model.Item, at time.Time) {
	t.Helper()
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 || !equalTime(item.LatestPrice.Timestamp, at) {
		t.Fatalf("item %s: latest price %+v, want 129.99 checked at %s", item.ID, item.LatestPrice, at)
	}
	if item.LastAttempt == nil || item.LastAttempt.Result != "success" || !item.LastAttempt.Timestamp.Equal(at) {
		t.Fatalf("item %s: last attempt %+v, want a success at %s", item.ID, item.LastAttempt, at)
	}
	if item.AmazonStoppedAt != nil {
		t.Fatalf("item %s is stopped at %s", item.ID, item.AmazonStoppedAt)
	}
}

// TC-010
func TestAmazonBrowserProfile_TC010_DataDirectoryMovedToAnotherMachine(t *testing.T) {
	t.Run("Refresh price after the move", func(t *testing.T) {
		fixture := browserProfileServer(t)
		dir := t.TempDir()
		options := browserProfileOptions(t, filepath.Join(dir, "amazon-browser"), fixture.URL)
		const id = "B0PROF0101"
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		earlier := amazon.NewBrowserCollector(options).Collect(ctx, model.Listing{ID: id, Platform: "amazon", ListingID: id, ASIN: id, Marketplace: "amazon.fr",
			URL: "https://www.amazon.fr/dp/" + id})
		cancel()
		if earlier.Result != "success" {
			t.Fatalf("fixture check: the earlier check failed: %+v", earlier)
		}
		leaveOtherHostLock(t, options.ProfileDir)

		database := openHumanStore(t, dir)
		insertCheckedAmazonItem(t, database, id, humanTime(20, 0))
		if err := database.SaveAmazonFailures(context.Background(), 4); err != nil {
			t.Fatal(err)
		}
		clock := &humanClock{now: humanTime(14, 0)}
		server, _ := openBrowserProfileAPI(t, dir, database, clock, amazon.NewBrowserCollector(options))
		fixture.Reset()

		refresh(t, server, id)
		waitBrowserCheck(t, server, id, humanTime(14, 0))
		assertBrowserPriceListed(t, listItems(t, server)[id], humanTime(14, 0))
		if body := callPath(server, http.MethodGet, "/api/v1/items", "").Body.String(); !strings.Contains(body, `"amazonStoppedAt":null`) {
			t.Fatalf("GET /api/v1/items does not show amazonStoppedAt null: %s", body)
		}
		if got := storedFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d after the check, want 0", got)
		}
		cookieSent := false
		for _, request := range fixture.Requests() {
			if request.Path == "/dp/"+id && strings.Contains(request.Header.Get("Cookie"), "session-id=fixture-session-180") {
				cookieSent = true
			}
		}
		if !cookieSent {
			t.Fatalf("the page request does not carry the cookie of the earlier check: %+v", fixture.Requests())
		}
	})

	t.Run("items stopped before the start", func(t *testing.T) {
		fixture := browserProfileServer(t)
		dir := t.TempDir()
		options := browserProfileOptions(t, filepath.Join(dir, "amazon-browser"), fixture.URL)
		leaveOtherHostLock(t, options.ProfileDir)
		database := openHumanStore(t, dir)
		ids := []string{"B0PROF0102", "B0PROF0103", "B0PROF0104"}
		for _, id := range ids {
			insertCheckedAmazonItem(t, database, id, humanTime(7, 0))
		}
		if err := database.StopAmazonItems(context.Background(), humanTime(6, 0)); err != nil {
			t.Fatal(err)
		}
		if err := database.SaveAmazonFailures(context.Background(), 5); err != nil {
			t.Fatal(err)
		}
		clock := &humanClock{now: humanTime(7, 30)}
		server, items := openBrowserProfileAPI(t, dir, database, clock, amazon.NewBrowserCollector(options))
		for _, id := range ids {
			if item := listItems(t, server)[id]; item.AmazonStoppedAt == nil {
				t.Fatalf("item %s was restarted by the start", id)
			}
		}

		scheduler, stopScheduler := context.WithCancel(context.Background())
		finished := make(chan struct{})
		go func() { items.RunScheduler(scheduler); close(finished) }() // its first run is at once; the items are due
		time.Sleep(time.Second)
		stopScheduler()
		<-finished
		for _, request := range fixture.Requests() {
			if strings.HasPrefix(request.Path, "/dp/") {
				t.Fatalf("the scheduler sent a request for a stopped item: %s", request.Path)
			}
		}
		for _, id := range ids {
			if item := listItems(t, server)[id]; item.AmazonStoppedAt == nil {
				t.Fatalf("item %s was restarted by the scheduler", id)
			}
		}

		refresh(t, server, ids[0])
		waitBrowserCheck(t, server, ids[0], humanTime(7, 30))
		listed := listItems(t, server)
		assertBrowserPriceListed(t, listed[ids[0]], humanTime(7, 30))
		for _, id := range ids[1:] {
			if listed[id].AmazonStoppedAt == nil {
				t.Fatalf("item %s was restarted by the Refresh price of %s", id, ids[0])
			}
		}
		if got := storedFailures(t, database); got != 0 {
			t.Fatalf("consecutive failure count %d after the check, want 0", got)
		}
	})
}
