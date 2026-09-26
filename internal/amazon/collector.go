package amazon

import (
	"context"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"pricefollower.local/internal/model"
)

var (
	hosts               = map[string]bool{"amazon.de": true, "amazon.fr": true, "amazon.es": true, "amazon.it": true, "amazon.nl": true, "amazon.be": true}
	tagPattern          = regexp.MustCompile(`(?s)<[^>]+>`)
	scriptPattern       = regexp.MustCompile(`(?is)<script\b[^>]*>[\s\S]*?</script>`)
	stylePattern        = regexp.MustCompile(`(?is)<style\b[^>]*>[\s\S]*?</style>`)
	spacePattern        = regexp.MustCompile(`\s+`)
	numberPattern       = regexp.MustCompile(`[0-9][0-9\s\x{00a0}.,]*[0-9]|[0-9]`)
	offscreenPattern    = regexp.MustCompile(`(?is)<span\b[^>]*class=["']([^"']*\ba-offscreen\b[^"']*)["'][^>]*>(.*?)</span>`)
	priceWrapperPattern = regexp.MustCompile(`(?is)<span\b[^>]*class=["']([^"']*\ba-price\b[^"']*)["'][^>]*>`)
	imagePattern        = regexp.MustCompile(`(?is)<img\b[^>]*>`)
)

type URLResult struct {
	Kind        string
	ASIN        string
	Marketplace string
	URL         string
	Canonical   string
}

func ParseURL(raw string) URLResult {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return URLResult{Kind: "invalid"}
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if strings.HasPrefix(host, "www.") {
		host = strings.TrimPrefix(host, "www.")
	}
	if !strings.EqualFold(parsed.Scheme, "https") || (parsed.Port() != "" && parsed.Port() != "443") || parsed.User != nil || !hosts[host] {
		return URLResult{Kind: "unsupported"}
	}
	pathPattern := regexp.MustCompile(`(?i)/(?:dp|gp/product|gp/aw/d)/([A-Z0-9]{10})(?:/|$)`)
	match := pathPattern.FindStringSubmatch(parsed.Path)
	if len(match) < 2 {
		return URLResult{Kind: "unsupported"}
	}
	asin := strings.ToUpper(match[1])
	canonical := "https://" + host + "/dp/" + asin
	return URLResult{Kind: "valid", ASIN: asin, Marketplace: host, URL: canonical, Canonical: canonical}
}

type Collector struct {
	client    *http.Client
	userAgent string
}

func NewCollector(userAgent string) *Collector {
	return &Collector{
		userAgent: userAgent,
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= 4 {
					return fmt.Errorf("too many Amazon redirects")
				}
				if request.URL.Scheme != "https" || !hosts[strings.ToLower(strings.TrimPrefix(request.URL.Hostname(), "www."))] {
					return fmt.Errorf("redirected to an unsupported host")
				}
				return nil
			},
		},
	}
}

func (c *Collector) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return requestError(item, err)
	}
	chromeMajor := "154"
	if match := regexp.MustCompile(`Chrome/(\d+)`).FindStringSubmatch(c.userAgent); len(match) > 1 {
		chromeMajor = match[1]
	}
	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	request.Header.Set("Accept-Language", locale(item.Marketplace))
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("Upgrade-Insecure-Requests", "1")
	request.Header.Set("Sec-CH-UA", `"Google Chrome";v="`+chromeMajor+`", "Chromium";v="`+chromeMajor+`", "Not_A Brand";v="99"`)
	request.Header.Set("Sec-CH-UA-Mobile", "?0")
	request.Header.Set("Sec-CH-UA-Platform", `"Linux"`)
	request.Header.Set("Sec-Fetch-Dest", "document")
	request.Header.Set("Sec-Fetch-Mode", "navigate")
	request.Header.Set("Sec-Fetch-Site", "none")
	request.Header.Set("Sec-Fetch-User", "?1")

	response, err := c.client.Do(request)
	if err != nil {
		return requestError(item, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		return model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available."}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return requestError(item, fmt.Errorf("Amazon returned HTTP %d", response.StatusCode))
	}
	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if mediaType != "text/html" && mediaType != "application/xhtml+xml" {
		return requestError(item, fmt.Errorf("Amazon returned an unexpected content type"))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 5<<20))
	if err != nil {
		return requestError(item, err)
	}
	page := string(body)
	if regexp.MustCompile(`(?i)captcha|robot check|enter the characters you see below`).MatchString(page[:min(len(page), 200_000)]) {
		return requestError(item, fmt.Errorf("Amazon blocked the product request"))
	}
	if regexp.MustCompile(`(?i)currently unavailable|temporarily out of stock|no longer available|this item is not available`).MatchString(textContent(page[:min(len(page), 200_000)])) {
		return model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available."}
	}
	amount, ok := productPrice(page)
	if !ok {
		return model.CollectionResult{Result: "price_not_found", Message: "Amazon did not show a detectable euro price."}
	}
	title := productTitle(page)
	thumbnail := productImage(page)
	return model.CollectionResult{Result: "success", AmountCents: amount, Title: title, ThumbnailURL: thumbnail}
}

func requestError(item model.Listing, err error) model.CollectionResult {
	fmt.Printf("Amazon collection failed for %s/%s: %v\n", item.Marketplace, item.ASIN, err)
	return model.CollectionResult{Result: "request_error", Message: "Amazon could not be reached for a price check."}
}

func locale(marketplace string) string {
	locales := map[string]string{
		"amazon.de": "de-DE,de;q=0.9,en;q=0.8", "amazon.fr": "fr-FR,fr;q=0.9,en;q=0.8",
		"amazon.es": "es-ES,es;q=0.9,en;q=0.8", "amazon.it": "it-IT,it;q=0.9,en;q=0.8",
		"amazon.nl": "nl-NL,nl;q=0.9,en;q=0.8", "amazon.be": "nl-BE,nl;q=0.9,fr-BE;q=0.8,en;q=0.7",
	}
	if value, ok := locales[marketplace]; ok {
		return value
	}
	return "en-GB,en;q=0.9"
}

func textContent(value string) string {
	value = scriptPattern.ReplaceAllString(value, " ")
	value = stylePattern.ReplaceAllString(value, " ")
	value = tagPattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	value = strings.NewReplacer("\u200e", "", "\u200f", "", "\u202a", "", "\u202b", "", "\u202c", "", "\u202d", "", "\u202e", "").Replace(value)
	return strings.TrimSpace(spacePattern.ReplaceAllString(value, " "))
}

func attribute(tag, name string) string {
	pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	match := pattern.FindStringSubmatch(tag)
	if len(match) == 0 {
		return ""
	}
	for _, value := range match[1:] {
		if value != "" {
			return html.UnescapeString(value)
		}
	}
	return ""
}

func elementContentByID(page, id string) string {
	openPattern := regexp.MustCompile(`(?is)<([a-z][\w:-]*)\b[^>]*\bid=["']` + regexp.QuoteMeta(id) + `["'][^>]*>`)
	open := openPattern.FindStringSubmatchIndex(page)
	if len(open) < 4 {
		return ""
	}
	tagName := page[open[2]:open[3]]
	start := open[1]
	closePattern := regexp.MustCompile(`(?is)</` + regexp.QuoteMeta(tagName) + `\s*>`)
	close := closePattern.FindStringIndex(page[start:])
	if close == nil {
		return ""
	}
	return page[start : start+close[0]]
}

func metaContent(page, property string) string {
	metas := regexp.MustCompile(`(?is)<meta\b[^>]*>`).FindAllString(page, -1)
	for _, tag := range metas {
		name := attribute(tag, "property")
		if name == "" {
			name = attribute(tag, "name")
		}
		if strings.EqualFold(name, property) {
			return textContent(attribute(tag, "content"))
		}
	}
	return ""
}

func productTitle(page string) *string {
	title := textContent(elementContentByID(page, "productTitle"))
	if title == "" {
		title = metaContent(page, "og:title")
	}
	if title == "" {
		return nil
	}
	return &title
}

func imageByID(page, id string) string {
	for _, tag := range imagePattern.FindAllString(page, -1) {
		if attribute(tag, "id") == id {
			if value := attribute(tag, "data-old-hires"); value != "" {
				return value
			}
			return attribute(tag, "src")
		}
	}
	return ""
}

func productImage(page string) *string {
	image := imageByID(page, "landingImage")
	if image == "" {
		for _, tag := range imagePattern.FindAllString(page, -1) {
			if attribute(tag, "data-a-dynamic-image") != "" {
				image = attribute(tag, "src")
				break
			}
		}
	}
	if image == "" {
		image = metaContent(page, "og:image")
	}
	if image == "" {
		return nil
	}
	return &image
}

type priceCandidate struct {
	amount   int64
	primary  bool
	priority int
}

func productPrice(page string) (int64, bool) {
	regions := []string{"corePriceDisplay_desktop_feature_div", "corePrice_feature_div", "price_inside_buybox", "apex_desktop", "desktop_buybox"}
	candidates := make([]priceCandidate, 0)
	for priority, id := range regions {
		start := strings.Index(page, `id="`+id+`"`)
		if start < 0 {
			start = strings.Index(page, `id='`+id+`'`)
		}
		if start < 0 {
			continue
		}
		end := min(len(page), start+20_000)
		for _, match := range offscreenPattern.FindAllStringSubmatchIndex(page[start:end], -1) {
			class := strings.ToLower(page[start+match[2] : start+match[3]])
			prefixStart := max(start, start+match[0]-2_000)
			prefix := page[prefixStart : start+match[0]]
			wrappers := priceWrapperPattern.FindAllStringSubmatch(prefix, -1)
			if len(wrappers) == 0 {
				continue
			}
			wrapper := strings.ToLower(wrappers[len(wrappers)-1][1])
			if strings.Contains(class, "a-text-price") || regexp.MustCompile(`a-text-price|price-per-unit|unit-price|installment|saving|coupon|listprice`).MatchString(wrapper) {
				continue
			}
			amount, ok := parseEuroPrice(page[start+match[4] : start+match[5]])
			if !ok {
				continue
			}
			primary := regexp.MustCompile(`\bpricetopay\b|\bpriceblock_(our|deal)price\b`).MatchString(wrapper)
			candidates = append(candidates, priceCandidate{amount, primary, priority})
		}
	}
	if len(candidates) == 0 {
		return 0, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].primary != candidates[j].primary {
			return candidates[i].primary
		}
		return candidates[i].priority < candidates[j].priority
	})
	return candidates[0].amount, true
}

func parseEuroPrice(value string) (int64, bool) {
	value = textContent(value)
	if !strings.Contains(value, "€") && !regexp.MustCompile(`(?i)\bEUR\b`).MatchString(value) {
		return 0, false
	}
	match := numberPattern.FindString(value)
	if match == "" {
		return 0, false
	}
	digits := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, match)
	comma, dot := strings.LastIndex(digits, ","), strings.LastIndex(digits, ".")
	decimalIndex := -1
	if comma >= 0 && dot >= 0 {
		decimalIndex = max(comma, dot)
	} else if comma >= 0 && len(digits)-comma-1 <= 2 {
		decimalIndex = comma
	} else if dot >= 0 && len(digits)-dot-1 <= 2 {
		decimalIndex = dot
	}
	euros, cents := digits, "00"
	if decimalIndex >= 0 {
		euros = strings.NewReplacer(",", "", ".", "").Replace(digits[:decimalIndex])
		cents = strings.NewReplacer(",", "", ".", "").Replace(digits[decimalIndex+1:])
		if len(cents) > 2 {
			cents = cents[:2]
		}
		cents += strings.Repeat("0", 2-len(cents))
	} else {
		euros = strings.NewReplacer(",", "", ".", "").Replace(euros)
	}
	euroValue, err := strconv.ParseInt(euros, 10, 64)
	if err != nil {
		return 0, false
	}
	centValue, err := strconv.ParseInt(cents, 10, 64)
	if err != nil {
		return 0, false
	}
	amount := euroValue*100 + centValue
	return amount, amount > 0
}
