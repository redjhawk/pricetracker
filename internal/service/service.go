package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"pricefollower.local/config"
	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

type Service struct {
	config        config.Config
	store         *store.Store
	collector     *amazon.Collector
	mu            sync.Mutex
	inFlight      map[string]time.Time
	workerContext context.Context
	stopWorkers   context.CancelFunc
	workers       sync.WaitGroup
}

func New(cfg config.Config, database *store.Store) *Service {
	workerContext, stopWorkers := context.WithCancel(context.Background())
	return &Service{config: cfg, store: database, collector: amazon.NewCollector(cfg.UserAgent), inFlight: make(map[string]time.Time), workerContext: workerContext, stopWorkers: stopWorkers}
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
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return item, err
	}
	return s.withInFlight(item), nil
}

func (s *Service) Add(ctx context.Context, rawURL string) (model.Item, error) {
	parsed := amazon.ParseURL(strings.TrimSpace(rawURL))
	if parsed.Kind == "invalid" {
		return model.Item{}, &Error{Status: 400, Code: "INVALID_URL", Message: "Enter a valid HTTPS Amazon listing URL."}
	}
	if parsed.Kind != "valid" {
		return model.Item{}, &Error{Status: 422, Code: "UNSUPPORTED_LISTING", Message: "This listing is outside the supported euro-priced Amazon marketplaces."}
	}
	exists, err := s.store.IsCanonicalTracked(ctx, parsed.Canonical)
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
	itemListing := model.Listing{ID: id, ASIN: parsed.ASIN, Marketplace: parsed.Marketplace, URL: parsed.URL}
	if err := s.store.Insert(ctx, itemListing, parsed.Canonical, s.NextCheckAt(time.Now())); err != nil {
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
	go func() { defer s.workers.Done(); s.Collect(s.workerContext, id) }()
	return s.withPending(item, time.Now().UTC()), nil
}

func (s *Service) Remove(ctx context.Context, id string) (bool, error) {
	return s.store.Delete(ctx, id)
}

func (s *Service) Collect(ctx context.Context, id string) {
	started := time.Now().UTC()
	s.mu.Lock()
	if _, exists := s.inFlight[id]; exists {
		s.mu.Unlock()
		return
	}
	s.inFlight[id] = started
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.inFlight, id); s.mu.Unlock() }()
	item, err := s.store.Listing(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("load listing %s for collection: %v", id, err)
		return
	}
	requestContext, cancel := context.WithTimeout(ctx, 16*time.Second)
	defer cancel()
	result := s.collector.Collect(requestContext, item)
	if _, err := s.store.Listing(ctx, id); errors.Is(err, sql.ErrNoRows) {
		return
	} else if err != nil {
		log.Printf("confirm listing %s after collection: %v", id, err)
		return
	}
	timestamp := time.Now().UTC()
	if result.Result == "success" {
		err = s.store.RecordSuccess(ctx, id, result, timestamp)
	} else {
		err = s.store.RecordFailure(ctx, id, result.Result, result.Message, timestamp)
	}
	if err != nil {
		log.Printf("save collection result for item %s: %v", id, err)
	}
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
		if err := s.store.SetNextCheck(ctx, id, s.NextCheckAt(time.Now())); err != nil {
			log.Printf("schedule next check for item %s: %v", id, err)
			continue
		}
		s.Collect(ctx, id)
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
