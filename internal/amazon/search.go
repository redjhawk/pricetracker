package amazon

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"pricefollower.local/internal/model"
)

// SearchLimit is the number of products kept from a results page.
const SearchLimit = 30

var (
	cardASINPattern = regexp.MustCompile(`(?i)\bdata-asin=["']([A-Z0-9]{10})["']`)
	linkASINPattern = regexp.MustCompile(`(?i)href=["'][^"']*/(?:dp|gp/product)/([A-Z0-9]{10})(?:[/?"']|$)`)
	headingPattern  = regexp.MustCompile(`(?is)<h2\b[^>]*>(.*?)</h2>`)
)

// SearchURLResult is a validated Amazon results URL.
type SearchURLResult struct {
	Kind        string // valid, invalid or unsupported
	URL         string // as entered, trimmed
	Marketplace string
}

// ParseSearchURL accepts an http(s) URL on a supported Amazon marketplace, with any path and query.
func ParseSearchURL(raw string) SearchURLResult {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if raw == "" || len(raw) > 2048 || err != nil || parsed.Host == "" {
		return SearchURLResult{Kind: "invalid"}
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return SearchURLResult{Kind: "invalid"}
	}
	host := strings.TrimPrefix(strings.ToLower(strings.TrimSuffix(parsed.Hostname(), ".")), "www.")
	if parsed.Port() != "" || parsed.User != nil || !hosts[host] {
		return SearchURLResult{Kind: "unsupported"}
	}
	return SearchURLResult{Kind: "valid", URL: raw, Marketplace: host}
}

// SearchProduct is one product card of a results page.
type SearchProduct struct {
	ASIN        string
	Marketplace string
	URL         string // https://www.amazon.<tld>/dp/<ASIN>
	Canonical   string // same key as ParseURL, so a tracked item is found
	Title       *string
	PriceCents  *int64
}

// FetchSearch opens a results page once and returns its first 30 distinct products in page order.
// The result is success, or request_error with diagnostics (also when the page shows no product).
func (c *Collector) FetchSearch(ctx context.Context, search SearchURLResult) ([]SearchProduct, model.CollectionResult) {
	target, err := url.Parse(search.URL)
	if err != nil {
		return nil, model.CollectionResult{Result: "request_error", Message: err.Error(), RequestURL: search.URL}
	}
	target.Scheme = "https" // the client follows only https redirects
	result, page := c.fetchPage(ctx, search.Marketplace, target.String(), "results")
	if result.Result == "unavailable" {
		result.Result, result.Message = "request_error", "Amazon returned the results page as not found"
	}
	if result.Result != "" {
		return nil, result
	}
	products := searchProducts(page, search.Marketplace)
	if len(products) == 0 {
		result.Result, result.Message, result.ResponseExcerpt = "request_error", "Amazon results page showed no product", excerpt([]byte(page))
		return nil, result
	}
	result.Result = "success"
	return products, result
}

// searchProducts splits the page at each product card (data-asin, or a product link when the page has
// no card) and reads the title and price inside the card.
func searchProducts(page, marketplace string) []SearchProduct {
	matches := cardASINPattern.FindAllStringSubmatchIndex(page, -1)
	if len(matches) == 0 {
		matches = linkASINPattern.FindAllStringSubmatchIndex(page, -1)
	}
	products := make([]SearchProduct, 0, SearchLimit)
	seen := map[string]bool{}
	for index, match := range matches {
		asin := strings.ToUpper(page[match[2]:match[3]])
		if seen[asin] {
			continue
		}
		seen[asin] = true
		end := len(page)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		card := page[match[0]:end]
		link := "https://www." + marketplace + "/dp/" + asin
		product := SearchProduct{ASIN: asin, Marketplace: marketplace, URL: link, Canonical: ParseURL(link).Canonical}
		if heading := headingPattern.FindStringSubmatch(card); heading != nil {
			if title := textContent(heading[1]); title != "" {
				product.Title = &title
			}
		}
		if amount, ok := cardPrice(card); ok {
			product.PriceCents = &amount
		}
		products = append(products, product)
		if len(products) == SearchLimit {
			break
		}
	}
	return products
}

// cardPrice returns the first euro price of a card that is not a struck-through or unit price.
func cardPrice(card string) (int64, bool) {
	wrappers := priceWrapperPattern.FindAllStringSubmatchIndex(card, -1)
	for _, wrapper := range wrappers {
		class := strings.ToLower(card[wrapper[2]:wrapper[3]])
		if strings.Contains(class, "a-text-price") {
			continue
		}
		if offscreen := offscreenPattern.FindStringSubmatch(card[wrapper[1]:]); offscreen != nil {
			if amount, ok := parseEuroPrice(offscreen[2]); ok {
				return amount, true
			}
		}
	}
	return 0, false
}
