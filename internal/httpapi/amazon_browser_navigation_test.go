package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

// Amazon success responses the browser does not open as a page (issue #186) through the HTTP API:
// a real browser collector checks a local fixture server whose product page answer is switched per check.

// Answers of the navigation fixture server for /dp/<ASIN>.
const (
	answerPage503    int32 = iota // HTTP 503 with an error page
	answerAttachment              // HTTP 200 text/html offered as a download
	answerEmpty503                // HTTP 503 with an empty body
	answerEmpty404                // HTTP 404 with an empty body
	answerEmpty410                // HTTP 410 with an empty body
)

// navigationFixture is a fixture server whose /dp/ answer is the value of answer.
type navigationFixture struct {
	*amazontest.Server
	answer atomic.Int32
}

func newNavigationFixture(t *testing.T) *navigationFixture {
	t.Helper()
	fixture := &navigationFixture{}
	page, noOffer := amazontest.Fixture(t, "price-found.html"), amazontest.Fixture(t, "aod-none.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/dp/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch fixture.answer.Load() {
		case answerPage503:
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, "<html><body><h1>Service Unavailable fixture</h1></body></html>")
		case answerAttachment:
			w.Header().Set("Content-Disposition", `attachment; filename="page.html"`)
			io.WriteString(w, page)
		case answerEmpty503:
			w.WriteHeader(http.StatusServiceUnavailable)
		case answerEmpty404:
			w.WriteHeader(http.StatusNotFound)
		case answerEmpty410:
			w.WriteHeader(http.StatusGone)
		}
	})
	usedOffers := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, noOffer)
	}
	mux.HandleFunc("/gp/aod/ajax", usedOffers)
	mux.HandleFunc("/gp/aod/ajax/", usedOffers)
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	fixture.Server = amazontest.NewServer(t, mux)
	return fixture
}

type navigationAPI struct {
	server   *Server
	items    *service.Service
	database *store.Store
	dir      string
	clock    *humanClock
	fixture  *navigationFixture
}

// openNavigationAPI serves the API of a service checking Amazon with a real browser collector on the
// fixture server, with one Amazon item per id (owned by owners[i], or the open-mode owner) whose last
// check found 19.99 and a second-hand offer at 89.50.
func openNavigationAPI(t *testing.T, ids []string, owners ...int64) navigationAPI {
	t.Helper()
	fixture := newNavigationFixture(t)
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	for index, id := range ids {
		var owner int64
		if index < len(owners) {
			owner = owners[index]
		}
		listing := model.Listing{ID: id, Platform: "amazon", ListingID: id, ASIN: id, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + id, OwnerID: owner}
		if err := database.Insert(context.Background(), listing, listing.URL, humanTime(20, 0)); err != nil {
			t.Fatal(err)
		}
		if err := database.RecordCollection(context.Background(), id, model.CollectionResult{Result: "success", AmountCents: 1999, SecondHandStatus: "available",
			SecondHandAmountCents: 8950, SecondHandCondition: "very_good", SecondHandConditionLabel: "Très bon état"}, humanTime(8, 0).AddDate(0, 0, -1)); err != nil {
			t.Fatal(err)
		}
	}
	clock := &humanClock{now: humanTime(14, 0)}
	collector := amazon.NewBrowserCollector(browserProfileOptions(t, filepath.Join(dir, "amazon-browser"), fixture.URL))
	server, items := openBrowserProfileAPI(t, dir, database, clock, collector)
	return navigationAPI{server: server, items: items, database: database, dir: dir, clock: clock, fixture: fixture}
}

// check runs the scheduled check of id at the time at with the fixture answering answer.
func (api navigationAPI) check(id string, at time.Time, answer int32) {
	api.fixture.answer.Store(answer)
	api.clock.Set(at)
	api.items.Collect(context.Background(), id)
}

// captureCheckOutput collects what the code under test logs or prints until the returned function is called.
func captureCheckOutput(t *testing.T) func() string {
	t.Helper()
	var logs syncBuffer
	previousLog := log.Writer()
	log.SetOutput(&logs)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = writer
	var printed bytes.Buffer
	copied := make(chan struct{})
	go func() { io.Copy(&printed, reader); close(copied) }()
	var once sync.Once
	stop := func() string {
		once.Do(func() {
			os.Stdout = previousStdout
			writer.Close()
			<-copied
			reader.Close()
			log.SetOutput(previousLog)
		})
		return logs.String() + printed.String()
	}
	t.Cleanup(func() { stop() })
	return stop
}

// linesWith returns the lines of output containing text.
func linesWith(output, text string) []string {
	var found []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, text) {
			found = append(found, line)
		}
	}
	return found
}

// assertBrowserErrorLogged checks that the browser's error and the request URL of id are logged on one line.
func assertBrowserErrorLogged(t *testing.T, output, id string) {
	t.Helper()
	for _, line := range linesWith(output, "Amazon page could not be opened: net::") {
		if strings.Contains(line, "https://www.amazon.fr/dp/"+id) {
			return
		}
	}
	t.Fatalf("no log line with the request URL of %s and the browser's error:\n%s", id, output)
}

// assertPriceAndOfferKept checks that every item listed keeps its price of 19.99 and its second-hand
// offer at 89.50, and is stopped (at stoppedAt) or not (zero stoppedAt).
func assertPriceAndOfferKept(t *testing.T, api navigationAPI, stoppedAt time.Time, step string) {
	t.Helper()
	for id, item := range listItems(t, api.server) {
		if item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
			t.Fatalf("%s, item %s: latest price %+v, want 19.99", step, id, item.LatestPrice)
		}
		if offer := item.SecondHandOffer; offer == nil || offer.LatestDetection == nil || offer.LatestDetection.AmountCents != 8950 {
			t.Fatalf("%s, item %s: second-hand offer %+v, want the offer at 89.50 kept", step, id, item.SecondHandOffer)
		}
		if stoppedAt.IsZero() && item.AmazonStoppedAt != nil {
			t.Fatalf("%s, item %s is stopped at %s", step, id, item.AmazonStoppedAt)
		}
		if !stoppedAt.IsZero() && !equalTime(item.AmazonStoppedAt, stoppedAt) {
			t.Fatalf("%s, item %s: stopped %v, want stopped at %s", step, id, item.AmazonStoppedAt, stoppedAt)
		}
	}
}

// assertRetrievalIssue checks that item shows the failed check of at: retrieval issue, second-hand
// check error, price and second-hand offer kept.
func assertRetrievalIssue(t *testing.T, item model.Item, at time.Time) {
	t.Helper()
	if item.Status != "retrieval_error" || item.LastAttempt == nil || item.LastAttempt.Result != "request_error" || !item.LastAttempt.Timestamp.Equal(at) {
		t.Fatalf("item %s: status %s, last attempt %+v, want a request_error at %s", item.ID, item.Status, item.LastAttempt, at)
	}
	if offer := item.SecondHandOffer; offer == nil || offer.Status != "check_error" || offer.LatestDetection == nil || offer.LatestDetection.AmountCents != 8950 {
		t.Fatalf("item %s: second-hand offer %+v, want check_error with the offer at 89.50 kept", item.ID, item.SecondHandOffer)
	}
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 1999 {
		t.Fatalf("item %s: latest price %+v, want 19.99", item.ID, item.LatestPrice)
	}
}

// storedResults counts the stored collection attempts with result.
func (api navigationAPI) storedResults(t *testing.T, result string) int {
	t.Helper()
	raw, err := sql.Open("sqlite", filepath.Join(api.dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var count int
	if err := raw.QueryRow("SELECT COUNT(*) FROM collection_attempts WHERE result = ?", result).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

// A download response is counted and does not reset the consecutive failure count.
// TC-004
func TestAmazonBrowserNavigation_TC004_DownloadIsACountedFailure(t *testing.T) {
	ids := []string{"B0NAVI0041", "B0NAVI0042"}

	t.Run("download in the middle of five failures", func(t *testing.T) {
		api := openNavigationAPI(t, ids)
		output := captureCheckOutput(t)
		answers := []int32{answerPage503, answerPage503, answerAttachment, answerPage503, answerPage503}
		for index, answer := range answers {
			id, at := ids[index%2], humanTime(14, index+1)
			api.check(id, at, answer)
			if answer == answerAttachment {
				assertRetrievalIssue(t, itemDetails(t, api.server, id), at)
				if got := storedFailures(t, api.database); got != 3 {
					t.Fatalf("consecutive failure count %d after the download, want 3", got)
				}
			}
			if index < len(answers)-1 {
				assertPriceAndOfferKept(t, api, time.Time{}, fmt.Sprintf("after check %d", index+1))
			}
		}
		assertPriceAndOfferKept(t, api, humanTime(14, 5), "after the 5th failure")
		if got := api.storedResults(t, "price_not_found"); got != 0 {
			t.Fatalf("%d checks stored as price not found", got)
		}
		logged := output()
		failures := linesWith(logged, "Amazon request failed")
		if len(failures) != 5 {
			t.Fatalf("%d failures logged, want 5:\n%s", len(failures), logged)
		}
		if !strings.Contains(failures[2], "url=https://www.amazon.fr/dp/"+ids[0]) || !strings.Contains(failures[2], "status=0") {
			t.Fatalf("the failure of the download is not logged with its request URL: %s", failures[2])
		}
		assertBrowserErrorLogged(t, logged, ids[0])
	})

	t.Run("download as the 5th failure", func(t *testing.T) {
		api := openNavigationAPI(t, ids)
		for index := range 4 {
			api.check(ids[index%2], humanTime(14, index+1), answerPage503)
		}
		assertPriceAndOfferKept(t, api, time.Time{}, "after 4 failures")
		api.check(ids[0], humanTime(14, 5), answerAttachment)
		assertPriceAndOfferKept(t, api, humanTime(14, 5), "after the download as the 5th failure")
	})

	t.Run("protected mode with two owners", func(t *testing.T) {
		api := openNavigationAPI(t, ids, 1, 2)
		answers := []int32{answerPage503, answerPage503, answerAttachment, answerPage503, answerPage503}
		for index, answer := range answers {
			api.check(ids[index%2], humanTime(14, index+1), answer)
			for _, id := range ids {
				stopped, err := api.database.AmazonStopped(context.Background(), id)
				if err != nil {
					t.Fatal(err)
				}
				if want := index == len(answers)-1; stopped != want {
					t.Fatalf("after check %d, item %s stopped %v, want %v", index+1, id, stopped, want)
				}
			}
		}
	})
}

// Refresh price and Refresh all prices answered with a download show a failed refresh.
// TC-005
func TestAmazonBrowserNavigation_TC005_RefreshAnsweredWithADownload(t *testing.T) {
	const id = "B0NAVI0051"

	t.Run("Refresh price, Refresh all prices, Refresh price again", func(t *testing.T) {
		api := openNavigationAPI(t, []string{id})
		api.fixture.answer.Store(answerAttachment)

		refresh(t, api.server, id)
		assertRetrievalIssue(t, waitBrowserCheck(t, api.server, id, humanTime(14, 0)), humanTime(14, 0))
		assertRetrievalIssue(t, listItems(t, api.server)[id], humanTime(14, 0))
		if got := storedFailures(t, api.database); got != 1 {
			t.Fatalf("consecutive failure count %d after Refresh price, want 1", got)
		}

		api.clock.Set(humanTime(14, 5))
		if recorder := callPath(api.server, http.MethodPost, "/api/v1/items/refresh", ""); recorder.Code != http.StatusAccepted {
			t.Fatalf("Refresh all prices: %d %s", recorder.Code, recorder.Body.String())
		}
		assertRetrievalIssue(t, waitBrowserCheck(t, api.server, id, humanTime(14, 5)), humanTime(14, 5))
		assertRetrievalIssue(t, listItems(t, api.server)[id], humanTime(14, 5))
		if got := storedFailures(t, api.database); got != 2 {
			t.Fatalf("consecutive failure count %d after Refresh all prices, want 2", got)
		}

		api.clock.Set(humanTime(14, 10))
		refresh(t, api.server, id)
		assertRetrievalIssue(t, waitBrowserCheck(t, api.server, id, humanTime(14, 10)), humanTime(14, 10))
		if got := storedFailures(t, api.database); got != 3 {
			t.Fatalf("consecutive failure count %d after a repeated Refresh price, want 3", got)
		}
		if got := api.storedResults(t, "price_not_found"); got != 0 {
			t.Fatalf("%d checks stored as price not found", got)
		}
	})

	t.Run("Refresh price of a stopped item", func(t *testing.T) {
		api := openNavigationAPI(t, []string{id})
		if err := api.database.StopAmazonItems(context.Background(), humanTime(13, 0)); err != nil {
			t.Fatal(err)
		}
		api.fixture.answer.Store(answerAttachment)
		refresh(t, api.server, id)
		item := waitBrowserCheck(t, api.server, id, humanTime(14, 0))
		assertRetrievalIssue(t, item, humanTime(14, 0))
		if !equalTime(item.AmazonStoppedAt, humanTime(13, 0)) {
			t.Fatalf("stopped %v after a failed Refresh price, want still stopped at %s", item.AmazonStoppedAt, humanTime(13, 0))
		}
	})
}

// A non-success response with an empty body keeps its effect on the failure count.
// TC-006
func TestAmazonBrowserNavigation_TC006_EmptyNonSuccessKeepsItsEffect(t *testing.T) {
	ids := []string{"B0NAVI0061", "B0NAVI0062"}
	for _, reset := range []struct {
		name   string
		answer int32
	}{{"HTTP 404", answerEmpty404}, {"HTTP 410", answerEmpty410}} {
		t.Run(reset.name, func(t *testing.T) {
			api := openNavigationAPI(t, ids)
			output := captureCheckOutput(t)
			for index := range 4 {
				api.check(ids[index%2], humanTime(14, index+1), answerPage503)
			}
			if got := storedFailures(t, api.database); got != 4 {
				t.Fatalf("consecutive failure count %d after 4 failures, want 4", got)
			}

			api.check(ids[0], humanTime(14, 10), reset.answer)
			if item := itemDetails(t, api.server, ids[0]); item.Status != "unavailable" || item.LastAttempt == nil || item.LastAttempt.Result != "unavailable" {
				t.Fatalf("%s with an empty body: status %s, last attempt %+v, want the listing unavailable", reset.name, item.Status, item.LastAttempt)
			}
			if got := storedFailures(t, api.database); got != 0 {
				t.Fatalf("consecutive failure count %d after %s, want 0", got, reset.name)
			}

			for index := range 4 {
				api.check(ids[1], humanTime(14, 11+index), answerEmpty503)
			}
			for id, item := range listItems(t, api.server) {
				if item.AmazonStoppedAt != nil {
					t.Fatalf("item %s stopped after 4 failures following %s", id, reset.name)
				}
			}
			api.check(ids[1], humanTime(14, 20), answerEmpty503)
			for id, item := range listItems(t, api.server) {
				if !equalTime(item.AmazonStoppedAt, humanTime(14, 20)) {
					t.Fatalf("item %s: stopped %v after the 5th failure following %s, want stopped at %s", id, item.AmazonStoppedAt, reset.name, humanTime(14, 20))
				}
			}

			failures := linesWith(output(), "Amazon request failed")
			if len(failures) != 9 {
				t.Fatalf("%d failures logged, want 9: %v", len(failures), failures)
			}
			for _, line := range failures {
				if !strings.Contains(line, "status=503") {
					t.Fatalf("failure not logged with status 503: %s", line)
				}
			}
		})
	}
}
