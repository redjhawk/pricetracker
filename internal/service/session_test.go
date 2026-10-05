package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/leboncoin"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

type sessionFixtureTransport func(*http.Request) (*http.Response, error)

func (f sessionFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const serviceListingHTML = `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","price":[12]}}}}</script>`

const (
	rejectedSessionMessage = "LeBoncoin rejected the saved session. Capture a new session and save it in Settings."
	unusableSessionMessage = "The saved LeBoncoin session has expired or was revoked. Save a new session in Settings."
	unreadSessionMessage   = "The LeBoncoin session could not be read. The check will be retried at the next scheduled time."
)

type fakeLeboncoin func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome)

func (f fakeLeboncoin) CollectWithSession(ctx context.Context, item model.Listing, session *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
	return f(ctx, item, session)
}

func newSessionService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	cfg := config.Config{DataDirectory: t.TempDir(), Location: time.UTC, CheckTimes: []string{"08:00", "20:00"}, StaleAfter: 36 * time.Hour, UserAgent: "test"}
	database, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	service := New(cfg, database)
	t.Cleanup(service.Close)
	return service, database
}

func insertListing(t *testing.T, database *store.Store, id, platform, url string) {
	t.Helper()
	listing := model.Listing{ID: id, Platform: platform, ListingID: "123", Marketplace: "leboncoin.fr", URL: url}
	if platform == "amazon" {
		listing.ListingID, listing.ASIN, listing.Marketplace = "B012345678", "B012345678", "amazon.fr"
	}
	if err := database.Insert(context.Background(), listing, url, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func withTransport(t *testing.T, transport sessionFixtureTransport) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
}

func saveSession(t *testing.T, service *Service, raw string) model.LeboncoinSession {
	t.Helper()
	current, err := service.LeboncoinSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	saved, err := service.SaveLeboncoinSession(context.Background(), raw, current.Revision)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

func lastAttempt(t *testing.T, service *Service, id string) model.Item {
	t.Helper()
	item, err := service.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if item.LastAttempt == nil {
		t.Fatal("no attempt recorded")
	}
	return item
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buffer)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &buffer
}

func TestSaveLeboncoinSessionValidationAndConflict(t *testing.T) {
	service, database := newSessionService(t)
	ctx := context.Background()
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	saved, err := service.SaveLeboncoinSession(ctx, " Cookie: a=1; datadome=synthetic-saved; b=2 ", 0)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Value == nil || *saved.Value != "synthetic-saved" || saved.Revision != 1 || saved.Status != "active" {
		t.Fatalf("saved %+v", saved)
	}
	var typed *Error
	if _, err := service.SaveLeboncoinSession(ctx, "a=1", 1); !errors.As(err, &typed) || typed.Status != 400 || typed.Code != "INVALID_SESSION" || typed.Message != "No datadome cookie was found in the pasted text." {
		t.Fatalf("invalid input error %v", err)
	}
	if _, err := service.SaveLeboncoinSession(ctx, "other", 0); !errors.As(err, &typed) || typed.Status != 409 || typed.Code != "SESSION_CHANGED" || typed.Message != "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving." {
		t.Fatalf("conflict error %v", err)
	}
	current, _ := service.LeboncoinSession(ctx)
	if *current.Value != "synthetic-saved" || current.Revision != 1 {
		t.Fatalf("rejected saves changed the session %+v", current)
	}
	cleared, err := service.SaveLeboncoinSession(ctx, "   ", 1)
	if err != nil || cleared.Value != nil || cleared.Status != "none" {
		t.Fatalf("clear %+v %v", cleared, err)
	}
	// Saving never queues a collection.
	service.mu.Lock()
	inFlight := len(service.inFlight)
	service.mu.Unlock()
	if item, _ := service.Get(ctx, "lbc"); inFlight != 0 || item.LastAttempt != nil {
		t.Fatal("saving started a check")
	}
}

func TestSessionlessAndAmazonCollectionSendNoCookie(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	insertListing(t, database, "amz", "amazon", "https://www.amazon.fr/dp/B012345678")
	var calls atomic.Int32
	withTransport(t, func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Header.Get("Cookie") != "" {
			t.Errorf("cookie sent to %s", r.URL.Host)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/html"}}, Body: io.NopCloser(strings.NewReader(serviceListingHTML)), Request: r}, nil
	})
	service.Collect(context.Background(), "lbc")
	if item := lastAttempt(t, service, "lbc"); item.LastAttempt.Result != "success" {
		t.Fatalf("sessionless result %+v", item.LastAttempt)
	}
	saveSession(t, service, "synthetic-amazon-guard")
	service.Collect(context.Background(), "amz")
	if calls.Load() < 2 {
		t.Fatal("Amazon was not requested")
	}
	session, _ := service.LeboncoinSession(context.Background())
	if session.LastAttempt != nil {
		t.Fatal("sessionless or Amazon attempt recorded as session attempt")
	}
}

func TestSessionUsedRenewedAndReusedAfterRestart(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	logs := captureLogs(t)
	saved := saveSession(t, service, "datadome=synthetic-first")
	var sent []string
	var mu sync.Mutex
	withTransport(t, func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		sent = append(sent, r.Header.Get("Cookie"))
		mu.Unlock()
		h := http.Header{"Content-Type": []string{"text/html"}}
		h.Add("Set-Cookie", "datadome=synthetic-renewed; Domain=.leboncoin.fr; Path=/; Max-Age=3600; Secure")
		return &http.Response{StatusCode: 200, Header: h, Body: io.NopCloser(strings.NewReader(serviceListingHTML)), Request: r}, nil
	})
	service.Collect(context.Background(), "lbc")
	session, _ := service.LeboncoinSession(context.Background())
	if session.Value == nil || *session.Value != "synthetic-renewed" || session.Revision != saved.Revision+1 || session.ExpiresAt == nil || session.LastAttempt == nil || session.LastAttempt.Outcome != "accepted" {
		t.Fatalf("renewal not stored %+v", session)
	}
	restarted := New(service.config, database)
	defer restarted.Close()
	restarted.Collect(context.Background(), "lbc")
	if len(sent) != 2 || sent[0] != "datadome=synthetic-first" || sent[1] != "datadome=synthetic-renewed" {
		t.Fatalf("cookies sent %q", sent)
	}
	item := lastAttempt(t, restarted, "lbc")
	body, _ := json.Marshal(item)
	if strings.Contains(string(body)+logs.String(), "synthetic-") {
		t.Fatal("session value disclosed in item response or logs")
	}
}

func TestSessionRejectionKeepsValueAndPrice(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	logs := captureLogs(t)
	status := 200
	withTransport(t, func(r *http.Request) (*http.Response, error) {
		h := http.Header{"Content-Type": []string{"text/html"}}
		if status == 403 {
			h.Add("Set-Cookie", "datadome=synthetic-unverified; Domain=.leboncoin.fr; Path=/")
		}
		return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(serviceListingHTML)), Request: r}, nil
	})
	service.Collect(context.Background(), "lbc")
	saved := saveSession(t, service, "synthetic-kept")
	status = 403
	service.Collect(context.Background(), "lbc")
	item := lastAttempt(t, service, "lbc")
	if item.LastAttempt.Result != "request_error" || item.LastAttempt.Message == nil || *item.LastAttempt.Message != rejectedSessionMessage || item.LatestPrice == nil || item.LatestPrice.AmountCents != 1200 {
		t.Fatalf("rejection %+v %+v", item.LastAttempt, item.LatestPrice)
	}
	session, _ := service.LeboncoinSession(context.Background())
	if *session.Value != "synthetic-kept" || session.Revision != saved.Revision || session.Status != "active" || session.LastAttempt == nil || session.LastAttempt.Outcome != "rejected" {
		t.Fatalf("session after rejection %+v", session)
	}
	if strings.Contains(logs.String(), "synthetic-") {
		t.Fatal("session value logged")
	}
}

func TestSessionRevocationStopsSendingUntilNewSave(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	saved := saveSession(t, service, "synthetic-revoked")
	var calls atomic.Int32
	withTransport(t, func(r *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		h := http.Header{"Content-Type": []string{"text/html"}}
		status := 200
		if call == 1 {
			status = 403
			h.Add("Set-Cookie", "datadome=; Domain=.leboncoin.fr; Path=/; Max-Age=0")
		} else if r.Header.Get("Cookie") != "datadome=synthetic-new" {
			t.Errorf("unexpected cookie %q", r.Header.Get("Cookie"))
		}
		return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(serviceListingHTML)), Request: r}, nil
	})
	service.Collect(context.Background(), "lbc")
	session, _ := service.LeboncoinSession(context.Background())
	if session.Status != "revoked" || session.Revision != saved.Revision+1 || session.RevokedAt == nil {
		t.Fatalf("revocation %+v", session)
	}
	service.Collect(context.Background(), "lbc")
	item := lastAttempt(t, service, "lbc")
	if calls.Load() != 1 || item.LastAttempt.Result != "request_error" || *item.LastAttempt.Message != unusableSessionMessage {
		t.Fatalf("revoked session used: calls=%d attempt=%+v", calls.Load(), item.LastAttempt)
	}
	saveSession(t, service, "synthetic-new")
	service.Collect(context.Background(), "lbc")
	if item := lastAttempt(t, service, "lbc"); calls.Load() != 2 || item.LastAttempt.Result != "success" {
		t.Fatalf("new session not used: calls=%d", calls.Load())
	}
}

func TestExpiredSessionMakesNoRequest(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	saved := saveSession(t, service, "synthetic-expiring")
	past := time.Now().Add(-time.Minute)
	if _, err := database.FinishLeboncoinSessionAttempt(context.Background(), 0, saved.Revision, store.LeboncoinSessionOutcome{Attempt: "accepted", Renewed: true, Value: "synthetic-expiring", ExpiresAt: &past}, time.Now()); err != nil {
		t.Fatal(err)
	}
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		t.Error("expired session reached the collector")
		return model.CollectionResult{}, leboncoin.SessionOutcome{}
	})
	service.Collect(context.Background(), "lbc")
	item := lastAttempt(t, service, "lbc")
	if item.LastAttempt.Result != "request_error" || *item.LastAttempt.Message != unusableSessionMessage {
		t.Fatalf("attempt %+v", item.LastAttempt)
	}
	if session, _ := service.LeboncoinSession(context.Background()); session.Status != "expired" {
		t.Fatalf("status %s", session.Status)
	}
}

func TestOperatorSaveDuringAttemptWins(t *testing.T) {
	for _, replacement := range []string{"synthetic-operator", ""} {
		t.Run("replacement="+replacement, func(t *testing.T) {
			service, database := newSessionService(t)
			insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
			saveSession(t, service, "synthetic-old")
			entered := make(chan struct{})
			release := make(chan struct{})
			service.leboncoin = fakeLeboncoin(func(_ context.Context, _ model.Listing, session *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
				if session == nil || session.Value != "synthetic-old" {
					t.Errorf("session %+v", session)
				}
				close(entered)
				<-release
				return model.CollectionResult{Result: "success", AmountCents: 100}, leboncoin.SessionOutcome{Attempt: "accepted", Renewed: true, Value: "synthetic-stale-renewal"}
			})
			done := make(chan struct{})
			go func() { defer close(done); service.Collect(context.Background(), "lbc") }()
			<-entered
			saved := saveSession(t, service, replacement)
			close(release)
			<-done
			session, _ := service.LeboncoinSession(context.Background())
			if session.Revision != saved.Revision || session.LastAttempt != nil {
				t.Fatalf("stale attempt applied %+v", session)
			}
			if replacement == "" && session.Value != nil || replacement != "" && *session.Value != replacement {
				t.Fatalf("operator value lost %+v", session.Value)
			}
			if item := lastAttempt(t, service, "lbc"); item.LastAttempt.Result != "success" {
				t.Fatal("collection result not recorded")
			}
		})
	}
}

func TestConcurrentRenewalsApplyOnce(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "first", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	insertListing(t, database, "second", "leboncoin", "https://www.leboncoin.fr/ad/other/123")
	saved := saveSession(t, service, "synthetic-shared")
	var started sync.WaitGroup
	started.Add(2)
	release := make(chan struct{})
	var counter atomic.Int32
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		started.Done()
		<-release
		value := "synthetic-renewal-" + string(rune('a'+counter.Add(1)))
		return model.CollectionResult{Result: "success", AmountCents: 100}, leboncoin.SessionOutcome{Attempt: "accepted", Renewed: true, Value: value}
	})
	var done sync.WaitGroup
	for _, id := range []string{"first", "second"} {
		done.Add(1)
		go func() { defer done.Done(); service.Collect(context.Background(), id) }()
	}
	started.Wait()
	close(release)
	done.Wait()
	session, _ := service.LeboncoinSession(context.Background())
	if session.Revision != saved.Revision+1 || !strings.HasPrefix(*session.Value, "synthetic-renewal-") {
		t.Fatalf("renewals not applied exactly once %+v", session)
	}
}

func TestSessionWriteFailureKeepsResultAndRow(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "lbc", "leboncoin", "https://www.leboncoin.fr/ad/test/123")
	logs := captureLogs(t)
	saved := saveSession(t, service, "synthetic-intact")
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		// An empty renewal violates the schema, forcing the completion write to fail.
		return model.CollectionResult{Result: "success", AmountCents: 4200}, leboncoin.SessionOutcome{Attempt: "accepted", Renewed: true, Value: ""}
	})
	service.Collect(context.Background(), "lbc")
	item := lastAttempt(t, service, "lbc")
	if item.LastAttempt.Result != "success" || item.LatestPrice == nil || item.LatestPrice.AmountCents != 4200 {
		t.Fatal("observation lost after session write failure")
	}
	session, _ := service.LeboncoinSession(context.Background())
	if *session.Value != "synthetic-intact" || session.Revision != saved.Revision || session.LastAttempt != nil {
		t.Fatalf("row changed %+v", session)
	}
	if !strings.Contains(logs.String(), "LeBoncoin session update could not be saved; the stored session is unchanged") || strings.Contains(logs.String(), "synthetic-") {
		t.Fatalf("diagnostic %q", logs.String())
	}
}

func TestSessionReadFailureSkipsRequest(t *testing.T) {
	service, database := newSessionService(t)
	logs := captureLogs(t)
	service.leboncoin = fakeLeboncoin(func(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome) {
		t.Error("collector called without a readable session")
		return model.CollectionResult{}, leboncoin.SessionOutcome{}
	})
	database.Close()
	result := service.collectLeboncoin(context.Background(), model.Listing{ID: "lbc", Platform: "leboncoin", ListingID: "123", URL: "https://www.leboncoin.fr/ad/test/123"})
	if result.Result != "request_error" || result.Message != unreadSessionMessage {
		t.Fatalf("result %+v", result)
	}
	if !strings.Contains(logs.String(), "LeBoncoin session could not be read; check skipped") {
		t.Fatalf("diagnostic %q", logs.String())
	}
}

func TestReviewNoRequestRecordsNoSessionOutcome(t *testing.T) {
	service, database := newSessionService(t)
	insertListing(t, database, "lbc", "leboncoin", "https://example.com/ad/test/123")
	saved := saveSession(t, service, "synthetic-unused")
	service.Collect(context.Background(), "lbc")
	if item := lastAttempt(t, service, "lbc"); item.LastAttempt.Result != "request_error" {
		t.Fatalf("attempt %+v", item.LastAttempt)
	}
	session, _ := service.LeboncoinSession(context.Background())
	if session.Revision != saved.Revision || session.LastAttempt != nil {
		t.Fatalf("session changed without a request %+v", session)
	}
}
