package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"pricefollower.local/internal/service"
)

func (s *Server) handleClaudeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
		return
	}
	token, err := s.service.ClaudeToken(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"claudeToken": token})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", "PUT")
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not allowed for the route.")
		return
	}
	var body struct {
		LeboncoinSession *struct {
			Value    *string `json:"value"`
			Revision *int64  `json:"revision"`
		} `json:"leboncoinSession"`
		ClaudeToken *struct {
			Value *string `json:"value"`
		} `json:"claudeToken"`
	}
	const invalidRequest = "Request must include the LeBoncoin session and/or the Claude token to save."
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
	var input service.SettingsInput
	if session := body.LeboncoinSession; session != nil {
		if session.Value == nil || session.Revision == nil || *session.Revision < 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", invalidRequest)
			return
		}
		input.LeboncoinSession = &service.SessionInput{Value: *session.Value, Revision: *session.Revision}
	}
	if token := body.ClaudeToken; token != nil {
		if token.Value == nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", invalidRequest)
			return
		}
		input.ClaudeToken = token.Value
	}
	if input.LeboncoinSession == nil && input.ClaudeToken == nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", invalidRequest)
		return
	}
	session, token, err := s.service.SaveSettings(r.Context(), input)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": session, "claudeToken": token})
}
