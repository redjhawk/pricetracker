package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"pricefollower.local/internal/model"
)

// BrowserOptions configures the browser collector (issue #168).
type BrowserOptions struct {
	ExecPath          string               // Chromium/Chrome executable; "" = FindBrowser()
	ProfileDir        string               // persistent profile (cookies); "" = temporary directory removed after the check
	NoSandbox         bool                 // adds --no-sandbox (containers, root)
	NavigationTimeout time.Duration        // 0 = 45 s; page load deadline
	Pause             func() time.Duration // nil = random 2–5 s after load ("reading"); tests return 0
	Origin            string               // tests only: replaces scheme+host of Amazon URLs and is accepted as a supported host
}

// BrowserCollector checks Amazon item pages through a headless Chromium, which runs the page's
// scripts, loads its resources and keeps its cookies like a person's browser. It starts one browser
// per check and never shows it.
type BrowserCollector struct {
	options BrowserOptions
	mu      sync.Mutex // one browser at a time (one profile)
}

// NewBrowserCollector returns a browser collector.
func NewBrowserCollector(options BrowserOptions) *BrowserCollector {
	return &BrowserCollector{options: options}
}

// FindBrowser returns PRICEFOLLOWER_BROWSER_PATH, or else the Chromium or Chrome executable found on PATH.
func FindBrowser() (string, error) {
	if path := os.Getenv("PRICEFOLLOWER_BROWSER_PATH"); path != "" {
		return path, nil
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no Chromium or Google Chrome found on PATH")
}

const (
	amazonCheckFailedMessage = "Amazon could not be reached for a price check."
	unavailableMessage       = "The listing is no longer available."
	maxPageBytes             = 5 << 20
)

var robotCheckPattern = regexp.MustCompile(`(?i)captcha|robot check|enter the characters you see below`)

// loadedPage is the main document of a page loaded by the browser.
type loadedPage struct {
	status   int
	mimeType string
	url      string // after redirects
	html     string // rendered DOM
}

// Collect checks one Amazon item page: the product page, then, without an Amazon second-hand offer
// on it, the used-offer requests made from inside the page.
func (c *BrowserCollector) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := model.CollectionResult{RequestURL: item.URL}
	tab, err := c.open(ctx, item)
	if err != nil {
		return requestFailure(item, result, err.Error())
	}
	defer tab.close()
	page, err := tab.load(ctx, c.target(item.URL), c.navigationTimeout())
	if err != nil {
		return requestFailure(item, result, err.Error())
	}
	result.HTTPStatus = page.status
	if !c.pause(ctx) {
		return requestFailure(item, result, ctx.Err().Error())
	}
	if page.html, err = tab.html(ctx); err != nil {
		return requestFailure(item, result, err.Error())
	}
	unavailable, err := c.classifyPage(page)
	if err != nil {
		result.ResponseExcerpt = excerpt([]byte(page.html))
		return requestFailure(item, result, err.Error())
	}
	product := ""
	if unavailable {
		result.Result, result.Message = "unavailable", unavailableMessage
	} else {
		result, product = parseProductPage(result, page.html)
	}
	if product != "" {
		result.SecondHandStatus, result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel = secondHandOfferFromProductPage(product)
		if result.SecondHandStatus == "available" {
			return result
		}
	}
	urls := usedOfferURLs(item)
	for index := range urls {
		urls[index] = c.target(urls[index])
	}
	pages, failedResponses := tab.fetchAll(ctx, urls)
	result.SecondHandStatus, result.SecondHandAmountCents, result.SecondHandCondition, result.SecondHandConditionLabel = parseUsedOffers(item, pages, failedResponses)
	return result
}

// requestFailure returns the failure of a check that got no usable Amazon answer.
func requestFailure(item model.Listing, result model.CollectionResult, reason string) model.CollectionResult {
	printRequestError(item, reason)
	result.Result, result.Message, result.SecondHandStatus = "request_error", amazonCheckFailedMessage, "check_error"
	return result
}

// classifyPage applies the Amazon answer rules to the loaded product page: 404/410 is an unavailable
// listing; another non-2xx status, an unexpected content type, a robot check or a redirect to an
// unsupported host is a failure.
func (c *BrowserCollector) classifyPage(page loadedPage) (bool, error) {
	if !c.supportedURL(page.url) {
		return false, errors.New("redirected to an unsupported host")
	}
	if page.status == 404 || page.status == 410 {
		return true, nil
	}
	if page.status < 200 || page.status >= 300 {
		return false, fmt.Errorf("Amazon returned HTTP %d", page.status)
	}
	if page.mimeType != "text/html" && page.mimeType != "application/xhtml+xml" {
		return false, errors.New("Amazon returned an unexpected content type")
	}
	if robotCheckPattern.MatchString(page.html[:min(len(page.html), 200_000)]) {
		return false, errors.New("Amazon blocked the product request")
	}
	return false, nil
}

// target returns the URL the browser requests for an Amazon URL (the test origin replaces Amazon's).
func (c *BrowserCollector) target(raw string) string {
	if c.options.Origin == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	origin, originErr := url.Parse(c.options.Origin)
	if err != nil || originErr != nil {
		return raw
	}
	parsed.Scheme, parsed.Host = origin.Scheme, origin.Host
	return parsed.String()
}

func (c *BrowserCollector) supportedURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if origin, err := url.Parse(c.options.Origin); c.options.Origin != "" && err == nil && parsed.Scheme == origin.Scheme && parsed.Host == origin.Host {
		return true
	}
	return parsed.Scheme == "https" && hosts[strings.ToLower(strings.TrimPrefix(parsed.Hostname(), "www."))]
}

func (c *BrowserCollector) navigationTimeout() time.Duration {
	if c.options.NavigationTimeout > 0 {
		return c.options.NavigationTimeout
	}
	return 45 * time.Second
}

// pause waits like a person reading the page; it returns false when ctx ends first.
func (c *BrowserCollector) pause(ctx context.Context) bool {
	delay := time.Duration(2000+rand.IntN(3001)) * time.Millisecond
	if c.options.Pause != nil {
		delay = c.options.Pause()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// browserTab is a page of a browser started for one check.
type browserTab struct {
	browser    *cdpBrowser
	session    string
	removeTemp func()
}

// open starts the browser and opens a page set up with the browser's own user agent (without its
// headless marker) and the marketplace language.
func (c *BrowserCollector) open(ctx context.Context, item model.Listing) (*browserTab, error) {
	executable := c.options.ExecPath
	if executable == "" {
		found, err := FindBrowser()
		if err != nil {
			return nil, fmt.Errorf("browser could not start: %w", err)
		}
		executable = found
	}
	profile, removeTemp, err := c.profile()
	if err != nil {
		return nil, fmt.Errorf("browser could not start: %w", err)
	}
	language, _, _ := strings.Cut(locale(item.Marketplace), ",")
	args := []string{"--headless=new", "--remote-debugging-pipe", "--user-data-dir=" + profile, "--no-first-run", "--no-default-browser-check",
		"--disable-blink-features=AutomationControlled", "--lang=" + language, "--window-size=1366,768"}
	if c.options.NoSandbox {
		args = append(args, "--no-sandbox")
	}
	browser, err := launchBrowser(executable, append(args, "about:blank"))
	if err != nil {
		removeTemp()
		return nil, fmt.Errorf("browser could not start: %w", err)
	}
	tab := &browserTab{browser: browser, removeTemp: removeTemp}
	if err := tab.setUp(ctx, locale(item.Marketplace)); err != nil {
		tab.close()
		return nil, fmt.Errorf("browser could not start: %w", err)
	}
	return tab, nil
}

// profile returns the browser profile directory and the function removing it when temporary.
func (c *BrowserCollector) profile() (string, func(), error) {
	if c.options.ProfileDir != "" {
		return c.options.ProfileDir, func() {}, os.MkdirAll(c.options.ProfileDir, 0o700)
	}
	directory, err := os.MkdirTemp("", "pricefollower-browser-")
	if err != nil {
		return "", nil, err
	}
	return directory, func() { os.RemoveAll(directory) }, nil
}

func (t *browserTab) setUp(ctx context.Context, acceptLanguage string) error {
	var version struct {
		Product   string `json:"product"`
		UserAgent string `json:"userAgent"`
	}
	if err := t.browser.call(ctx, "", "Browser.getVersion", nil, &version); err != nil {
		return err
	}
	var target struct {
		TargetID string `json:"targetId"`
	}
	if err := t.browser.call(ctx, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &target); err != nil {
		return err
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := t.browser.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &attached); err != nil {
		return err
	}
	t.session = attached.SessionID
	for _, method := range []string{"Network.enable", "Page.enable"} {
		if err := t.browser.call(ctx, t.session, method, nil, nil); err != nil {
			return err
		}
	}
	return t.browser.call(ctx, t.session, "Emulation.setUserAgentOverride", userAgentOverride(version.Product, version.UserAgent, acceptLanguage), nil)
}

// userAgentOverride returns the browser's own user agent and client hints without the headless marker.
func userAgentOverride(product, userAgent, acceptLanguage string) map[string]any {
	fullVersion := strings.TrimPrefix(strings.TrimPrefix(product, "HeadlessChrome/"), "Chrome/")
	major, _, _ := strings.Cut(fullVersion, ".")
	platform, architecture := "Linux x86_64", "x86"
	if runtime.GOARCH == "arm64" {
		platform, architecture = "Linux aarch64", "arm"
	}
	brands := []map[string]string{{"brand": "Chromium", "version": major}, {"brand": "Google Chrome", "version": major}, {"brand": "Not_A Brand", "version": "99"}}
	fullVersions := []map[string]string{{"brand": "Chromium", "version": fullVersion}, {"brand": "Google Chrome", "version": fullVersion}, {"brand": "Not_A Brand", "version": "99.0.0.0"}}
	return map[string]any{
		"userAgent":      strings.Replace(userAgent, "HeadlessChrome/", "Chrome/", 1),
		"acceptLanguage": acceptLanguage,
		"platform":       platform,
		"userAgentMetadata": map[string]any{"brands": brands, "fullVersionList": fullVersions, "fullVersion": fullVersion, "platform": "Linux",
			"platformVersion": "", "architecture": architecture, "model": "", "mobile": false, "bitness": "64"},
	}
}

// load navigates to target and waits for the page's load event within timeout. Without an error the
// page has the status, content type and final URL of the main document.
func (t *browserTab) load(ctx context.Context, target string, timeout time.Duration) (loadedPage, error) {
	var mu sync.Mutex
	documents := map[string]loadedPage{} // last document response by frame
	responded := make(chan struct{}, 1)  // signalled when a document response is recorded
	loaded := make(chan struct{})
	var loadedOnce sync.Once
	t.browser.on(t.session, func(method string, params json.RawMessage) {
		switch method {
		case "Network.responseReceived":
			var event struct {
				Type     string `json:"type"`
				FrameID  string `json:"frameId"`
				Response struct {
					URL      string `json:"url"`
					Status   int    `json:"status"`
					MimeType string `json:"mimeType"`
				} `json:"response"`
			}
			if json.Unmarshal(params, &event) == nil && event.Type == "Document" {
				mu.Lock()
				documents[event.FrameID] = loadedPage{status: event.Response.Status, mimeType: event.Response.MimeType, url: event.Response.URL}
				mu.Unlock()
				select {
				case responded <- struct{}{}:
				default:
				}
			}
		case "Page.loadEventFired":
			mu.Lock()
			navigated := len(documents) > 0 // not the load of the initial about:blank
			mu.Unlock()
			if navigated {
				loadedOnce.Do(func() { close(loaded) })
			}
		}
	})
	loadContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var navigation struct {
		FrameID   string `json:"frameId"`
		ErrorText string `json:"errorText"`
	}
	if err := t.browser.call(loadContext, t.session, "Page.navigate", map[string]any{"url": target}, &navigation); err != nil {
		return loadedPage{}, loadError(ctx, err)
	}
	document := func() (loadedPage, bool) {
		mu.Lock()
		defer mu.Unlock()
		page, found := documents[navigation.FrameID]
		return page, found
	}
	if navigation.ErrorText != "" {
		// A non-2xx answer with an empty body also ends with an error text (Chromium then shows its own
		// error page): its status still classifies the page, so only a navigation without an answer fails.
		if page, found := awaitDocument(loadContext, document, responded); found {
			return page, nil
		}
		return loadedPage{}, fmt.Errorf("Amazon page could not be opened: %s", navigation.ErrorText)
	}
	select {
	case <-loaded:
	case <-loadContext.Done():
		return loadedPage{}, loadError(ctx, loadContext.Err())
	}
	page, found := document()
	if !found {
		return loadedPage{}, errors.New("Amazon page loaded without a response")
	}
	return page, nil
}

// awaitDocument returns the navigated document's response, waiting up to a second for it since the
// browser may report the failed navigation before the response event.
func awaitDocument(ctx context.Context, document func() (loadedPage, bool), responded <-chan struct{}) (loadedPage, bool) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		if page, found := document(); found {
			return page, true
		}
		select {
		case <-responded:
		case <-timer.C:
			return loadedPage{}, false
		case <-ctx.Done():
			return loadedPage{}, false
		}
	}
}

// loadError keeps the cancellation of the check and reports the end of the load deadline.
func loadError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("Amazon page did not finish loading")
	}
	return err
}

// html returns the rendered DOM of the page (at most 5 MB).
func (t *browserTab) html(ctx context.Context) (string, error) {
	var html string
	err := t.evaluate(ctx, fmt.Sprintf(`document.documentElement ? document.documentElement.outerHTML.slice(0, %d) : ""`, maxPageBytes), &html)
	return html, err
}

// fetchAll requests urls in turn from inside the page, with its cookies, and returns the bodies of
// the 2xx answers and the reasons of the failed requests.
func (t *browserTab) fetchAll(ctx context.Context, urls []string) ([]string, []string) {
	encoded, err := json.Marshal(urls)
	if err != nil {
		return nil, []string{err.Error()}
	}
	script := fmt.Sprintf(`(async () => {
  const responses = [];
  for (const url of %s) {
    try {
      const response = await fetch(url, {credentials: "include", headers: {"X-Requested-With": "XMLHttpRequest"}});
      responses.push({status: response.status, body: (await response.text()).slice(0, %d)});
    } catch (error) {
      responses.push({status: 0, error: String(error)});
    }
  }
  return responses;
})()`, encoded, maxPageBytes)
	var responses []struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
		Error  string `json:"error"`
	}
	if err := t.evaluate(ctx, script, &responses); err != nil {
		return nil, []string{err.Error()}
	}
	var pages, failedResponses []string
	for _, response := range responses {
		switch {
		case response.Error != "":
			failedResponses = append(failedResponses, response.Error)
		case response.Status >= 200 && response.Status < 300:
			pages = append(pages, response.Body)
		default:
			failedResponses = append(failedResponses, fmt.Sprintf("HTTP %d", response.Status))
		}
	}
	return pages, failedResponses
}

// evaluate runs a script in the page, awaiting its promise, and stores its value in value.
func (t *browserTab) evaluate(ctx context.Context, script string, value any) error {
	var evaluation struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := t.browser.call(ctx, t.session, "Runtime.evaluate", map[string]any{"expression": script, "awaitPromise": true, "returnByValue": true}, &evaluation); err != nil {
		return err
	}
	if evaluation.ExceptionDetails != nil {
		return fmt.Errorf("page script failed: %s", evaluation.ExceptionDetails.Text)
	}
	return json.Unmarshal(evaluation.Result.Value, value)
}

// close ends the browser and removes a temporary profile.
func (t *browserTab) close() {
	t.browser.close()
	t.removeTemp()
}
