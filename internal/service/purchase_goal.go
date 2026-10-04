package service

import (
	"context"
	"database/sql"
	"log"
	"strings"
)

// SetPurchaseGoal replaces a LeBoncoin item's purchase goal. A changed goal
// requests a new AI review; if one is running, another runs after it.
func (s *Service) SetPurchaseGoal(ctx context.Context, id, goal string) (string, bool, bool, error) {
	listing, err := s.store.Listing(ctx, id)
	if err != nil {
		return "", false, false, err
	}
	if listing.Platform != "leboncoin" {
		return "", false, false, &Error{Status: 422, Code: "PURCHASE_GOAL_UNSUPPORTED", Message: "Purchase goals are available for LeBoncoin items only."}
	}
	goal = strings.TrimSpace(goal)
	if goal == listing.PurchaseGoal {
		return goal, false, false, nil
	}
	found, err := s.store.SetPurchaseGoal(ctx, id, goal)
	if err != nil {
		return "", false, false, err
	}
	if !found {
		return "", false, false, sql.ErrNoRows
	}
	// The goal is saved: a review start failure follows the AI review rules
	// and is not reported as a save failure.
	outcome, err := s.startReview(id, nil, true)
	if err != nil {
		log.Printf("start AI review for item %s: %v", id, err)
		return goal, true, false, nil
	}
	if outcome == reviewAlreadyRunning {
		s.mu.Lock()
		running := s.reviewing[id]
		if running {
			s.reviewAgain[id] = true
		}
		s.mu.Unlock()
		if !running {
			// The running review finished meanwhile; start a new one now.
			outcome, err = s.startReview(id, nil, true)
			if err != nil {
				log.Printf("start AI review for item %s: %v", id, err)
				return goal, true, false, nil
			}
		}
	}
	return goal, true, outcome != reviewNoToken, nil
}
