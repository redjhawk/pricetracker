package leboncoin

import (
	"encoding/json"
	"testing"
)

func TestOldPriceCents(t *testing.T) {
	cases := []struct {
		raw    string
		amount int64
		ok     bool
	}{
		{`320`, 32000, true},
		{`[12.5]`, 1250, true},
		{``, 0, false},
		{`null`, 0, false},
		{`"320"`, 0, false},
		{`-5`, 0, false},
		{`[1, 2]`, 0, false},
	}
	for _, test := range cases {
		amount, ok := oldPriceCents(json.RawMessage(test.raw))
		if amount != test.amount || ok != test.ok {
			t.Errorf("oldPriceCents(%q) = %d, %t; want %d, %t", test.raw, amount, ok, test.amount, test.ok)
		}
	}
}

func TestCollectReadsOldPrice(t *testing.T) {
	page := `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","price":[12],"old_price":[15]}}}}</script>`
	result, _ := collectFixture(t, 200, page)
	if result.Result != "success" || result.OldPriceCents == nil || *result.OldPriceCents != 1500 || result.OldPriceAt != nil {
		t.Fatalf("unexpected result %+v", result)
	}
	result, _ = collectFixture(t, 200, listingHTML)
	if result.Result != "success" || result.OldPriceCents != nil {
		t.Fatalf("listing without old price: %+v", result)
	}
}
