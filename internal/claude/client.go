// Package claude calls the Anthropic Messages API with a Claude subscription token.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	// Model is the Claude model used for verification and reviews.
	Model          = "claude-sonnet-5-5"
	apiURL         = "https://api.anthropic.com/v1/messages"
	systemPreamble = "You are Claude Code, Anthropic's official CLI for Claude."
)

var (
	ErrRejected    = errors.New("claude rejected the token")
	ErrUsageLimit  = errors.New("claude usage limit reached")
	ErrUnreachable = errors.New("claude could not be reached")
	ErrBadResponse = errors.New("claude returned an unusable response")
)

// Client sends Messages API requests. It never logs tokens, prompts or responses.
type Client struct {
	http *http.Client
	url  string
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 120 * time.Second}, url: apiURL}
}

type textBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type request struct {
	Model     string      `json:"model"`
	MaxTokens int         `json:"max_tokens"`
	System    []textBlock `json:"system"`
	Messages  []message   `json:"messages"`
}

type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

// Verify checks that Claude accepts the token with a minimal request.
func (c *Client) Verify(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err := c.send(ctx, "verify", token, request{
		Model:     Model,
		MaxTokens: 1,
		System:    []textBlock{{Type: "text", Text: systemPreamble}},
		Messages:  []message{{Role: "user", Content: "ping"}},
	})
	return err
}

// send posts body and returns the decoded 2xx response or a typed error.
func (c *Client) send(ctx context.Context, operation, token string, body request) (response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return response{}, ErrBadResponse
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return response{}, ErrUnreachable
	}
	httpRequest.Header.Set("Authorization", "Bearer "+token)
	httpRequest.Header.Set("anthropic-version", "2023-06-01")
	httpRequest.Header.Set("anthropic-beta", "oauth-2025-04-20")
	httpRequest.Header.Set("content-type", "application/json")
	httpResponse, err := c.http.Do(httpRequest)
	if err != nil {
		log.Printf("Claude %s request failed before a response", operation)
		return response{}, ErrUnreachable
	}
	defer httpResponse.Body.Close()
	log.Printf("Claude %s response status=%d", operation, httpResponse.StatusCode)
	switch status := httpResponse.StatusCode; {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return response{}, ErrRejected
	case status == http.StatusTooManyRequests:
		return response{}, ErrUsageLimit
	case status >= 500:
		return response{}, ErrUnreachable
	case status < 200 || status >= 300:
		if operation == "verify" {
			return response{}, ErrUnreachable
		}
		return response{}, ErrBadResponse
	}
	var decoded response
	if err := json.NewDecoder(io.LimitReader(httpResponse.Body, 1<<20)).Decode(&decoded); err != nil {
		return response{}, ErrBadResponse
	}
	return decoded, nil
}
