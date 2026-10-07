package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const searchesPath = "/api/v1/amazon-searches"

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
}

func (s *Server) handleAmazonRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"amazonRequests": s.service.AmazonRequests()})
}

// handleSearches routes /api/v1/amazon-searches and the paths below it.
func (s *Server) handleSearches(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == searchesPath {
		s.handleSearchCollection(w, r)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, searchesPath+"/"), "/")
	switch {
	case len(parts) == 1 && parts[0] != "":
		s.handleSearch(w, r, parts[0])
	case len(parts) == 2 && parts[0] != "" && parts[1] == "refresh":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		requestedAt, search, err := s.service.RestartAmazonSearch(r.Context(), parts[0])
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"requestedAt": requestedAt, "search": search})
	case len(parts) == 4 && parts[0] != "" && parts[1] == "items" && parts[2] != "" && parts[3] == "track":
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		item, err := s.service.TrackSearchItem(r.Context(), parts[0], parts[2])
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		writeError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "API route was not found.")
	}
}

func (s *Server) handleSearchCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		searches, err := s.service.Searches(r.Context())
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"searches": searches, "amazonRequests": s.service.AmazonRequests()})
	case http.MethodPost:
		var body struct {
			URL string `json:"url"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&body)
		if err == nil && !errors.Is(decoder.Decode(&struct{}{}), io.EOF) {
			err = errors.New("several JSON values")
		}
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large.")
			return
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}
		search, err := s.service.AddSearch(r.Context(), body.URL)
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, search)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		search, items, err := s.service.SearchDetails(r.Context(), id)
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"search": search, "amazonRequests": s.service.AmazonRequests(), "items": items})
	case http.MethodDelete:
		if err := s.service.DeleteSearch(r.Context(), id); err != nil {
			serviceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, "GET, DELETE")
	}
}
