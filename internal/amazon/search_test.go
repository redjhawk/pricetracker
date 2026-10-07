package amazon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pricefollower.local/internal/model"
)

const issue56URL = "https://www.amazon.fr/joursprime/?_encoding=UTF8&discounts-widget=x&promotionsSearchLastSeenAsin=B0HBWZ42P9&promotionsSearchStartIndex=0&promotionsSearchPageSize=60"

func TestParseSearchURL(t *testing.T) {
	tests := []struct{ raw, kind, marketplace string }{
		{"  " + issue56URL + " ", "valid", "amazon.fr"},
		{"http://amazon.de/s?k=lego", "valid", "amazon.de"},
		{"https://www.leboncoin.fr/recherche?text=lego", "unsupported", ""},
		{"https://www.amazon.com/s?k=lego", "unsupported", ""},
		{"https://www.amazon.fr:8443/s?k=lego", "unsupported", ""},
		{"ftp://www.amazon.fr/s", "invalid", ""},
		{"lego", "invalid", ""},
		{"", "invalid", ""},
		{"https://www.amazon.fr/s?k=" + strings.Repeat("a", 2048), "invalid", ""},
	}
	for _, test := range tests {
		result := ParseSearchURL(test.raw)
		if result.Kind != test.kind || result.Marketplace != test.marketplace || (test.kind == "valid" && result.URL != strings.TrimSpace(test.raw)) {
			t.Errorf("ParseSearchURL(%q) = %+v", test.raw, result)
		}
	}
}

// resultsPage is a results page fixture with count product cards; the second card repeats the first ASIN.
func resultsPage(count int) string {
	var page strings.Builder
	page.WriteString(`<html><body><div data-asin="" class="header"></div>`)
	for index := 1; index <= count; index++ {
		asin := fmt.Sprintf("B%09d", index)
		if index == 2 {
			asin = "B000000001"
		}
		page.WriteString(`<div data-asin="` + asin + `" data-component-type="s-search-result"><h2><a href="/dp/` + asin + `"><span>Product ` + fmt.Sprint(index) + `</span></a></h2>`)
		if index%10 != 0 {
			page.WriteString(priceSpan("a-text-price", "99,00 €") + priceSpan("", fmt.Sprintf("%d,50 €", index)))
		}
		page.WriteString(`</div>`)
	}
	page.WriteString(`</body></html>`)
	return page.String()
}

func TestSearchProductsKeepsFirst30DistinctInOrder(t *testing.T) {
	products := searchProducts(resultsPage(60), "amazon.fr")
	if len(products) != 30 {
		t.Fatalf("got %d products", len(products))
	}
	first, second, last := products[0], products[1], products[29]
	if first.ASIN != "B000000001" || second.ASIN != "B000000003" || last.ASIN != "B000000031" {
		t.Fatalf("unexpected order %s %s %s", first.ASIN, second.ASIN, last.ASIN)
	}
	if first.URL != "https://www.amazon.fr/dp/B000000001" || first.Canonical != "https://amazon.fr/dp/B000000001" || *first.Title != "Product 1" || *first.PriceCents != 150 {
		t.Fatalf("unexpected first product %+v", first)
	}
	if products[8].ASIN != "B000000010" || products[8].PriceCents != nil {
		t.Fatalf("card without price %+v", products[8])
	}
	links := searchProducts(`<a href="/fr/dp/B0HBWZ42P9?ref=x"><img alt="Deal"></a><a href="https://www.amazon.fr/gp/product/B0HBWZ42P8/">x</a>`, "amazon.fr")
	if len(links) != 2 || links[0].ASIN != "B0HBWZ42P9" || links[1].ASIN != "B0HBWZ42P8" {
		t.Fatalf("unexpected link products %+v", links)
	}
}

func testCollector(server *httptest.Server) *Collector {
	collector := NewCollector("Mozilla/5.0 Chrome/150.0")
	collector.client = server.Client()
	return collector
}

func TestFetchSearchFailuresKeepDiagnostics(t *testing.T) {
	status, body := http.StatusServiceUnavailable, "<html>Service Unavailable\x00 try later</html>"
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.WriteHeader(status)
		response.Write([]byte(body))
	}))
	defer server.Close()
	collector := testCollector(server)
	search := SearchURLResult{Kind: "valid", URL: server.URL + "/s?k=lego", Marketplace: "amazon.fr"}
	products, result := collector.FetchSearch(context.Background(), search)
	if products != nil || result.Result != "request_error" || result.HTTPStatus != 503 || result.RequestURL != search.URL ||
		result.ResponseExcerpt != "<html>Service Unavailable  try later</html>" || result.Message != "Amazon returned HTTP 503" {
		t.Fatalf("unexpected failure %+v", result)
	}
	status, body = http.StatusOK, "<html>no results</html>"
	if _, result = collector.FetchSearch(context.Background(), search); result.Result != "request_error" || result.HTTPStatus != 200 || result.ResponseExcerpt != body {
		t.Fatalf("unexpected empty page result %+v", result)
	}
	body = resultsPage(3)
	if products, result = collector.FetchSearch(context.Background(), search); result.Result != "success" || len(products) != 2 {
		t.Fatalf("unexpected success %+v %+v", products, result)
	}
}

func TestCollectProductSendsOneRequestAndReturnsDetails(t *testing.T) {
	requests := 0
	page := `<html><span id="productTitle"> Lego Castle </span><div id="corePriceDisplay_desktop_feature_div">` + priceSpan("priceToPay", "49,99 €") + `</div>
<div id="feature-bullets"><ul><li><span>1 200 pieces</span></li><li> </li><li><div>Age 12+</div></li></ul></div>
<div id="productDescription"><div><p>A large <b>castle</b>.</p></div><p>Gift box.</p></div></html>`
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.Header().Set("Content-Type", "text/html")
		response.Write([]byte(page))
	}))
	defer server.Close()
	item := model.Listing{ASIN: "B000000001", Marketplace: "amazon.fr", URL: server.URL + "/dp/B000000001"}
	result := testCollector(server).CollectProduct(context.Background(), item)
	if requests != 1 || result.Result != "success" || result.AmountCents != 4999 || result.SecondHandStatus != "check_error" || result.Product == nil {
		t.Fatalf("unexpected result after %d requests: %+v", requests, result)
	}
	details := result.Product
	if details.Title != "Lego Castle" || *details.PriceCents != 4999 || strings.Join(details.Features, "|") != "1 200 pieces|Age 12+" || details.Description != "A large castle . Gift box." {
		t.Fatalf("unexpected details %+v", details)
	}
}
