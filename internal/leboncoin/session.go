package leboncoin

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxSessionInputLength = 8192
	maxSessionValueLength = 4096
)

// Session is the saved datadome cookie sent with a LeBoncoin attempt. It is
// always sent with domain .leboncoin.fr and path /.
type Session struct {
	Value     string
	ExpiresAt *time.Time // nil: unknown expiry (pasted value or session cookie)
}

// SessionOutcome reports what a session-assisted attempt learned about the session.
type SessionOutcome struct {
	Attempt   string // "accepted", "rejected" or "failed"
	Renewed   bool
	Value     string     // renewed value, when Renewed
	ExpiresAt *time.Time // renewed expiry, when Renewed
	Revoked   bool
}

// SessionInputError describes unusable pasted session input; Message is safe to display.
type SessionInputError struct {
	Message string
}

func (e *SessionInputError) Error() string { return e.Message }

// ParseSessionInput extracts the datadome value from a raw value or a cookie
// string. An empty result without error means the session must be cleared.
func ParseSessionInput(raw string) (string, error) {
	if len(raw) > maxSessionInputLength {
		return "", &SessionInputError{"The pasted text is too long (maximum 8192 characters)."}
	}
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", nil
	}
	for _, prefix := range []string{"cookie:", "set-cookie:"} {
		if len(text) >= len(prefix) && strings.EqualFold(text[:len(prefix)], prefix) {
			text = strings.TrimSpace(text[len(prefix):])
			if text == "" {
				return "", &SessionInputError{"No datadome cookie was found in the pasted text."}
			}
			break
		}
	}
	value := text
	if strings.ContainsAny(text, "=;") {
		extracted, err := datadomeFromCookieString(text)
		if err != nil {
			return "", err
		}
		value = extracted
	}
	if len(value) > maxSessionValueLength {
		return "", &SessionInputError{"The datadome value is too long (maximum 4096 characters)."}
	}
	if !validCookieValue(value) {
		return "", &SessionInputError{"The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value."}
	}
	return value, nil
}

func datadomeFromCookieString(text string) (string, error) {
	var values []string
	for _, part := range strings.Split(text, ";") {
		name, value, isPair := strings.Cut(strings.TrimSpace(part), "=")
		if !isPair || strings.TrimSpace(name) != "datadome" {
			continue // other cookies and attributes such as Secure are ignored
		}
		values = append(values, strings.TrimSpace(value))
	}
	if len(values) == 0 {
		return "", &SessionInputError{"No datadome cookie was found in the pasted text."}
	}
	for _, value := range values {
		if value == "" {
			return "", &SessionInputError{"The datadome cookie value is empty."}
		}
		if value != values[0] {
			return "", &SessionInputError{"The pasted text contains different datadome values; paste only one."}
		}
	}
	return values[0], nil
}

func validCookieValue(value string) bool {
	if len(value) == 0 || len(value) > maxSessionValueLength {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b < 0x21 || b > 0x7e || strings.ContainsRune("\",;\\", rune(b)) {
			return false
		}
	}
	return true
}

func allowedSessionURL(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return u.Scheme == "https" && u.User == nil && (u.Port() == "" || u.Port() == "443") && (host == "leboncoin.fr" || host == "www.leboncoin.fr")
}

func expired(session *Session) bool {
	return session.ExpiresAt != nil && !session.ExpiresAt.After(time.Now())
}

// sessionAttempt is the per-attempt cookie jar. It only sends and accepts the
// datadome cookie with domain .leboncoin.fr and path / on allowed HTTPS origins.
type sessionAttempt struct {
	candidate *Session // value sent on the next hop; nil after a deletion
	replaced  bool     // candidate came from a response
	revoked   bool
	rejected  bool
	verified  bool
}

func (a *sessionAttempt) Cookies(u *url.URL) []*http.Cookie {
	if a.candidate == nil || !allowedSessionURL(u) || expired(a.candidate) {
		return nil
	}
	return []*http.Cookie{{Name: "datadome", Value: a.candidate.Value}}
}

func (a *sessionAttempt) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if !allowedSessionURL(u) {
		return
	}
	var replacement *Session
	firstValue := ""
	ambiguous := false
	for _, raw := range cookies {
		if raw.Name != "datadome" || hasMalformedMetadata(raw) || raw.Valid() != nil || !sessionCookieScope(raw) {
			continue
		}
		expiresAt, deletion, ok := cookieExpiry(raw)
		if !ok {
			continue
		}
		if deletion {
			a.revoked = true
			a.candidate = nil
			replacement = nil
			continue
		}
		if !validCookieValue(raw.Value) {
			continue
		}
		if firstValue == "" {
			firstValue = raw.Value
		} else if raw.Value != firstValue {
			ambiguous = true
		}
		replacement = &Session{Value: raw.Value, ExpiresAt: expiresAt}
	}
	if replacement != nil && !ambiguous {
		a.candidate = replacement
		a.replaced = true
	}
}

// hasMalformedMetadata reports a known cookie attribute that the parser could
// not interpret. Unknown attribute names (e.g. Priority) are ignored.
func hasMalformedMetadata(cookie *http.Cookie) bool {
	for _, attribute := range cookie.Unparsed {
		name, _, _ := strings.Cut(attribute, "=")
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "expires", "max-age", "domain", "path", "samesite", "secure", "httponly", "partitioned":
			return true
		}
	}
	return false
}

// sessionCookieScope accepts only the cookie identity used by the saved session.
func sessionCookieScope(cookie *http.Cookie) bool {
	return strings.TrimPrefix(strings.ToLower(cookie.Domain), ".") == "leboncoin.fr" && cookie.Path == "/"
}

// cookieExpiry applies Max-Age before Expires and rejects durations that overflow.
func cookieExpiry(cookie *http.Cookie) (expiresAt *time.Time, deletion bool, ok bool) {
	now := time.Now().UTC()
	switch {
	case cookie.MaxAge < 0:
		return nil, true, true
	case cookie.MaxAge > 0:
		if int64(cookie.MaxAge) > int64(time.Duration(1<<63-1)/time.Second) {
			return nil, false, false
		}
		deadline := now.Add(time.Duration(cookie.MaxAge) * time.Second)
		return &deadline, false, true
	case !cookie.Expires.IsZero():
		deadline := cookie.Expires.UTC()
		return &deadline, !deadline.After(now), true
	}
	return nil, false, true
}

func (a *sessionAttempt) outcome() SessionOutcome {
	outcome := SessionOutcome{Attempt: "failed"}
	if a.verified {
		outcome.Attempt = "accepted"
	} else if a.rejected {
		outcome.Attempt = "rejected"
	}
	if a.verified && a.candidate != nil {
		if a.replaced {
			outcome.Renewed = true
			outcome.Value = a.candidate.Value
			outcome.ExpiresAt = a.candidate.ExpiresAt
		}
	} else if a.revoked {
		outcome.Revoked = true
	}
	return outcome
}
