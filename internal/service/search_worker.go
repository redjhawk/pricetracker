package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	mathrand "math/rand"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// browsingClock lets tests replace real time and pauses.
type browsingClock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) bool // false when ctx is cancelled
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) Sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// searchCollector is the part of the Amazon collector the search worker uses.
type searchCollector interface {
	FetchSearch(context.Context, amazon.SearchURLResult) ([]amazon.SearchProduct, model.CollectionResult)
	CollectProduct(context.Context, model.Listing) model.CollectionResult
}

// searchWorker reads Amazon searches like a person: results page, then each item in order,
// with random pauses, only in the Paris request windows, one search at a time.
type searchWorker struct {
	service   *Service
	collector searchCollector
	clock     browsingClock
	pause     func() time.Duration
}

// humanPause is a random pause from 30 to 120 seconds inclusive.
func humanPause() time.Duration { return time.Duration(30+mathrand.Intn(91)) * time.Second }

// RunSearchWorker processes due Amazon searches until ctx is cancelled.
func (s *Service) RunSearchWorker(ctx context.Context) {
	worker := &searchWorker{service: s, collector: amazon.NewCollector(s.config.UserAgent), clock: realClock{}, pause: humanPause}
	worker.run(ctx)
}

func (w *searchWorker) run(ctx context.Context) {
	for ctx.Err() == nil {
		now := w.clock.Now()
		wait := time.Minute
		if !inSearchWindow(now) {
			wait = min(wait, nextSearchWindowStart(now).Sub(now))
		} else if !w.service.amazonGate.current().Blocked {
			due, err := w.service.store.DueSearches(ctx, now)
			if err != nil && ctx.Err() == nil {
				log.Printf("load due Amazon searches: %v", err)
			}
			if len(due) > 0 && w.runPass(ctx, due[0]) {
				continue
			}
		}
		if !w.clock.Sleep(ctx, wait) {
			return
		}
	}
}

// canContinue reports whether the pass may take its next action; otherwise the pass stops and
// resumes at its saved position.
func (w *searchWorker) canContinue(ctx context.Context, search store.Search) bool {
	if ctx.Err() != nil || !inSearchWindow(w.clock.Now()) || w.service.amazonGate.current().Blocked {
		return false
	}
	_, err := w.service.store.Search(ctx, search.OwnerID, search.ID)
	return err == nil
}

// runPass runs one pass of a search: results page, every item, then one retry of each failed action.
// It reports false when the pass stopped before its end, so the worker waits before trying again.
func (w *searchWorker) runPass(ctx context.Context, search store.Search) bool {
	w.service.setRunningSearch(search.ID)
	defer w.service.setRunningSearch("")
	var failed []int // positions put aside; 0 is the results page
	position := search.PassPosition
	if position == 0 {
		if !w.canContinue(ctx, search) {
			return false
		}
		if !w.openResults(ctx, &search) {
			failed = append(failed, 0)
		}
		position = 1
		w.saveProgress(ctx, search.ID, position, search.NextRunAt)
	}
	ids, err := w.service.store.SearchItemIDs(ctx, search.ID)
	if err != nil {
		log.Printf("load items of Amazon search %s: %v", search.ID, err)
		return false
	}
	for ; position <= len(ids); position++ {
		if !w.clock.Sleep(ctx, w.pause()) || !w.canContinue(ctx, search) {
			return false
		}
		if !w.openItem(ctx, ids[position-1]) {
			failed = append(failed, position)
		}
		w.saveProgress(ctx, search.ID, position+1, search.NextRunAt)
	}
	for _, position := range failed {
		if !w.clock.Sleep(ctx, w.pause()) || !w.canContinue(ctx, search) {
			return false
		}
		if position == 0 {
			w.openResults(ctx, &search)
		} else {
			w.openItem(ctx, ids[position-1])
		}
	}
	w.saveProgress(ctx, search.ID, 0, nextSearchWindowStart(w.clock.Now()))
	return true
}

func (w *searchWorker) saveProgress(ctx context.Context, id string, position int, nextRunAt time.Time) {
	if err := w.service.store.SetSearchProgress(ctx, id, position, nextRunAt); err != nil {
		log.Printf("save progress of Amazon search %s: %v", id, err)
	}
}

// openResults reads the results page; the first success captures the search items.
func (w *searchWorker) openResults(ctx context.Context, search *store.Search) bool {
	var products []amazon.SearchProduct
	result, err := w.service.amazonGate.do(ctx, func() model.CollectionResult {
		requestContext, cancel := context.WithTimeout(ctx, 31*time.Second)
		defer cancel()
		var result model.CollectionResult
		products, result = w.collector.FetchSearch(requestContext, amazon.ParseSearchURL(search.URL))
		return result
	})
	if err != nil {
		return false
	}
	now := w.clock.Now()
	if result.Result != "success" {
		w.saveSearchError(ctx, search.ID, now, result.Message)
		return false
	}
	if search.CapturedAt != nil {
		if err := w.service.store.ClearSearchError(ctx, search.ID); err != nil { // the item set is frozen
			log.Printf("clear error of Amazon search %s: %v", search.ID, err)
		}
		return true
	}
	if err := w.captureSearch(ctx, *search, products, now); err != nil {
		log.Printf("save results of Amazon search %s: %v", search.ID, err)
		w.saveSearchError(ctx, search.ID, now, "The search results could not be saved.")
		return false
	}
	search.CapturedAt = &now
	return true
}

func (w *searchWorker) saveSearchError(ctx context.Context, id string, now time.Time, message string) {
	if err := w.service.store.SetSearchError(ctx, id, now, message); err != nil {
		log.Printf("save error of Amazon search %s: %v", id, err)
	}
}

func (w *searchWorker) captureSearch(ctx context.Context, search store.Search, products []amazon.SearchProduct, now time.Time) error {
	items := make([]store.CapturedItem, 0, len(products))
	for _, product := range products {
		id, err := newID()
		if err != nil {
			return err
		}
		items = append(items, store.CapturedItem{ID: id, ASIN: product.ASIN, Marketplace: product.Marketplace, CanonicalURL: product.Canonical,
			URL: product.URL, Title: product.Title, PriceCents: product.PriceCents})
	}
	return w.service.store.CaptureSearch(ctx, search.ID, search.OwnerID, items, now)
}

// openItem opens one item, records what Amazon showed, reads it, then closes it.
// It reports false when the request failed and should be retried at the end of the pass.
func (w *searchWorker) openItem(ctx context.Context, id string) bool {
	item, err := w.service.store.Listing(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	if err != nil {
		log.Printf("load search item %s: %v", id, err)
		return false
	}
	result, err := w.service.amazonGate.do(ctx, func() model.CollectionResult {
		requestContext, cancel := context.WithTimeout(ctx, 31*time.Second)
		defer cancel()
		return w.collector.CollectProduct(requestContext, item)
	})
	if err != nil {
		return false
	}
	if err := w.service.store.RecordCollection(ctx, id, result, w.clock.Now().UTC()); err != nil {
		log.Printf("save collection result for search item %s: %v", id, err)
	}
	if result.Result == "success" {
		w.service.reviewSearchItem(id, result)
	}
	w.clock.Sleep(ctx, w.pause()) // reading the item
	log.Printf("close item %s", item.ASIN)
	return result.Result != "request_error"
}
