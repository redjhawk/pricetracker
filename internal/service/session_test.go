package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

type sessionFixtureTransport func(*http.Request) (*http.Response, error)

func (f sessionFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestSessionSharedByImmediateManualAndScheduledCollection(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	sessionFile := filepath.Join(dir, "session.json")
	data, _ := json.Marshal(map[string]any{"version": 1, "capturedAt": time.Now().UTC().Format(time.RFC3339Nano), "cookie": map[string]any{"name": "datadome", "value": "synthetic-service", "domain": ".leboncoin.fr", "path": "/", "secure": true, "expiresAt": nil}})
	if err := os.WriteFile(sessionFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{DataDirectory: dir, Location: time.UTC, CheckTimes: []string{"08:00", "20:00"}, StaleAfter: 36 * time.Hour, LeboncoinSessionFile: sessionFile}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var calls atomic.Int32
	originalTransport := http.DefaultTransport
	http.DefaultTransport = sessionFixtureTransport(func(r *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		if r.URL.Host != "www.leboncoin.fr" || r.Header.Get("Cookie") != "datadome=synthetic-service" {
			t.Error("configured session not scoped to LeBoncoin")
		}
		status := 200
		body := `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","price":[12]}}}}</script>`
		if call == 3 {
			status = 403
			body = "challenge"
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"text/html"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	defer func() { http.DefaultTransport = originalTransport }()
	service := New(cfg, database)
	defer service.Close()
	item, err := service.Add(context.Background(), "https://www.leboncoin.fr/ad/test/123")
	if err != nil {
		t.Fatal(err)
	}
	waitForResult := func(want string) model.Item {
		t.Helper()
		// Every call is made after scheduling has completed; wait for the real
		// worker before reading the store's independently assembled item fields.
		service.workers.Wait()
		current, err := service.Get(context.Background(), item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.LastAttempt == nil || current.LastAttempt.Result != want {
			t.Fatalf("did not observe %s", want)
		}
		return current
	}
	first := waitForResult("success")
	if first.LatestPrice == nil || first.LatestPrice.AmountCents != 1200 {
		t.Fatal("initial observation missing")
	}
	if _, err := service.RefreshItem(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	waitForResult("success")
	if err := database.SetNextCheck(context.Background(), item.ID, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	service.collectDue(context.Background())
	failed := waitForResult("request_error")
	if calls.Load() != 3 || failed.LatestPrice == nil || failed.LatestPrice.AmountCents != 1200 || len(failed.PriceHistory) != 2 {
		t.Fatalf("calls=%d or latest successful price lost", calls.Load())
	}
	// A broken opt-in does not prevent startup or collection on another platform.
	cfg.LeboncoinSessionFile = filepath.Join(dir, "missing.json")
	unrelated := New(cfg, database)
	defer unrelated.Close()
	http.DefaultTransport = sessionFixtureTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Error("session leaked to Amazon")
		}
		return &http.Response{StatusCode: 403, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	unrelated.collectors["amazon"].Collect(context.Background(), model.Listing{Platform: "amazon", ListingID: "B012345678", ASIN: "B012345678", URL: "https://www.amazon.fr/dp/B012345678"})
}
