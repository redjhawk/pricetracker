package service

import (
	"cmp"
	"context"
	"database/sql"
	"log"
	"slices"
	"strings"
)

// baseLetters maps lower-case Latin letters with diacritics and ligatures to
// their base letters, so that "écran" sorts with the e's.
var baseLetters = strings.NewReplacer(
	"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "æ", "ae",
	"ç", "c", "ð", "d",
	"è", "e", "é", "e", "ê", "e", "ë", "e",
	"ì", "i", "í", "i", "î", "i", "ï", "i",
	"ñ", "n",
	"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o", "œ", "oe",
	"ß", "ss",
	"ù", "u", "ú", "u", "û", "u", "ü", "u",
	"ý", "y", "ÿ", "y",
)

// compareGoals is the alphabetical display order: base letters (letter case and
// accents ignored), then letter case ignored, then the exact text.
func compareGoals(a, b string) int {
	lowerA, lowerB := strings.ToLower(a), strings.ToLower(b)
	return cmp.Or(
		strings.Compare(baseLetters.Replace(lowerA), baseLetters.Replace(lowerB)),
		strings.Compare(lowerA, lowerB),
		strings.Compare(a, b),
	)
}

// PurchaseGoals returns the requester's existing purchase goals in alphabetical order.
func (s *Service) PurchaseGoals(ctx context.Context) ([]string, error) {
	stored, err := s.store.PurchaseGoals(ctx, ownerFrom(ctx))
	if err != nil {
		return nil, err
	}
	goals := make([]string, 0, len(stored))
	for _, goal := range stored {
		if strings.TrimSpace(goal) != "" {
			goals = append(goals, goal)
		}
	}
	slices.SortFunc(goals, compareGoals)
	return goals, nil
}

// SetPurchaseGoal replaces an item's purchase goal. A changed LeBoncoin goal
// requests a new AI review; if one is running, another runs after it. A changed
// Amazon goal starts no review: the next requested review uses it.
func (s *Service) SetPurchaseGoal(ctx context.Context, id, goal string) (string, bool, bool, error) {
	listing, err := s.ownedListing(ctx, id)
	if err != nil {
		return "", false, false, err
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
	if listing.Platform == "amazon" {
		return goal, true, false, nil
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
