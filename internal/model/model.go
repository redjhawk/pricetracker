package model

import "time"

type Observation struct {
	AmountCents int64     `json:"amountCents"`
	Currency    string    `json:"currency"`
	Timestamp   time.Time `json:"timestamp"`
}

type Attempt struct {
	Result    string    `json:"result"`
	Timestamp time.Time `json:"timestamp"`
	Message   *string   `json:"message"`
}

type SecondHandObservation struct {
	AmountCents    int64     `json:"amountCents"`
	Currency       string    `json:"currency"`
	Condition      string    `json:"condition"`
	ConditionLabel string    `json:"conditionLabel"`
	Timestamp      time.Time `json:"timestamp"`
}

type SecondHandOffer struct {
	Status              string                  `json:"status"`
	LatestDetection     *SecondHandObservation  `json:"latestDetection"`
	LastThreeDetections []SecondHandObservation `json:"lastThreeDetections"`
	PriceHistory        []SecondHandObservation `json:"priceHistory,omitempty"`
	LastCheckedAt       *time.Time              `json:"lastCheckedAt"`
}

type Item struct {
	ID                  string           `json:"id"`
	Title               *string          `json:"title"`
	Platform            string           `json:"platform"`
	ListingID           string           `json:"listingId"`
	ASIN                *string          `json:"asin"`
	Marketplace         string           `json:"marketplace"`
	URL                 string           `json:"url"`
	ThumbnailURL        *string          `json:"thumbnailUrl"`
	Status              string           `json:"status"`
	LatestPrice         *Observation     `json:"latestPrice"`
	LastThreeDetections []Observation    `json:"lastThreeDetections"`
	PriceHistory        []Observation    `json:"priceHistory,omitempty"`
	SecondHandOffer     *SecondHandOffer `json:"secondHandOffer"`
	LastAttempt         *Attempt         `json:"lastAttempt"`
	NextCheckAt         *time.Time       `json:"nextCheckAt"`
	AddedAt             time.Time        `json:"addedAt"`
}

type Listing struct {
	ID          string
	Platform    string
	ListingID   string
	ASIN        string
	Marketplace string
	URL         string
}

type CollectionResult struct {
	Result                   string
	AmountCents              int64
	Title                    *string
	ThumbnailURL             *string
	Message                  string
	SecondHandStatus         string
	SecondHandAmountCents    int64
	SecondHandCondition      string
	SecondHandConditionLabel string
}
