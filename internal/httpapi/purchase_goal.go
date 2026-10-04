package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxGoalBodyBytes caps add and purchase goal bodies; it is not a goal length rule.
const maxGoalBodyBytes = 1 << 20

func (s *Server) handlePurchaseGoal(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", "PUT")
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
		return
	}
	var body struct {
		PurchaseGoal *string `json:"purchaseGoal"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxGoalBodyBytes)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err == nil && !errors.Is(decoder.Decode(&struct{}{}), io.EOF) {
		err = errors.New("more than one JSON value")
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large.")
		return
	}
	if err != nil || body.PurchaseGoal == nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
		return
	}
	goal, changed, reviewStarted, err := s.service.SetPurchaseGoal(r.Context(), id, *body.PurchaseGoal)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Tracked item was not found.")
		return
	}
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchaseGoal": goal, "changed": changed, "reviewStarted": reviewStarted})
}
