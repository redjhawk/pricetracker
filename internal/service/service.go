package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	mathrand "math/rand"
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
}

func New(cfg config.Config, database *store.Store) *Service {
	workerContext, stopWorkers := context.WithCancel(context.Background())
	return &Service{
		config: cfg, store: database, leboncoin: leboncoin.NewCollector(cfg.UserAgent), claude: claude.NewClient(),
		collectors: map[string]collector{
			"amazon": amazon.NewCollector(cfg.UserAgent),
		},
		inFlight: make(map[string]time.Time), reviewing: make(map[string]bool), reviewAgain: make(map[string]bool), workerContext: workerContext, stopWorkers: stopWorkers,
		refreshSlots: make(chan struct{}, 2),
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
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = s.withInFlight(items[index])
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, id string) (model.Item, error) {
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
	exists, err := s.store.IsCanonicalTracked(ctx, canonical)
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
	if err := s.store.Insert(ctx, itemListing, canonical, s.NextCheckAt(time.Now())); err != nil {
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
	return s.withPending(item, time.Now().UTC()), nil
}

func (s *Service) Remove(ctx context.Context, id string) (bool, error) {
	return s.store.Delete(ctx, id)
}

func (s *Service) RefreshAll(ctx context.Context) (time.Time, int, error) {
	ids, err := s.store.IDs(ctx)
	if err != nil {
		return time.Time{}, 0, err
	}
	requestedAt := time.Now().UTC()
	if err := s.store.SetNextChecks(ctx, ids, s.NextCheckAt(requestedAt)); err != nil {
		return time.Time{}, 0, err
	}
	queued := make([]string, 0, len(ids))
	s.mu.Lock()
	for _, id := range ids {
		if _, active := s.inFlight[id]; active {
			continue
		}
		s.inFlight[id] = requestedAt
		queued = append(queued, id)
	}
	s.mu.Unlock()
	for _, id := range queued {
		s.workers.Add(1)
		go s.runQueuedCollection(id)
	}
	return requestedAt, len(ids), nil
}

func (s *Service) RefreshItem(ctx context.Context, id string) (time.Time, error) {
	if _, err := s.store.Listing(ctx, id); err != nil {
		return time.Time{}, err
	}
	requestedAt := time.Now().UTC()
	if err := s.store.SetNextCheck(ctx, id, s.NextCheckAt(requestedAt)); err != nil {
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
		go s.runQueuedCollection(id)
	}
	return requestedAt, nil
}

func (s *Service) runQueuedCollection(id string) {
	defer s.workers.Done()
	select {
	case s.refreshSlots <- struct{}{}:
		defer func() { <-s.refreshSlots }()
		s.collectReserved(s.workerContext, id, false)
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
	started := time.Now().UTC()
	s.mu.Lock()
	if _, exists := s.inFlight[id]; exists {
		s.mu.Unlock()
		return
	}
	s.inFlight[id] = started
	s.mu.Unlock()
	s.collectReserved(ctx, id, newItem)
}

func (s *Service) collectReserved(ctx context.Context, id string, newItem bool) {
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
		requestContext, cancel := context.WithTimeout(ctx, 31*time.Second)
		result = collector.Collect(requestContext, item)
		cancel()
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
	timestamp := time.Now().UTC()
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
	stored, err := s.store.LeboncoinSession(ctx)
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
	applied, err := s.store.FinishLeboncoinSessionAttempt(ctx, stored.Revision, store.LeboncoinSessionOutcome(outcome), time.Now().UTC())
	if err != nil {
		log.Print("LeBoncoin session update could not be saved; the stored session is unchanged")
	} else if !applied {
		log.Print("LeBoncoin session changed during a check; that check's session update was discarded")
	}
	return result
}

// LeboncoinSession returns the saved LeBoncoin session settings.
func (s *Service) LeboncoinSession(ctx context.Context) (model.LeboncoinSession, error) {
	return s.store.LeboncoinSession(ctx)
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
	session, err := s.store.SaveLeboncoinSession(ctx, stored, revision, time.Now().UTC())
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
	ids, err := s.store.DueIDs(ctx, time.Now())
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("load due listings: %v", err)
		}
		return
	}
	collected := false
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		_, active := s.inFlight[id]
		s.mu.Unlock()
		if active {
			continue
		}
		if collected && !waitScheduledRequest(ctx) {
			return
		}
		if err := s.store.SetNextCheck(ctx, id, s.NextCheckAt(time.Now())); err != nil {
			log.Printf("schedule next check for item %s: %v", id, err)
			continue
		}
		s.Collect(ctx, id)
		collected = true
	}
}

func waitScheduledRequest(ctx context.Context) bool {
	delay := time.Duration(5+mathrand.Intn(56)) * time.Second
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
