package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
)

func (s *Server) handleAIReviewRequest(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
		return
	}
	requestedAt, alreadyRunning, err := s.service.RequestAIReview(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Tracked item was not found.")
		return
	}
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"requestedAt": requestedAt, "alreadyRunning": alreadyRunning})
}
