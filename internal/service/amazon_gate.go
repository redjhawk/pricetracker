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

// amazonFailureLimit is the number of consecutive failed Amazon requests that stops every Amazon item.
const amazonFailureLimit = 5

var errAmazonStopped = errors.New("Amazon checks of the item are stopped")

// amazonGate serializes every Amazon request, counts consecutive failures across all items and,
// at the limit, stops every Amazon item of every owner. The count and the stops are persisted in SQLite.
type amazonGate struct {
	request  sync.Mutex // held for the whole Amazon request, so the 5th failure and the stop are one step
	failures int        // guarded by request
	store    *store.Store
	now      func() time.Time
}

func newAmazonGate(database *store.Store, now func() time.Time) *amazonGate {
	state, err := database.AmazonRequestState(context.Background())
	if err != nil {
		log.Printf("load Amazon request state: %v", err)
	}
	return &amazonGate{failures: state.ConsecutiveFailures, store: database, now: now}
}

// do runs the request of item id unless the item is stopped; a manual check (Refresh price) runs
// even then. A request_error result is a failure; any other result (success, price_not_found,
// unavailable) is an Amazon answer: it resets the count and restarts the item.
// A request cancelled through ctx (shutdown) is neither: it returns ctx.Err() and leaves the state unchanged.
func (g *amazonGate) do(ctx context.Context, id string, manual bool, request func() model.CollectionResult) (model.CollectionResult, error) {
	g.request.Lock()
	defer g.request.Unlock()
	stopped, err := g.store.AmazonStopped(ctx, id)
	if err != nil {
		return model.CollectionResult{}, err
	}
	if stopped && !manual {
		return model.CollectionResult{}, errAmazonStopped
	}
	result := request()
	if err := ctx.Err(); err != nil {
		return model.CollectionResult{}, err
	}
	now := g.now().UTC()
	if result.Result == "request_error" {
		log.Printf("Amazon request failed url=%s at=%s status=%d error=%s response=%s",
			result.RequestURL, now.Format(time.RFC3339), result.HTTPStatus, result.Message, result.ResponseExcerpt)
		g.failures++
		if g.failures >= amazonFailureLimit {
			if err := g.store.StopAmazonItems(context.Background(), now); err != nil {
				log.Printf("stop Amazon items: %v", err)
			}
			g.failures = 0
			log.Printf("Amazon checks stopped after %d consecutive failures at %s", amazonFailureLimit, now.Format(time.RFC3339))
		}
	} else {
		g.failures = 0
		if stopped {
			if err := g.store.ClearAmazonStop(context.Background(), id); err != nil {
				log.Printf("restart Amazon item %s: %v", id, err)
			}
		}
	}
	if err := g.store.SaveAmazonFailures(context.Background(), g.failures); err != nil {
		log.Printf("save Amazon request state: %v", err)
	}
	return result, nil
}
