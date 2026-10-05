package model

import "time"

type Observation struct {
	AmountCents int64      `json:"amountCents"`
	Currency    string     `json:"currency"`
	Timestamp   *time.Time `json:"timestamp"` // nil only for an undated old price
	OldPrice    bool       `json:"oldPrice"`
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
	PurchaseGoal        string           `json:"purchaseGoal"`
}

type Listing struct {
	ID           string
	Platform     string
	ListingID    string
	ASIN         string
	Marketplace  string
	URL          string
	PurchaseGoal string
	OwnerID      int64 // 0 is the open-mode owner
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
	Listing                  *ListingDetails // LeBoncoin only; kept in memory for AI reviews, never persisted
	OldPriceCents            *int64          // LeBoncoin old price; recorded only for a new item
	OldPriceAt               *time.Time      // date of the old price; nil when the listing gives none
}

// ListingDetails is the full LeBoncoin listing content sent to an AI review.
type ListingDetails struct {
	Title       string
	Description string
	PriceCents  *int64 // nil if not detected; 0 for free/donation listings
	ImageURLs   []string
	Attributes  []ListingAttribute
	Category    string
	City        string
	Zipcode     string
	Department  string
	Region      string
	PublishedAt string
	SellerType  string // "private", "pro" or ""
}

type ListingAttribute struct {
	Label string
	Value string
}

// AIReviewContent is a validated Claude review; amounts are euro cents.
type AIReviewContent struct {
	Price               AIRating            `json:"price"`
	Condition           AIConditionRating   `json:"condition"`
	Recommendation      AIRating            `json:"recommendation"`
	FairPrice           AIFairPrice         `json:"fairPrice"`
	Risks               []string            `json:"risks"`
	MissingInformation  []string            `json:"missingInformation"`
	SellerQuestions     []string            `json:"sellerQuestions"`
	DescriptionVsPhotos AIDescriptionPhotos `json:"descriptionVsPhotos"`
}

type AIRating struct {
	Rating      string `json:"rating"`
	Explanation string `json:"explanation"`
}

type AIConditionRating struct {
	Rating      *string `json:"rating"` // nil when the condition cannot be assessed
	Explanation string  `json:"explanation"`
}

type AIFairPrice struct {
	MinCents            int64 `json:"minCents"`
	MaxCents            int64 `json:"maxCents"`
	SuggestedOfferCents int64 `json:"suggestedOfferCents"`
}

// AIReview is one stored review attempt.
type AIReview struct {
	ID           int64            `json:"id"`
	Status       string           `json:"status"` // pending, succeeded or failed
	PriceCents   *int64           `json:"priceCents"`
	CreatedAt    time.Time        `json:"createdAt"`
	CompletedAt  *time.Time       `json:"completedAt"`
	Review       *AIReviewContent `json:"review"`
	ErrorMessage *string          `json:"errorMessage"`
}

// AIReviewState is the aiReview field of a LeBoncoin item details response.
type AIReviewState struct {
	TokenConfigured bool       `json:"tokenConfigured"`
	Running         bool       `json:"running"`
	LastAttempt     *AIReview  `json:"lastAttempt"`
	Latest          *AIReview  `json:"latest"`
	History         []AIReview `json:"history"`
}

// ItemDetails is the item details response; AIReview is null for Amazon items.
type ItemDetails struct {
	Item
	AIReview *AIReviewState `json:"aiReview"`
}

type AIDescriptionPhotos struct {
	Matches     *bool    `json:"matches"`
	Explanation string   `json:"explanation"`
	Mismatches  []string `json:"mismatches"`
}

// LeboncoinSession is the saved LeBoncoin datadome session as returned by the settings API.
type LeboncoinSession struct {
	Value       *string                  `json:"value"`
	Revision    int64                    `json:"revision"`
	UpdatedAt   *time.Time               `json:"updatedAt"`
	Status      string                   `json:"status"` // none, active, expired or revoked
	ExpiresAt   *time.Time               `json:"expiresAt"`
	RevokedAt   *time.Time               `json:"revokedAt"`
	LastAttempt *LeboncoinSessionAttempt `json:"lastAttempt"`
}

// LeboncoinSessionAttempt is the latest LeBoncoin check that sent the saved session.
type LeboncoinSessionAttempt struct {
	Outcome     string    `json:"outcome"` // accepted, rejected or failed
	AttemptedAt time.Time `json:"attemptedAt"`
}

// ClaudeToken is the saved Claude subscription token as returned by the settings API.
type ClaudeToken struct {
	Value          *string    `json:"value"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	LastRejectedAt *time.Time `json:"lastRejectedAt"`
	Revision       int64      `json:"-"` // used by AI reviews to ignore outcomes for a replaced token
	OwnerID        int64      `json:"-"` // the user whose token this is; 0 in open mode
}
