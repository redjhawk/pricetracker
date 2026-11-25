package service

import (
	"time"

	"pricefollower.local/internal/model"
)

// amazonReviewAvailability returns why an Amazon item cannot be reviewed now: "stopped",
// "no_current_price", or "" when a review can be requested. Not implemented yet (issue #305).
func amazonReviewAvailability(stoppedAt *time.Time, latestPrice *model.Observation, lastAttempt *model.Attempt) string {
	return ""
}
