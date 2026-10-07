package model

import "time"

// AmazonRequests is the application-wide Amazon request state.
type AmazonRequests struct {
	Stopped             bool       `json:"stopped"`
	StoppedAt           *time.Time `json:"stoppedAt"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
}

// AmazonSearch is the search object of the API.
type AmazonSearch struct {
	ID           string       `json:"id"`
	URL          string       `json:"url"`
	Label        string       `json:"label"`
	AddedAt      time.Time    `json:"addedAt"`
	CapturedAt   *time.Time   `json:"capturedAt"`
	ItemCount    int          `json:"itemCount"`
	State        string       `json:"state"` // waiting, running, done or stopped
	WaitingUntil *time.Time   `json:"waitingUntil"`
	LastError    *SearchError `json:"lastError"`
}

type SearchError struct {
	At      time.Time `json:"at"`
	Message string    `json:"message"`
}

// SearchItem is an item of a search, in Amazon order.
type SearchItem struct {
	Item
	Position        int             `json:"position"`
	AIReviewSummary AIReviewSummary `json:"aiReviewSummary"`
}

// AIReviewSummary is the review state shown in a search item list.
type AIReviewSummary struct {
	Status      string  `json:"status"` // none, pending, available, failed or no_token
	PriceRating *string `json:"priceRating"`
	PriceCents  *int64  `json:"priceCents"`
}
