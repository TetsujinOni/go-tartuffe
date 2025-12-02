package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
)

// CallbackHandler handles HTTP callbacks from out-of-process plugins.
// It implements the protocol.CallbackClient interface.
type CallbackHandler struct {
	repo    repository.Repository
	baseURL string
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(repo repository.Repository, baseURL string) *CallbackHandler {
	return &CallbackHandler{
		repo:    repo,
		baseURL: baseURL,
	}
}

// GetCallbackURL returns the callback URL for an imposter port
func (h *CallbackHandler) GetCallbackURL(port int) string {
	return fmt.Sprintf("%s/imposters/%d/_requests", h.baseURL, port)
}

// RecordRequest records a request for the imposter at the given port
func (h *CallbackHandler) RecordRequest(port int, request interface{}) error {
	// Convert to Request model
	reqMap, ok := request.(map[string]interface{})
	if !ok {
		return fmt.Errorf("request must be a map")
	}

	req := models.Request{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Extract standard fields
	if rf, ok := reqMap["requestFrom"].(string); ok {
		req.RequestFrom = rf
	}
	if ip, ok := reqMap["ip"].(string); ok {
		req.IP = ip
	}
	if method, ok := reqMap["method"].(string); ok {
		req.Method = method
	}
	if path, ok := reqMap["path"].(string); ok {
		req.Path = path
	}
	if body, ok := reqMap["body"].(string); ok {
		req.Body = body
	}

	return h.repo.AddRequest(port, req)
}

// MatchStub finds a matching stub for a request and returns the response
func (h *CallbackHandler) MatchStub(port int, request map[string]interface{}) (*protocol.MatchResult, error) {
	imp, err := h.repo.Get(port)
	if err != nil {
		return nil, fmt.Errorf("imposter not found: %w", err)
	}

	// Simple stub matching for out-of-process plugins
	for i, stub := range imp.Stubs {
		if h.stubMatches(&stub, request) {
			resp := stub.NextResponse()
			return &protocol.MatchResult{
				Response:  resp,
				StubIndex: i,
				Matched:   true,
			}, nil
		}
	}

	// Return default response if no match
	return &protocol.MatchResult{
		Response:  imp.DefaultResponse,
		StubIndex: -1,
		Matched:   false,
	}, nil
}

// stubMatches checks if a stub matches the request data
func (h *CallbackHandler) stubMatches(stub *models.Stub, data map[string]interface{}) bool {
	if len(stub.Predicates) == 0 {
		return true
	}

	// Evaluate all predicates
	for _, pred := range stub.Predicates {
		if !h.evaluatePredicate(&pred, data) {
			return false
		}
	}

	return true
}

// evaluatePredicate evaluates a single predicate against request data
func (h *CallbackHandler) evaluatePredicate(pred *models.Predicate, data map[string]interface{}) bool {
	// Handle logical operators
	if pred.And != nil {
		for _, p := range pred.And {
			if !h.evaluatePredicate(&p, data) {
				return false
			}
		}
		return true
	}

	if pred.Or != nil {
		for _, p := range pred.Or {
			if h.evaluatePredicate(&p, data) {
				return true
			}
		}
		return false
	}

	if pred.Not != nil {
		return !h.evaluatePredicate(pred.Not, data)
	}

	// Handle comparison operators
	if pred.Equals != nil {
		return h.evaluateEquals(pred.Equals, data, pred.CaseSensitive)
	}

	if pred.Contains != nil {
		return h.evaluateContains(pred.Contains, data, pred.CaseSensitive)
	}

	if pred.StartsWith != nil {
		return h.evaluateStartsWith(pred.StartsWith, data, pred.CaseSensitive)
	}

	if pred.EndsWith != nil {
		return h.evaluateEndsWith(pred.EndsWith, data, pred.CaseSensitive)
	}

	// Default to true if no predicate type matched
	return true
}

// evaluateEquals checks field equality
func (h *CallbackHandler) evaluateEquals(value interface{}, data map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := data[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if actualStr != expectedStr {
				return false
			}
		} else {
			if !strings.EqualFold(actualStr, expectedStr) {
				return false
			}
		}
	}

	return true
}

// evaluateContains checks if field contains value
func (h *CallbackHandler) evaluateContains(value interface{}, data map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := data[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if !strings.Contains(actualStr, expectedStr) {
				return false
			}
		} else {
			if !strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr)) {
				return false
			}
		}
	}

	return true
}

// evaluateStartsWith checks if field starts with value
func (h *CallbackHandler) evaluateStartsWith(value interface{}, data map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := data[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if !strings.HasPrefix(actualStr, expectedStr) {
				return false
			}
		} else {
			if !strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(expectedStr)) {
				return false
			}
		}
	}

	return true
}

// evaluateEndsWith checks if field ends with value
func (h *CallbackHandler) evaluateEndsWith(value interface{}, data map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := data[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if !strings.HasSuffix(actualStr, expectedStr) {
				return false
			}
		} else {
			if !strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(expectedStr)) {
				return false
			}
		}
	}

	return true
}

// HandleCallback handles POST /imposters/:port/_requests
// This is the mountebank-compatible callback endpoint for out-of-process plugins.
func (h *CallbackHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Extract port from path: /imposters/{port}/_requests
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	port, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid port", http.StatusBadRequest)
		return
	}

	// Parse request body
	var callbackReq protocol.CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&callbackReq); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Get imposter
	imp, err := h.repo.Get(port)
	if err != nil {
		http.Error(w, "imposter not found", http.StatusNotFound)
		return
	}

	// Record request if configured
	if imp.RecordRequests {
		h.RecordRequest(port, callbackReq.Request)
	}

	// Match against stubs
	result, err := h.MatchStub(port, callbackReq.Request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response
	callbackResp := protocol.CallbackResponse{
		StubIndex: result.StubIndex,
		Matched:   result.Matched,
	}

	// Extract response details
	if result.Response != nil {
		// Check response type
		if result.Response.Is != nil {
			callbackResp.Response = result.Response.Is
		}
		if result.Response.Proxy != nil {
			callbackResp.Proxy = result.Response.Proxy
		}
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(callbackResp)
}
