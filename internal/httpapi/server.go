package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"

	"pricefollower.local/config"
	"pricefollower.local/internal/model"
	"pricefollower.local/internal/service"
	"pricefollower.local/web"
)

type Server struct {
	config  config.Config
	service *service.Service
	handler http.Handler
}

func New(cfg config.Config, items *service.Service) *Server {
	s := &Server{config: cfg, service: items}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", s.handleAPI)
	mux.HandleFunc("/", s.handleFrontend)
	s.handler = securityHeaders(mux)
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

// handleData routes the item and settings endpoints after handleAPI's access checks.
func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/api/v1/settings/leboncoin-session" {
		s.handleLeboncoinSession(w, r)
		return
	}
	if path == "/api/v1/settings/claude-token" {
		s.handleClaudeToken(w, r)
		return
	}
	if path == "/api/v1/settings" {
		s.handleSettings(w, r)
		return
	}
	if path == "/api/v1/items" && r.Method == http.MethodGet {
		items, err := s.service.List(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
		return
	}
	if path == "/api/v1/items" && r.Method == http.MethodPost {
		var body struct {
			URL          string `json:"url"`
			PurchaseGoal string `json:"purchaseGoal"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxGoalBodyBytes)
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large.")
			} else {
				writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			}
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must contain one JSON value.")
			return
		}
		if strings.TrimSpace(body.URL) == "" {
			writeError(w, http.StatusBadRequest, "INVALID_URL", "A listing URL is required.")
			return
		}
		item, err := s.service.Add(r.Context(), body.URL, body.PurchaseGoal)
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
		return
	}
	if path == "/api/v1/items/refresh" {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
			return
		}
		requestedAt, count, err := s.service.RefreshAll(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"requestedAt": requestedAt, "itemsQueued": count})
		return
	}
	const itemPrefix = "/api/v1/items/"
	const refreshSuffix = "/refresh"
	if strings.HasPrefix(path, itemPrefix) && strings.HasSuffix(path, refreshSuffix) {
		id := strings.TrimSuffix(strings.TrimPrefix(path, itemPrefix), refreshSuffix)
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
			return
		}
		requestedAt, err := s.service.RefreshItem(r.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Tracked item was not found.")
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"requestedAt": requestedAt, "itemsQueued": 1})
		return
	}
	const purchaseGoalSuffix = "/purchase-goal"
	if strings.HasPrefix(path, itemPrefix) && strings.HasSuffix(path, purchaseGoalSuffix) {
		id := strings.TrimSuffix(strings.TrimPrefix(path, itemPrefix), purchaseGoalSuffix)
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
			return
		}
		s.handlePurchaseGoal(w, r, id)
		return
	}
	const aiReviewSuffix = "/ai-review"
	if strings.HasPrefix(path, itemPrefix) && strings.HasSuffix(path, aiReviewSuffix) {
		id := strings.TrimSuffix(strings.TrimPrefix(path, itemPrefix), aiReviewSuffix)
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
			return
		}
		s.handleAIReviewRequest(w, r, id)
		return
	}
	if strings.HasPrefix(path, itemPrefix) {
		id := strings.TrimPrefix(path, itemPrefix)
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
			return
		}
		switch r.Method {
		case http.MethodGet:
			item, err := s.service.Get(r.Context(), id)
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Tracked item was not found.")
				return
			}
			if err != nil {
				serverError(w, err)
				return
			}
			review, err := s.service.AIReview(r.Context(), item)
			if err != nil {
				serverError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, model.ItemDetails{Item: item, AIReview: review})
		case http.MethodDelete:
			deleted, err := s.service.Remove(r.Context(), id)
			if err != nil {
				serverError(w, err)
				return
			}
			if !deleted {
				writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Tracked item was not found.")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Allow", "GET, DELETE")
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
		}
		return
	}
	writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
}

func (s *Server) handleLeboncoinSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		session, err := s.service.LeboncoinSession(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": session})
	case http.MethodPut:
		var body struct {
			Value    *string `json:"value"`
			Revision *int64  `json:"revision"`
		}
		const invalidRequest = "Request must include a session value (use an empty string to remove it) and the revision from Settings."
		r.Body = http.MaxBytesReader(w, r.Body, 32_768)
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			var tooLarge *http.MaxBytesError
			var wrongType *json.UnmarshalTypeError
			switch {
			case errors.As(err, &tooLarge):
				writeError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large.")
			case errors.As(err, &wrongType):
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", invalidRequest)
			default:
				writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			}
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large.")
				return
			}
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must contain one JSON value.")
			return
		}
		if body.Value == nil || body.Revision == nil || *body.Revision < 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", invalidRequest)
			return
		}
		session, err := s.service.SaveLeboncoinSession(r.Context(), *body.Value, *body.Revision)
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": session})
	default:
		w.Header().Set("Allow", "GET, PUT")
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
	}
}

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if s.config.Development || r.Method != http.MethodGet {
		writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route was not found.")
		return
	}
	staticFS, err := fs.Sub(web.Files, "static/dist")
	if err != nil {
		http.Error(w, "Frontend build is not embedded. Run scripts/build-release.sh.", http.StatusNotFound)
		return
	}
	requested := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requested == "." || requested == "" {
		requested = "index.html"
	}
	if !fs.ValidPath(requested) {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	file, err := staticFS.Open(requested)
	if err != nil {
		requested = "index.html" // Support direct navigation to client-side item pages.
		file, err = staticFS.Open(requested)
	}
	if err != nil {
		http.Error(w, "Frontend build not found. Run scripts/build-release.sh.", http.StatusNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	content, ok := file.(io.ReadSeeker)
	if !ok {
		http.Error(w, "Frontend file cannot be served", http.StatusInternalServerError)
		return
	}
	if requested == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	contentType := mime.TypeByExtension(path.Ext(requested))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, info.Name(), info.ModTime(), content)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func serviceError(w http.ResponseWriter, err error) {
	var typed *service.Error
	if errors.As(err, &typed) {
		writeError(w, typed.Status, typed.Code, typed.Message)
		return
	}
	serverError(w, err)
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("request failed: %v", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The server could not complete the request.")
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
