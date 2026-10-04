package leboncoin

import (
	"reflect"
	"strings"
	"testing"

	"pricefollower.local/internal/model"
)

const detailedListingHTML = `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active",
"subject":" Vélo ","body":" Très bon état ","price":[150],"category_name":"Vélos","first_publication_date":"2026-10-01 10:00:00",
"images":{"urls":["https://img.leboncoin.fr/s/1.jpg"],"urls_large":["https://img.leboncoin.fr/l/1.jpg","https://evil.example/2.jpg","https://img.leboncoin.fr/l/1.jpg","https://img.leboncoin.fr/l/3.jpg"]},
"attributes":[{"key":"condition","key_label":"État","value":"2","value_label":"Très bon état"},{"key":"brand","value":"Decathlon"},{"key":"odd","value":12},{"key":"empty","value":""}],
"location":{"city":"Lyon","zipcode":"69001","department_name":"Rhône","region_name":"Auvergne-Rhône-Alpes","lat":45.7},
"owner":{"type":"private","name":"x"}}}}}</script>`

func TestCollectorParsesListingDetails(t *testing.T) {
	result, _ := collectFixture(t, 200, detailedListingHTML)
	if result.Result != "success" || result.AmountCents != 15000 || result.Listing == nil {
		t.Fatalf("unexpected result %+v", result)
	}
	price := int64(15000)
	want := model.ListingDetails{
		Title: "Vélo", Description: "Très bon état", PriceCents: &price,
		ImageURLs:  []string{"https://img.leboncoin.fr/l/1.jpg", "https://img.leboncoin.fr/l/3.jpg"},
		Attributes: []model.ListingAttribute{{Label: "État", Value: "Très bon état"}, {Label: "brand", Value: "Decathlon"}},
		Category:   "Vélos", City: "Lyon", Zipcode: "69001", Department: "Rhône", Region: "Auvergne-Rhône-Alpes",
		PublishedAt: "2026-10-01 10:00:00", SellerType: "private",
	}
	if !reflect.DeepEqual(*result.Listing, want) {
		t.Fatalf("details\n got %+v\nwant %+v", *result.Listing, want)
	}
}

func TestCollectorListingDetailsWithoutPriceOrPhotos(t *testing.T) {
	page := `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","subject":"Lampe","body":"` +
		strings.Repeat("é", 20_005) + `","location":"odd"}}}}</script>`
	result, _ := collectFixture(t, 200, page)
	if result.Result != "price_not_found" || result.Listing == nil || result.Listing.PriceCents != nil || len(result.Listing.ImageURLs) != 0 {
		t.Fatalf("unexpected result %+v", result)
	}
	if len([]rune(result.Listing.Description)) != 20_000 || result.Listing.City != "" {
		t.Fatalf("description not capped or odd location used: %d", len([]rune(result.Listing.Description)))
	}
}

func TestCollectorDonationAndUnavailableDetails(t *testing.T) {
	donation := `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","subject":"Donne table"}}}}</script>`
	result, _ := collectFixture(t, 200, donation)
	if result.Listing == nil || result.Listing.PriceCents == nil || *result.Listing.PriceCents != 0 {
		t.Fatalf("donation details %+v", result.Listing)
	}
	inactive := `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"deleted"}}}}</script>`
	if result, _ := collectFixture(t, 200, inactive); result.Listing != nil {
		t.Fatal("details set for an unavailable listing")
	}
}
