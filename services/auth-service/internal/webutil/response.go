package webutil

import (
	"encoding/json"
	"net/http"
)

// envelope mirrors the response shape mandated by
// docs/architecture/API_DESIGN.md §4: {"data": ...} on success.
type envelope struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

// errorEnvelope mirrors docs/architecture/API_DESIGN.md §6.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteJSON writes a successful {"data": ...} response.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data})
}

// WriteJSONWithMeta writes a successful {"data": ..., "meta": ...} response.
func WriteJSONWithMeta(w http.ResponseWriter, status int, data, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data, Meta: meta})
}

// WriteError writes the standard error envelope described in
// docs/architecture/API_DESIGN.md §6.
func WriteError(w http.ResponseWriter, status int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}})
}
