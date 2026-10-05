package amazon

import (
	"context"
	"fmt"
	"html"
	"io"
	"log"
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
	aodOfferPattern     = regexp.MustCompile(`(?is)<div\b[^>]*\bid=["']aod-offer["'][^>]*>`)
	htmlTokenPattern    = regexp.MustCompile(`(?is)<!--.*?-->|<![^>]*>|</?[a-z][^>]*>|[^<]+`)
	tagNamePattern      = regexp.MustCompile(`(?is)^</?([a-z][\w:-]*)`)
	instalmentText      = regexp.MustCompile(`(?i)^\s*(€)?\s*x\s*\d+(\D|$)`)
	instalmentContainer = regexp.MustCompile(`(?i)installment|inemi|price-block-message|price-block-amount`)
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
	request, err := c.newRequest(ctx, item, item.URL)
	if err != nil {
		return requestError(item, err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return requestError(item, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		return c.withSecondHand(ctx, item, model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available."})
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
		return c.withSecondHand(ctx, item, model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available."})
	}
	result := model.CollectionResult{Result: "price_not_found", Message: "Amazon did not show a detectable euro price.", SecondHandStatus: "check_error"}
	if amount, ok := productPrice(page); ok {
		result.Result = "success"
		result.AmountCents = amount
		result.Title = productTitle(page)
		result.ThumbnailURL = productImage(page)
	} else {
		result.Title = productTitle(page)
		result.ThumbnailURL = productImage(page)
	}
	result.SecondHandStatus, result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel = secondHandOfferFromProductPage(page)
	if result.SecondHandStatus == "available" {
		return result
	}
	return c.withSecondHand(ctx, item, result)
}

func (c *Collector) withSecondHand(ctx context.Context, item model.Listing, result model.CollectionResult) model.CollectionResult {
	result.SecondHandStatus, result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel = c.secondHandOffer(ctx, item)
	return result
}

func (c *Collector) newRequest(ctx context.Context, item model.Listing, target string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
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
	return request, nil
}

type usedCondition struct {
	code  string
	label string
}

type offerHTMLNode struct {
	tag      string
	id       string
	openTag  string
	text     string
	children []*offerHTMLNode
	parent   *offerHTMLNode
}

func secondHandOfferFromProductPage(page string) (string, int64, string, string) {
	root := &offerHTMLNode{tag: "root"}
	stack := []*offerHTMLNode{root}
	page = scriptPattern.ReplaceAllString(page, " ")
	page = stylePattern.ReplaceAllString(page, " ")
	for _, token := range htmlTokenPattern.FindAllString(page, -1) {
		if strings.HasPrefix(token, "<!--") || strings.HasPrefix(token, "<!") {
			continue
		}
		if strings.HasPrefix(token, "</") {
			matches := tagNamePattern.FindStringSubmatch(token)
			if len(matches) < 2 {
				continue
			}
			for index := len(stack) - 1; index > 0; index-- {
				if stack[index].tag == strings.ToLower(matches[1]) {
					stack = stack[:index]
					break
				}
			}
			continue
		}
		if strings.HasPrefix(token, "<") {
			matches := tagNamePattern.FindStringSubmatch(token)
			if len(matches) < 2 {
				continue
			}
			node := &offerHTMLNode{tag: strings.ToLower(matches[1]), id: attribute(token, "id"), openTag: token, parent: stack[len(stack)-1]}
			node.parent.children = append(node.parent.children, node)
			if !strings.HasSuffix(strings.TrimSpace(token), "/>") && !isVoidElement(node.tag) {
				stack = append(stack, node)
			}
			continue
		}
		stack[len(stack)-1].text += " " + html.UnescapeString(token)
	}

	var merchantNodes []*offerHTMLNode
	collectOfferNodesByID(root, "merchant-info", &merchantNodes)
	for _, merchant := range merchantNodes {
		if !hasAmazonSecondHandSeller(merchant) {
			continue
		}
		var amount int64
		amountFound := false
		condition := usedCondition{code: "unknown", label: "Condition unavailable"}
		for ancestor, depth := merchant, 0; ancestor != nil && ancestor.tag != "root" && depth < 12; ancestor, depth = ancestor.parent, depth+1 {
			if !amountFound {
				amount, amountFound = customerVisibleEuroPrice(ancestor)
				if !amountFound {
					if priceNode := findOfferNodeByID(ancestor, "price_feature_div"); priceNode != nil {
						amount, amountFound = parseEuroPrice(offerNodeText(priceNode))
					}
				}
			}
			if candidate, found := parseUsedCondition(offerNodeText(ancestor)); found {
				condition = candidate
			}
			if amountFound && condition.code != "unknown" {
				return "available", amount, condition.code, condition.label
			}
			if ancestor.id == "usedAccordionRow" {
				break
			}
		}
		if amountFound {
			return "available", amount, condition.code, condition.label
		}
	}
	return "not_found", 0, "", ""
}

func collectOfferNodesByID(node *offerHTMLNode, id string, matches *[]*offerHTMLNode) {
	if node.id == id {
		*matches = append(*matches, node)
	}
	for _, child := range node.children {
		collectOfferNodesByID(child, id, matches)
	}
}

func hasAmazonSecondHandSeller(node *offerHTMLNode) bool {
	if node.tag == "a" && isAmazonSeller(offerNodeText(node)) && strings.Contains(strings.ToLower(offerNodeText(node)), "seconde main") {
		return true
	}
	for _, child := range node.children {
		if hasAmazonSecondHandSeller(child) {
			return true
		}
	}
	return false
}

func findOfferNodeByID(node *offerHTMLNode, id string) *offerHTMLNode {
	if node.id == id {
		return node
	}
	for _, child := range node.children {
		if match := findOfferNodeByID(child, id); match != nil {
			return match
		}
	}
	return nil
}

func offerNodeText(node *offerHTMLNode) string {
	var text strings.Builder
	text.WriteString(node.text)
	for _, child := range node.children {
		text.WriteByte(' ')
		text.WriteString(offerNodeText(child))
	}
	return textContent(text.String())
}

func customerVisibleEuroPrice(node *offerHTMLNode) (int64, bool) {
	var amount, currency string
	var findInputs func(*offerHTMLNode)
	findInputs = func(current *offerHTMLNode) {
		if current.tag == "input" {
			name := attribute(current.openTag, "name")
			id := attribute(current.openTag, "id")
			switch {
			case name == "items[0.base][customerVisiblePrice][amount]" || id == "items[0.base][customerVisiblePrice][amount]":
				amount = attribute(current.openTag, "value")
			case name == "items[0.base][customerVisiblePrice][currencyCode]" || id == "items[0.base][customerVisiblePrice][currencyCode]":
				currency = attribute(current.openTag, "value")
			}
		}
		for _, child := range current.children {
			findInputs(child)
		}
	}
	findInputs(node)
	if amount == "" || !strings.EqualFold(currency, "EUR") {
		return 0, false
	}
	return parseEuroPrice(amount + " €")
}

func isVoidElement(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func (c *Collector) secondHandOffer(ctx context.Context, item model.Listing) (string, int64, string, string) {
	parsed, err := url.Parse(item.URL)
	if err != nil {
		return "check_error", 0, "", ""
	}
	parsed.Path = "/gp/aod/ajax/ref=tmm_pap_used_aod_0"
	parsed.RawQuery = ""
	// Amazon's used-offer link uses this referral path and a double-encoded
	// filters parameter. A bare /gp/aod/ajax request can return an empty/new
	// offer response even when the product page shows an Amazon Seconde main offer.
	queries := []url.Values{}
	usedQuery := url.Values{}
	usedQuery.Set("asin", item.ASIN)
	usedQuery.Set("pc", "dp")
	usedQuery.Set("condition", "used")
	usedQuery.Set("filters", `%7B%22all%22%3Atrue%2C%22usedLikeNew%22%3Atrue%2C%22usedVeryGood%22%3Atrue%2C%22usedGood%22%3Atrue%2C%22usedAcceptable%22%3Atrue%7D`)
	queries = append(queries, usedQuery)
	allQuery := url.Values{}
	allQuery.Set("asin", item.ASIN)
	allQuery.Set("pc", "dp")
	allQuery.Set("condition", "ALL")
	allQuery.Set("experienceId", "aodAjaxMain")
	queries = append(queries, allQuery)

	var pages []string
	var failedResponses []string
	for _, query := range queries {
		endpoint := *parsed
		if query.Get("condition") == "ALL" {
			endpoint.Path = "/gp/aod/ajax"
		}
		endpoint.RawQuery = query.Encode()
		request, requestErr := c.newRequest(ctx, item, endpoint.String())
		if requestErr != nil {
			continue
		}
		request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		request.Header.Set("Referer", item.URL)
		request.Header.Set("Sec-Fetch-Site", "same-origin")
		request.Header.Set("Sec-Fetch-Mode", "cors")
		request.Header.Set("Sec-Fetch-Dest", "empty")
		request.Header.Set("X-Requested-With", "XMLHttpRequest")
		response, requestErr := c.client.Do(request)
		if requestErr != nil {
			failedResponses = append(failedResponses, requestErr.Error())
			continue
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 5<<20))
			if readErr == nil {
				pages = append(pages, string(body))
			} else {
				failedResponses = append(failedResponses, readErr.Error())
			}
		} else {
			failedResponses = append(failedResponses, fmt.Sprintf("HTTP %d", response.StatusCode))
		}
		response.Body.Close()
	}
	if len(pages) == 0 {
		log.Printf("Amazon used-offer check failed for %s/%s: %s", item.Marketplace, item.ASIN, strings.Join(failedResponses, "; "))
		return "check_error", 0, "", ""
	}
	checkError := false
	offerCount, usedCount, amazonSellerCount, malformedCount := 0, 0, 0, 0
	for _, page := range pages {
		starts := aodOfferPattern.FindAllStringIndex(page, -1)
		offerCount += len(starts)
		if len(starts) == 0 {
			lower := strings.ToLower(page)
			if strings.Contains(lower, "aod-offer") || strings.Contains(lower, "no featured offers") || strings.Contains(lower, "aucune offre") {
				continue
			}
			checkError = true
			continue
		}
		var lowest int64
		var best usedCondition
		ambiguous := false
		for index, match := range starts {
			end := len(page)
			if index+1 < len(starts) {
				end = starts[index+1][0]
			}
			block := page[match[0]:end]
			conditionText := textContent(elementContentByID(block, "aod-offer-condition"))
			if conditionText == "" {
				conditionText = textContent(elementContentByID(block, "aod-offer-heading"))
			}
			condition, isUsed := parseUsedCondition(conditionText)
			if !isUsed {
				condition, isUsed = parseUsedCondition(textContent(block))
			}
			if !isUsed {
				continue
			}
			usedCount++
			sellerText := textContent(elementContentByID(block, "aod-offer-soldBy"))
			if sellerText == "" {
				ambiguous = true
				continue
			}
			if !isAmazonSeller(sellerText) {
				continue
			}
			amazonSellerCount++
			priceRegion := elementContentByID(block, "aod-offer-price")
			if priceRegion == "" {
				malformedCount++
				ambiguous = true
				continue
			}
			amount, ok := lowestEuroAmount(priceRegion)
			if !ok {
				malformedCount++
				ambiguous = true
				continue
			}
			if lowest == 0 || amount < lowest {
				lowest, best = amount, condition
			}
		}
		if lowest > 0 {
			return "available", lowest, best.code, best.label
		}
		if ambiguous {
			checkError = true
		}
	}
	if checkError {
		log.Printf("Amazon used-offer check incomplete for %s/%s: offers=%d used=%d amazon_sellers=%d malformed=%d", item.Marketplace, item.ASIN, offerCount, usedCount, amazonSellerCount, malformedCount)
		return "check_error", 0, "", ""
	}
	log.Printf("Amazon used-offer check found none for %s/%s: offers=%d used=%d amazon_sellers=%d", item.Marketplace, item.ASIN, offerCount, usedCount, amazonSellerCount)
	return "not_found", 0, "", ""
}

func parseUsedCondition(value string) (usedCondition, bool) {
	value = textContent(value)
	lower := strings.ToLower(value)
	patterns := []struct {
		code string
		re   *regexp.Regexp
	}{
		{"like_new", regexp.MustCompile(`(?i)(comme neuf|wie neu|como nuevo|come nuovo|als nieuw|like new)`)},
		{"very_good", regexp.MustCompile(`(?i)(tr[eè]s bon [eé]tat|sehr gut|muy bueno|ottim[ae] condizioni|zeer goed|very good)`)},
		{"good", regexp.MustCompile(`(?i)(bon [eé]tat|gut(?:er zustand)?|bueno|buon[ae] condizioni|goed|condition: good|good)`)},
		{"acceptable", regexp.MustCompile(`(?i)([eé]tat correct|akzeptabel|aceptable|accettabile|acceptabel|acceptable)`)},
	}
	for _, pattern := range patterns {
		match := pattern.re.FindString(value)
		if match != "" {
			return usedCondition{code: pattern.code, label: match}, true
		}
	}
	if strings.Contains(lower, "used") || strings.Contains(lower, "gebraucht") || strings.Contains(lower, "d'occasion") || strings.Contains(lower, "usado") || strings.Contains(lower, "usato") || strings.Contains(lower, "gebruikt") {
		return usedCondition{code: "unknown", label: "Condition unavailable"}, true
	}
	return usedCondition{}, false
}

func lowestEuroAmount(region string) (int64, bool) {
	var lowest int64
	for _, match := range offscreenPattern.FindAllStringSubmatch(region, -1) {
		amount, ok := parseEuroPrice(match[2])
		if ok && (lowest == 0 || amount < lowest) {
			lowest = amount
		}
	}
	return lowest, lowest > 0
}

func isAmazonSeller(value string) bool {
	normalized := strings.ToLower(textContent(value))
	normalized = regexp.MustCompile(`[\s.,'’()\-]`).ReplaceAllString(normalized, "")
	known := map[string]bool{
		"amazon": true, "amazonfr": true, "amazonde": true, "amazones": true, "amazonit": true, "amazonnl": true, "amazoncombe": true,
		"amazoneusarl": true, "amazoneusàrl": true, "amazoneuropesarl": true, "amazoneuropecoresarl": true,
		"amazonwarehouse": true, "amazonresale": true, "amazonsecondemain": true,
	}
	return known[normalized]
}

func requestError(item model.Listing, err error) model.CollectionResult {
	fmt.Printf("Amazon collection failed for %s/%s: %v\n", item.Marketplace, item.ASIN, err)
	return model.CollectionResult{Result: "request_error", Message: "Amazon could not be reached for a price check.", SecondHandStatus: "check_error"}
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
			wrappers := priceWrapperPattern.FindAllStringSubmatchIndex(prefix, -1)
			if len(wrappers) == 0 {
				continue
			}
			lastWrapper := wrappers[len(wrappers)-1]
			wrapper := strings.ToLower(prefix[lastWrapper[2]:lastWrapper[3]])
			if strings.Contains(class, "a-text-price") || regexp.MustCompile(`a-text-price|price-per-unit|unit-price|installment|saving|coupon|listprice`).MatchString(wrapper) {
				continue
			}
			if isInstalmentPrice(page, prefixStart+lastWrapper[0], prefix) {
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

// isInstalmentPrice reports whether the a-price element starting at wrapperStart
// is a monthly-payment amount: followed by text such as "x4 mois" or "x4 (", or enclosed
// in an instalment container opened within the prefix window.
func isInstalmentPrice(page string, wrapperStart int, prefix string) bool {
	end := elementEnd(page, wrapperStart)
	text := textContent(page[end:min(len(page), end+2_000)])
	if instalmentText.MatchString(text[:min(len(text), 300)]) {
		return true
	}
	for _, tag := range openTags(prefix) {
		if instalmentContainer.MatchString(attribute(tag, "id")) || instalmentContainer.MatchString(attribute(tag, "class")) {
			return true
		}
	}
	return false
}

// elementEnd returns the index just after the closing tag of the element opened at start.
func elementEnd(page string, start int) int {
	depth := 0
	tagName := ""
	position := start
	for _, token := range htmlTokenPattern.FindAllString(page[start:min(len(page), start+20_000)], -1) {
		position += len(token)
		matches := tagNamePattern.FindStringSubmatch(token)
		if len(matches) < 2 {
			continue
		}
		name := strings.ToLower(matches[1])
		if tagName == "" {
			tagName = name
		}
		if name != tagName {
			continue
		}
		if strings.HasPrefix(token, "</") {
			depth--
			if depth == 0 {
				return position
			}
		} else {
			depth++
		}
	}
	return position
}

// openTags returns the opening tags still unclosed at the end of fragment.
func openTags(fragment string) []string {
	var stack []string
	for _, token := range htmlTokenPattern.FindAllString(fragment, -1) {
		matches := tagNamePattern.FindStringSubmatch(token)
		if len(matches) < 2 {
			continue
		}
		name := strings.ToLower(matches[1])
		if strings.HasPrefix(token, "</") {
			for index := len(stack) - 1; index >= 0; index-- {
				if strings.ToLower(tagNamePattern.FindStringSubmatch(stack[index])[1]) == name {
					stack = stack[:index]
					break
				}
			}
			continue
		}
		if !strings.HasSuffix(strings.TrimSpace(token), "/>") && !isVoidElement(name) {
			stack = append(stack, token)
		}
	}
	return stack
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
