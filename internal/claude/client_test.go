package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const testToken = "sk-ant-oat01-synthetic"

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewClient()
	client.url = server.URL
	return client
}

func TestVerifySendsSubscriptionHeadersAndPreamble(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+testToken ||
			r.Header.Get("anthropic-version") != "2023-06-01" || r.Header.Get("anthropic-beta") != "oauth-2025-04-20" ||
			r.Header.Get("content-type") != "application/json" {
			t.Errorf("unexpected request headers %v", r.Header)
		}
		var body struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
			System    []struct{ Type, Text string }
			Messages  []struct{ Role, Content string }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != Model || body.MaxTokens != 1 || len(body.System) != 1 || body.System[0].Text != systemPreamble || len(body.Messages) != 1 {
			t.Errorf("unexpected body %+v", body)
		}
		w.Write([]byte(`{"content":[{"type":"text","text":"p"}],"stop_reason":"max_tokens"}`))
	})
	if err := client.Verify(context.Background(), testToken); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if strings.Contains(logs.String(), testToken) {
		t.Fatal("token was logged")
	}
}

func TestVerifyStatusMapping(t *testing.T) {
	for _, test := range []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrRejected},
		{http.StatusForbidden, ErrRejected},
		{http.StatusTooManyRequests, ErrUsageLimit},
		{http.StatusInternalServerError, ErrUnreachable},
		{http.StatusServiceUnavailable, ErrUnreachable},
		{http.StatusBadRequest, ErrUnreachable},
	} {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.status)
			w.Write([]byte(`{"error":{"message":"` + testToken + `"}}`))
		})
		err := client.Verify(context.Background(), testToken)
		if !errors.Is(err, test.want) {
			t.Errorf("status %d: got %v, want %v", test.status, err, test.want)
		}
		if err != nil && strings.Contains(err.Error(), testToken) {
			t.Errorf("status %d: error discloses the token", test.status)
		}
	}
}

func TestVerifyNetworkErrorIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	client := NewClient()
	client.url = server.URL
	server.Close()
	if err := client.Verify(context.Background(), testToken); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("got %v, want ErrUnreachable", err)
	}
}
