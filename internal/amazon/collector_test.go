package amazon

import "testing"

func priceSpan(class, value string) string {
	return `<span class="a-price ` + class + `"><span class="a-offscreen">` + value + `</span><span aria-hidden="true">` + value + `</span></span>`
}

func priceBlockMessage() string {
	return `<span id="price-block-message" class="a-size-base price-block-message"><span id="price-block-amount-prefix">Ou </span><span id="price-block-amount">` +
		priceSpan("a-text-normal", "128,65€") + `</span> x4 (0,0% de frais inclus)</span>`
}

func TestProductPriceIgnoresInstalments(t *testing.T) {
	tests := []struct {
		name   string
		page   string
		amount int64
		found  bool
	}{
		{
			name: "issue 45 instalment after price",
			page: `<div id="corePriceDisplay_desktop_feature_div">` + priceSpan("priceToPay", "514,63 €") +
				`<div>Ou ` + priceSpan("priceToPay", "128,65 €") + ` x4 mois (2,3% de frais inclus)</div></div>`,
			amount: 51463, found: true,
		},
		{
			name: "instalment before price",
			page: `<div id="corePriceDisplay_desktop_feature_div"><div>Ou ` + priceSpan("priceToPay", "128,65 €") +
				` x4 mois</div>` + priceSpan("priceToPay", "514,63 €") + `</div>`,
			amount: 51463, found: true,
		},
		{
			name: "instalment container",
			page: `<div id="corePrice_feature_div"><div id="inemi_feature_div">` + priceSpan("", "128,65 €") +
				`</div>` + priceSpan("", "514,63 €") + `</div>`,
			amount: 51463, found: true,
		},
		{
			name: "german instalment",
			page: `<div id="corePriceDisplay_desktop_feature_div">` + priceSpan("priceToPay", "514,63 €") +
				`<div>Oder ` + priceSpan("", "128,65 €") + ` x4 Monate</div></div>`,
			amount: 51463, found: true,
		},
		{
			name:   "live price block message after price",
			page:   `<div id="corePriceDisplay_desktop_feature_div">` + priceSpan("priceToPay", "514,63 €") + priceBlockMessage() + `</div>`,
			amount: 51463, found: true,
		},
		{
			name:   "live price block message before price",
			page:   `<div id="corePriceDisplay_desktop_feature_div">` + priceBlockMessage() + priceSpan("priceToPay", "514,63 €") + `</div>`,
			amount: 51463, found: true,
		},
		{
			name: "instalment only",
			page: `<div id="corePriceDisplay_desktop_feature_div"><div>Ou ` + priceSpan("priceToPay", "128,65 €") +
				` x4 mois (2,3% de frais inclus)</div></div>`,
			found: false,
		},
		{
			name: "normal price",
			page: `<div id="corePriceDisplay_desktop_feature_div">` + priceSpan("priceToPay", "49,99 €") +
				`<span>Livraison gratuite</span></div>`,
			amount: 4999, found: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			amount, found := productPrice(test.page)
			if found != test.found || amount != test.amount {
				t.Fatalf("productPrice() = %d, %v; want %d, %v", amount, found, test.amount, test.found)
			}
		})
	}
}
