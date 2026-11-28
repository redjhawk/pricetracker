package amazon

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/amazon/amazontest"
	"pricefollower.local/internal/model"
)

// Amazon browser retrieval of issue #204: product pages whose content mentions "captcha" or "robot"
// are read as product pages, robot checks stay failures, and every failure has its own cause.

const (
	retrievalSecret  = "fixture-secret-token-204" // session-token cookie value; never in a result or the log
	retrievalSession = "fixture-session-204"      // session-id cookie set by the product pages
	mentionTitle     = "Robot aspirateur laveur X200"
	mentionImage     = "https://m.media-amazon.com/images/I/mentions-captcha.jpg"
	rootSandboxError = "Running as root without --no-sandbox is not supported. See https://crbug.com/638180."
)

// domMentionScript adds a text naming a captcha and a robot check to the page while it loads, so it is
// only in the rendered page.
const domMentionScript = `<script>
(function () {
  var note = document.createElement("p");
  note.className = "captcha-note";
  note.textContent = ["Captcha", "aucun robot check requis pour ce produit."].join(" : ");
  document.body.appendChild(note);
})();
</script>
`

// mentionSnippets are the mentions of the mentioning page (page script, footer link, review text)
// added to another product page by withMentions.
const mentionSnippets = `<script>var captchaEndpoint = "/errors/validateCaptcha"; var robotWidget = "CaptchaWidget";</script>
<div id="navFooter"><a href="/gp/help/customer/display.html?nodeId=captcha-help">Aide captcha</a></div>
<p data-hook="review-body">Pas de robot check à la connexion.</p>
`

func withMentions(page string) string {
	return strings.Replace(page, "</body>", mentionSnippets+"</body>", 1)
}

// retrievalSite is the local stand-in for Amazon of the issue #204 browser tests.
type retrievalSite struct {
	pages  map[string]string // product page by ASIN; another ASIN answers 404
	status map[string]int    // status of the product page by ASIN; 200 when absent
	used   map[string]string // used-offer answer by ASIN: content, "503" or "close"; aod-none.html when absent
}

func newRetrievalSite(t *testing.T) *retrievalSite {
	t.Helper()
	mention := amazontest.ExpandedFixture(t, "mentions-captcha.html")
	priceFound := amazontest.Fixture(t, "price-found.html")
	amazonUsed := amazontest.Fixture(t, "aod-amazon-used.html")
	site := &retrievalSite{
		pages: map[string]string{
			"B0MENTION1": mention,
			"B0CONSENT1": amazontest.ExpandedFixture(t, "consent-banner.html"),
			"B0DOMROBOT": strings.Replace(mention, "</body>", domMentionScript+"</body>", 1),
			"B0ROBOTEN1": amazontest.Fixture(t, "robot-check-en.html"),
			"B0ROBOTFR1": amazontest.Fixture(t, "robot-check-fr.html"),
			"B0ROBOT503": amazontest.Fixture(t, "robot-check-en.html"),
			"B0GONE0410": "<html><body><h1>Gone fixture</h1></body></html>",
			"B0MUSEDAMZ": withMentions(amazontest.Fixture(t, "used-amazon.html")),
			"B0MAODAMZ1": withMentions(priceFound),
			"B0MAODOTH1": withMentions(priceFound),
			"B0MAODNONE": withMentions(priceFound),
			"B0MAOD5031": withMentions(priceFound),
			"B0MAODCLOS": withMentions(priceFound),
			"B0MAODCAP1": withMentions(priceFound),
		},
		status: map[string]int{"B0ROBOT503": http.StatusServiceUnavailable, "B0GONE0410": http.StatusGone},
		used: map[string]string{
			"B0USEDOTH1": amazontest.Fixture(t, "aod-other-seller.html"),
			"B0MAODOTH1": amazontest.Fixture(t, "aod-other-seller.html"),
			"B0MAODNONE": amazontest.Fixture(t, "aod-none.html"),
			"B0MAODAMZ1": amazonUsed,
			"B0MAODCAP1": `<script>var captcha = "validateCaptcha"; var robot = "Captcha";</script>` + "\n" + amazonUsed,
			"B0MAOD5031": "503",
			"B0MAODCLOS": "close",
		},
	}
	for asin, name := range fixturePages {
		site.pages[asin] = amazontest.Fixture(t, name)
	}
	return site
}

// usedAnswer returns the used-offer answer of asin.
func (s *retrievalSite) usedAnswer(t *testing.T, asin string) string {
	t.Helper()
	if answer, ok := s.used[asin]; ok {
		return answer
	}
	return amazontest.Fixture(t, "aod-none.html")
}

// handler serves the product pages at /dp/<ASIN>, setting the session-id cookie and the secret
// session-token cookie, the used-offer answers at /gp/aod/ajax and the page resources.
func (s *retrievalSite) handler(t *testing.T) http.Handler {
	t.Helper()
	pixel, err := base64.StdEncoding.DecodeString(pixelPNG)
	if err != nil {
		t.Fatal(err)
	}
	noOffer := amazontest.Fixture(t, "aod-none.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/dp/", func(w http.ResponseWriter, r *http.Request) {
		asin := strings.TrimPrefix(r.URL.Path, "/dp/")
		http.SetCookie(w, &http.Cookie{Name: "session-id", Value: retrievalSession, Path: "/", MaxAge: 3600})
		http.SetCookie(w, &http.Cookie{Name: "session-token", Value: retrievalSecret, Path: "/", MaxAge: 3600})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page, ok := s.pages[asin]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "<html><body><h1>Page introuvable</h1></body></html>")
			return
		}
		if status, ok := s.status[asin]; ok {
			w.WriteHeader(status)
		}
		io.WriteString(w, page)
	})
	usedOffers := func(w http.ResponseWriter, r *http.Request) {
		answer, ok := s.used[r.URL.Query().Get("asin")]
		if !ok {
			answer = noOffer
		}
		switch answer {
		case "503":
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, "<html><body>Service Unavailable fixture</body></html>")
		case "close":
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("the fixture server cannot close the connection")
				return
			}
			if connection, _, err := hijacker.Hijack(); err == nil {
				connection.Close()
			}
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, answer)
		}
	}
	mux.HandleFunc("/gp/aod/ajax", usedOffers)
	mux.HandleFunc("/gp/aod/ajax/", usedOffers)
	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		io.WriteString(w, "body { font-family: sans-serif; }")
	})
	mux.HandleFunc("/static/pixel.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pixel)
	})
	return mux
}

// usedOfferRequests returns the used-offer requests received by server for asin.
func usedOfferRequests(server *amazontest.Server, asin string) []amazontest.Request {
	var found []amazontest.Request
	for _, request := range server.Requests() {
		if strings.HasPrefix(request.Path, "/gp/aod/ajax") && strings.Contains(request.Query, "asin="+asin) {
			found = append(found, request)
		}
	}
	return found
}

// readPage classifies html as a loaded product page and reads it, as the browser collector does.
func readPage(html string) (model.CollectionResult, bool, error) {
	unavailable, err := (&BrowserCollector{}).classifyPage(loadedPage{status: 200, mimeType: "text/html", url: "https://www.amazon.fr/dp/B0MENTION1", html: html})
	result, _ := parseProductPage(model.CollectionResult{}, html)
	return result, unavailable, err
}

// parsedResult is the result of the product page parser on page answered with status, with the
// used-offer answers usedPages: the result the browser collector must give for that page.
func parsedResult(item model.Listing, page string, status int, usedPages []string) model.CollectionResult {
	result := model.CollectionResult{RequestURL: item.URL}
	product := ""
	if status == http.StatusNotFound || status == http.StatusGone {
		result.Result, result.Message = "unavailable", unavailableMessage
	} else {
		result, product = parseProductPage(result, page)
	}
	if product != "" {
		result.SecondHandStatus, result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel = secondHandOfferFromProductPage(product)
		if result.SecondHandStatus == "available" {
			return result
		}
	}
	result.SecondHandStatus, result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel = parseUsedOffers(item, usedPages, nil)
	return result
}

// describe returns the fields of a result that the tests compare, without the response excerpt.
func describe(result model.CollectionResult) string {
	return fmt.Sprintf("{result %s, amount %d, title %s, image %s, message %q, status %d, second-hand %s %d %s %q, cause %q}", result.Result,
		result.AmountCents, pointerValue(result.Title), pointerValue(result.ThumbnailURL), result.Message, result.HTTPStatus, result.SecondHandStatus,
		result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel, result.FailureCause)
}

// sameResult reports the differences between got and want on the fields compared by TC-004.
func sameResult(got, want model.CollectionResult) string {
	if got.Result != want.Result || got.AmountCents != want.AmountCents || pointerValue(got.Title) != pointerValue(want.Title) ||
		pointerValue(got.ThumbnailURL) != pointerValue(want.ThumbnailURL) || got.Message != want.Message ||
		got.SecondHandStatus != want.SecondHandStatus || got.SecondHandAmountCents != want.SecondHandAmountCents ||
		got.SecondHandCondition != want.SecondHandCondition || got.SecondHandConditionLabel != want.SecondHandConditionLabel {
		return fmt.Sprintf("got  %s\nwant %s", describe(got), describe(want))
	}
	return ""
}

// assertMentionPageRead checks the result of a check of the mentioning page.
func assertMentionPageRead(t *testing.T, got model.CollectionResult) {
	t.Helper()
	if got.Result != "success" || got.AmountCents != 12999 || got.Message != "" || got.HTTPStatus != http.StatusOK ||
		pointerValue(got.Title) != mentionTitle || pointerValue(got.ThumbnailURL) != mentionImage || got.SecondHandStatus != "not_found" || got.FailureCause != "" {
		t.Fatalf("mentioning page not read as a product page: %s", describe(got))
	}
}

// assertNoFailureLogged checks that logged has no failure, robot check or block.
func assertNoFailureLogged(t *testing.T, logged string) {
	t.Helper()
	lower := strings.ToLower(logged)
	if strings.Contains(logged, "Amazon collection failed") || strings.Contains(lower, "robot check") || strings.Contains(lower, "blocked") {
		t.Fatalf("a failure was logged for a product page:\n%s", logged)
	}
}

// TC-001
func TestBrowserRetrieval_TC001_MentionsAreNotARobotCheck(t *testing.T) {
	t.Run("mentioning page", func(t *testing.T) {
		page := amazontest.ExpandedFixture(t, "mentions-captcha.html")
		head, tail := strings.ToLower(page[:200_000]), strings.ToLower(page[200_000:])
		for _, word := range []string{"captcha", "robot"} {
			if !strings.Contains(head, word) || !strings.Contains(tail, word) {
				t.Fatalf("fixture check: %q is not mentioned both within and after the first 200 kB", word)
			}
		}
		result, unavailable, err := readPage(page)
		if err != nil || unavailable {
			t.Fatalf("the mentioning page is classified as a failure or unavailable: %v, %v", unavailable, err)
		}
		if result.Result != "success" || result.AmountCents != 12999 || pointerValue(result.Title) != mentionTitle || pointerValue(result.ThumbnailURL) != mentionImage {
			t.Fatalf("mentioning page read as %s", describe(result))
		}
	})

	t.Run("each mention alone", func(t *testing.T) {
		base := amazontest.Fixture(t, "price-found.html")
		mentions := []struct {
			name string
			add  func(word string) string
		}{
			{"page script", func(word string) string {
				return strings.Replace(base, "</head>", `<script>var check = "`+word+`";</script></head>`, 1)
			}},
			{"link URL", func(word string) string {
				return strings.Replace(base, "</body>", `<a href="/gp/help/customer/display.html?nodeId=`+word+`-help">Aide</a></body>`, 1)
			}},
			{"product title", func(word string) string {
				return strings.Replace(base, "  Price found product  ", word+" aspirateur laveur", 1)
			}},
			{"image alt", func(word string) string {
				return strings.Replace(base, `alt="Price found product"`, `alt="`+word+` vu de dessus"`, 1)
			}},
			{"review text", func(word string) string {
				return strings.Replace(base, "</body>", `<p data-hook="review-body">Aucun `+word+` check demandé.</p></body>`, 1)
			}},
		}
		for _, mention := range mentions {
			for _, word := range []string{"captcha", "CAPTCHA", "robot", "Robot"} {
				t.Run(mention.name+" "+word, func(t *testing.T) {
					page := mention.add(word)
					if page == base {
						t.Fatal("fixture check: the mention was not added")
					}
					result, unavailable, err := readPage(page)
					if err != nil || unavailable {
						t.Fatalf("classified as a failure or unavailable: %v, %v", unavailable, err)
					}
					if result.Result != "success" || result.AmountCents != 12999 {
						t.Fatalf("read as %s", describe(result))
					}
				})
			}
		}
	})

	t.Run("robot pages", func(t *testing.T) {
		for _, name := range []string{"robot-check-en.html", "robot-check-fr.html"} {
			robot := amazontest.Fixture(t, name)
			pages := map[string]string{
				"as served":            robot,
				"after a larger page":  amazontest.Expand(strings.Replace(robot, "<body>", "<body>\n<div id=\"navbar\"><a href=\"/\">Amazon.fr</a></div>\n<!--PADDING-->", 1)),
				"before a larger page": amazontest.Expand(strings.Replace(robot, "</body>", "<!--PADDING-->\n</body>", 1)),
				"with a page script":   strings.Replace(robot, "</head>", `<script>window.ue_csm = {};</script></head>`, 1),
			}
			for variant, page := range pages {
				t.Run(name+" "+variant, func(t *testing.T) {
					if _, _, err := readPage(page); err == nil {
						t.Fatal("the robot page is not classified as a failure")
					}
				})
			}
		}
	})

	t.Run("empty page", func(t *testing.T) {
		for _, page := range []string{"", "<html><head></head><body></body></html>"} {
			result, unavailable, err := readPage(page)
			if err != nil || unavailable {
				t.Fatalf("empty page %q classified as a failure or unavailable: %v, %v", page, unavailable, err)
			}
			if result.Result != "price_not_found" {
				t.Fatalf("empty page %q read as %s", page, describe(result))
			}
		}
	})

	t.Run("existing fixtures", func(t *testing.T) {
		tests := []struct {
			name   string
			result string
			amount int64
		}{
			{"price-found.html", "success", 12999},
			{"price-not-found.html", "price_not_found", 0},
			{"unavailable.html", "unavailable", 0},
			{"script-price.html", "price_not_found", 0}, // the price exists only once the page script has run
			{"used-amazon.html", "success", 11900},
			{"used-no-price.html", "price_not_found", 0},
			{"used-other-seller.html", "success", 7500},
		}
		for _, test := range tests {
			result, unavailable, err := readPage(amazontest.Fixture(t, test.name))
			if err != nil || unavailable {
				t.Fatalf("%s classified as a failure or unavailable: %v, %v", test.name, unavailable, err)
			}
			if result.Result != test.result || result.AmountCents != test.amount {
				t.Fatalf("%s read as %s, want %s %d", test.name, describe(result), test.result, test.amount)
			}
		}
	})
}

// TC-002
func TestBrowserRetrieval_TC002_MentioningPageGivesItsPrice(t *testing.T) {
	site := newRetrievalSite(t)
	server := amazontest.NewServer(t, site.handler(t))
	collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
	tests := []struct {
		name string
		asin string
	}{
		{"mentioning page", "B0MENTION1"},
		{"consent banner over the product", "B0CONSENT1"},
		{"mentions added to the DOM by a page script", "B0DOMROBOT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := amazontest.CaptureOutput(t)
			got := browserCheck(t, collector, fixtureItem(test.asin))
			logged := output()
			assertMentionPageRead(t, got)
			assertNoFailureLogged(t, logged)
		})
	}

	t.Run("two checks in a row on the same profile", func(t *testing.T) {
		output := amazontest.CaptureOutput(t)
		first := browserCheck(t, collector, fixtureItem("B0MENTION1"))
		second := browserCheck(t, collector, fixtureItem("B0MENTION1"))
		logged := output()
		assertMentionPageRead(t, first)
		assertMentionPageRead(t, second)
		assertNoFailureLogged(t, logged)
	})
}

// TC-003
func TestBrowserRetrieval_TC003_RobotChecksStayFailures(t *testing.T) {
	site := newRetrievalSite(t)
	server := amazontest.NewServer(t, site.handler(t))
	collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
	tests := []struct {
		name   string
		asin   string
		status int
		cause  string
	}{
		{"English robot page", "B0ROBOTEN1", http.StatusOK, "robot check"},
		{"French robot page", "B0ROBOTFR1", http.StatusOK, "robot check"},
		{"robot page answered with HTTP 503", "B0ROBOT503", http.StatusServiceUnavailable, "HTTP 503"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := amazontest.CaptureOutput(t)
			got := browserCheck(t, collector, fixtureItem(test.asin))
			logged := output()
			if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
				t.Fatalf("not a failure: %s", describe(got))
			}
			if !strings.HasSuffix(got.RequestURL, "/dp/"+test.asin) || got.HTTPStatus != test.status || got.ResponseExcerpt == "" {
				t.Fatalf("missing diagnostics: url=%q status=%d excerpt=%q", got.RequestURL, got.HTTPStatus, got.ResponseExcerpt)
			}
			if !strings.Contains(strings.ToLower(got.FailureCause), strings.ToLower(test.cause)) {
				t.Fatalf("failure cause %q does not name %q", got.FailureCause, test.cause)
			}
			if !strings.Contains(logged, "Amazon collection failed for amazon.fr/"+test.asin+": "+got.FailureCause) {
				t.Fatalf("the logged failure does not carry the cause %q:\n%s", got.FailureCause, logged)
			}
			if requests := usedOfferRequests(server, test.asin); len(requests) != 0 {
				t.Fatalf("used-offer requests sent after a failed product page: %+v", requests)
			}
			if strings.Contains(got.ResponseExcerpt+got.Message+got.RequestURL+got.FailureCause+logged, retrievalSecret) {
				t.Fatalf("a cookie value reached the diagnostics or the log: %s\nexcerpt %s\n%s", describe(got), got.ResponseExcerpt, logged)
			}
		})
	}
}

// TC-004
func TestBrowserRetrieval_TC004_BrowserResultsEqualParserResults(t *testing.T) {
	site := newRetrievalSite(t)
	server := amazontest.NewServer(t, site.handler(t))
	collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
	// The parser reads script-price.html as the browser renders it: with the price added by its script.
	renderedScriptPrice := strings.Replace(site.pages["B0SCRIPT01"], `<div id="corePriceDisplay_desktop_feature_div"></div>`,
		`<div id="corePriceDisplay_desktop_feature_div"><span class="a-price priceToPay"><span class="a-offscreen">49,99 €</span></span></div>`, 1)
	tests := []struct {
		name   string
		asin   string
		page   string
		status int
	}{
		{"price found", "B0PRICE001", site.pages["B0PRICE001"], http.StatusOK},
		{"price added by a page script", "B0SCRIPT01", renderedScriptPrice, http.StatusOK},
		{"instalment amount only", "B0NOPRICE1", site.pages["B0NOPRICE1"], http.StatusOK},
		{"unavailable text", "B0UNAVAIL1", site.pages["B0UNAVAIL1"], http.StatusOK},
		{"second-hand offer sold by Amazon", "B0USEDAMZ1", site.pages["B0USEDAMZ1"], http.StatusOK},
		{"second-hand offer and no new price", "B0USEDNOP1", site.pages["B0USEDNOP1"], http.StatusOK},
		{"second-hand offer by another seller", "B0USEDOTH1", site.pages["B0USEDOTH1"], http.StatusOK},
		{"mentioning page", "B0MENTION1", site.pages["B0MENTION1"], http.StatusOK},
		{"consent banner", "B0CONSENT1", site.pages["B0CONSENT1"], http.StatusOK},
		{"HTTP 404", "B0GONE0001", "", http.StatusNotFound},
		{"HTTP 410", "B0GONE0410", site.pages["B0GONE0410"], http.StatusGone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := fixtureItem(test.asin)
			used := site.usedAnswer(t, test.asin)
			want := parsedResult(item, test.page, test.status, []string{used, used})
			got := browserCheck(t, collector, item)
			if difference := sameResult(got, want); difference != "" {
				t.Fatal(difference)
			}
			switch test.asin {
			case "B0NOPRICE1":
				if got.Result != "price_not_found" {
					t.Fatalf("instalment-only page: %s", describe(got))
				}
			case "B0UNAVAIL1", "B0GONE0001", "B0GONE0410":
				if got.Result != "unavailable" || got.Message != "The listing is no longer available." {
					t.Fatalf("unavailable listing: %s", describe(got))
				}
			}
		})
	}
}

// TC-005
func TestBrowserRetrieval_TC005_SecondHandOfferResultIsUnchanged(t *testing.T) {
	site := newRetrievalSite(t)
	server := amazontest.NewServer(t, site.handler(t))
	collector := NewBrowserCollector(testBrowserOptions(t, server.URL))
	tests := []struct {
		name           string
		asin           string
		amount         int64
		status         string
		usedAmount     int64
		condition      string
		conditionLabel string
	}{
		{"used offer sold by Amazon on the page", "B0MUSEDAMZ", 11900, "available", 8950, "very_good", "Très bon état"},
		{"used-offer answer with an offer sold by Amazon", "B0MAODAMZ1", 12999, "available", 9550, "like_new", "Comme neuf"},
		{"used-offer answer with another seller's offer only", "B0MAODOTH1", 12999, "not_found", 0, "", ""},
		{"used-offer answer without offer", "B0MAODNONE", 12999, "not_found", 0, "", ""},
		{"used-offer endpoint answering 503", "B0MAOD5031", 12999, "check_error", 0, "", ""},
		{"used-offer connection closed", "B0MAODCLOS", 12999, "check_error", 0, "", ""},
		{"used-offer answer mentioning captcha", "B0MAODCAP1", 12999, "available", 9550, "like_new", "Comme neuf"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := browserCheck(t, collector, fixtureItem(test.asin))
			if got.Result != "success" || got.AmountCents != test.amount {
				t.Fatalf("price result %s, want success %d", describe(got), test.amount)
			}
			if got.SecondHandStatus != test.status || got.SecondHandAmountCents != test.usedAmount || got.SecondHandCondition != test.condition ||
				got.SecondHandConditionLabel != test.conditionLabel {
				t.Fatalf("second-hand result %s %d %s %q, want %s %d %s %q", got.SecondHandStatus, got.SecondHandAmountCents, got.SecondHandCondition,
					got.SecondHandConditionLabel, test.status, test.usedAmount, test.condition, test.conditionLabel)
			}
			requests := usedOfferRequests(server, test.asin)
			if test.asin == "B0MUSEDAMZ" {
				if len(requests) != 0 {
					t.Fatalf("used-offer requests sent although the page shows an Amazon offer: %+v", requests)
				}
				return
			}
			if len(requests) == 0 {
				t.Fatal("no used-offer request sent")
			}
			if test.asin == "B0MAODAMZ1" && !strings.Contains(requests[0].Header.Get("Cookie"), "session-id="+retrievalSession) {
				t.Fatalf("the used-offer request does not carry the page's cookies: %+v", requests[0].Header)
			}
		})
	}
}

// TC-006
func TestBrowserRetrieval_TC006_BrowserStartFailureLogsItsReason(t *testing.T) {
	stub := func(t *testing.T, script string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "chromium-stub")
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	// check runs one check with executable and returns the result and what was logged.
	check := func(t *testing.T, executable string) (model.CollectionResult, string) {
		t.Helper()
		profile := filepath.Join(t.TempDir(), "profile")
		output := amazontest.CaptureOutput(t)
		goroutines := runtime.NumGoroutine()
		collector := NewBrowserCollector(BrowserOptions{ExecPath: executable, ProfileDir: profile, NoSandbox: true,
			NavigationTimeout: 5 * time.Second, Pause: func() time.Duration { return 0 }, Origin: "http://127.0.0.1:9"})
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		started := time.Now()
		got := collector.Collect(ctx, fixtureItem("B0PRICE001"))
		if elapsed := time.Since(started); elapsed > 20*time.Second {
			t.Fatalf("the failed start took %s", elapsed)
		}
		logged := output()
		if executable != "" {
			assertNothingLeft(t, executable, goroutines)
		}
		assertNothingLeft(t, profile, goroutines)
		if got.Result != "request_error" || got.Message != amazonCheckFailed || got.SecondHandStatus != "check_error" {
			t.Fatalf("not a failure: %s", describe(got))
		}
		if !strings.HasPrefix(got.FailureCause, "browser could not start") {
			t.Fatalf("failure cause %q does not say the browser could not start", got.FailureCause)
		}
		if strings.ContainsAny(got.FailureCause, "\r\n") || len(got.FailureCause) > 1024 {
			t.Fatalf("failure cause is not one bounded line (%d bytes): %q", len(got.FailureCause), got.FailureCause)
		}
		if !strings.Contains(logged, "Amazon collection failed for amazon.fr/B0PRICE001: "+got.FailureCause) {
			t.Fatalf("the logged failure does not carry the cause %q:\n%s", got.FailureCause, logged)
		}
		return got, logged
	}

	t.Run("missing executable", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing-chromium")
		got, _ := check(t, missing)
		if !strings.Contains(got.FailureCause, missing) || !strings.Contains(got.FailureCause, "no such file") {
			t.Fatalf("failure cause %q does not name the missing file %s", got.FailureCause, missing)
		}
	})

	t.Run("browser refusing to run as root", func(t *testing.T) {
		got, _ := check(t, stub(t, "echo '"+rootSandboxError+"' >&2\nexit 1\n"))
		if !strings.Contains(got.FailureCause, rootSandboxError) || !strings.Contains(got.FailureCause, "exit status 1") {
			t.Fatalf("failure cause %q does not carry the browser's message and exit status", got.FailureCause)
		}
	})

	t.Run("browser exiting without output", func(t *testing.T) {
		got, _ := check(t, stub(t, "exit 3\n"))
		if !strings.Contains(got.FailureCause, "exit status 3") {
			t.Fatalf("failure cause %q does not carry the exit status", got.FailureCause)
		}
	})

	t.Run("browser writing several lines", func(t *testing.T) {
		script := "i=1\nwhile [ $i -le 40 ]; do echo \"stub line $i: the browser reports a long diagnostic message on its error output\" >&2; i=$((i+1)); done\nexit 1\n"
		got, logged := check(t, stub(t, script))
		if !strings.Contains(got.FailureCause, "exit status 1") || !strings.Contains(got.FailureCause, "stub line") {
			t.Fatalf("failure cause %q does not carry the browser's output", got.FailureCause)
		}
		for _, line := range strings.Split(logged, "\n") {
			if strings.Contains(line, "stub line") && !strings.Contains(line, "Amazon collection failed") {
				t.Fatalf("the browser's output broke the log entry over several lines:\n%s", logged)
			}
		}
	})

	t.Run("no browser found", func(t *testing.T) {
		t.Setenv(BrowserPathVariable, "")
		t.Setenv("PATH", t.TempDir())
		got, _ := check(t, "")
		if !strings.Contains(got.FailureCause, "no Chromium or Google Chrome found") {
			t.Fatalf("failure cause %q does not say no browser was found", got.FailureCause)
		}
	})
}
