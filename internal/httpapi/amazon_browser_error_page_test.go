package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

// Amazon product pages answered 404/410 with an empty body (issue #187) through the HTTP API: the
// browser shows its own error page, the listing is unavailable and no used-offer request is sent.

const usedOfferFailedLine = "Amazon used-offer check failed for"

// usedOfferCount counts the requests the fixture server received on both used-offer paths, whatever
// their method (a CORS preflight counts too).
func (api navigationAPI) usedOfferCount() int {
	return api.fixture.Count("/gp/aod/ajax/ref=tmm_pap_used_aod_0") + api.fixture.Count("/gp/aod/ajax")
}

// assertUnavailableOfferKept checks that item shows the unavailable listing checked at the time at,
// with the second-hand check error and the second-hand offer at 89.50 kept.
func assertUnavailableOfferKept(t *testing.T, item model.Item, at time.Time) {
	t.Helper()
	if item.Status != "unavailable" || item.LastAttempt == nil || item.LastAttempt.Result != "unavailable" || !item.LastAttempt.Timestamp.Equal(at) {
		t.Fatalf("item %s: status %s, last attempt %+v, want the listing unavailable at %s", item.ID, item.Status, item.LastAttempt, at)
	}
	if offer := item.SecondHandOffer; offer == nil || offer.Status != "check_error" || offer.LatestDetection == nil || offer.LatestDetection.AmountCents != 8950 {
		t.Fatalf("item %s: second-hand offer %+v, want check_error with the offer at 89.50 kept", item.ID, item.SecondHandOffer)
	}
}

// A scheduled check answered with an empty-body 404 or 410 keeps the last second-hand offer, resets
// the consecutive failure count and sends no used-offer request.
// TC-003
func TestAmazonBrowserErrorPage_TC003_ScheduledCheckSendsNoUsedOfferRequest(t *testing.T) {
	ids := []string{"B0ERRP0031", "B0ERRP0032"}
	for _, unavailable := range []struct {
		name   string
		answer int32
	}{{"HTTP 404", answerEmpty404}, {"HTTP 410", answerEmpty410}} {
		t.Run(unavailable.name, func(t *testing.T) {
			api := openNavigationAPI(t, ids)
			if err := api.database.SaveAmazonFailures(context.Background(), 4); err != nil {
				t.Fatal(err)
			}
			api.fixture.Reset()
			output := captureCheckOutput(t)
			at := humanTime(14, 10)
			api.check(ids[0], at, unavailable.answer)
			logged := output()

			assertUnavailableOfferKept(t, itemDetails(t, api.server, ids[0]), at)
			assertUnavailableOfferKept(t, listItems(t, api.server)[ids[0]], at)
			if got := storedFailures(t, api.database); got != 0 {
				t.Fatalf("consecutive failure count %d after %s, want 0", got, unavailable.name)
			}
			if got := api.fixture.Count("/dp/" + ids[0]); got != 1 {
				t.Fatalf("%d product page requests, want 1: %+v", got, api.fixture.Requests())
			}
			if got := api.usedOfferCount(); got != 0 {
				t.Fatalf("%d used-offer requests after %s with an empty body: %+v", got, unavailable.name, api.fixture.Requests())
			}
			if lines := linesWith(logged, usedOfferFailedLine); len(lines) != 0 {
				t.Fatalf("used-offer failure logged: %v", lines)
			}
			if lines := linesWith(logged, "Amazon request failed"); len(lines) != 0 {
				t.Fatalf("request failure logged for an unavailable listing: %v", lines)
			}

			api.check(ids[1], humanTime(14, 11), answerEmpty503)
			for id, item := range listItems(t, api.server) {
				if item.AmazonStoppedAt != nil {
					t.Fatalf("item %s stopped by one failure after %s", id, unavailable.name)
				}
			}
		})
	}

	t.Run("item without an earlier second-hand offer", func(t *testing.T) {
		const id = "B0ERRP0033"
		api := openNavigationAPI(t, nil)
		listing := model.Listing{ID: id, Platform: "amazon", ListingID: id, ASIN: id, Marketplace: "amazon.fr", URL: "https://www.amazon.fr/dp/" + id}
		if err := api.database.Insert(context.Background(), listing, listing.URL, humanTime(20, 0)); err != nil {
			t.Fatal(err)
		}
		at := humanTime(14, 10)
		api.check(id, at, answerEmpty404)
		item := itemDetails(t, api.server, id)
		if item.Status != "unavailable" || item.LastAttempt == nil || item.LastAttempt.Result != "unavailable" || !item.LastAttempt.Timestamp.Equal(at) {
			t.Fatalf("status %s, last attempt %+v, want the listing unavailable at %s", item.Status, item.LastAttempt, at)
		}
		if offer := item.SecondHandOffer; offer == nil || offer.Status != "check_error" || offer.LatestDetection != nil || len(offer.LastThreeDetections) != 0 {
			t.Fatalf("second-hand offer %+v, want check_error without an offer", item.SecondHandOffer)
		}
		if got := api.usedOfferCount(); got != 0 {
			t.Fatalf("%d used-offer requests: %+v", got, api.fixture.Requests())
		}
	})

	t.Run("protected mode with two owners", func(t *testing.T) {
		api := openNavigationAPI(t, ids, 1, 2)
		for index, id := range ids {
			at := humanTime(14, 10+index)
			api.check(id, at, answerEmpty404)
			item, err := api.database.Get(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			assertUnavailableOfferKept(t, item, at)
		}
		if got := api.usedOfferCount(); got != 0 {
			t.Fatalf("%d used-offer requests: %+v", got, api.fixture.Requests())
		}
	})
}

// Refresh price and Refresh all prices answered with an empty-body 404 send no used-offer request.
// TC-004
func TestAmazonBrowserErrorPage_TC004_RefreshSendsNoUsedOfferRequest(t *testing.T) {
	ids := []string{"B0ERRP0041", "B0ERRP0042"}

	t.Run("Refresh price, Refresh all prices, Refresh price again", func(t *testing.T) {
		api := openNavigationAPI(t, ids)
		api.fixture.answer.Store(answerEmpty404)
		output := captureCheckOutput(t)

		refresh(t, api.server, ids[0])
		assertUnavailableOfferKept(t, waitBrowserCheck(t, api.server, ids[0], humanTime(14, 0)), humanTime(14, 0))

		api.clock.Set(humanTime(14, 5))
		if recorder := callPath(api.server, http.MethodPost, "/api/v1/items/refresh", ""); recorder.Code != http.StatusAccepted {
			t.Fatalf("Refresh all prices: %d %s", recorder.Code, recorder.Body.String())
		}
		for _, id := range ids {
			assertUnavailableOfferKept(t, waitBrowserCheck(t, api.server, id, humanTime(14, 5)), humanTime(14, 5))
		}

		for attempt, minute := range []int{10, 15} {
			api.clock.Set(humanTime(14, minute))
			refresh(t, api.server, ids[0])
			item := waitBrowserCheck(t, api.server, ids[0], humanTime(14, minute))
			assertUnavailableOfferKept(t, item, humanTime(14, minute))
			if got := api.usedOfferCount(); got != 0 {
				t.Fatalf("%d used-offer requests after repeated Refresh price %d: %+v", got, attempt+1, api.fixture.Requests())
			}
		}
		if got := api.usedOfferCount(); got != 0 {
			t.Fatalf("%d used-offer requests in total: %+v", got, api.fixture.Requests())
		}
		if lines := linesWith(output(), usedOfferFailedLine); len(lines) != 0 {
			t.Fatalf("used-offer failure logged: %v", lines)
		}
	})

	t.Run("Refresh price of a stopped item", func(t *testing.T) {
		api := openNavigationAPI(t, ids)
		if err := api.database.StopAmazonItems(context.Background(), humanTime(13, 0)); err != nil {
			t.Fatal(err)
		}
		api.fixture.answer.Store(answerEmpty404)
		refresh(t, api.server, ids[0])
		item := waitBrowserCheck(t, api.server, ids[0], humanTime(14, 0))
		assertUnavailableOfferKept(t, item, humanTime(14, 0))
		if item.AmazonStoppedAt != nil {
			t.Fatalf("stopped %v after Refresh price answered by Amazon, want restarted", item.AmazonStoppedAt)
		}
		if other := itemDetails(t, api.server, ids[1]); !equalTime(other.AmazonStoppedAt, humanTime(13, 0)) {
			t.Fatalf("item %s: stopped %v, want still stopped at %s", ids[1], other.AmazonStoppedAt, humanTime(13, 0))
		}
		if got := api.usedOfferCount(); got != 0 {
			t.Fatalf("%d used-offer requests: %+v", got, api.fixture.Requests())
		}
	})
}
