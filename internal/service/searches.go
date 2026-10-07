package service

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"pricefollower.local/internal/amazon"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

var errSearchNotFound = &Error{Status: 404, Code: "SEARCH_NOT_FOUND", Message: "Amazon search was not found."}

// AmazonRequests returns the application-wide Amazon request state.
func (s *Service) AmazonRequests() model.AmazonRequests {
	state := s.amazonGate.current()
	return model.AmazonRequests{Stopped: state.Blocked, StoppedAt: state.StoppedAt, ConsecutiveFailures: state.ConsecutiveFailures}
}

// Searches returns the owner's searches, most recently added first.
func (s *Service) Searches(ctx context.Context) ([]model.AmazonSearch, error) {
	searches, err := s.store.Searches(ctx, ownerFrom(ctx))
	if err != nil {
		return nil, err
	}
	result := make([]model.AmazonSearch, 0, len(searches))
	for _, search := range searches {
		result = append(result, s.searchResponse(search))
	}
	return result, nil
}

// AddSearch saves a search; the first retrieval happens later, in a request window.
func (s *Service) AddSearch(ctx context.Context, rawURL string) (model.AmazonSearch, error) {
	parsed := amazon.ParseSearchURL(rawURL)
	switch parsed.Kind {
	case "invalid":
		return model.AmazonSearch{}, &Error{Status: 400, Code: "INVALID_URL", Message: "Enter an Amazon search URL starting with https://."}
	case "unsupported":
		return model.AmazonSearch{}, &Error{Status: 422, Code: "UNSUPPORTED_SEARCH", Message: "Only searches on the supported Amazon euro marketplaces can be added."}
	}
	id, err := newID()
	if err != nil {
		return model.AmazonSearch{}, err
	}
	now := s.now().UTC()
	search := store.Search{ID: id, OwnerID: ownerFrom(ctx), URL: parsed.URL, AddedAt: now, NextRunAt: now}
	if err := s.store.InsertSearch(ctx, search); err != nil {
		if store.IsUniqueConstraint(err) {
			return model.AmazonSearch{}, &Error{Status: 409, Code: "SEARCH_ALREADY_ADDED", Message: "This search is already added: " + searchLabel(parsed.URL) + "."}
		}
		return model.AmazonSearch{}, err
	}
	return s.searchResponse(search), nil
}

// SearchDetails returns an owned search and its items in Amazon order.
func (s *Service) SearchDetails(ctx context.Context, id string) (model.AmazonSearch, []model.SearchItem, error) {
	search, err := s.ownedSearch(ctx, id)
	if err != nil {
		return model.AmazonSearch{}, nil, err
	}
	ids, err := s.store.SearchItemIDs(ctx, id)
	if err != nil {
		return model.AmazonSearch{}, nil, err
	}
	token, err := s.store.ClaudeToken(ctx, search.OwnerID)
	if err != nil {
		return model.AmazonSearch{}, nil, err
	}
	items := make([]model.SearchItem, 0, len(ids))
	for index, itemID := range ids {
		item, err := s.store.Get(ctx, itemID)
		if err != nil {
			return model.AmazonSearch{}, nil, err
		}
		summary, err := s.reviewSummary(ctx, itemID, token.Value != nil)
		if err != nil {
			return model.AmazonSearch{}, nil, err
		}
		items = append(items, model.SearchItem{Item: s.withInFlight(item), Position: index + 1, AIReviewSummary: summary})
	}
	return s.searchResponse(search), items, nil
}

func (s *Service) reviewSummary(ctx context.Context, itemID string, tokenConfigured bool) (model.AIReviewSummary, error) {
	lastAttempt, succeeded, err := s.store.AIReviews(ctx, itemID, "amazon")
	if err != nil {
		return model.AIReviewSummary{}, err
	}
	summary := model.AIReviewSummary{Status: "none"}
	if len(succeeded) > 0 {
		review := succeeded[0].Review.(*model.AmazonAIReviewContent)
		summary.Status, summary.PriceRating, summary.PriceCents = "available", &review.Price.Rating, succeeded[0].PriceCents
	}
	switch {
	case s.reviewRunning(itemID):
		summary.Status = "pending"
	case lastAttempt != nil && lastAttempt.Status == "failed":
		summary.Status = "failed"
	case len(succeeded) == 0 && !tokenConfigured:
		summary.Status = "no_token"
	}
	return summary, nil
}

// DeleteSearch deletes an owned search and its items that are neither tracked nor in another search.
func (s *Service) DeleteSearch(ctx context.Context, id string) error {
	deleted, err := s.store.DeleteSearch(ctx, ownerFrom(ctx), id)
	if err == nil && !deleted {
		return errSearchNotFound
	}
	return err
}

// TrackSearchItem moves an item of the search to the tracked Amazon items; it stays in the search.
func (s *Service) TrackSearchItem(ctx context.Context, searchID, itemID string) (model.Item, error) {
	if _, err := s.ownedSearch(ctx, searchID); err != nil {
		return model.Item{}, err
	}
	ids, err := s.store.SearchItemIDs(ctx, searchID)
	if err != nil {
		return model.Item{}, err
	}
	if !slices.Contains(ids, itemID) {
		return model.Item{}, &Error{Status: 404, Code: "ITEM_NOT_FOUND", Message: "Tracked item was not found."}
	}
	moved, err := s.store.TrackItem(ctx, ownerFrom(ctx), itemID, s.NextCheckAt(s.now()))
	if err != nil {
		return model.Item{}, err
	}
	if !moved {
		return model.Item{}, &Error{Status: 409, Code: "ITEM_ALREADY_TRACKED", Message: "This listing is already being tracked."}
	}
	item, err := s.store.Get(ctx, itemID)
	return s.withInFlight(item), err
}

// RestartAmazonSearch restarts all Amazon requests after a stop and makes the search due now.
func (s *Service) RestartAmazonSearch(ctx context.Context, id string) (time.Time, model.AmazonSearch, error) {
	search, err := s.ownedSearch(ctx, id)
	if err != nil {
		return time.Time{}, model.AmazonSearch{}, err
	}
	if state := s.amazonGate.current(); !state.Blocked && state.StoppedAt == nil {
		return time.Time{}, model.AmazonSearch{}, &Error{Status: 409, Code: "AMAZON_NOT_STOPPED", Message: "Amazon requests are not stopped."}
	}
	s.amazonGate.restart()
	now := s.now().UTC()
	if err := s.store.SetSearchProgress(ctx, id, 0, now); err != nil {
		return time.Time{}, model.AmazonSearch{}, err
	}
	search.PassPosition, search.NextRunAt = 0, now
	return now, s.searchResponse(search), nil
}

func (s *Service) ownedSearch(ctx context.Context, id string) (store.Search, error) {
	search, err := s.store.Search(ctx, ownerFrom(ctx), id)
	if errors.Is(err, sql.ErrNoRows) {
		return search, errSearchNotFound
	}
	return search, err
}

func (s *Service) setRunningSearch(id string) {
	s.mu.Lock()
	s.runningSearch = id
	s.mu.Unlock()
}

// searchResponse computes the search state; it is never stored.
func (s *Service) searchResponse(search store.Search) model.AmazonSearch {
	now := s.now()
	s.mu.Lock()
	running := s.runningSearch == search.ID
	s.mu.Unlock()
	unfinished := search.CapturedAt == nil || search.PassPosition > 0 || !search.NextRunAt.After(now)
	response := model.AmazonSearch{ID: search.ID, URL: search.URL, Label: searchLabel(search.URL), AddedAt: search.AddedAt,
		CapturedAt: search.CapturedAt, ItemCount: search.ItemCount, State: "done"}
	switch {
	case running:
		response.State = "running"
	case unfinished && s.amazonGate.current().Blocked:
		response.State = "stopped"
	case unfinished:
		response.State = "waiting"
		if !inSearchWindow(now) {
			next := nextSearchWindowStart(now)
			response.WaitingUntil = &next
		}
	}
	if search.LastErrorAt != nil {
		response.LastError = &model.SearchError{At: *search.LastErrorAt, Message: "The Amazon search page could not be retrieved."}
	}
	return response
}

// searchLabel is the host without "www." followed by the path, e.g. amazon.fr/joursprime/.
func searchLabel(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.") + parsed.EscapedPath()
}
