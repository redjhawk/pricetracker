package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	mathrand "math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/leboncoin"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

type Error struct {
	Status  int
	Code    string
	Message string
}

type collector interface {
	Collect(context.Context, model.Listing) model.CollectionResult
}

// leboncoinCollector lets tests replace the LeBoncoin collector.
type leboncoinCollector interface {
	CollectWithSession(context.Context, model.Listing, *leboncoin.Session) (model.CollectionResult, leboncoin.SessionOutcome)
}

func (e *Error) Error() string { return e.Message }

type Service struct {
	config        config.Config
	store         *store.Store
	collectors    map[string]collector
	leboncoin     leboncoinCollector
	claude        claudeClient
	mu            sync.Mutex
	inFlight      map[string]time.Time
	reviewing     map[string]bool // items with a running AI review
	reviewAgain   map[string]bool // items whose goal changed during a running review
	workerContext context.Context
	stopWorkers   context.CancelFunc
	workers       sync.WaitGroup
	refreshSlots  chan struct{}
	loginFailures map[string]loginFailures // by lowercased username
	now           func() time.Time         // replaced by tests
	random        func(n int64) int64      // draws the Amazon check times
	schedulePause func(context.Context) bool
	amazonGate    *amazonGate // every Amazon request goes through it
}

// amazonCheckTimeout bounds one Amazon check: browser start, page and resources, reading pause and
// used-offer requests.
const amazonCheckTimeout = 90 * time.Second

func New(cfg config.Config, database *store.Store) *Service {
	return NewWithOptions(cfg, database, Options{})
}

// AmazonCollector checks one Amazon item page.
type AmazonCollector interface {
	Collect(context.Context, model.Listing) model.CollectionResult
}

// Options replaces dependencies of the service (tests); zero values keep the defaults.
type Options struct {
	Amazon        AmazonCollector            // nil = browser collector from the configuration
	Now           func() time.Time           // nil = time.Now
	Random        func(n int64) int64        // nil = math/rand/v2; value in [0, n), goroutine-safe
	SchedulePause func(context.Context) bool // nil = 5–60 s pause between scheduled checks; false = cancelled
	// AmazonBrowserOrigin is for tests only: BrowserOptions.Origin of the default browser collector.
	AmazonBrowserOrigin string
}

// browserNoSandboxVariable set to "true" starts the browser without its sandbox (containers, root).
const browserNoSandboxVariable = "PRICEFOLLOWER_BROWSER_NO_SANDBOX"

// NewWithOptions is New with replaced dependencies. Amazon items without a next check (restarted
// once by the store, issue #168) or with a pre-#168 check time get a time in the next Amazon window.
func NewWithOptions(cfg config.Config, database *store.Store, options Options) *Service {
	if options.Amazon == nil {
		options.Amazon = newBrowserCollector(cfg, options.AmazonBrowserOrigin)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Random == nil {
		options.Random = mathrand.Int64N
	}
	if options.SchedulePause == nil {
		options.SchedulePause = waitScheduledRequest
	}
	workerContext, stopWorkers := context.WithCancel(context.Background())
	s := &Service{
		config: cfg, store: database, leboncoin: leboncoin.NewCollector(cfg.UserAgent), claude: claude.NewClient(),
		collectors: map[string]collector{
			"amazon": options.Amazon,
		},
		inFlight: make(map[string]time.Time), reviewing: make(map[string]bool), reviewAgain: make(map[string]bool), workerContext: workerContext, stopWorkers: stopWorkers,
		refreshSlots: make(chan struct{}, 2), loginFailures: make(map[string]loginFailures), now: options.Now, random: options.Random,
		schedulePause: options.SchedulePause, amazonGate: newAmazonGate(database, options.Now),
	}
	s.scheduleUnscheduledAmazonItems()
	return s
}

// newBrowserCollector returns the default Amazon collector. The browser is PRICEFOLLOWER_BROWSER_PATH
// or found on PATH at each check; without one the application still starts and Amazon checks fail.
func newBrowserCollector(cfg config.Config, origin string) *amazon.BrowserCollector {
	if _, err := amazon.FindBrowser(); err != nil {
		log.Print("Amazon browser not found; Amazon checks will fail until Chromium is installed or PRICEFOLLOWER_BROWSER_PATH is set")
	}
	return amazon.NewBrowserCollector(amazon.BrowserOptions{
		ProfileDir: filepath.Join(cfg.DataDirectory, "amazon-browser"),
		NoSandbox:  os.Getenv(browserNoSandboxVariable) == "true",
		Origin:     origin,
	})
}

// scheduleUnscheduledAmazonItems gives each Amazon item without a next check, or marked for a redraw
// by the upgrade, its own time in the next window. The redraw mark is kept if a time is not saved.
func (s *Service) scheduleUnscheduledAmazonItems() {
	ctx := context.Background()
	ids, err := s.store.AmazonIDsToSchedule(ctx)
	if err != nil {
		log.Printf("load unscheduled Amazon items: %v", err)
		return
	}
	for _, id := range ids {
		if err := s.store.SetNextCheck(ctx, id, s.nextCheckAt("amazon", s.now())); err != nil {
			log.Printf("schedule next check for item %s: %v", id, err)
			return
		}
	}
	if err := s.store.ClearAmazonRedraw(ctx); err != nil {
		log.Printf("record the Amazon check times redraw: %v", err)
	}
}

func (s *Service) Close() {
	s.stopWorkers()
	s.workers.Wait()
}

func (s *Service) NextCheckAt(after time.Time) time.Time {
	local := after.In(s.config.Location)
	date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.config.Location)
	slots := make([]int, 0, len(s.config.CheckTimes))
	for _, raw := range s.config.CheckTimes {
		parsed, _ := time.Parse("15:04", raw)
		slots = append(slots, parsed.Hour()*60+parsed.Minute())
	}
	sort.Ints(slots)
	for offset := 0; offset < 4; offset++ {
		day := date.AddDate(0, 0, offset)
		for _, slot := range slots {
			candidate := time.Date(day.Year(), day.Month(), day.Day(), slot/60, slot%60, 0, 0, s.config.Location)
			if candidate.After(after) {
				return candidate.UTC()
			}
		}
	}
	return after.Add(24 * time.Hour)
}

func (s *Service) List(ctx context.Context) ([]model.Item, error) {
	items, err := s.store.List(ctx, ownerFrom(ctx))
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = s.withInFlight(items[index])
	}
	return items, nil
}

// ownedListing loads an item of the request's owner; another owner's item is sql.ErrNoRows, like an unknown one.
func (s *Service) ownedListing(ctx context.Context, id string) (model.Listing, error) {
	listing, err := s.store.Listing(ctx, id)
	if err == nil && listing.OwnerID != ownerFrom(ctx) {
		return model.Listing{}, sql.ErrNoRows
	}
	return listing, err
}

func (s *Service) Get(ctx context.Context, id string) (model.Item, error) {
	if _, err := s.ownedListing(ctx, id); err != nil {
		return model.Item{}, err
	}
	item, err := s.store.GetWithHistory(ctx, id)
	if err != nil {
		return item, err
	}
	return s.withInFlight(item), nil
}

// Add tracks a listing; the trimmed purchase goal is kept for LeBoncoin items only.
func (s *Service) Add(ctx context.Context, rawURL, purchaseGoal string) (model.Item, error) {
	rawURL = strings.TrimSpace(rawURL)
	amazonURL := amazon.ParseURL(rawURL)
	leboncoinURL := leboncoin.ParseURL(rawURL)
	var itemListing model.Listing
	var canonical string
	switch {
	case amazonURL.Kind == "valid":
		itemListing = model.Listing{Platform: "amazon", ListingID: amazonURL.ASIN, ASIN: amazonURL.ASIN, Marketplace: amazonURL.Marketplace, URL: rawURL}
		canonical = amazonURL.Canonical
	case leboncoinURL.Kind == "valid":
		itemListing = model.Listing{Platform: "leboncoin", ListingID: leboncoinURL.ListingID, Marketplace: leboncoinURL.Marketplace, URL: leboncoinURL.URL, PurchaseGoal: strings.TrimSpace(purchaseGoal)}
		canonical = leboncoinURL.Canonical
	case amazonURL.Kind == "invalid" && leboncoinURL.Kind == "invalid":
		return model.Item{}, &Error{Status: 400, Code: "INVALID_URL", Message: "Enter a valid HTTPS Amazon or LeBoncoin listing URL."}
	default:
		return model.Item{}, &Error{Status: 422, Code: "UNSUPPORTED_LISTING", Message: "This listing is outside the supported Amazon euro marketplaces and LeBoncoin listings."}
	}
	itemListing.OwnerID = ownerFrom(ctx)
	exists, err := s.store.IsCanonicalTracked(ctx, itemListing.OwnerID, canonical)
	if err != nil {
		return model.Item{}, err
	}
	if exists {
		return model.Item{}, &Error{Status: 409, Code: "ITEM_ALREADY_TRACKED", Message: "This listing is already being tracked."}
	}
	id, err := newID()
	if err != nil {
		return model.Item{}, err
	}
	itemListing.ID = id
	if err := s.store.Insert(ctx, itemListing, canonical, s.nextCheckAt(itemListing.Platform, s.now())); err != nil {
		if store.IsUniqueConstraint(err) {
			return model.Item{}, &Error{Status: 409, Code: "ITEM_ALREADY_TRACKED", Message: "This listing is already being tracked."}
		}
		return model.Item{}, err
	}
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return model.Item{}, err
	}
	s.workers.Add(1)
	go func() { defer s.workers.Done(); s.collect(s.workerContext, id, true) }()
	return s.withPending(item, s.now().UTC()), nil
}

func (s *Service) Remove(ctx context.Context, id string) (bool, error) {
	return s.store.Delete(ctx, ownerFrom(ctx), id)
}

// RefreshAll checks every item of the owner at once. Amazon items keep their scheduled check.
func (s *Service) RefreshAll(ctx context.Context) (time.Time, int, error) {
	refs, err := s.store.ItemRefs(ctx, ownerFrom(ctx))
	if err != nil {
		return time.Time{}, 0, err
	}
	requestedAt := s.now().UTC()
	// Stopped Amazon items are not reserved: the gate would skip them, and a reservation would
	// hide a later Refresh price (manual check) of the item. They still count as queued.
	rescheduled := make([]string, 0, len(refs))
	unstopped := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Platform != "amazon" {
			rescheduled = append(rescheduled, ref.ID)
		}
		if !ref.Stopped {
			unstopped = append(unstopped, ref.ID)
		}
	}
	if err := s.store.SetNextChecks(ctx, rescheduled, s.NextCheckAt(requestedAt)); err != nil {
		return time.Time{}, 0, err
	}
	queued := make([]string, 0, len(unstopped))
	s.mu.Lock()
	for _, id := range unstopped {
		if _, active := s.inFlight[id]; active {
			continue
		}
		s.inFlight[id] = requestedAt
		queued = append(queued, id)
	}
	s.mu.Unlock()
	for _, id := range queued {
		s.workers.Add(1)
		go s.runQueuedCollection(id, false)
	}
	return requestedAt, len(refs), nil
}

// RefreshItem checks one item at once. An Amazon item keeps a scheduled check that is not due yet.
func (s *Service) RefreshItem(ctx context.Context, id string) (time.Time, error) {
	listing, err := s.ownedListing(ctx, id)
	if err != nil {
		return time.Time{}, err
	}
	requestedAt := s.now().UTC()
	if listing.Platform != "amazon" {
		err = s.store.SetNextCheck(ctx, id, s.NextCheckAt(requestedAt))
	} else {
		err = s.rescheduleOverdueAmazonCheck(ctx, id, requestedAt)
	}
	if err != nil {
		return time.Time{}, err
	}
	s.mu.Lock()
	_, active := s.inFlight[id]
	if !active {
		s.inFlight[id] = requestedAt
	}
	s.mu.Unlock()
	if !active {
		s.workers.Add(1)
		go s.runQueuedCollection(id, true)
	}
	return requestedAt, nil
}

// rescheduleOverdueAmazonCheck draws a time in the next window for an Amazon item whose scheduled
// check is already due, as when it was skipped while the item was stopped: once Refresh price
// restarts the item, that check must not run right after the manual one.
func (s *Service) rescheduleOverdueAmazonCheck(ctx context.Context, id string, now time.Time) error {
	due, ok, err := s.store.NextCheck(ctx, id)
	if err != nil || !ok || due.After(now) {
		return err
	}
	return s.store.SetNextCheck(ctx, id, nextAmazonCheckAt(now, s.config.Location, s.random))
}

// runQueuedCollection runs a reserved collection; manual (Refresh price) checks a stopped Amazon item too.
func (s *Service) runQueuedCollection(id string, manual bool) {
	defer s.workers.Done()
	select {
	case s.refreshSlots <- struct{}{}:
		defer func() { <-s.refreshSlots }()
		s.collectReserved(s.workerContext, id, false, manual)
	case <-s.workerContext.Done():
		s.releaseCollection(id)
	}
}

func (s *Service) Collect(ctx context.Context, id string) {
	s.collect(ctx, id, false)
}

// collect runs one collection unless one is in flight; newItem marks the first
// collection of a newly added item, which triggers an AI review.
func (s *Service) collect(ctx context.Context, id string, newItem bool) {
	started := s.now().UTC()
	s.mu.Lock()
	if _, exists := s.inFlight[id]; exists {
		s.mu.Unlock()
		return
	}
	s.inFlight[id] = started
	s.mu.Unlock()
	s.collectReserved(ctx, id, newItem, false)
}

func (s *Service) collectReserved(ctx context.Context, id string, newItem, manual bool) {
	defer func() { s.mu.Lock(); delete(s.inFlight, id); s.mu.Unlock() }()
	item, err := s.store.Listing(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("load listing %s for collection: %v", id, err)
		return
	}
	var result model.CollectionResult
	if item.Platform == "leboncoin" {
		result = s.collectLeboncoin(ctx, item)
	} else {
		collector, exists := s.collectors[item.Platform]
		if !exists {
			log.Printf("no collector registered for platform %q on item %s", item.Platform, id)
			return
		}
		var stopped error
		result, stopped = s.amazonGate.do(ctx, id, manual, func() model.CollectionResult {
			requestContext, cancel := context.WithTimeout(ctx, amazonCheckTimeout)
			defer cancel()
			return collector.Collect(requestContext, item)
		})
		if stopped != nil {
			log.Printf("check of item %s skipped: %v", id, stopped)
			return
		}
	}
	if _, err := s.store.Listing(ctx, id); errors.Is(err, sql.ErrNoRows) {
		return
	} else if err != nil {
		log.Printf("confirm listing %s after collection: %v", id, err)
		return
	}
	var previousPrice *int64
	if item.Platform == "leboncoin" {
		if previousPrice, err = s.store.LatestPriceCents(ctx, id); err != nil {
			log.Printf("load previous price for item %s: %v", id, err)
			return
		}
	}
	// Reserve the review before recording, so a client never sees the new
	// collection result without the review already shown as running.
	priceChanged := result.Result == "success" && previousPrice != nil && *previousPrice != result.AmountCents
	var token model.ClaudeToken
	reserved := false
	if item.Platform == "leboncoin" && (newItem || priceChanged) {
		var outcome reviewOutcome
		token, outcome, err = s.reserveReview(id)
		if err != nil {
			log.Printf("reserve AI review for item %s: %v", id, err)
		}
		reserved = outcome == reviewStarted
	}
	timestamp := s.now().UTC()
	if !newItem || result.Result != "success" {
		result.OldPriceCents, result.OldPriceAt = nil, nil
	}
	if result.OldPriceAt != nil && !result.OldPriceAt.Before(timestamp) {
		result.OldPriceAt = nil // keep the collected price as the latest
	}
	err = s.store.RecordCollection(ctx, id, result, timestamp)
	if err != nil {
		log.Printf("save collection result for item %s: %v", id, err)
		if reserved {
			s.releaseReview(id)
		}
		return
	}
	if reserved {
		if err := s.launchReview(id, token, result.Listing, false); err != nil {
			log.Printf("start AI review for item %s: %v", id, err)
		}
	}
}

// collectLeboncoin reads the saved session at the start of every attempt,
// before the request timeout starts, and records the session outcome only if
// the session did not change during the attempt.
func (s *Service) collectLeboncoin(ctx context.Context, item model.Listing) model.CollectionResult {
	stored, err := s.store.LeboncoinSession(ctx, item.OwnerID)
	if err != nil {
		log.Print("LeBoncoin session could not be read; check skipped")
		return model.CollectionResult{Result: "request_error", Message: "The LeBoncoin session could not be read. The check will be retried at the next scheduled time."}
	}
	if stored.Status == "expired" || stored.Status == "revoked" {
		log.Print("LeBoncoin session expired or revoked; check skipped")
		return model.CollectionResult{Result: "request_error", Message: "The saved LeBoncoin session has expired or was revoked. Save a new session in Settings."}
	}
	requestContext, cancel := context.WithTimeout(ctx, 31*time.Second)
	defer cancel()
	if stored.Status == "none" {
		result, _ := s.leboncoin.CollectWithSession(requestContext, item, nil)
		return result
	}
	session := &leboncoin.Session{Value: *stored.Value, ExpiresAt: stored.ExpiresAt}
	result, outcome := s.leboncoin.CollectWithSession(requestContext, item, session)
	if outcome.Attempt == "" {
		return result // no request was sent; nothing to record for the session
	}
	applied, err := s.store.FinishLeboncoinSessionAttempt(ctx, item.OwnerID, stored.Revision, store.LeboncoinSessionOutcome(outcome), time.Now().UTC())
	if err != nil {
		log.Print("LeBoncoin session update could not be saved; the stored session is unchanged")
	} else if !applied {
		log.Print("LeBoncoin session changed during a check; that check's session update was discarded")
	}
	return result
}

// LeboncoinSession returns the saved LeBoncoin session settings.
func (s *Service) LeboncoinSession(ctx context.Context) (model.LeboncoinSession, error) {
	return s.store.LeboncoinSession(ctx, ownerFrom(ctx))
}

// SaveLeboncoinSession saves pasted session input, or clears the session when
// the input is blank. It never starts a price check.
func (s *Service) SaveLeboncoinSession(ctx context.Context, raw string, revision int64) (model.LeboncoinSession, error) {
	value, err := leboncoin.ParseSessionInput(raw)
	var inputError *leboncoin.SessionInputError
	if errors.As(err, &inputError) {
		return model.LeboncoinSession{}, &Error{Status: 400, Code: "INVALID_SESSION", Message: inputError.Message}
	}
	if err != nil {
		return model.LeboncoinSession{}, err
	}
	var stored *string
	if value != "" {
		stored = &value
	}
	session, err := s.store.SaveLeboncoinSession(ctx, ownerFrom(ctx), stored, revision, time.Now().UTC())
	if errors.Is(err, store.ErrSessionChanged) {
		return model.LeboncoinSession{}, &Error{Status: 409, Code: "SESSION_CHANGED", Message: "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving."}
	}
	return session, err
}

func (s *Service) releaseCollection(id string) {
	s.mu.Lock()
	delete(s.inFlight, id)
	s.mu.Unlock()
}

func (s *Service) RunScheduler(ctx context.Context) {
	s.collectDue(ctx)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectDue(ctx)
		}
	}
}

func (s *Service) collectDue(ctx context.Context) {
	checks, err := s.store.DueChecks(ctx, s.now())
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("load due listings: %v", err)
		}
		return
	}
	collected := false
	for _, check := range checks {
		if ctx.Err() != nil {
			return
		}
		id := check.ID
		item, err := s.store.Listing(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			log.Printf("load listing %s for its scheduled check: %v", id, err)
			continue
		}
		if s.skipScheduledCheck(ctx, item, check.DueAt) {
			continue
		}
		if collected && !s.schedulePause(ctx) {
			return
		}
		// The next check is stored before the request, so a restart never repeats this one.
		if err := s.store.SetNextCheck(ctx, id, s.nextCheckAt(item.Platform, s.now())); err != nil {
			log.Printf("schedule next check for item %s: %v", id, err)
			continue
		}
		s.Collect(ctx, id)
		collected = true
	}
}

// skipScheduledCheck reports whether the item due at dueAt must not be checked now: a check of the
// item is running, or (Amazon) the window it was due in has passed while the application was down or
// busy. A skipped Amazon item gets a time in the next window, so it is checked once per window.
func (s *Service) skipScheduledCheck(ctx context.Context, item model.Listing, dueAt time.Time) bool {
	s.mu.Lock()
	_, active := s.inFlight[item.ID]
	s.mu.Unlock()
	now := s.now()
	if item.Platform != "amazon" || (!active && inDueAmazonWindow(dueAt, now, s.config.Location)) {
		return active
	}
	next := s.nextCheckAt(item.Platform, now)
	if err := s.store.SetNextCheck(ctx, item.ID, next); err != nil {
		log.Printf("schedule next check for item %s: %v", item.ID, err)
	} else if !active {
		log.Printf("scheduled Amazon check of item %s missed its window; next check %s", item.ID, next.Format(time.RFC3339))
	}
	return true
}

func waitScheduledRequest(ctx context.Context) bool {
	delay := time.Duration(5+mathrand.IntN(56)) * time.Second // #nosec G404 -- pause jitter, not security
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *Service) withInFlight(item model.Item) model.Item {
	s.mu.Lock()
	started, active := s.inFlight[item.ID]
	s.mu.Unlock()
	if active {
		return s.withPending(item, started)
	}
	return item
}

func (s *Service) withPending(item model.Item, timestamp time.Time) model.Item {
	item.Status = "pending"
	if item.SecondHandOffer != nil {
		item.SecondHandOffer.Status = "pending"
	}
	item.LastAttempt = &model.Attempt{Result: "pending", Timestamp: timestamp.UTC(), Message: nil}
	return item
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate item ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
