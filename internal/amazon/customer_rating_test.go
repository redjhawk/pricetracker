package amazon

import (
	"strconv"
	"strings"
	"testing"

	"pricefollower.local/internal/model"
)

// Customer rating of Amazon product pages (issue #305, TC-001). The fixtures rating-fr.html and
// rating-en.html carry the product's own rating block, a price and a recommended product with its
// own rating; other languages and corner cases are inline pages.

// pricedPage is a product page priced 49,99 € with block inserted before the price.
func pricedPage(block string) string {
	return `<html><body><h1><span id="productTitle">Rated product</span></h1>` + block +
		`<div id="corePriceDisplay_desktop_feature_div">` + priceSpan("priceToPay", "49,99 €") + `</div></body></html>`
}

// ratingBlock is the product's rating block; an empty title or count leaves that part out.
func ratingBlock(title, count string) string {
	block := `<div id="averageCustomerReviews">`
	if title != "" {
		block += `<span id="acrPopover" title="` + title + `"><i class="a-icon a-icon-star"><span class="a-icon-alt">` + title + `</span></i></span>`
	}
	if count != "" {
		block += `<a id="acrCustomerReviewLink" href="#customerReviews"><span id="acrCustomerReviewText">` + count + `</span></a>`
	}
	return block + `</div>`
}

func showValue(value *int64) string {
	if value == nil {
		return "absent"
	}
	return strconv.FormatInt(*value, 10)
}

func showRating(rating *model.CustomerRating) string {
	if rating == nil {
		return "<nil>"
	}
	return showValue(rating.StarsTenths) + "/" + showValue(rating.Count)
}

// absent marks an expected rating value that is not shown or unreadable.
const absent = -1

func sameValue(got *int64, want int64) bool {
	if want == absent {
		return got == nil
	}
	return got != nil && *got == want
}

func showWant(value int64) string {
	if value == absent {
		return "absent"
	}
	return strconv.FormatInt(value, 10)
}

// assertRating checks a parsed rating that must be set (a product page was read).
func assertRating(t *testing.T, name string, rating *model.CustomerRating, tenths, count int64) {
	t.Helper()
	if rating == nil || !sameValue(rating.StarsTenths, tenths) || !sameValue(rating.Count, count) {
		t.Errorf("%s: rating %s, want %s/%s", name, showRating(rating), showWant(tenths), showWant(count))
	}
}

func TestCustomerRating_TC001_ProductPagesInSupportedLanguages(t *testing.T) {
	for _, test := range []struct {
		name          string
		page          string
		tenths, count int64
		amount        int64
	}{
		{"amazon.fr fixture", thumbnailFixture(t, "rating-fr.html"), 45, 1234, 11999},
		{"English fixture", thumbnailFixture(t, "rating-en.html"), 45, 1234, 11999},
		{"German", pricedPage(ratingBlock("4,5 von 5 Sternen", "1.234 Sternebewertungen")), 45, 1234, 4999},
		{"Spanish", pricedPage(ratingBlock("4,5 de 5 estrellas", "1.234 valoraciones")), 45, 1234, 4999},
		{"Italian", pricedPage(ratingBlock("4,5 su 5 stelle", "1.234 voti")), 45, 1234, 4999},
		{"Dutch", pricedPage(ratingBlock("4,5 van 5 sterren", "1.234 beoordelingen")), 45, 1234, 4999},
		{"French no-break space", pricedPage(ratingBlock("4,2 sur 5 étoiles", "12 345 évaluations")), 42, 12345, 4999},
		{"French narrow no-break space", pricedPage(ratingBlock("4,2 sur 5 étoiles", "12 345 évaluations")), 42, 12345, 4999},
		{"single rating", pricedPage(ratingBlock("4.0 out of 5 stars", "1 rating")), 40, 1, 4999},
		{"whole-number rating", pricedPage(ratingBlock("5 out of 5 stars", "7 ratings")), 50, 7, 4999},
		{"rating without a count", pricedPage(ratingBlock("4,5 sur 5 étoiles", "")), 45, absent, 4999},
		{"count without a rating", pricedPage(ratingBlock("", "1 234 évaluations")), absent, 1234, 4999},
		{"rating in the text only", pricedPage(`<span id="acrPopover"><span class="a-icon-alt">3,5 sur 5 étoiles</span></span>`), 35, absent, 4999},
		{"malformed rating block", pricedPage(ratingBlock("sur 5 étoiles", "évaluations")), absent, absent, 4999},
		{"no rating block", pricedPage(""), absent, absent, 4999},
		{"rating of a recommended product only", pricedPage(`<div class="a-carousel-card"><i class="a-icon a-icon-star-small"><span class="a-icon-alt">3,0 sur 5 étoiles</span></i><span>987 évaluations</span></div>`), absent, absent, 4999},
	} {
		result, product := parseProductPage(model.CollectionResult{}, test.page)
		if result.Result != "success" || result.AmountCents != test.amount || product == "" {
			t.Errorf("%s: result %q %d, want success %d", test.name, result.Result, result.AmountCents, test.amount)
		}
		assertRating(t, test.name, result.CustomerRating, test.tenths, test.count)
	}
}

func TestCustomerRating_TC001_PriceNotFoundPageKeepsItsRating(t *testing.T) {
	page := `<html><body><h1><span id="productTitle">No price product</span></h1>` + ratingBlock("4,5 sur 5 étoiles", "1 234 évaluations") +
		`<div id="buybox"><a href="/gp/offer-listing/B0NOPRICE1">Voir les offres</a></div></body></html>`
	result, _ := parseProductPage(model.CollectionResult{}, page)
	if result.Result != "price_not_found" {
		t.Fatalf("result %q", result.Result)
	}
	assertRating(t, "price not found", result.CustomerRating, 45, 1234)
}

// The existing product page fixtures give the same price, failure and second-hand results as before
// the rating, and an absent rating; an unavailable page has no rating at all (no product was read).
func TestCustomerRating_TC001_ExistingFixturesUnchanged(t *testing.T) {
	for _, test := range []struct {
		fixture, result string
		amount          int64
		title           string
		usedStatus      string
		usedAmount      int64
		usedCondition   string
	}{
		{"price-found.html", "success", 12999, "Price found product", "not_found", 0, ""},
		{"price-not-found.html", "price_not_found", 0, "No price product", "not_found", 0, ""},
		{"unavailable.html", "unavailable", 0, "", "", 0, ""},
		{"used-amazon.html", "success", 11900, "Used Amazon product", "available", 8950, "very_good"},
		{"used-no-price.html", "price_not_found", 0, "Used only product", "available", 4500, "good"},
		{"used-other-seller.html", "success", 7500, "Other seller product", "not_found", 0, ""},
		{"script-price.html", "price_not_found", 0, "Script product", "not_found", 0, ""},
	} {
		result, product := parseProductPage(model.CollectionResult{}, thumbnailFixture(t, test.fixture))
		if result.Result != test.result || result.AmountCents != test.amount || (test.title != "" && pointerValue(result.Title) != test.title) {
			t.Errorf("%s: result %q %d %s", test.fixture, result.Result, result.AmountCents, pointerValue(result.Title))
		}
		if test.result == "unavailable" {
			if product != "" || result.CustomerRating != nil {
				t.Errorf("%s: unavailable page read as a product (rating %s)", test.fixture, showRating(result.CustomerRating))
			}
			continue
		}
		assertRating(t, test.fixture, result.CustomerRating, absent, absent)
		status, amount, condition, _ := secondHandOfferFromProductPage(product)
		if status != test.usedStatus || amount != test.usedAmount || condition != test.usedCondition {
			t.Errorf("%s: second-hand %s %d %s, want %s %d %s", test.fixture, status, amount, condition, test.usedStatus, test.usedAmount, test.usedCondition)
		}
	}
}

func TestCustomerRating_TC001_RatingNeverChangesPriceOrSecondHand(t *testing.T) {
	used := thumbnailFixture(t, "used-amazon.html")
	rated := strings.Replace(used, `<div id="usedAccordionRow">`, ratingBlock("4,5 sur 5 étoiles", "1 234 évaluations")+`<div id="usedAccordionRow">`, 1)
	plainResult, plainProduct := parseProductPage(model.CollectionResult{}, used)
	ratedResult, ratedProduct := parseProductPage(model.CollectionResult{}, rated)
	if ratedResult.Result != plainResult.Result || ratedResult.AmountCents != plainResult.AmountCents || pointerValue(ratedResult.Title) != pointerValue(plainResult.Title) ||
		pointerValue(ratedResult.ThumbnailURL) != pointerValue(plainResult.ThumbnailURL) || ratedResult.Message != plainResult.Message {
		t.Fatalf("the rating changed the result: %+v vs %+v", ratedResult, plainResult)
	}
	plainStatus, plainAmount, plainCondition, plainLabel := secondHandOfferFromProductPage(plainProduct)
	ratedStatus, ratedAmount, ratedCondition, ratedLabel := secondHandOfferFromProductPage(ratedProduct)
	if plainStatus != ratedStatus || plainAmount != ratedAmount || plainCondition != ratedCondition || plainLabel != ratedLabel {
		t.Fatalf("the rating changed the second-hand offer: %s %d vs %s %d", ratedStatus, ratedAmount, plainStatus, plainAmount)
	}
	assertRating(t, "used Amazon page with a rating", ratedResult.CustomerRating, 45, 1234)

	// An unreadable rating never turns a usable answer into a failure.
	broken := pricedPage(`<span id="acrPopover" title="9,9 sur 5 étoiles"></span><span id="acrCustomerReviewText">beaucoup</span>`)
	result, _ := parseProductPage(model.CollectionResult{}, broken)
	if result.Result != "success" || result.AmountCents != 4999 || result.Message != "" {
		t.Fatalf("an unreadable rating changed the answer: %+v", result)
	}
	assertRating(t, "out-of-range rating", result.CustomerRating, absent, absent)
}
