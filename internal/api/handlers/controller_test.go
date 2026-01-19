package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
)

// TestDeleteImposters_ReplayableMode tests that DELETE /imposters returns replayable body by default
func TestDeleteImposters_ReplayableMode(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImpostersHandler(repo, manager, 2525)

	// Create two imposters
	imp1 := &models.Imposter{
		Port:           3001,
		Protocol:       "http",
		Name:           "imposter 1",
		RecordRequests: false,
		Stubs:          []models.Stub{},
	}
	imp2 := &models.Imposter{
		Port:           3002,
		Protocol:       "http",
		Name:           "imposter 2",
		RecordRequests: false,
		Stubs:          []models.Stub{},
	}

	repo.Add(imp1)
	repo.Add(imp2)

	// Make DELETE /imposters request (no query params, should default to replayable=true)
	req := httptest.NewRequest("DELETE", "/imposters", nil)
	w := httptest.NewRecorder()

	handler.DeleteImposters(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Imposters []map[string]interface{} `json:"imposters"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// In replayable mode, the response should NOT contain requests, numberOfRequests, or _links fields
	for i, imp := range response.Imposters {
		// Check that requests field is not present
		if _, hasRequests := imp["requests"]; hasRequests {
			t.Errorf("Imposter %d should not have 'requests' field in replayable mode, but got: %v", i, imp)
		}

		// Check that numberOfRequests field is not present
		if _, hasCount := imp["numberOfRequests"]; hasCount {
			t.Errorf("Imposter %d should not have 'numberOfRequests' field in replayable mode", i)
		}

		// Check that _links field is not present
		if _, hasLinks := imp["_links"]; hasLinks {
			t.Errorf("Imposter %d should not have '_links' field in replayable mode", i)
		}

		// These fields SHOULD be present
		expectedFields := []string{"protocol", "port", "name", "recordRequests", "stubs"}
		for _, field := range expectedFields {
			if _, ok := imp[field]; !ok {
				t.Errorf("Imposter %d missing required field: %s", i, field)
			}
		}
	}
}

// TestDeleteImposters_NonReplayableMode tests that DELETE /imposters?replayable=false includes full details
func TestDeleteImposters_NonReplayableMode(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImpostersHandler(repo, manager, 2525)

	// Create an imposter with a stub
	imp := &models.Imposter{
		Port:     3001,
		Protocol: "http",
		Name:     "test-imposter",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Body: "Hello, World!"}},
				},
			},
		},
		RecordRequests: false,
	}
	count := 0
	imp.NumberOfRequests = &count

	repo.Add(imp)

	// Make DELETE /imposters?replayable=false&removeProxies=true request
	req := httptest.NewRequest("DELETE", "/imposters?replayable=false&removeProxies=true", nil)
	w := httptest.NewRecorder()

	handler.DeleteImposters(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Imposters []map[string]interface{} `json:"imposters"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// In non-replayable mode, these fields SHOULD be present
	if len(response.Imposters) == 0 {
		t.Fatal("Expected at least one imposter in response")
	}

	imp0 := response.Imposters[0]

	// Check that requests field IS present
	if _, hasRequests := imp0["requests"]; !hasRequests {
		t.Error("Imposter should have 'requests' field in non-replayable mode")
	}

	// Check that numberOfRequests field IS present
	if _, hasCount := imp0["numberOfRequests"]; !hasCount {
		t.Error("Imposter should have 'numberOfRequests' field in non-replayable mode")
	}

	// Check that _links field IS present
	if _, hasLinks := imp0["_links"]; !hasLinks {
		t.Error("Imposter should have '_links' field in non-replayable mode")
	}
}

// TestDeleteImposter_ReplayableMode tests that DELETE /imposters/:id returns replayable body by default
func TestDeleteImposter_ReplayableMode(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImposterHandler(repo, manager)

	// Create imposter with proxy and non-proxy responses
	imp := &models.Imposter{
		Port:     3001,
		Protocol: "http",
		Name:     "test-imposter",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Proxy: &models.ProxyResponse{To: "http://www.google.com"}},
					{Is: &models.IsResponse{Body: "Hello, World!"}},
				},
			},
		},
		RecordRequests: false,
	}

	repo.Add(imp)

	// Make DELETE /imposters/3001?removeProxies=true&replayable=true request
	req := httptest.NewRequest("DELETE", "/imposters/3001?removeProxies=true&replayable=true", nil)
	req.URL.RawQuery = "_param_id=3001&removeProxies=true&replayable=true"
	w := httptest.NewRecorder()

	handler.DeleteImposter(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// In replayable mode with removeProxies=true, requests field should not be present
	if _, hasRequests := response["requests"]; hasRequests {
		t.Errorf("Response should not have 'requests' field in replayable mode, got: %v", response)
	}

	// Verify proxy response was removed
	stubs, ok := response["stubs"].([]interface{})
	if !ok {
		t.Fatal("Expected stubs to be an array")
	}

	if len(stubs) != 1 {
		t.Fatalf("Expected 1 stub (proxy removed), got %d", len(stubs))
	}

	stub0 := stubs[0].(map[string]interface{})
	responses := stub0["responses"].([]interface{})
	if len(responses) != 1 {
		t.Fatalf("Expected 1 response (proxy removed), got %d", len(responses))
	}

	// Verify it's the 'is' response
	resp0 := responses[0].(map[string]interface{})
	if _, hasIs := resp0["is"]; !hasIs {
		t.Error("Expected 'is' response, proxy should have been removed")
	}
}

// TestCreateImposter_HypermediaLinks tests that POST /imposters returns consistent hypermedia
func TestCreateImposter_HypermediaLinks(t *testing.T) {
	repo := repository.NewInMemory()
	manager := imposter.NewManager()
	handler := NewImpostersHandler(repo, manager, 2525)

	// Create imposter
	imposterJSON := `{"protocol": "http", "port": 3001}`
	req := httptest.NewRequest("POST", "/imposters", bytes.NewBufferString(imposterJSON))
	req.Host = "localhost:2525"
	w := httptest.NewRecorder()

	handler.CreateImposter(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Check Location header
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header to be set")
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify _links.self.href matches Location header
	links, ok := response["_links"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected _links field in response")
	}

	self, ok := links["self"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected _links.self in response")
	}

	href, ok := self["href"].(string)
	if !ok {
		t.Fatal("Expected _links.self.href to be a string")
	}

	if location != href {
		t.Errorf("Location header (%s) should match _links.self.href (%s)", location, href)
	}
}
