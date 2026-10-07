package service

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"pricefollower.local/internal/model"
	"pricefollower.local/internal/store"
)

// amazonFailureLimit is the number of consecutive failed Amazon requests that stops all Amazon requests.
const amazonFailureLimit = 5

var errAmazonStopped = errors.New("Amazon requests are stopped")

// amazonGate serializes every Amazon request (tracked items and searches), counts consecutive
// failures and stops all Amazon requests at the limit. The state is persisted in SQLite.
type amazonGate struct {
	request sync.Mutex // held for the whole Amazon request, so the 5th failure and the stop are one step
	mu      sync.Mutex // guards state
	state   store.AmazonRequestState
	store   *store.Store
	now     func() time.Time
}

func newAmazonGate(database *store.Store, now func() time.Time) *amazonGate {
	state, err := database.AmazonRequestState(context.Background())
	if err != nil {
		log.Printf("load Amazon request state: %v", err)
	}
	return &amazonGate{state: state, store: database, now: now}
}

// do runs request unless Amazon requests are stopped. A request_error result is a failure;
// any other result (success, price_not_found, unavailable) is an Amazon answer and resets the count.
// A request cancelled through ctx (shutdown) is neither: it returns ctx.Err() and leaves the state unchanged.
func (g *amazonGate) do(ctx context.Context, request func() model.CollectionResult) (model.CollectionResult, error) {
	g.request.Lock()
	defer g.request.Unlock()
	if g.current().Blocked {
		return model.CollectionResult{}, errAmazonStopped
	}
	result := request()
	if err := ctx.Err(); err != nil {
		return model.CollectionResult{}, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now().UTC()
	if result.Result == "request_error" {
		log.Printf("Amazon request failed url=%s at=%s status=%d error=%s response=%s",
			result.RequestURL, now.Format(time.RFC3339), result.HTTPStatus, result.Message, result.ResponseExcerpt)
		g.state.ConsecutiveFailures++
		if g.state.ConsecutiveFailures >= amazonFailureLimit {
			g.state.Blocked, g.state.StoppedAt = true, &now
			log.Printf("Amazon requests stopped after %d consecutive failures at %s", amazonFailureLimit, now.Format(time.RFC3339))
		}
	} else {
		g.state.ConsecutiveFailures, g.state.StoppedAt = 0, nil
	}
	g.save()
	return result, nil
}

// restart allows Amazon requests again; stoppedAt stays until the next Amazon success.
func (g *amazonGate) restart() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.state.Blocked, g.state.ConsecutiveFailures = false, 0
	g.save()
}

func (g *amazonGate) current() store.AmazonRequestState {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.state
}

func (g *amazonGate) save() {
	if err := g.store.SaveAmazonRequestState(context.Background(), g.state); err != nil {
		log.Printf("save Amazon request state: %v", err)
	}
}
