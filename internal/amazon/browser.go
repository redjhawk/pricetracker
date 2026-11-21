package amazon

import (
	"context"
	"errors"
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

// BrowserCollector checks Amazon item pages through a headless Chromium.
type BrowserCollector struct {
	options BrowserOptions
}

// NewBrowserCollector returns a browser collector.
func NewBrowserCollector(options BrowserOptions) *BrowserCollector {
	return &BrowserCollector{options: options}
}

// Collect checks one Amazon item page.
// TDD red stub (issue #168): not implemented.
func (c *BrowserCollector) Collect(ctx context.Context, item model.Listing) model.CollectionResult {
	return model.CollectionResult{}
}

// FindBrowser returns the Chromium or Chrome executable found on PATH.
// TDD red stub (issue #168): not implemented.
func FindBrowser() (string, error) {
	return "", errors.New("not implemented")
}
