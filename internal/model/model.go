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

type Item struct {
	ID                  string        `json:"id"`
	Title               *string       `json:"title"`
	ASIN                string        `json:"asin"`
	Marketplace         string        `json:"marketplace"`
	URL                 string        `json:"url"`
	ThumbnailURL        *string       `json:"thumbnailUrl"`
	Status              string        `json:"status"`
	LatestPrice         *Observation  `json:"latestPrice"`
	LastThreeDetections []Observation `json:"lastThreeDetections"`
	LastAttempt         *Attempt      `json:"lastAttempt"`
	NextCheckAt         *time.Time    `json:"nextCheckAt"`
	AddedAt             time.Time     `json:"addedAt"`
}

type Listing struct {
	ID          string
	ASIN        string
	Marketplace string
	URL         string
}

type CollectionResult struct {
	Result       string
	AmountCents  int64
	Title        *string
	ThumbnailURL *string
	Message      string
}
