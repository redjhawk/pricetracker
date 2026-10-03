package leboncoin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"pricefollower.local/internal/model"
)

type sessionTransport func(*http.Request) (*http.Response, error)

func (f sessionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

var sessionListing = model.Listing{Platform: "leboncoin", ListingID: "123", URL: "https://www.leboncoin.fr/ad/test/123"}

const listingHTML = `<script id="__NEXT_DATA__">{"props":{"pageProps":{"ad":{"list_id":123,"status":"active","price":[12]}}}}</script>`

func sessionResponse(r *http.Request, status int, body string, cookies ...string) *http.Response {
	h := http.Header{"Content-Type": []string{"text/html"}}
	for _, cookie := range cookies {
		h.Add("Set-Cookie", cookie)
	}
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func syntheticSession() *Session { return &Session{Value: "synthetic-original"} }

func collectFixture(t *testing.T, status int, body string, cookies ...string) (model.CollectionResult, SessionOutcome) {
	t.Helper()
	c := NewCollector("test")
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		return sessionResponse(r, status, body, cookies...), nil
	})
	return c.CollectWithSession(context.Background(), sessionListing, syntheticSession())
}

func TestParseSessionInput(t *testing.T) {
	accepted := []struct{ input, want string }{
		{"abc123", "abc123"},
		{"  abc123 \n", "abc123"},
		{"datadome=abc123", "abc123"},
		{"Cookie: a=1; datadome=abc123; b=2", "abc123"},
		{"cookie:datadome=abc123", "abc123"},
		{"SET-COOKIE: datadome=abc123; Max-Age=31536000; Domain=.leboncoin.fr; Path=/; Secure", "abc123"},
		{"datadome=abc123; Max-Age=31536000; Domain=.leboncoin.fr; Path=/; Secure", "abc123"},
		{"datadome=abc123; HttpOnly; datadome=abc123", "abc123"},
		{"Xyz~AbC123_example", "Xyz~AbC123_example"},
		{"", ""},
		{"   \n\t ", ""},
		{strings.Repeat("v", 4096), strings.Repeat("v", 4096)},
	}
	for _, tc := range accepted {
		got, err := ParseSessionInput(tc.input)
		if err != nil || got != tc.want {
			t.Errorf("ParseSessionInput(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
		}
	}
	rejected := []struct{ input, message string }{
		{"a=1; b=2", "No datadome cookie was found in the pasted text."},
		{"Cookie: Datadome=abc", "No datadome cookie was found in the pasted text."},
		{"datadome=", "The datadome cookie value is empty."},
		{"datadome=abc; datadome=", "The datadome cookie value is empty."},
		{"datadome=abc; datadome=def", "The pasted text contains different datadome values; paste only one."},
		{"abc def", "The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."},
		{"abc\ndef", "The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."},
		{"datadome=\"abc\"", "The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."},
		{"datadome=a,b", "The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."},
		{"datadome=a\\b", "The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."},
		{"datadome=a\x01b", "The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."},
		{"datadome=café", "The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."},
		{strings.Repeat("v", 4097), "The datadome value is too long (maximum 4096 characters)."},
		{strings.Repeat("v", 8193), "The pasted text is too long (maximum 8192 characters)."},
	}
	for _, tc := range rejected {
		got, err := ParseSessionInput(tc.input)
		var inputError *SessionInputError
		if !errors.As(err, &inputError) || inputError.Message != tc.message || got != "" {
			t.Errorf("ParseSessionInput(%.40q) = %.40q, %v; want error %q", tc.input, got, err, tc.message)
		}
	}
}

func TestCollectWithoutSessionSendsNoCookie(t *testing.T) {
	c := NewCollector("test")
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Error("unexpected cookie")
		}
		return sessionResponse(r, 200, listingHTML, "datadome=synthetic-new; Domain=.leboncoin.fr; Path=/"), nil
	})
	result, outcome := c.CollectWithSession(context.Background(), sessionListing, nil)
	if result.Result != "success" || result.AmountCents != 1200 || outcome != (SessionOutcome{}) {
		t.Fatalf("result=%+v outcome=%+v", result, outcome)
	}
}

func TestSessionCookieAttachedToLeboncoinOrigins(t *testing.T) {
	c := NewCollector("test")
	calls := 0
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if got := r.Header.Get("Cookie"); got != "datadome=synthetic-original" {
			t.Errorf("request %d cookie = %q", calls, got)
		}
		if calls == 1 {
			res := sessionResponse(r, 302, "")
			res.Header.Set("Location", "https://leboncoin.fr/ad/test/123")
			return res, nil
		}
		return sessionResponse(r, 200, listingHTML), nil
	})
	result, outcome := c.CollectWithSession(context.Background(), sessionListing, syntheticSession())
	if result.Result != "success" || calls != 2 || outcome.Attempt != "accepted" || outcome.Renewed || outcome.Revoked {
		t.Fatalf("result=%+v outcome=%+v calls=%d", result, outcome, calls)
	}
}

func TestSessionExpiredValueNeverSent(t *testing.T) {
	c := NewCollector("test")
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Error("expired session sent")
		}
		return sessionResponse(r, 403, "blocked"), nil
	})
	past := time.Now().Add(-time.Minute)
	c.CollectWithSession(context.Background(), sessionListing, &Session{Value: "synthetic-original", ExpiresAt: &past})
}

func TestSessionOriginAndRedirectIsolation(t *testing.T) {
	unsafe := []string{"http://www.leboncoin.fr/ad/test/123", "https://user@www.leboncoin.fr/ad/test/123", "https://www.leboncoin.fr:444/ad/test/123", "https://evil.leboncoin.fr/ad/test/123", "https://leboncoin.fr.example/ad/test/123", "https://www.amazon.fr/ad/test/123"}
	for _, destination := range unsafe {
		t.Run(destination, func(t *testing.T) {
			c := NewCollector("test")
			calls := 0
			c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					t.Errorf("transport saw redirect to %s", r.URL)
				}
				res := sessionResponse(r, 302, "")
				res.Header.Set("Location", destination)
				return res, nil
			})
			result, outcome := c.CollectWithSession(context.Background(), sessionListing, syntheticSession())
			if result.Result != "request_error" || calls != 1 || outcome.Attempt != "failed" {
				t.Fatalf("redirect result=%s outcome=%+v calls=%d", result.Result, outcome, calls)
			}
			bad := sessionListing
			bad.URL = destination
			calls = 0
			if result, _ := c.CollectWithSession(context.Background(), bad, syntheticSession()); result.Result != "request_error" || calls != 0 {
				t.Fatalf("initial result=%s calls=%d", result.Result, calls)
			}
		})
	}
	c := NewCollector("test")
	calls := 0
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		res := sessionResponse(r, 302, "")
		res.Header.Set("Location", "https://www.leboncoin.fr/ad/test/123?hop="+strconv.Itoa(calls))
		return res, nil
	})
	if result, _ := c.CollectWithSession(context.Background(), sessionListing, syntheticSession()); result.Result != "request_error" || calls != 4 {
		t.Fatalf("redirect bound=%d result=%s", calls, result.Result)
	}
}

func TestSessionOutcomeClassificationAndVerifiedRenewal(t *testing.T) {
	const rejectedMessage = "LeBoncoin rejected the saved session. Capture a new session and save it in Settings."
	const failedMessage = "LeBoncoin could not be reached for a price check."
	challenge := `<html><script src="https://ct.captcha-delivery.com/c.js"></script></html>`
	cases := []struct {
		name, body, result, message, attempt string
		status                               int
		renewed                              bool
	}{
		{"success", listingHTML, "success", "", "accepted", 200, true},
		{"inactive", strings.Replace(listingHTML, `active`, `deleted`, 1), "unavailable", "The listing is no longer available.", "accepted", 200, true},
		{"missing-price", strings.Replace(listingHTML, `,"price":[12]`, "", 1), "price_not_found", "LeBoncoin did not show a detectable euro price.", "accepted", 200, true},
		{"donation", strings.Replace(listingHTML, `"price":[12]`, `"subject":"don gratuit"`, 1), "success", "", "accepted", 200, true},
		{"403", "blocked", "request_error", rejectedMessage, "rejected", 403, false},
		{"challenge-page", challenge, "request_error", rejectedMessage, "rejected", 200, false},
		{"malformed", `<script id="__NEXT_DATA__">synthetic-secret</script>`, "request_error", failedMessage, "failed", 200, false},
		{"wrong-id", strings.Replace(listingHTML, `123`, `999`, 1), "request_error", failedMessage, "failed", 200, false},
		{"404", "", "unavailable", "The listing is no longer available.", "failed", 404, false},
		{"410", "", "unavailable", "The listing is no longer available.", "failed", 410, false},
		{"500", "", "request_error", failedMessage, "failed", 500, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, outcome := collectFixture(t, tc.status, tc.body, "datadome=synthetic-new; Domain=leboncoin.fr; Path=/; Secure; Max-Age=3600")
			if result.Result != tc.result || result.Message != tc.message || outcome.Attempt != tc.attempt || outcome.Renewed != tc.renewed || outcome.Revoked {
				t.Fatalf("result=%+v outcome=%+v", result, outcome)
			}
			if tc.renewed && (outcome.Value != "synthetic-new" || outcome.ExpiresAt == nil || outcome.ExpiresAt.Before(time.Now().Add(59*time.Minute))) {
				t.Fatalf("renewal %+v", outcome)
			}
		})
	}
}

func TestSessionNetworkErrorIsFailedAndUndisclosed(t *testing.T) {
	c := NewCollector("test")
	var diagnostics bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&diagnostics)
	defer log.SetOutput(previous)
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("synthetic-private-cookie")
	})
	result, outcome := c.CollectWithSession(context.Background(), sessionListing, syntheticSession())
	if result.Result != "request_error" || outcome.Attempt != "failed" || strings.Contains(result.Message+diagnostics.String(), "synthetic-private-cookie") || strings.Contains(diagnostics.String(), "synthetic-original") {
		t.Fatalf("unsafe diagnostic or outcome %+v", outcome)
	}
}

func TestSessionResponseCookieMetadata(t *testing.T) {
	cases := []struct {
		name, header string
		renewed      bool
		value        string
	}{
		{"domain-with-dot", "datadome=new; Domain=.leboncoin.fr; Path=/", true, "new"},
		{"session-cookie", "datadome=new; Domain=leboncoin.fr; Path=/", true, "new"},
		{"host-only", "datadome=new; Path=/", false, ""},
		{"default-path", "datadome=new; Domain=leboncoin.fr", false, ""},
		{"other-path", "datadome=new; Domain=leboncoin.fr; Path=/ad", false, ""},
		{"wrong-domain", "datadome=new; Domain=evil.leboncoin.fr; Path=/", false, ""},
		{"www-domain", "datadome=new; Domain=www.leboncoin.fr; Path=/", false, ""},
		{"unrelated", "account=secret; Domain=leboncoin.fr; Path=/", false, ""},
		{"invalid-value", "datadome=\"a b\"; Domain=leboncoin.fr; Path=/", false, ""},
		{"overflow", "datadome=new; Domain=leboncoin.fr; Path=/; Max-Age=9999999999999999999999999999999", false, ""},
		{"duration-overflow", "datadome=new; Domain=leboncoin.fr; Path=/; Max-Age=9223372036854775807", false, ""},
		{"maxage-priority", "datadome=new; Domain=leboncoin.fr; Path=/; Max-Age=3600; Expires=Thu, 01 Jan 1970 00:00:00 GMT", true, "new"},
		{"wrong-path-deletion", "datadome=; Domain=leboncoin.fr; Path=/other; Max-Age=0", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, outcome := collectFixture(t, 200, listingHTML, tc.header)
			if result.Result != "success" || outcome.Attempt != "accepted" || outcome.Renewed != tc.renewed || outcome.Value != tc.value || outcome.Revoked {
				t.Fatalf("result=%+v outcome=%+v", result, outcome)
			}
		})
	}
	t.Run("expires", func(t *testing.T) {
		expiry := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
		_, outcome := collectFixture(t, 200, listingHTML, "datadome=new; Domain=leboncoin.fr; Path=/; Expires="+expiry.Format(http.TimeFormat))
		if !outcome.Renewed || outcome.ExpiresAt == nil || !outcome.ExpiresAt.Equal(expiry) {
			t.Fatalf("outcome=%+v", outcome)
		}
	})
	t.Run("ambiguous", func(t *testing.T) {
		_, outcome := collectFixture(t, 200, listingHTML, "datadome=first; Domain=leboncoin.fr; Path=/", "datadome=second; Domain=leboncoin.fr; Path=/")
		if outcome.Renewed || outcome.Attempt != "accepted" {
			t.Fatalf("ambiguous renewal accepted %+v", outcome)
		}
	})
	t.Run("identical-duplicates", func(t *testing.T) {
		_, outcome := collectFixture(t, 200, listingHTML, "datadome=same; Domain=leboncoin.fr; Path=/", "datadome=same; Domain=.leboncoin.fr; Path=/")
		if !outcome.Renewed || outcome.Value != "same" {
			t.Fatalf("outcome=%+v", outcome)
		}
	})
}

func TestSessionDeletionIsRevocationOnAnyStatus(t *testing.T) {
	for _, status := range []int{200, 403, 404, 410} {
		for _, header := range []string{"datadome=; Domain=leboncoin.fr; Path=/; Max-Age=0", "datadome=gone; Domain=.leboncoin.fr; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT"} {
			t.Run(strconv.Itoa(status)+header, func(t *testing.T) {
				_, outcome := collectFixture(t, status, listingHTML, header)
				if !outcome.Revoked || outcome.Renewed {
					t.Fatalf("outcome=%+v", outcome)
				}
			})
		}
	}
	t.Run("replacement-before-deletion-in-same-response", func(t *testing.T) {
		_, outcome := collectFixture(t, 200, listingHTML, "datadome=new; Domain=leboncoin.fr; Path=/", "datadome=; Domain=leboncoin.fr; Path=/; Max-Age=0")
		if !outcome.Revoked || outcome.Renewed {
			t.Fatalf("outcome=%+v", outcome)
		}
	})
}

type failingSessionBody struct{}

func (failingSessionBody) Read([]byte) (int, error) { return 0, errors.New("synthetic-private-error") }
func (failingSessionBody) Close() error             { return nil }

func TestSessionDeletionSurvivesReadError(t *testing.T) {
	c := NewCollector("test")
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		res := sessionResponse(r, 200, listingHTML, "datadome=; Domain=leboncoin.fr; Path=/; Max-Age=0")
		res.Body = failingSessionBody{}
		return res, nil
	})
	result, outcome := c.CollectWithSession(context.Background(), sessionListing, syntheticSession())
	if result.Result != "request_error" || !outcome.Revoked || outcome.Attempt != "failed" {
		t.Fatalf("result=%+v outcome=%+v", result, outcome)
	}
}

func TestSessionRedirectStagingAndDeletion(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		deletion, success, escape bool
		revoked, renewed          bool
		value                     string
	}{
		{"accepted", false, true, false, false, true, "final"},
		{"discarded", false, false, false, false, false, ""},
		{"revoked", true, false, false, true, false, ""},
		{"revoked-before-redirect-error", true, false, true, true, false, ""},
		{"replacement-after-revocation", true, true, false, false, true, "final"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCollector("test")
			calls := 0
			c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					cookie := "datadome=redirect; Domain=leboncoin.fr; Path=/"
					if tc.deletion {
						cookie = "datadome=; Domain=leboncoin.fr; Path=/; Max-Age=0"
					}
					res := sessionResponse(r, 302, "", cookie)
					destination := "https://www.leboncoin.fr/ad/test/123?final"
					if tc.escape {
						destination = "https://example.com"
					}
					res.Header.Set("Location", destination)
					return res, nil
				}
				want := "datadome=redirect"
				if tc.deletion {
					want = ""
				}
				if r.Header.Get("Cookie") != want {
					t.Errorf("staged cookie %q", r.Header.Get("Cookie"))
				}
				status := 403
				if tc.success {
					status = 200
				}
				return sessionResponse(r, status, listingHTML, "datadome=final; Domain=leboncoin.fr; Path=/"), nil
			})
			_, outcome := c.CollectWithSession(context.Background(), sessionListing, syntheticSession())
			if outcome.Revoked != tc.revoked || outcome.Renewed != tc.renewed || outcome.Value != tc.value {
				t.Fatalf("outcome=%+v", outcome)
			}
		})
	}
}

func TestReviewBarePrefixHasNoDatadome(t *testing.T) {
	for _, input := range []string{"Cookie:", "set-cookie:  ", "Cookie: ;"} {
		_, err := ParseSessionInput(input)
		var inputError *SessionInputError
		if !errors.As(err, &inputError) || inputError.Message != "No datadome cookie was found in the pasted text." {
			t.Errorf("ParseSessionInput(%q) error %v", input, err)
		}
	}
}

func TestReviewUnknownAttributesIgnored(t *testing.T) {
	_, outcome := collectFixture(t, 200, listingHTML, "datadome=new; Domain=leboncoin.fr; Path=/; Priority=High")
	if !outcome.Renewed || outcome.Value != "new" {
		t.Fatalf("unknown attribute blocked renewal %+v", outcome)
	}
	_, outcome = collectFixture(t, 200, listingHTML, "datadome=; Domain=leboncoin.fr; Path=/; Max-Age=0; Priority=High")
	if !outcome.Revoked {
		t.Fatalf("unknown attribute blocked deletion %+v", outcome)
	}
	future := time.Now().Add(48 * time.Hour).UTC().Format(http.TimeFormat)
	for _, header := range []string{"datadome=new; Domain=leboncoin.fr; Path=/; Max-Age=abc; Expires=" + future, "datadome=new; Domain=leboncoin.fr; Path=/; Expires=garbage"} {
		if _, outcome := collectFixture(t, 200, listingHTML, header); outcome.Renewed || outcome.Revoked {
			t.Fatalf("malformed metadata accepted %q %+v", header, outcome)
		}
	}
}

func TestReviewInvalidURLReturnsZeroOutcome(t *testing.T) {
	c := NewCollector("test")
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		t.Error("request sent")
		return nil, errors.New("unexpected")
	})
	bad := sessionListing
	bad.URL = "https://example.com/ad/test/123"
	result, outcome := c.CollectWithSession(context.Background(), bad, syntheticSession())
	if result.Result != "request_error" || outcome != (SessionOutcome{}) {
		t.Fatalf("result=%+v outcome=%+v", result, outcome)
	}
}
