package leboncoin

import (
	"bytes"
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
	"syscall"
	"time"
	"unicode/utf8"
)

const sessionLimit = 16 << 10

var (
	errSession        = errors.New("LeBoncoin session format is invalid; renew the session")
	errSessionFile    = errors.New("LeBoncoin session files are missing, unreadable or unsafe; check private ownership and permissions")
	errSessionExpired = errors.New("LeBoncoin session is expired or revoked; renew the session")
)

type sessionCookie struct {
	Name      string     `json:"name"`
	Value     string     `json:"value"`
	Domain    string     `json:"domain"`
	Path      string     `json:"path"`
	Secure    bool       `json:"secure"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type sessionState struct {
	Version           int            `json:"version"`
	ImportFingerprint string         `json:"importFingerprint"`
	Cookie            *sessionCookie `json:"cookie"`
}

type sessionStore struct {
	path    string
	gate    chan struct{}
	current *sessionState
}

type sessionAttempt struct {
	baseline  *sessionCookie
	candidate *sessionCookie
	revoked   bool
	verified  bool
}

// exactObject rejects missing, duplicate and unknown fields, including trailing JSON.
func exactObject(data []byte, fields ...string) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) {
		return nil, errSession
	}
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errSession
	}
	result := make(map[string]json.RawMessage, len(fields))
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, errSession
		}
		name, ok := key.(string)
		if !ok {
			return nil, errSession
		}
		if _, exists := result[name]; exists {
			return nil, errSession
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return nil, errSession
		}
		result[name] = value
	}
	if _, err = d.Token(); err != nil {
		return nil, errSession
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, errSession
	}
	if len(result) != len(fields) {
		return nil, errSession
	}
	for _, field := range fields {
		if _, ok := result[field]; !ok {
			return nil, errSession
		}
	}
	return result, nil
}

func utcTimestamp(data []byte) (*time.Time, error) {
	var value string
	if json.Unmarshal(data, &value) != nil || !strings.HasSuffix(value, "Z") {
		return nil, errSession
	}
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, errSession
	}
	return &timestamp, nil
}

func decodeCookie(data []byte) (*sessionCookie, error) {
	fields, err := exactObject(data, "name", "value", "domain", "path", "secure", "expiresAt")
	if err != nil {
		return nil, err
	}
	var cookie sessionCookie
	if json.Unmarshal(data, &cookie) != nil {
		return nil, errSession
	}
	for _, field := range []string{"name", "value", "domain", "path", "secure"} {
		if bytes.Equal(bytes.TrimSpace(fields[field]), []byte("null")) {
			return nil, errSession
		}
	}
	if !bytes.Equal(bytes.TrimSpace(fields["expiresAt"]), []byte("null")) {
		cookie.ExpiresAt, err = utcTimestamp(fields["expiresAt"])
		if err != nil {
			return nil, err
		}
	}
	if !validCookie(&cookie) {
		return nil, errSession
	}
	return &cookie, nil
}

func validCookie(c *sessionCookie) bool {
	if c.Name != "datadome" || len(c.Value) == 0 || len(c.Value) > 4096 {
		return false
	}
	for i := 0; i < len(c.Value); i++ {
		b := c.Value[i]
		if b < 0x21 || b > 0x7e || strings.ContainsRune("\",;\\", rune(b)) {
			return false
		}
	}
	switch c.Domain {
	case "leboncoin.fr", ".leboncoin.fr", "www.leboncoin.fr", ".www.leboncoin.fr":
	default:
		return false
	}
	if !strings.HasPrefix(c.Path, "/") {
		return false
	}
	for _, r := range c.Path {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func allowedSessionURL(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return u.Scheme == "https" && u.User == nil && (u.Port() == "" || u.Port() == "443") && (host == "leboncoin.fr" || host == "www.leboncoin.fr")
}

func cookieApplies(c *sessionCookie, u *url.URL) bool {
	if c == nil || !allowedSessionURL(u) {
		return false
	}
	host := strings.ToLower(u.Hostname())
	domain := strings.TrimPrefix(c.Domain, ".")
	if host != domain && !(strings.HasPrefix(c.Domain, ".") && strings.HasSuffix(host, "."+domain)) {
		return false
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	return path == c.Path || (strings.HasPrefix(path, c.Path) && (strings.HasSuffix(c.Path, "/") || path[len(c.Path)] == '/'))
}

func expired(c *sessionCookie) bool {
	return c == nil || (c.ExpiresAt != nil && !c.ExpiresAt.After(time.Now()))
}

func owned(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

// Open a stable directory handle after rejecting symlink components. Reads and
// atomic sidecar replacement remain relative to this handle.
func privateSessionDirectory(path string) (*os.Root, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, "", errSessionFile
	}
	parent := filepath.Dir(absolute)
	for component := parent; ; component = filepath.Dir(component) {
		info, err := os.Lstat(component)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, "", errSessionFile
		}
		if component == filepath.Dir(component) {
			break
		}
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, "", errSessionFile
	}
	info, err := root.Stat(".")
	if err != nil || !owned(info) || info.Mode().Perm()&0077 != 0 {
		root.Close()
		return nil, "", errSessionFile
	}
	return root, filepath.Base(absolute), nil
}

func readPrivate(root *os.Root, name string) ([]byte, error) {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, errSessionFile
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !owned(info) || info.Mode().Perm()&0177 != 0 || info.Size() > sessionLimit {
		return nil, errSessionFile
	}
	data, err := io.ReadAll(io.LimitReader(f, sessionLimit+1))
	if err != nil || len(data) > sessionLimit {
		return nil, errSessionFile
	}
	return data, nil
}

func (s *sessionStore) load() error {
	root, name, err := privateSessionDirectory(s.path)
	if err != nil {
		return errSessionFile
	}
	defer root.Close()
	data, err := readPrivate(root, name)
	if err != nil {
		return errSessionFile
	}
	fields, err := exactObject(data, "version", "capturedAt", "cookie")
	if err != nil {
		return err
	}
	if string(fields["version"]) != "1" {
		return errSession
	}
	if _, err := utcTimestamp(fields["capturedAt"]); err != nil {
		return err
	}
	cookie, err := decodeCookie(fields["cookie"])
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	fingerprint := hex.EncodeToString(digest[:])
	if s.current != nil && s.current.ImportFingerprint == fingerprint {
		if expired(s.current.Cookie) {
			return errSessionExpired
		}
		return nil
	}
	state := &sessionState{Version: 1, ImportFingerprint: fingerprint, Cookie: cookie}
	saved, err := readPrivate(root, name+".state.json")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errSessionFile
	}
	if err == nil {
		fields, err := exactObject(saved, "version", "importFingerprint", "cookie")
		if err != nil {
			return err
		}
		var savedState sessionState
		if json.Unmarshal(saved, &savedState) != nil || string(fields["version"]) != "1" || len(savedState.ImportFingerprint) != 64 {
			return errSession
		}
		decoded, err := hex.DecodeString(savedState.ImportFingerprint)
		if err != nil || hex.EncodeToString(decoded) != savedState.ImportFingerprint {
			return errSession
		}
		if !bytes.Equal(bytes.TrimSpace(fields["cookie"]), []byte("null")) {
			savedState.Cookie, err = decodeCookie(fields["cookie"])
			if err != nil {
				return err
			}
		}
		if savedState.ImportFingerprint == fingerprint {
			state = &savedState
		}
	}
	s.current = state
	if expired(state.Cookie) {
		return errSessionExpired
	}
	return nil
}

func (s *sessionStore) save() error {
	root, name, err := privateSessionDirectory(s.path)
	if err != nil {
		return err
	}
	defer root.Close()
	target := name + ".state.json"
	if _, err := readPrivate(root, target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errSession
	}
	data, err := json.Marshal(s.current)
	if err != nil {
		return errSession
	}
	// Exclusive creation and atomic rename use the same stable directory handle.
	temporary := target + ".tmp-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	f, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errSession
	}
	defer root.Remove(temporary)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errSession
	}
	if _, err := readPrivate(root, target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errSession
	}
	if err = root.Rename(temporary, target); err != nil {
		return errSession
	}
	return nil
}

func (s *sessionStore) finish(a *sessionAttempt) {
	cookie := s.current.Cookie
	if a.verified {
		cookie = a.candidate
	} else if a.revoked {
		cookie = nil
	}
	before, _ := json.Marshal(s.current.Cookie)
	after, _ := json.Marshal(cookie)
	if bytes.Equal(before, after) {
		return
	}
	s.current.Cookie = cookie
	if s.save() != nil {
		log.Print("LeBoncoin session state could not be saved; the current process retains the update, but restart may require renewal")
	}
}

func (a *sessionAttempt) Cookies(u *url.URL) []*http.Cookie {
	if !cookieApplies(a.candidate, u) || expired(a.candidate) {
		return nil
	}
	return []*http.Cookie{{Name: "datadome", Value: a.candidate.Value}}
}

func sameIdentity(a, b *sessionCookie) bool {
	return a != nil && b != nil && a.Domain == b.Domain && a.Path == b.Path
}

func (a *sessionAttempt) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if !allowedSessionURL(u) {
		return
	}
	candidate := a.candidate
	replacements := 0
	deletedOriginalCandidate := false
	for _, raw := range cookies {
		if raw.Name != "datadome" || len(raw.Unparsed) > 0 || raw.Valid() != nil {
			continue
		}
		domain := strings.ToLower(raw.Domain)
		if domain == "" {
			domain = strings.ToLower(u.Hostname())
		} else {
			domain = "." + strings.TrimPrefix(domain, ".")
		}
		path := raw.Path
		if path == "" {
			path = u.Path
			if end := strings.LastIndex(path, "/"); end > 0 {
				path = path[:end]
			} else {
				path = "/"
			}
		}
		c := &sessionCookie{Name: raw.Name, Value: raw.Value, Domain: domain, Path: path, Secure: raw.Secure}
		deletion := raw.MaxAge < 0
		if raw.MaxAge > 0 {
			now := time.Now().UTC()
			if int64(raw.MaxAge) > int64((time.Duration(1<<63-1))/time.Second) {
				continue
			}
			deadline := now.Add(time.Duration(raw.MaxAge) * time.Second)
			c.ExpiresAt = &deadline
		} else if !raw.Expires.IsZero() {
			deadline := raw.Expires.UTC()
			c.ExpiresAt = &deadline
			deletion = deletion || !deadline.After(time.Now())
		}
		// Deletion commonly uses an empty value, which is never sent as a session.
		check := *c
		if deletion && check.Value == "" {
			check.Value = "deleted"
		}
		if !validCookie(&check) || !cookieApplies(&check, u) {
			continue
		}
		if deletion {
			if sameIdentity(c, a.baseline) {
				a.revoked = true
			}
			if sameIdentity(c, a.candidate) {
				deletedOriginalCandidate = true
			}
			if sameIdentity(c, candidate) {
				candidate = nil
			}
			continue
		}
		replacements++
		candidate = c
	}
	if replacements <= 1 {
		a.candidate = candidate
	} else if deletedOriginalCandidate {
		// Ambiguous replacements cannot undo an explicit revocation.
		a.candidate = nil
	}
}
