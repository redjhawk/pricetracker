package leboncoin

import (
	"encoding/json"
	"strings"

	"pricefollower.local/internal/model"
)

const (
	maxDescriptionCharacters = 20_000
	maxAttributes            = 100
)

type adAttribute struct {
	Key        string `json:"key"`
	KeyLabel   string `json:"key_label"`
	Value      string `json:"value"`
	ValueLabel string `json:"value_label"`
}

type adLocation struct {
	City           string `json:"city"`
	Zipcode        string `json:"zipcode"`
	DepartmentName string `json:"department_name"`
	RegionName     string `json:"region_name"`
}

// listingDetails copies the listing content used by AI reviews.
func listingDetails(ad listingData, priceCents *int64) *model.ListingDetails {
	var location adLocation
	var owner struct {
		Type string `json:"type"`
	}
	json.Unmarshal(ad.Location, &location) // a missing or odd value leaves the fields empty
	json.Unmarshal(ad.Owner, &owner)
	details := &model.ListingDetails{
		Title:       strings.TrimSpace(ad.Subject),
		Description: truncateCharacters(strings.TrimSpace(ad.Body), maxDescriptionCharacters),
		ImageURLs:   listingImages(ad.Images),
		Attributes:  make([]model.ListingAttribute, 0),
		Category:    strings.TrimSpace(ad.CategoryName),
		City:        strings.TrimSpace(location.City),
		Zipcode:     strings.TrimSpace(location.Zipcode),
		Department:  strings.TrimSpace(location.DepartmentName),
		Region:      strings.TrimSpace(location.RegionName),
		PublishedAt: strings.TrimSpace(ad.FirstPublicationDate),
		SellerType:  strings.TrimSpace(owner.Type),
	}
	if priceCents != nil {
		price := *priceCents
		details.PriceCents = &price
	}
	for _, raw := range ad.Attributes {
		if len(details.Attributes) == maxAttributes {
			break
		}
		var attribute adAttribute
		if json.Unmarshal(raw, &attribute) != nil {
			continue
		}
		label := firstNonEmpty(attribute.KeyLabel, attribute.Key)
		value := firstNonEmpty(attribute.ValueLabel, attribute.Value)
		if label != "" && value != "" {
			details.Attributes = append(details.Attributes, model.ListingAttribute{Label: label, Value: value})
		}
	}
	return details
}

// listingImages returns all LeBoncoin image URLs in listing order, large versions preferred.
func listingImages(images adImages) []string {
	source := images.URLsLarge
	if len(source) == 0 {
		source = images.URLs
	}
	urls := make([]string, 0, len(source))
	seen := make(map[string]bool)
	for _, candidate := range source {
		candidate = strings.TrimSpace(candidate)
		if strings.HasPrefix(candidate, "https://img.leboncoin.fr/") && !seen[candidate] {
			seen[candidate] = true
			urls = append(urls, candidate)
		}
	}
	return urls
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func truncateCharacters(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
