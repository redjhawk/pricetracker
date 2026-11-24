package amazon

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"pricefollower.local/internal/model"
)

// Amazon thumbnails of issue #275: only a URL starting with https://m.media-amazon.com/ or
// https://images-eu.ssl-images-amazon.com/ is kept, the first such candidate in the page's order.

// thumbnailImages are the image candidates of a test page; an empty field leaves its tag out.
type thumbnailImages struct {
	landingHires string // data-old-hires of landingImage
	landingSrc   string // src of landingImage
	dynamic      string // src of an <img> carrying data-a-dynamic-image
	og           string // content of the og:image meta
	emptyLanding bool   // a landingImage with an empty data-old-hires and no src
	emptyOG      bool   // an og:image meta with an empty content
}

// thumbnailPage returns a product page with a euro price, a title and the given image candidates.
func thumbnailPage(images thumbnailImages) string {
	var head, body strings.Builder
	if images.og != "" || images.emptyOG {
		head.WriteString(`<meta property="og:image" content="` + html.EscapeString(images.og) + `">` + "\n")
	}
	switch {
	case images.emptyLanding:
		body.WriteString(`<img id="landingImage" data-old-hires="" alt="Thumbnail product">` + "\n")
	case images.landingHires != "" || images.landingSrc != "":
		body.WriteString(`<img id="landingImage"`)
		if images.landingSrc != "" {
			body.WriteString(` src="` + html.EscapeString(images.landingSrc) + `"`)
		}
		if images.landingHires != "" {
			body.WriteString(` data-old-hires="` + html.EscapeString(images.landingHires) + `"`)
		}
		body.WriteString(` alt="Thumbnail product">` + "\n")
	}
	if images.dynamic != "" {
		body.WriteString(`<img class="a-dynamic-image" data-a-dynamic-image='{"` + html.EscapeString(images.dynamic) + `":[300,300]}' src="` +
			html.EscapeString(images.dynamic) + `" alt="Thumbnail product">` + "\n")
	}
	return `<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="utf-8">
<title>Thumbnail product</title>
` + head.String() + `</head>
<body>
<div id="centerCol">
<h1><span id="productTitle">Thumbnail product</span></h1>
` + body.String() + `<div id="corePriceDisplay_desktop_feature_div">
<span class="a-price aok-align-center priceToPay"><span class="a-offscreen">129,99 €</span><span aria-hidden="true">129,99 €</span></span>
</div>
</div>
</body>
</html>
`
}

// parseThumbnail parses page, checks that the check succeeds and returns its thumbnail.
func parseThumbnail(t *testing.T, page string) *string {
	t.Helper()
	result, _ := parseProductPage(model.CollectionResult{}, page)
	if result.Result != "success" || result.AmountCents != 12999 || result.Message != "" || result.FailureCause != "" {
		t.Fatalf("result %q amount %d message %q cause %q, want a success at 129.99", result.Result, result.AmountCents, result.Message, result.FailureCause)
	}
	return result.ThumbnailURL
}

func thumbnailFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func describeThumbnail(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return "\"" + *value + "\""
}

// TC-001
func TestAmazonThumbnail_TC001_FixturePagesKeepTheirAmazonImage(t *testing.T) {
	for name, want := range map[string]string{
		"price-found.html":      "https://m.media-amazon.com/images/I/price-found.jpg",
		"mentions-captcha.html": "https://m.media-amazon.com/images/I/mentions-captcha.jpg",
		"consent-banner.html":   "https://m.media-amazon.com/images/I/mentions-captcha.jpg",
	} {
		t.Run(name, func(t *testing.T) {
			result, _ := parseProductPage(model.CollectionResult{}, thumbnailFixture(t, name))
			if result.ThumbnailURL == nil || *result.ThumbnailURL != want {
				t.Fatalf("thumbnail %s, want %q", describeThumbnail(result.ThumbnailURL), want)
			}
		})
	}
	t.Run("script-price.html", func(t *testing.T) {
		result, _ := parseProductPage(model.CollectionResult{}, thumbnailFixture(t, "script-price.html"))
		if result.ThumbnailURL != nil {
			t.Fatalf("thumbnail %s for the relative /static/pixel.png, want none", describeThumbnail(result.ThumbnailURL))
		}
	})
}

// TC-002
func TestAmazonThumbnail_TC002_AcceptedURLsAreKeptExactlyAsFound(t *testing.T) {
	for _, row := range []struct {
		name   string
		images thumbnailImages
		want   string
	}{
		{"landingImage data-old-hires on images-eu", thumbnailImages{landingHires: "https://images-eu.ssl-images-amazon.com/images/I/x.jpg"}, "https://images-eu.ssl-images-amazon.com/images/I/x.jpg"},
		{"landingImage src without data-old-hires", thumbnailImages{landingSrc: "https://m.media-amazon.com/images/I/src.jpg"}, "https://m.media-amazon.com/images/I/src.jpg"},
		{"dynamic image", thumbnailImages{dynamic: "https://m.media-amazon.com/images/I/dyn._AC_SX300_.jpg"}, "https://m.media-amazon.com/images/I/dyn._AC_SX300_.jpg"},
		{"og:image on images-eu", thumbnailImages{og: "https://images-eu.ssl-images-amazon.com/images/I/og.jpg"}, "https://images-eu.ssl-images-amazon.com/images/I/og.jpg"},
		{"path, query and fragment", thumbnailImages{landingHires: "https://m.media-amazon.com/images/I/q.jpg?v=1&w=500#f"}, "https://m.media-amazon.com/images/I/q.jpg?v=1&w=500#f"},
	} {
		t.Run(row.name, func(t *testing.T) {
			if got := parseThumbnail(t, thumbnailPage(row.images)); got == nil || *got != row.want {
				t.Fatalf("thumbnail %s, want %q", describeThumbnail(got), row.want)
			}
		})
	}
}

// TC-003
func TestAmazonThumbnail_TC003_ARejectedCandidateIsSkippedForTheNextAcceptedOne(t *testing.T) {
	for _, row := range []struct {
		name   string
		images thumbnailImages
		want   string
	}{
		{"http landingImage, accepted og:image", thumbnailImages{landingHires: "http://m.media-amazon.com/images/I/a.jpg", og: "https://m.media-amazon.com/images/I/b.jpg"},
			"https://m.media-amazon.com/images/I/b.jpg"},
		{"other host landingImage, first accepted wins", thumbnailImages{landingHires: "https://example.com/a.jpg", dynamic: "https://m.media-amazon.com/images/I/d.jpg",
			og: "https://m.media-amazon.com/images/I/b.jpg"}, "https://m.media-amazon.com/images/I/d.jpg"},
		{"two rejections before og:image", thumbnailImages{landingHires: "data:image/png;base64,AAAA", dynamic: "https://example.com/d.jpg",
			og: "https://images-eu.ssl-images-amazon.com/images/I/og.jpg"}, "https://images-eu.ssl-images-amazon.com/images/I/og.jpg"},
		{"rejected data-old-hires with a relative src", thumbnailImages{landingHires: "https://example.com/h.jpg", landingSrc: "/static/pixel.png",
			og: "https://m.media-amazon.com/images/I/b.jpg"}, "https://m.media-amazon.com/images/I/b.jpg"},
	} {
		t.Run(row.name, func(t *testing.T) {
			if got := parseThumbnail(t, thumbnailPage(row.images)); got == nil || *got != row.want {
				t.Fatalf("thumbnail %s, want %q", describeThumbnail(got), row.want)
			}
		})
	}
}

// TC-004
func TestAmazonThumbnail_TC004_RejectedURLsGiveNoThumbnail(t *testing.T) {
	rejected := []string{
		"http://m.media-amazon.com/images/I/a.jpg",
		"https://example.com/a.jpg",
		"https://m.media-amazon.com.example.com/a.jpg",
		"https://evil.com/https://m.media-amazon.com/a.jpg",
		"https://M.MEDIA-AMAZON.COM/images/I/a.jpg",
		"HTTPS://m.media-amazon.com/images/I/a.jpg",
		"https://m.media-amazon.com:443/images/I/a.jpg",
		"https://m.media-amazon.com:8443/a.jpg",
		"https://user@m.media-amazon.com/a.jpg",
		"https://images-na.ssl-images-amazon.com/images/I/a.jpg",
		"https://m.media-amazon.com",
		"//m.media-amazon.com/images/I/a.jpg",
		"/images/I/a.jpg",
		"data:image/jpeg;base64,AAAA",
		"javascript:alert(1)",
		" https://m.media-amazon.com/images/I/a.jpg",
		"",
	}
	for _, candidate := range []struct {
		name   string
		images func(value string) thumbnailImages
	}{
		{"landingImage", func(value string) thumbnailImages {
			return thumbnailImages{landingHires: value, emptyLanding: value == ""}
		}},
		{"og:image", func(value string) thumbnailImages { return thumbnailImages{og: value, emptyOG: value == ""} }},
	} {
		for _, value := range rejected {
			t.Run(fmt.Sprintf("%s %q", candidate.name, value), func(t *testing.T) {
				if got := parseThumbnail(t, thumbnailPage(candidate.images(value))); got != nil {
					t.Fatalf("thumbnail %s for the rejected %q, want none", describeThumbnail(got), value)
				}
			})
		}
	}
	t.Run("no image at all", func(t *testing.T) {
		if got := parseThumbnail(t, thumbnailPage(thumbnailImages{})); got != nil {
			t.Fatalf("thumbnail %s for a page without image, want none", describeThumbnail(got))
		}
	})
}

// TC-005
func TestAmazonThumbnail_TC005_TheRestOfTheResultIsUnaffectedByARejectedImage(t *testing.T) {
	page := thumbnailFixture(t, "price-found.html")
	original := `data-old-hires="https://m.media-amazon.com/images/I/price-found.jpg"`
	if !strings.Contains(page, original) {
		t.Fatalf("price-found.html has no %s", original)
	}
	rejectedPage := strings.Replace(page, original, `data-old-hires="https://example.com/a.jpg"`, 1)

	rejected, _ := parseProductPage(model.CollectionResult{}, rejectedPage)
	accepted, _ := parseProductPage(model.CollectionResult{}, page)
	if rejected.ThumbnailURL != nil {
		t.Fatalf("thumbnail %s for the rejected image, want none", describeThumbnail(rejected.ThumbnailURL))
	}
	if accepted.ThumbnailURL == nil || *accepted.ThumbnailURL != "https://m.media-amazon.com/images/I/price-found.jpg" {
		t.Fatalf("thumbnail %s for the unchanged page", describeThumbnail(accepted.ThumbnailURL))
	}
	if rejected.Result != "success" || rejected.AmountCents != 12999 || rejected.Title == nil || *rejected.Title != "Price found product" ||
		rejected.Message != "" || rejected.FailureCause != "" {
		t.Fatalf("result with a rejected image: %+v", rejected)
	}
	rejected.ThumbnailURL, accepted.ThumbnailURL = nil, nil
	if !reflect.DeepEqual(rejected, accepted) {
		t.Fatalf("results differ beyond the thumbnail:\nrejected image %+v\naccepted image %+v", rejected, accepted)
	}
}
