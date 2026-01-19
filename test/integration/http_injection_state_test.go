package integration

import (
	"io"
	"net/http"
	"testing"
	"time"
)

// TestHTTPInjectionState_PersistsAcrossRequests tests that imposter state persists across HTTP requests
// This corresponds to mountebank's responseResolverTest.js:
// "should allow injection imposter state across calls to resolve"
func TestHTTPInjectionState_PersistsAcrossRequests(t *testing.T) {
	defer cleanup(t)

	// Create imposter with inject response that uses counter
	// Using new interface: function(config) where config.state is the imposterState
	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7200,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(config) {
							config.state.counter = (config.state.counter || 0) + 1;
							return { statusCode: 200, body: String(config.state.counter) };
						}`,
					},
				},
			},
		},
	}

	resp, _, err := post("/imposters", imposter)
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	time.Sleep(50 * time.Millisecond)

	// Make 3 requests, each should increment counter
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 1; i <= 3; i++ {
		resp, err := client.Get("http://localhost:7200/")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		want := string(rune('0' + i)) // "1", "2", "3"
		if string(body) != want {
			t.Errorf("Request %d: body = %q, want %q (state should persist)", i, string(body), want)
		}
	}
}

// TestHTTPInjectionState_SharedBetweenPredicateAndResponse tests state sharing
// between predicate and response injection within same imposter
func TestHTTPInjectionState_SharedBetweenPredicateAndResponse(t *testing.T) {
	defer cleanup(t)

	// Create imposter where predicate sets state and response reads it
	// Using new interface for both predicate and response
	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7201,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"inject": `function(config) {
							config.state.message = 'PREDICATE_' + config.path.substring(1);
							return true;
						}`,
					},
				},
				"responses": []map[string]interface{}{
					{
						"inject": `function(config) {
							return { statusCode: 200, body: config.state.message || 'NOT_SET' };
						}`,
					},
				},
			},
		},
	}

	resp, _, err := post("/imposters", imposter)
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	time.Sleep(50 * time.Millisecond)

	// Request with path /test
	client := &http.Client{Timeout: 5 * time.Second}
	resp1, err := client.Get("http://localhost:7201/test")
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "PREDICATE_test" {
		t.Errorf("First response = %q, want %q (state should flow from predicate to response)", string(body1), "PREDICATE_test")
	}

	// Request with path /other - state should be updated
	resp2, err := client.Get("http://localhost:7201/other")
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "PREDICATE_other" {
		t.Errorf("Second response = %q, want %q", string(body2), "PREDICATE_other")
	}
}

// TestHTTPInjectionState_IsolatedBetweenImposters tests that different imposters have isolated state
func TestHTTPInjectionState_IsolatedBetweenImposters(t *testing.T) {
	defer cleanup(t)

	// Script that increments counter using new interface
	counterScript := `function(config) {
		config.state.counter = (config.state.counter || 0) + 1;
		return { statusCode: 200, body: String(config.state.counter) };
	}`

	// Create first imposter
	imposter1 := map[string]interface{}{
		"protocol": "http",
		"port":     7202,
		"stubs": []map[string]interface{}{
			{"responses": []map[string]interface{}{{"inject": counterScript}}},
		},
	}

	resp, _, err := post("/imposters", imposter1)
	if err != nil {
		t.Fatalf("failed to create imposter1: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201 for imposter1, got %d", resp.StatusCode)
	}

	// Create second imposter
	imposter2 := map[string]interface{}{
		"protocol": "http",
		"port":     7203,
		"stubs": []map[string]interface{}{
			{"responses": []map[string]interface{}{{"inject": counterScript}}},
		},
	}

	resp, _, err = post("/imposters", imposter2)
	if err != nil {
		t.Fatalf("failed to create imposter2: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201 for imposter2, got %d", resp.StatusCode)
	}

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 5 * time.Second}

	// Make 2 requests to imposter1
	for i := 1; i <= 2; i++ {
		resp, _ := client.Get("http://localhost:7202/")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		want := string(rune('0' + i))
		if string(body) != want {
			t.Errorf("Imposter1 request %d: body = %q, want %q", i, string(body), want)
		}
	}

	// Imposter2 should have its own counter starting at 1
	resp1, _ := client.Get("http://localhost:7203/")
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "1" {
		t.Errorf("Imposter2 first request: body = %q, want %q (should have isolated state)", string(body1), "1")
	}
}

// TestHTTPInjectionState_PredicateModifiesState tests that predicate can modify state
// that persists across requests (mountebank's predicates/injectTest.js:
// "should allow changing the state in the injection")
func TestHTTPInjectionState_PredicateModifiesState(t *testing.T) {
	defer cleanup(t)

	// First stub: predicate sets state, returns specific response
	// Second stub: matches always, returns state value
	// Using new interface for both
	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7204,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"path": "/set",
						},
					},
					{
						"inject": `function(config) {
							config.state.foo = 'barbar';
							return true;
						}`,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"statusCode": 200, "body": "SET"}},
				},
			},
			{
				// Default: return current state value
				"responses": []map[string]interface{}{
					{
						"inject": `function(config) {
							return { statusCode: 200, body: config.state.foo || 'NOT_SET' };
						}`,
					},
				},
			},
		},
	}

	resp, _, err := post("/imposters", imposter)
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 5 * time.Second}

	// First request to /get - should return NOT_SET
	resp1, _ := client.Get("http://localhost:7204/get")
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "NOT_SET" {
		t.Errorf("First request: body = %q, want %q", string(body1), "NOT_SET")
	}

	// Request to /set - predicate sets state
	resp2, _ := client.Get("http://localhost:7204/set")
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "SET" {
		t.Errorf("Set request: body = %q, want %q", string(body2), "SET")
	}

	// Second request to /get - should now return 'barbar' (set by predicate)
	resp3, _ := client.Get("http://localhost:7204/get")
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if string(body3) != "barbar" {
		t.Errorf("After set request: body = %q, want %q (state should be modified by predicate)", string(body3), "barbar")
	}
}

// TestHTTPInjectionState_BackwardCompatibility tests that old 3-parameter scripts still work
func TestHTTPInjectionState_BackwardCompatibility(t *testing.T) {
	defer cleanup(t)

	// Old-style script without imposterState parameter
	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7205,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state, logger) {
							return { statusCode: 200, body: 'HELLO ' + request.path };
						}`,
					},
				},
			},
		},
	}

	resp, _, err := post("/imposters", imposter)
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 5 * time.Second}
	resp1, err := client.Get("http://localhost:7205/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	body, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body) != "HELLO /test" {
		t.Errorf("body = %q, want %q (old-style script should work)", string(body), "HELLO /test")
	}
}
