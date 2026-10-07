package store

import (
	"context"
	"testing"
)

func TestAmazonRequestStatePersists(t *testing.T) {
	dir := t.TempDir()
	database := openTestStore(t, dir)
	ctx := context.Background()
	if state, err := database.AmazonRequestState(ctx); err != nil || state.Blocked || state.ConsecutiveFailures != 0 || state.StoppedAt != nil {
		t.Fatalf("unexpected initial state %+v %v", state, err)
	}
	must(t, database.SaveAmazonRequestState(ctx, AmazonRequestState{ConsecutiveFailures: 5, Blocked: true, StoppedAt: &searchTime}))
	database.Close()
	state, err := openTestStore(t, dir).AmazonRequestState(ctx)
	if err != nil || !state.Blocked || state.ConsecutiveFailures != 5 || !state.StoppedAt.Equal(searchTime) {
		t.Fatalf("unexpected reopened state %+v %v", state, err)
	}
}
