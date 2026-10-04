package leboncoin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pricefollower.local/internal/model"
)

var (
	listingPathPattern = regexp.MustCompile(`(?i)^/ad/([a-z0-9_-]+)/([0-9]+),?$`)
	nextDataPattern    = regexp.MustCompile(`(?is)<script\b[^>]*\bid=["']__NEXT_DATA__["'][^>]*>(.*?)</script>`)
	donationPattern    = regexp.MustCompile(`(?i)\b(?:don|donne|donnent|donnons|offert|offerte|offerts|offertes|gratuit|gratuite|gratuitement|à donner|a donner)\b`)
)

type URLResult struct {
	Kind        string
	ListingID   string
	Marketplace string
	URL         string
	Canonical   string
}

func ParseURL(raw string) URLResult {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return URLResult{Kind: "invalid"}
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if strings.HasPrefix(host, "www.") {
		host = strings.TrimPrefix(host, "www.")
	}
	if !strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil || parsed.Port() != "" && parsed.Port() != "443" || host != "leboncoin.fr" {
		return URLResult{Kind: "unsupported"}
	}
	match := listingPathPattern.FindStringSubmatch(parsed.Path)
	if len(match) != 3 {
		return URLResult{Kind: "unsupported"}
	}
	category, listingID := strings.ToLower(match[1]), match[2]
	canonical := "https://www.leboncoin.fr/ad/" + category + "/" + listingID
	return URLResult{Kind: "valid", ListingID: listingID, Marketplace: "leboncoin.fr", URL: canonical, Canonical: canonical}
}

type Collector struct {
	client    *http.Client
	userAgent string
}

func NewCollector(userAgent string) *Collector {
	return &Collector{
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= 4 {
					return fmt.Errorf("too many LeBoncoin redirects")
				}
				host := strings.ToLower(strings.TrimPrefix(request.URL.Hostname(), "www."))
				if request.URL.Scheme != "https" || host != "leboncoin.fr" {
					return fmt.Errorf("redirected to an unsupported host")
				}
				return nil
			},
		},
		userAgent: userAgent,
	}
}

type pageData struct {
	Props struct {
		PageProps struct {
			Ad listingData `json:"ad"`
		} `json:"pageProps"`
	} `json:"props"`
}

type listingData struct {
	ListingID  int64         `json:"list_id"`
	Status     string        `json:"status"`
	Subject    string        `json:"subject"`
	Body       string        `json:"body"`
	PriceCents *int64        `json:"price_cents"`
	Price      []json.Number `json:"price"`
	Images     adImages      `json:"images"`
	// Fields below are only used for AI review listing details.
	// Raw so that an unexpected shape never breaks price parsing; see listingDetails.
	Attributes           []json.RawMessage `json:"attributes"`
	Location             json.RawMessage   `json:"location"`
	Owner                json.RawMessage   `json:"owner"`
	FirstPublicationDate string            `json:"first_publication_date"`
	CategoryName         string            `json:"category_name"`
}

type adImages struct {
	ThumbnailURL string   `json:"thumb_url"`
	SmallURL     string   `json:"small_url"`
	URLsLarge    []string `json:"urls_large"`
	URLs         []string `json:"urls"`
}

// CollectWithSession collects a listing. With a nil session it uses the
// sessionless path; otherwise it sends the saved datadome cookie only to
// allowed LeBoncoin HTTPS origins and reports the session outcome.
func (c *Collector) CollectWithSession(ctx context.Context, item model.Listing, session *Session) (model.CollectionResult, SessionOutcome) {
	if session == nil {
		return c.collect(ctx, item, c.client, nil), SessionOutcome{}
	}
	attempt := &sessionAttempt{candidate: &Session{Value: session.Value, ExpiresAt: session.ExpiresAt}}
	parsed := ParseURL(item.URL)
	if parsed.Kind != "valid" || parsed.ListingID != item.ListingID {
		// No request was sent, so there is no session outcome to record.
		return c.collectionError(item, fmt.Errorf("unsupported listing URL"), attempt), SessionOutcome{}
	}
	item.URL = parsed.Canonical
	local := *c.client
	local.Jar = attempt
	local.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		request.Header.Del("Cookie")
		if len(via) >= 4 || !allowedSessionURL(request.URL) {
			return fmt.Errorf("unsupported LeBoncoin session redirect")
		}
		return nil
	}
	result := c.collect(ctx, item, &local, attempt)
	return result, attempt.outcome()
}

func (c *Collector) collect(ctx context.Context, item model.Listing, client *http.Client, attempt *sessionAttempt) model.CollectionResult {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return c.collectionError(item, err, attempt)
	}
	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	request.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
	request.Header.Set("Upgrade-Insecure-Requests", "1")
	chromeMajor := "154"
	if match := regexp.MustCompile(`Chrome/(\d+)`).FindStringSubmatch(c.userAgent); len(match) > 1 {
		chromeMajor = match[1]
	}
	request.Header.Set("Sec-CH-UA", `"Google Chrome";v="`+chromeMajor+`", "Chromium";v="`+chromeMajor+`", "Not_A Brand";v="99"`)
	request.Header.Set("Sec-CH-UA-Mobile", "?0")
	request.Header.Set("Sec-CH-UA-Platform", `"Linux"`)
	request.Header.Set("Sec-Fetch-Dest", "document")
	request.Header.Set("Sec-Fetch-Mode", "navigate")
	request.Header.Set("Sec-Fetch-Site", "none")
	request.Header.Set("Sec-Fetch-User", "?1")

	startedAt := time.Now()
	if attempt == nil {
		log.Printf("LeBoncoin request started listing=%s url=%s", item.ListingID, request.URL.String())
	} else {
		log.Printf("LeBoncoin session request started listing=%s", item.ListingID)
	}
	response, err := client.Do(request)
	if err != nil {
		return c.collectionError(item, err, attempt)
	}
	defer response.Body.Close()
	if attempt == nil {
		log.Printf("LeBoncoin response listing=%s url=%s status=%d content_type=%q duration=%s", item.ListingID, request.URL.String(), response.StatusCode, response.Header.Get("Content-Type"), time.Since(startedAt).Round(time.Millisecond))
	} else {
		log.Printf("LeBoncoin session response listing=%s status=%d duration=%s", item.ListingID, response.StatusCode, time.Since(startedAt).Round(time.Millisecond))
	}
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		log.Printf("LeBoncoin listing unavailable listing=%s status=%d", item.ListingID, response.StatusCode)
		return model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available."}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if attempt != nil && response.StatusCode == http.StatusForbidden {
			attempt.rejected = true
		}
		return c.collectionError(item, fmt.Errorf("LeBoncoin returned HTTP %d", response.StatusCode), attempt)
	}
	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if mediaType != "text/html" && mediaType != "application/xhtml+xml" {
		return c.collectionError(item, fmt.Errorf("LeBoncoin returned an unexpected content type: %q", response.Header.Get("Content-Type")), attempt)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 5<<20))
	if err != nil {
		return c.collectionError(item, err, attempt)
	}
	if attempt != nil && bytes.Contains(body, []byte("captcha-delivery.com")) {
		attempt.rejected = true // DataDome challenge marker; verification below still takes precedence
	}
	match := nextDataPattern.FindSubmatch(body)
	if len(match) != 2 {
		return c.collectionError(item, fmt.Errorf("LeBoncoin __NEXT_DATA__ listing script was not present in the %d-byte page", len(body)), attempt)
	}
	var data pageData
	if err := json.Unmarshal(match[1], &data); err != nil {
		return c.collectionError(item, fmt.Errorf("parse LeBoncoin listing data: %w", err), attempt)
	}
	ad := data.Props.PageProps.Ad
	if ad.ListingID == 0 {
		return c.collectionError(item, fmt.Errorf("LeBoncoin page did not contain an ad"), attempt)
	}
	expectedID, err := strconv.ParseInt(item.ListingID, 10, 64)
	if err != nil || ad.ListingID != expectedID {
		return c.collectionError(item, fmt.Errorf("LeBoncoin page ad ID %d did not match requested listing %s", ad.ListingID, item.ListingID), attempt)
	}
	if attempt != nil {
		attempt.verified = true
	}
	if ad.Status != "active" {
		if attempt == nil {
			log.Printf("LeBoncoin listing unavailable listing=%s ad_status=%q", item.ListingID, ad.Status)
		}
		return model.CollectionResult{Result: "unavailable", Message: "The listing is no longer available."}
	}
	title := strings.TrimSpace(ad.Subject)
	result := model.CollectionResult{Result: "price_not_found", Message: "LeBoncoin did not show a detectable euro price."}
	if title != "" {
		result.Title = &title
	}
	if thumbnail := firstImage(ad.Images); thumbnail != "" {
		result.ThumbnailURL = &thumbnail
	}
	if amount, ok := listedPriceCents(ad); ok {
		result.Result = "success"
		result.AmountCents = amount
		result.Message = ""
		result.Listing = listingDetails(ad, &amount)
		log.Printf("LeBoncoin price parsed listing=%s amount_cents=%d", item.ListingID, amount)
		return result
	}
	if donationPattern.MatchString(title + " " + ad.Body) {
		result.Result = "success"
		result.AmountCents = 0
		result.Message = ""
		result.Listing = listingDetails(ad, &result.AmountCents)
		log.Printf("LeBoncoin donation parsed listing=%s amount_cents=0", item.ListingID)
		return result
	}
	result.Listing = listingDetails(ad, nil)
	log.Printf("LeBoncoin price not found listing=%s price_cents_present=%t price_values=%d", item.ListingID, ad.PriceCents != nil, len(ad.Price))
	return result
}

func listedPriceCents(ad listingData) (int64, bool) {
	if ad.PriceCents != nil {
		return *ad.PriceCents, *ad.PriceCents >= 0
	}
	if len(ad.Price) != 1 {
		return 0, false
	}
	return parsePrice(ad.Price[0].String())
}

func firstImage(images adImages) string {
	for _, candidate := range []string{images.ThumbnailURL, images.SmallURL} {
		if strings.HasPrefix(candidate, "https://img.leboncoin.fr/") {
			return candidate
		}
	}
	for _, group := range [][]string{images.URLsLarge, images.URLs} {
		for _, candidate := range group {
			if strings.HasPrefix(candidate, "https://img.leboncoin.fr/") {
				return candidate
			}
		}
	}
	return ""
}

// collectionError never logs session-assisted error details, which could echo
// cookie values or upstream content.
func (c *Collector) collectionError(item model.Listing, err error, attempt *sessionAttempt) model.CollectionResult {
	if attempt != nil {
		if attempt.rejected {
			log.Printf("LeBoncoin rejected the saved session listing=%s", item.ListingID)
			return model.CollectionResult{Result: "request_error", Message: "LeBoncoin rejected the saved session. Capture a new session and save it in Settings."}
		}
		log.Printf("LeBoncoin session-assisted check failed listing=%s", item.ListingID)
		return model.CollectionResult{Result: "request_error", Message: "LeBoncoin could not be reached for a price check."}
	}
	log.Printf("LeBoncoin collection error listing=%s url=%s: %v", item.ListingID, item.URL, err)
	return model.CollectionResult{Result: "request_error", Message: "LeBoncoin could not be reached for a price check."}
}

func parsePrice(value string) (int64, bool) {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil || amount < 0 {
		return 0, false
	}
	return int64(amount*100 + 0.5), true
}
