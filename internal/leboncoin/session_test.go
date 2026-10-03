package leboncoin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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
func writeImport(t *testing.T, path, value string) []byte {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"version": 1, "capturedAt": time.Now().UTC().Format(time.RFC3339Nano), "cookie": map[string]any{"name": "datadome", "value": value, "domain": ".leboncoin.fr", "path": "/", "secure": true, "expiresAt": nil}})
	temporary := path + ".renewal"
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporary, path); err != nil {
		t.Fatal(err)
	}
	return data
}
func privateImport(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "session.json")
	writeImport(t, path, "synthetic-original")
	return path
}
func TestSessionCookieAttached(t *testing.T) {
	path := privateImport(t)
	c := NewCollectorWithSession("test", path)
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Cookie"); got != "datadome=synthetic-original" {
			t.Errorf("cookie = %q", got)
		}
		return sessionResponse(r, 200, listingHTML), nil
	})
	if got := c.Collect(context.Background(), sessionListing); got.Result != "success" || got.AmountCents != 1200 {
		t.Fatalf("result %+v", got)
	}
}
func TestSessionMissingFailsBeforeRequest(t *testing.T) {
	c := NewCollectorWithSession("test", filepath.Join(t.TempDir(), "missing"))
	calls := 0
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return sessionResponse(r, 200, listingHTML), nil
	})
	if got := c.Collect(context.Background(), sessionListing); got.Result != "request_error" || calls != 0 {
		t.Fatalf("result=%s requests=%d", got.Result, calls)
	}
}
func TestSessionVerifiedUpdateSurvivesRestart(t *testing.T) {
	path := privateImport(t)
	original, _ := os.ReadFile(path)
	c := NewCollectorWithSession("test", path)
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		return sessionResponse(r, 200, listingHTML, "datadome=synthetic-new; Domain=leboncoin.fr; Path=/; Secure; Max-Age=3600"), nil
	})
	if got := c.Collect(context.Background(), sessionListing); got.Result != "success" {
		t.Fatal(got)
	}
	current, _ := os.ReadFile(path)
	if string(current) != string(original) {
		t.Fatal("import overwritten")
	}
	restarted := NewCollectorWithSession("test", path)
	restarted.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "datadome=synthetic-new" {
			t.Error("restart did not use saved update")
		}
		return sessionResponse(r, 200, listingHTML), nil
	})
	restarted.Collect(context.Background(), sessionListing)
}

func runFixture(t *testing.T, c *Collector, status int, body string, cookies ...string) model.CollectionResult {
	t.Helper()
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		return sessionResponse(r, status, body, cookies...), nil
	})
	return c.Collect(context.Background(), sessionListing)
}
func assertSent(t *testing.T, c *Collector, value string) {
	t.Helper()
	called := false
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		called = true
		if got := r.Header.Get("Cookie"); got != "datadome="+value {
			t.Errorf("unexpected cookie %q", got)
		}
		return sessionResponse(r, 200, listingHTML), nil
	})
	if got := c.Collect(context.Background(), sessionListing); got.Result != "success" || !called {
		t.Fatalf("collection failed: %+v", got)
	}
}
func TestSessionInvalidImports(t *testing.T) {
	cases := map[string]func(string){
		"malformed": func(p string) { os.WriteFile(p, []byte(`{"secret":"synthetic-private"`), 0600) },
		"oversized": func(p string) { os.WriteFile(p, []byte(strings.Repeat(" ", sessionLimit+1)), 0600) },
		"unknown-version": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), `"version":1`, `"version":2`, 1)), 0600)
		},
		"unknown-field": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), `"version":1`, `"version":1,"extra":true`, 1)), 0600)
		},
		"missing-secure": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), `"secure":true,`, "", 1)), 0600)
		},
		"null-secure": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), `"secure":true`, `"secure":null`, 1)), 0600)
		},
		"duplicate": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), `"version":1`, `"version":1,"version":1`, 1)), 0600)
		},
		"trailing": func(p string) { d, _ := os.ReadFile(p); os.WriteFile(p, append(d, []byte(` {}`)...), 0600) },
		"expired": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), `"expiresAt":null`, `"expiresAt":"2000-01-01T00:00:00Z"`, 1)), 0600)
		},
		"bad-timestamp": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), `"expiresAt":null`, `"expiresAt":"2030-01-01T00:00:00+01:00"`, 1)), 0600)
		},
		"unsafe-mode":      func(p string) { os.Chmod(p, 0640) },
		"unsafe-directory": func(p string) { os.Chmod(filepath.Dir(p), 0750) },
		"symlink":          func(p string) { os.Rename(p, p+".real"); os.Symlink(p+".real", p) },
		"directory":        func(p string) { os.Remove(p); os.Mkdir(p, 0700) },
		"fifo":             func(p string) { os.Remove(p); syscall.Mkfifo(p, 0600) },
		"unsafe-cookie": func(p string) {
			d, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.Replace(string(d), "synthetic-original", "bad;cookie", 1)), 0600)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := privateImport(t)
			mutate(p)
			c := NewCollectorWithSession("test", p)
			calls := 0
			c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return sessionResponse(r, 200, listingHTML), nil
			})
			if got := c.Collect(context.Background(), sessionListing); got.Result != "request_error" || calls != 0 {
				t.Fatalf("result=%s calls=%d", got.Result, calls)
			}
		})
	}
}
func TestSessionOriginAndRedirectIsolation(t *testing.T) {
	unsafe := []string{"http://www.leboncoin.fr/ad/test/123", "https://user@www.leboncoin.fr/ad/test/123", "https://www.leboncoin.fr:444/ad/test/123", "https://evil.leboncoin.fr/ad/test/123", "https://leboncoin.fr.example/ad/test/123", "https://www.amazon.fr/ad/test/123"}
	for _, destination := range unsafe {
		t.Run(destination, func(t *testing.T) {
			p := privateImport(t)
			c := NewCollectorWithSession("test", p)
			calls := 0
			c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				res := sessionResponse(r, 302, "")
				res.Header.Set("Location", destination)
				return res, nil
			})
			if got := c.Collect(context.Background(), sessionListing); got.Result != "request_error" || calls != 1 {
				t.Fatalf("redirect result=%s calls=%d", got.Result, calls)
			}
			bad := sessionListing
			bad.URL = destination
			calls = 0
			if got := c.Collect(context.Background(), bad); got.Result != "request_error" || calls != 0 {
				t.Fatalf("initial result=%s calls=%d", got.Result, calls)
			}
		})
	}
	c := NewCollectorWithSession("test", privateImport(t))
	calls := 0
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		res := sessionResponse(r, 302, "")
		res.Header.Set("Location", "https://www.leboncoin.fr/ad/test/123?hop="+strconv.Itoa(calls))
		return res, nil
	})
	if result := c.Collect(context.Background(), sessionListing); result.Result != "request_error" || calls != 4 {
		t.Fatalf("redirect bound=%d result=%s", calls, result.Result)
	}
}
func TestSessionScope(t *testing.T) {
	for _, tc := range []struct {
		domain, path, destination string
		want                      bool
	}{
		{".leboncoin.fr", "/", "https://leboncoin.fr/ad/test/123", true},
		{"www.leboncoin.fr", "/", "https://leboncoin.fr/ad/test/123", false},
		{".www.leboncoin.fr", "/", "https://evil.www.leboncoin.fr/ad/test/123", false},
		{".leboncoin.fr", "/ad/test", "https://www.leboncoin.fr/ad/test/123", true},
		{".leboncoin.fr", "/ad/tes", "https://www.leboncoin.fr/ad/test/123", false},
		{".leboncoin.fr", "/other", "https://www.leboncoin.fr/ad/test/123", false},
	} {
		u, _ := url.Parse(tc.destination)
		cookie := &sessionCookie{Domain: tc.domain, Path: tc.path}
		if got := cookieApplies(cookie, u); got != tc.want {
			t.Fatalf("scope %+v: %t", tc, got)
		}
	}
}
func TestSessionUpdatesRequireVerifiedListing(t *testing.T) {
	cases := []struct {
		name         string
		status       int
		body, result string
	}{
		{"403", 403, "challenge", "request_error"}, {"malformed", 200, `<script id="__NEXT_DATA__">synthetic-secret</script>`, "request_error"}, {"wrong-id", 200, strings.Replace(listingHTML, `123`, `999`, 1), "request_error"}, {"404", 404, "", "unavailable"}, {"inactive", 200, strings.Replace(listingHTML, `active`, `deleted`, 1), "unavailable"}, {"missing-price", 200, strings.Replace(listingHTML, `,"price":[12]`, "", 1), "price_not_found"}, {"donation", 200, strings.Replace(listingHTML, `"price":[12]`, `"subject":"don gratuit"`, 1), "success"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := privateImport(t)
			c := NewCollectorWithSession("test", p)
			result := runFixture(t, c, tc.status, tc.body, "datadome=synthetic-new; Domain=leboncoin.fr; Path=/; Secure")
			if result.Result != tc.result {
				t.Fatal(result)
			}
			want := "synthetic-original"
			if tc.name == "inactive" || tc.name == "missing-price" || tc.name == "donation" {
				want = "synthetic-new"
			}
			assertSent(t, NewCollectorWithSession("test", p), want)
		})
	}
}
func TestSessionDeletionPersistsEvenOnErrors(t *testing.T) {
	for _, status := range []int{200, 403, 404, 410} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			p := privateImport(t)
			c := NewCollectorWithSession("test", p)
			runFixture(t, c, status, listingHTML, "datadome=; Domain=leboncoin.fr; Path=/; Max-Age=0")
			for _, collector := range []*Collector{c, NewCollectorWithSession("test", p)} {
				calls := 0
				collector.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					return sessionResponse(r, 200, listingHTML), nil
				})
				if got := collector.Collect(context.Background(), sessionListing); got.Result != "request_error" || calls != 0 {
					t.Fatalf("deleted cookie restored: %+v calls=%d", got, calls)
				}
			}
			data, _ := os.ReadFile(p + ".state.json")
			if !strings.Contains(string(data), `"cookie":null`) {
				t.Fatal("missing tombstone")
			}
		})
	}
}
func TestSessionResponseMetadata(t *testing.T) {
	cases := []struct{ name, header, want string }{
		{"wrong-path-deletion", "datadome=; Domain=leboncoin.fr; Path=/other; Max-Age=0", "synthetic-original"},
		{"wrong-domain", "datadome=new; Domain=evil.leboncoin.fr; Path=/", "synthetic-original"},
		{"unrelated", "account=secret; Domain=leboncoin.fr; Path=/", "synthetic-original"},
		{"overflow", "datadome=new; Domain=leboncoin.fr; Path=/; Max-Age=9999999999999999999999999999999", "synthetic-original"},
		{"duration-overflow", "datadome=new; Domain=leboncoin.fr; Path=/; Max-Age=9223372036854775807", "synthetic-original"},
		{"maxage-priority", "datadome=new; Domain=leboncoin.fr; Path=/; Max-Age=3600; Expires=Thu, 01 Jan 1970 00:00:00 GMT", "new"},
		{"default-path", "datadome=new; Secure", "new"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := privateImport(t)
			c := NewCollectorWithSession("test", p)
			if got := runFixture(t, c, 200, listingHTML, tc.header); got.Result != "success" {
				t.Fatal(got)
			}
			assertSent(t, NewCollectorWithSession("test", p), tc.want)
		})
	}
	p := privateImport(t)
	c := NewCollectorWithSession("test", p)
	runFixture(t, c, 200, listingHTML, "datadome=first; Domain=leboncoin.fr; Path=/", "datadome=second; Domain=leboncoin.fr; Path=/ad")
	assertSent(t, NewCollectorWithSession("test", p), "synthetic-original")
}
func TestSessionRedirectStagingAndDeletion(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		deletion, success, escape bool
	}{{"accepted", false, true, false}, {"discarded", false, false, false}, {"revoked", true, false, false}, {"revoked-before-redirect-error", true, false, true}, {"replacement-after-revocation", true, true, false}} {
		t.Run(tc.name, func(t *testing.T) {
			p := privateImport(t)
			c := NewCollectorWithSession("test", p)
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
			c.Collect(context.Background(), sessionListing)
			restarted := NewCollectorWithSession("test", p)
			if tc.deletion && !tc.success {
				if result := restarted.Collect(context.Background(), sessionListing); result.Result != "request_error" {
					t.Fatal("revocation lost")
				}
			} else {
				want := "synthetic-original"
				if tc.success {
					want = "final"
				}
				assertSent(t, restarted, want)
			}
		})
	}
}
func TestSessionRenewalDuringRequestAndCancellation(t *testing.T) {
	p := privateImport(t)
	c := NewCollectorWithSession("test", p)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan model.CollectionResult, 1)
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		close(entered)
		<-release
		return sessionResponse(r, 200, listingHTML, "datadome=stale; Domain=leboncoin.fr; Path=/"), nil
	})
	go func() { done <- c.Collect(context.Background(), sessionListing) }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := c.Collect(ctx, sessionListing); result.Result != "request_error" {
		t.Fatal(result)
	}
	newImport := writeImport(t, p, "renewed")
	close(release)
	if got := <-done; got.Result != "success" {
		t.Fatal(got)
	}
	data, _ := os.ReadFile(p)
	if string(data) != string(newImport) {
		t.Fatal("renewal overwritten")
	}
	assertSent(t, c, "renewed")
	assertSent(t, NewCollectorWithSession("test", p), "renewed")
}
func TestSessionParallelAttempts(t *testing.T) {
	c := NewCollectorWithSession("test", privateImport(t))
	var active atomic.Int32
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		if active.Add(1) != 1 {
			t.Error("requests overlap")
		}
		defer active.Add(-1)
		return sessionResponse(r, 200, listingHTML, "datadome=next; Domain=leboncoin.fr; Path=/"), nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := c.Collect(context.Background(), sessionListing); got.Result != "success" {
				t.Error(got)
			}
		}()
	}
	wg.Wait()
}
func TestSessionPersistenceFailureRetainsPriceAndMemory(t *testing.T) {
	p := privateImport(t)
	original, _ := os.ReadFile(p)
	c := NewCollectorWithSession("test", p)
	var diagnostics bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&diagnostics)
	defer log.SetOutput(previous)
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		os.Symlink(p, p+".state.json")
		return sessionResponse(r, 200, listingHTML, "datadome=synthetic-sensitive-update; Domain=leboncoin.fr; Path=/"), nil
	})
	if result := c.Collect(context.Background(), sessionListing); result.Result != "success" || result.AmountCents != 1200 {
		t.Fatal(result)
	}
	data, _ := os.ReadFile(p)
	if string(data) != string(original) {
		t.Fatal("import overwritten")
	}
	if !strings.Contains(diagnostics.String(), "could not be saved") || strings.Contains(diagnostics.String(), "synthetic-sensitive") {
		t.Fatal("unsafe or absent diagnostic")
	}
	assertSent(t, c, "synthetic-sensitive-update")
	if result := NewCollectorWithSession("test", p).Collect(context.Background(), sessionListing); result.Result != "request_error" {
		t.Fatal("unsafe sidecar accepted")
	}
}
func TestSessionSidecarSelectionAndProtection(t *testing.T) {
	p := privateImport(t)
	data, _ := os.ReadFile(p)
	data = []byte(strings.Replace(string(data), `"expiresAt":null`, `"expiresAt":"2000-01-01T00:00:00Z"`, 1))
	os.WriteFile(p, data, 0600)
	digest := sha256.Sum256(data)
	fingerprint := hex.EncodeToString(digest[:])
	cookie := map[string]any{"name": "datadome", "value": "extended", "domain": ".leboncoin.fr", "path": "/", "secure": true, "expiresAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	state, _ := json.Marshal(map[string]any{"version": 1, "importFingerprint": fingerprint, "cookie": cookie})
	os.WriteFile(p+".state.json", state, 0600)
	assertSent(t, NewCollectorWithSession("test", p), "extended")
	for _, mutate := range []func(){func() { os.WriteFile(p+".state.json", []byte(`bad`), 0600) }, func() { os.WriteFile(p+".state.json", state, 0600); os.Chmod(p+".state.json", 0644) }} {
		mutate()
		if got := NewCollectorWithSession("test", p).Collect(context.Background(), sessionListing); got.Result != "request_error" {
			t.Fatal("unsafe state accepted")
		}
	}
}
func TestSessionlessUnchanged(t *testing.T) {
	c := NewCollectorWithSession("test", "")
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Error("unexpected cookie")
		}
		return sessionResponse(r, 200, listingHTML), nil
	})
	if got := c.Collect(context.Background(), sessionListing); got.Result != "success" {
		t.Fatal(got)
	}
}

type failingSessionBody struct{}

func (failingSessionBody) Read([]byte) (int, error) { return 0, errors.New("synthetic-private-error") }
func (failingSessionBody) Close() error             { return nil }
func TestSessionDeletionSurvivesReadErrorAndHeaderOrdering(t *testing.T) {
	for _, readFailure := range []bool{false, true} {
		p := privateImport(t)
		c := NewCollectorWithSession("test", p)
		c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
			res := sessionResponse(r, 200, listingHTML, "datadome=new; Domain=leboncoin.fr; Path=/", "datadome=; Domain=leboncoin.fr; Path=/; Max-Age=0")
			if readFailure {
				res.Body = failingSessionBody{}
			}
			return res, nil
		})
		c.Collect(context.Background(), sessionListing)
		restarted := NewCollectorWithSession("test", p)
		restarted.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) { t.Fatal("revoked session sent"); return nil, nil })
		if got := restarted.Collect(context.Background(), sessionListing); got.Result != "request_error" {
			t.Fatal(got)
		}
	}
}
func TestSessionFailedSavePreservesPreviousCompleteState(t *testing.T) {
	p := privateImport(t)
	c := NewCollectorWithSession("test", p)
	runFixture(t, c, 200, listingHTML, "datadome=first; Domain=leboncoin.fr; Path=/")
	previous, _ := os.ReadFile(p + ".state.json")
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) {
		if err := os.Chmod(filepath.Dir(p), 0750); err != nil {
			t.Fatal(err)
		}
		return sessionResponse(r, 200, listingHTML, "datadome=second; Domain=leboncoin.fr; Path=/"), nil
	})
	if got := c.Collect(context.Background(), sessionListing); got.Result != "success" {
		t.Fatal(got)
	}
	current, _ := os.ReadFile(p + ".state.json")
	if string(current) != string(previous) {
		t.Fatal("previous state damaged")
	}
	os.Chmod(filepath.Dir(p), 0700)
	assertSent(t, c, "second")
	assertSent(t, NewCollectorWithSession("test", p), "first")
	info, err := os.Stat(p + ".state.json")
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("sidecar permissions")
	}
}
func TestSessionErrorsDoNotDiscloseUpstreamDetails(t *testing.T) {
	c := NewCollectorWithSession("test", privateImport(t))
	var diagnostics bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&diagnostics)
	defer log.SetOutput(previous)
	c.client.Transport = sessionTransport(func(r *http.Request) (*http.Response, error) { return nil, errors.New("synthetic-private-cookie") })
	result := c.Collect(context.Background(), sessionListing)
	if result.Result != "request_error" || strings.Contains(result.Message+diagnostics.String(), "synthetic-private-cookie") {
		t.Fatal("unsafe diagnostic")
	}
}
