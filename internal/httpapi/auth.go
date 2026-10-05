package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"pricefollower.local/internal/service"
)

const sessionCookie = "pricefollower_session"

type userJSON struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type adminUserJSON struct {
	Username    string     `json:"username"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
}

// handleAPI applies the access rules before routing: auth endpoints are always open,
// admin endpoints need the admin, and other endpoints need a regular user in protected mode.
func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/auth/") {
		s.handleAuth(w, r)
		return
	}
	user, loggedIn, err := s.service.SessionUser(r.Context(), sessionToken(r))
	if err != nil {
		serverError(w, err)
		return
	}
	isAdmin := loggedIn && user.Role == "admin"
	if strings.HasPrefix(path, "/api/v1/admin/") {
		switch {
		case !loggedIn:
			writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Log in to continue.")
		case !isAdmin:
			writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this page.")
		default:
			s.handleAdmin(w, r)
		}
		return
	}
	if isAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this page.")
		return
	}
	if loggedIn {
		r = r.WithContext(service.WithPrincipal(r.Context(), service.Principal{UserID: user.ID, Role: user.Role}))
	} else {
		protected, err := s.service.ProtectedMode(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		if protected {
			writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Log in to continue.")
			return
		}
	}
	s.handleData(w, r)
}

func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: value, Path: "/", MaxAge: maxAge,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil})
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/auth/session":
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		protected, err := s.service.ProtectedMode(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		mode := "open"
		if protected {
			mode = "protected"
		}
		var current *userJSON
		user, loggedIn, err := s.service.SessionUser(r.Context(), sessionToken(r))
		if err != nil {
			serverError(w, err)
			return
		}
		if loggedIn {
			current = &userJSON{Username: user.Username, Role: user.Role}
		}
		writeJSON(w, http.StatusOK, map[string]any{"mode": mode, "user": current})
	case "/api/v1/auth/login":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var body struct{ Username, Password string }
		if !decodeCredentials(w, r, &body) {
			return
		}
		token, user, err := s.service.Login(r.Context(), body.Username, body.Password)
		if err != nil {
			serviceError(w, err)
			return
		}
		setSessionCookie(w, r, token, int(service.SessionDuration.Seconds()))
		writeJSON(w, http.StatusOK, map[string]any{"user": userJSON{Username: user.Username, Role: user.Role}})
	case "/api/v1/auth/logout":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		if err := s.service.Logout(r.Context(), sessionToken(r)); err != nil {
			serverError(w, err)
			return
		}
		setSessionCookie(w, r, "", -1)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
	}
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/admin/users" {
		writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
		return
	}
	switch r.Method {
	case http.MethodGet:
		users, err := s.service.ListUsers(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		list := make([]adminUserJSON, 0, len(users))
		for _, user := range users {
			list = append(list, adminUserJSON{Username: user.Username, LastLoginAt: user.LastLoginAt})
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": list})
	case http.MethodPost:
		var body struct{ Username, Password string }
		if !decodeCredentials(w, r, &body) {
			return
		}
		user, err := s.service.CreateUser(r.Context(), body.Username, body.Password)
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"user": adminUserJSON{Username: user.Username}})
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
	}
}

func allowMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
	return false
}

// decodeCredentials reads one small JSON object with string username and password fields.
func decodeCredentials(w http.ResponseWriter, r *http.Request, body any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(body)
	if err == nil && !errors.Is(decoder.Decode(&struct{}{}), io.EOF) {
		err = errors.New("more than one JSON value")
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
		return false
	}
	return true
}
