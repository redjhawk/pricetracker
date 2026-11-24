package httpapi

import (
	"database/sql"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

// Amazon thumbnails of issue #275 through the HTTP API: a real browser collector checks product
// pages served by a local fixture server, and only an https URL on an Amazon image host is stored.

const (
	priceFoundImage = "https://m.media-amazon.com/images/I/price-found.jpg"
	oldAmazonImage  = "https://m.media-amazon.com/images/I/old.jpg"
	oldForeignImage = "http://example.com/old.jpg"
	newAmazonImage  = "https://m.media-amazon.com/images/I/new.jpg"
)

var landingImageLine = regexp.MustCompile(`(?m)^<img id="landingImage"[^\n]*\n`)

// imagePage returns price-found.html with the data-old-hires value of landingImage set to image.
func imagePage(t *testing.T, image string) string {
	t.Helper()
	page, original := amazontest.Fixture(t, "price-found.html"), `data-old-hires="`+priceFoundImage+`"`
	if !strings.Contains(page, original) {
		t.Fatalf("price-found.html has no %s", original)
	}
	return strings.Replace(page, original, `data-old-hires="`+html.EscapeString(image)+`"`, 1)
}

// imagelessPage returns price-found.html without its landingImage tag.
func imagelessPage(t *testing.T) string {
	t.Helper()
	page := amazontest.Fixture(t, "price-found.html")
	if !landingImageLine.MatchString(page) {
		t.Fatal("price-found.html has no landingImage line")
	}
	return landingImageLine.ReplaceAllString(page, "")
}

// withOGImage adds an og:image meta with content to the <head> of page.
func withOGImage(t *testing.T, page, content string) string {
	t.Helper()
	if !strings.Contains(page, "</head>") {
		t.Fatal("the page has no </head>")
	}
	return strings.Replace(page, "</head>", `<meta property="og:image" content="`+html.EscapeString(content)+`">`+"\n</head>", 1)
}

// failingFixture answers HTTP 503 for the product pages of the ASINs in failing and delegates to
// the retrieval fixture otherwise.
type failingFixture struct {
	*retrievalFixture
	mu      sync.Mutex
	failing map[string]bool
}

func (f *failingFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	failing := f.failing[strings.TrimPrefix(r.URL.Path, "/dp/")]
	f.mu.Unlock()
	if strings.HasPrefix(r.URL.Path, "/dp/") && failing {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, "<html><body><h1>Service Unavailable fixture</h1></body></html>")
		return
	}
	f.retrievalFixture.ServeHTTP(w, r)
}

func (f *failingFixture) setFailing(asin string, failing bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failing[asin] = failing
}

type thumbnailAPI struct {
	fixture  *failingFixture
	server   *Server
	items    *service.Service
	database *store.Store
	dir      string
	clock    *humanClock
	logs     *syncBuffer
	at       time.Time // time of the latest check
}

// openThumbnailAPI serves the API in open mode with a browser collector checking pages (by ASIN;
// price-found.html otherwise) and aod-none.html as used-offer answer.
func openThumbnailAPI(t *testing.T, pages map[string]string) *thumbnailAPI {
	t.Helper()
	fixture := &failingFixture{failing: map[string]bool{}, retrievalFixture: &retrievalFixture{page: amazontest.Fixture(t, "price-found.html"),
		used: amazontest.Fixture(t, "aod-none.html"), pages: pages}}
	served := amazontest.NewServer(t, fixture)
	dir := t.TempDir()
	database := openHumanStore(t, dir)
	clock := &humanClock{now: humanTime(14, 0)}
	logs := captureRetrievalLog(t)
	server, items := openBrowserProfileAPI(t, dir, database, clock, retrievalCollector(t, served.URL))
	return &thumbnailAPI{fixture: fixture, server: server, items: items, database: database, dir: dir, clock: clock, logs: logs, at: humanTime(14, 0)}
}

// next moves the clock one minute forward and returns the new time.
func (api *thumbnailAPI) next() time.Time {
	api.at = api.at.Add(time.Minute)
	api.clock.Set(api.at)
	return api.at
}

// add adds the Amazon listing of asin and returns the item once its first check is recorded.
func (api *thumbnailAPI) add(t *testing.T, asin string) model.Item {
	t.Helper()
	at := api.next()
	added := addItem(t, api.server, "https://www.amazon.fr/dp/"+asin)
	return waitBrowserCheck(t, api.server, added.ID, at)
}

// refresh runs Refresh price on id and returns the item once the check is recorded.
func (api *thumbnailAPI) refresh(t *testing.T, id string) model.Item {
	t.Helper()
	at := api.next()
	refresh(t, api.server, id)
	return waitBrowserCheck(t, api.server, id, at)
}

func (api *thumbnailAPI) setPage(asin, page string) {
	api.fixture.retrievalFixture.mu.Lock()
	defer api.fixture.retrievalFixture.mu.Unlock()
	api.fixture.retrievalFixture.pages[asin] = page
}

// storeThumbnail writes value as the stored thumbnail of id, as a check before issue #275 could.
func (api *thumbnailAPI) storeThumbnail(t *testing.T, id, value string) {
	t.Helper()
	raw, err := sql.Open("sqlite", filepath.Join(api.dir, "pricefollower.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if result, err := raw.Exec("UPDATE items SET thumbnail_url = ? WHERE id = ?", value, id); err != nil {
		t.Fatal(err)
	} else if count, err := result.RowsAffected(); err != nil || count != 1 {
		t.Fatalf("store the thumbnail of %s: %d rows, %v", id, count, err)
	}
}

func showThumbnail(value *string) string {
	if value == nil {
		return "null"
	}
	return `"` + *value + `"`
}

// assertThumbnail checks the thumbnailUrl of id in GET /api/v1/items and GET /api/v1/items/{id}.
func assertThumbnail(t *testing.T, server *Server, id string, want *string) {
	t.Helper()
	for source, item := range map[string]model.Item{"GET /api/v1/items": listItems(t, server)[id], "GET /api/v1/items/" + id: itemDetails(t, server, id)} {
		if item.ID != id {
			t.Fatalf("%s: item %s not found", source, id)
		}
		if (item.ThumbnailURL == nil) != (want == nil) || (want != nil && *item.ThumbnailURL != *want) {
			t.Fatalf("%s: thumbnailUrl %s, want %s", source, showThumbnail(item.ThumbnailURL), showThumbnail(want))
		}
	}
}

// assertSucceededAt checks that item has the price-found.html price, title and offer, checked at the time at.
func assertSucceededAt(t *testing.T, item model.Item, at time.Time) {
	t.Helper()
	if item.LatestPrice == nil || item.LatestPrice.AmountCents != 12999 || !equalTime(item.LatestPrice.Timestamp, at) {
		t.Fatalf("item %s: latest price %+v, want 129.99 checked at %s", item.ID, item.LatestPrice, at)
	}
	if item.Title == nil || *item.Title != "Price found product" {
		t.Fatalf("item %s: title %v", item.ID, item.Title)
	}
	if item.Status != "active" || item.LastAttempt == nil || item.LastAttempt.Result != "success" || !item.LastAttempt.Timestamp.Equal(at) || item.LastAttempt.Message != nil {
		t.Fatalf("item %s: status %s, last attempt %+v", item.ID, item.Status, item.LastAttempt)
	}
}

// TC-006
func TestAmazonThumbnailURLs_TC006_AddingAnItemWithAnAcceptedImageStoresIt(t *testing.T) {
	euImage := "https://images-eu.ssl-images-amazon.com/images/I/x.jpg"
	api := openThumbnailAPI(t, map[string]string{"B0THUMB061": amazontest.Fixture(t, "price-found.html"), "B0THUMB062": imagePage(t, euImage)})
	for _, row := range []struct{ asin, want string }{{"B0THUMB061", priceFoundImage}, {"B0THUMB062", euImage}} {
		item := api.add(t, row.asin)
		listed := listItems(t, api.server)[item.ID]
		assertSucceededAt(t, listed, api.at)
		assertThumbnail(t, api.server, item.ID, &row.want)
	}
	assertNoRetrievalFailure(t, api.database, api.logs)
}

// TC-007
func TestAmazonThumbnailURLs_TC007_AddingAnItemWithOnlyRejectedImagesStoresNone(t *testing.T) {
	api := openThumbnailAPI(t, map[string]string{
		"B0THUMB070": amazontest.Fixture(t, "price-found.html"),
		"B0THUMB071": imagePage(t, "https://example.com/a.jpg"),
		"B0THUMB072": withOGImage(t, imagePage(t, "http://m.media-amazon.com/images/I/a.jpg"), "data:image/png;base64,AAAA"),
		"B0THUMB073": imagelessPage(t),
	})

	control := api.add(t, "B0THUMB070")
	controlListed := listItems(t, api.server)[control.ID]
	assertSucceededAt(t, controlListed, api.at)
	if controlListed.SecondHandOffer == nil || controlListed.SecondHandOffer.Status != "not_found" || controlListed.NextCheckAt == nil {
		t.Fatalf("precondition: unchanged page: second-hand offer %+v, next check %v", controlListed.SecondHandOffer, controlListed.NextCheckAt)
	}

	for _, asin := range []string{"B0THUMB071", "B0THUMB072", "B0THUMB073"} {
		t.Run(asin, func(t *testing.T) {
			item := api.add(t, asin)
			at := api.at
			assertThumbnail(t, api.server, item.ID, nil)
			details := callPath(api.server, http.MethodGet, "/api/v1/items/"+item.ID, "")
			if details.Code != http.StatusOK || !strings.Contains(details.Body.String(), `"thumbnailUrl":null`) {
				t.Fatalf("GET /api/v1/items/%s: %d %s, want \"thumbnailUrl\":null", item.ID, details.Code, details.Body.String())
			}
			var list struct {
				Items []map[string]json.RawMessage `json:"items"`
			}
			recorder := callPath(api.server, http.MethodGet, "/api/v1/items", "")
			if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &list) != nil {
				t.Fatalf("GET /api/v1/items: %d %s", recorder.Code, recorder.Body.String())
			}
			found := false
			for _, entry := range list.Items {
				if string(entry["id"]) == `"`+item.ID+`"` {
					found = true
					if raw, ok := entry["thumbnailUrl"]; !ok || string(raw) != "null" {
						t.Fatalf("GET /api/v1/items: item %s thumbnailUrl %s, want null", item.ID, raw)
					}
				}
			}
			if !found {
				t.Fatalf("GET /api/v1/items: item %s not listed", item.ID)
			}

			listed := listItems(t, api.server)[item.ID]
			assertSucceededAt(t, listed, at)
			if listed.SecondHandOffer == nil || listed.SecondHandOffer.Status != controlListed.SecondHandOffer.Status {
				t.Fatalf("item %s: second-hand offer %+v, want status %s", item.ID, listed.SecondHandOffer, controlListed.SecondHandOffer.Status)
			}
			if listed.NextCheckAt == nil {
				t.Fatalf("item %s: no next check", item.ID)
			}
			assertNoRetrievalFailure(t, api.database, api.logs)
		})
	}
}

// TC-008
func TestAmazonThumbnailURLs_TC008_RecheckReplacesTheStoredThumbnailWithTheFirstAcceptedImage(t *testing.T) {
	pages := map[string]func(t *testing.T) string{
		"accepted landingImage": func(t *testing.T) string { return imagePage(t, newAmazonImage) },
		"rejected landingImage, accepted og:image": func(t *testing.T) string {
			return withOGImage(t, imagePage(t, "https://example.com/a.jpg"), newAmazonImage)
		},
	}
	rows := []struct {
		asin, stored, page string
	}{
		{"B0THUMB081", oldAmazonImage, "accepted landingImage"},
		{"B0THUMB082", oldAmazonImage, "rejected landingImage, accepted og:image"},
		{"B0THUMB083", oldForeignImage, "accepted landingImage"},
		{"B0THUMB084", oldForeignImage, "rejected landingImage, accepted og:image"},
	}
	initial := map[string]string{}
	for _, row := range rows {
		initial[row.asin] = imagePage(t, oldAmazonImage)
	}
	api := openThumbnailAPI(t, initial)

	for _, row := range rows {
		t.Run(row.asin+" "+row.stored+" then "+row.page, func(t *testing.T) {
			item := api.add(t, row.asin)
			if row.stored != oldAmazonImage {
				api.storeThumbnail(t, item.ID, row.stored)
			}
			assertThumbnail(t, api.server, item.ID, &row.stored)

			api.setPage(row.asin, pages[row.page](t))
			checked := api.refresh(t, item.ID)
			assertSucceededAt(t, checked, api.at)
			want := newAmazonImage
			assertThumbnail(t, api.server, item.ID, &want)

			checked = api.refresh(t, item.ID)
			assertSucceededAt(t, checked, api.at)
			assertThumbnail(t, api.server, item.ID, &want)
		})
	}
	assertNoRetrievalFailure(t, api.database, api.logs)

	t.Run("scheduled check", func(t *testing.T) {
		api := openThumbnailAPI(t, map[string]string{"B0THUMB085": amazontest.Fixture(t, "price-found.html")})
		item := api.add(t, "B0THUMB085")
		api.storeThumbnail(t, item.ID, oldForeignImage)
		stored := oldForeignImage
		assertThumbnail(t, api.server, item.ID, &stored)
		api.setPage("B0THUMB085", imagePage(t, newAmazonImage))
		next := listItems(t, api.server)[item.ID].NextCheckAt
		if next == nil {
			t.Fatal("no scheduled check after the first check")
		}
		at := next.Add(time.Second)
		api.at = at
		api.clock.Set(at)
		checked := runScheduler(t, api.server, api.items, item.ID, at)
		assertSucceededAt(t, checked, at)
		want := newAmazonImage
		assertThumbnail(t, api.server, item.ID, &want)
		assertNoRetrievalFailure(t, api.database, api.logs)
	})
}

// TC-009
func TestAmazonThumbnailURLs_TC009_RecheckWithNoAcceptedImageKeepsTheStoredThumbnail(t *testing.T) {
	rows := []struct{ asin, stored string }{{"B0THUMB091", oldAmazonImage}, {"B0THUMB092", oldForeignImage}}
	initial := map[string]string{}
	for _, row := range rows {
		initial[row.asin] = imagePage(t, oldAmazonImage)
	}
	api := openThumbnailAPI(t, initial)

	for _, row := range rows {
		t.Run(row.asin+" "+row.stored, func(t *testing.T) {
			item := api.add(t, row.asin)
			if row.stored != oldAmazonImage {
				api.storeThumbnail(t, item.ID, row.stored)
			}
			assertThumbnail(t, api.server, item.ID, &row.stored)

			for name, page := range map[string]string{"rejected image": imagePage(t, "https://example.com/a.jpg"), "page without image": imagelessPage(t)} {
				api.setPage(row.asin, page)
				checked := api.refresh(t, item.ID)
				assertSucceededAt(t, checked, api.at)
				if checked.ThumbnailURL == nil || *checked.ThumbnailURL != row.stored {
					t.Fatalf("%s: thumbnailUrl %s, want %q", name, showThumbnail(checked.ThumbnailURL), row.stored)
				}
				assertThumbnail(t, api.server, item.ID, &row.stored)
			}

			api.fixture.setFailing(row.asin, true)
			api.setPage(row.asin, imagePage(t, newAmazonImage))
			failed := api.refresh(t, item.ID)
			api.fixture.setFailing(row.asin, false)
			if failed.LastAttempt == nil || failed.LastAttempt.Result == "success" {
				t.Fatalf("precondition: the HTTP 503 check succeeded: %+v", failed.LastAttempt)
			}
			assertThumbnail(t, api.server, item.ID, &row.stored)

			checked := api.refresh(t, item.ID)
			assertSucceededAt(t, checked, api.at)
			want := newAmazonImage
			assertThumbnail(t, api.server, item.ID, &want)
		})
	}
}
