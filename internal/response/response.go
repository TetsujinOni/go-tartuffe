package response

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard error format
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Error represents a single error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes matching mountebank
const (
	ErrCodeBadData          = "bad data"
	ErrCodeResourceConflict = "resource conflict"
	ErrCodeNoSuchResource   = "no such resource"
	ErrCodeInvalidJSON      = "invalid JSON"
	ErrCodeInvalidInjection = "invalid injection"
)

// WriteError writes an error response
func WriteError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := ErrorResponse{
		Errors: []Error{{Code: code, Message: message}},
	}

	json.NewEncoder(w).Encode(resp)
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
