package integration

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// TestHTTPProxy_ProxyAlwaysBasic tests proxyAlways mode with basic predicate generators
func TestHTTPProxy_ProxyAlwaysBasic(t *testing.T) {
	defer cleanup(t)

	// Create origin server with stateful responses
	originPort := 8100
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin server",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							state.count = state.count || 0;
							state.count += 1;
							return { body: state.count + ". " + request.path };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with proxyAlways mode
	proxyPort := 8101
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyAlways",
							"predicateGenerators": []map[string]interface{}{
								{"matches": map[string]interface{}{"path": true}},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make requests through proxy
	// First: GET /first (response: "1. /first")
	resp1, err := http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "1. /first" {
		t.Errorf("expected '1. /first', got %q", string(body1))
	}

	// Second: GET /second (response: "2. /second")
	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d/second", proxyPort))
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "2. /second" {
		t.Errorf("expected '2. /second', got %q", string(body2))
	}

	// Third: GET /first again (response: "3. /first")
	resp3, err := http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if string(body3) != "3. /first" {
		t.Errorf("expected '3. /first', got %q", string(body3))
	}

	// Delete proxy and check recorded stubs
	delResp, delBody, err := del(fmt.Sprintf("/imposters/%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to delete proxy: %v", err)
	}
	if delResp.StatusCode != 200 {
		t.Fatalf("expected status 200 for delete, got %d", delResp.StatusCode)
	}

	// Verify stubs were recorded
	stubs, ok := delBody["stubs"].([]interface{})
	if !ok {
		t.Fatal("stubs field is not an array")
	}

	// Should have 3 stubs: original proxy + 2 recorded stubs (one for /first with 2 responses, one for /second)
	if len(stubs) != 3 {
		t.Errorf("expected 3 stubs, got %d", len(stubs))
	}

	// Verify recorded stubs have correct responses
	// Skip first stub (original proxy stub)
	if len(stubs) < 3 {
		t.Fatal("not enough stubs recorded")
	}

	// Check first recorded stub (for /first path) - should have 2 responses
	stub1 := stubs[1].(map[string]interface{})
	responses1 := stub1["responses"].([]interface{})
	if len(responses1) != 2 {
		t.Errorf("expected 2 responses for /first stub, got %d", len(responses1))
	}

	// Verify first response for /first
	resp1Map := responses1[0].(map[string]interface{})
	is1 := resp1Map["is"].(map[string]interface{})
	if is1["body"] != "1. /first" {
		t.Errorf("expected first response '1. /first', got %q", is1["body"])
	}

	// Verify second response for /first
	resp1Map2 := responses1[1].(map[string]interface{})
	is1_2 := resp1Map2["is"].(map[string]interface{})
	if is1_2["body"] != "3. /first" {
		t.Errorf("expected second response '3. /first', got %q", is1_2["body"])
	}

	// Check second recorded stub (for /second path) - should have 1 response
	stub2 := stubs[2].(map[string]interface{})
	responses2 := stub2["responses"].([]interface{})
	if len(responses2) != 1 {
		t.Errorf("expected 1 response for /second stub, got %d", len(responses2))
	}

	// Verify response for /second
	resp2Map := responses2[0].(map[string]interface{})
	is2 := resp2Map["is"].(map[string]interface{})
	if is2["body"] != "2. /second" {
		t.Errorf("expected response '2. /second', got %q", is2["body"])
	}
}

// TestHTTPProxy_ProxyAlwaysComplexPredicates tests proxyAlways with complex predicate generators
func TestHTTPProxy_ProxyAlwaysComplexPredicates(t *testing.T) {
	defer cleanup(t)

	// Create origin server with stateful responses
	originPort := 8102
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin server",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							state.count = state.count || 0;
							state.count += 1;
							return { body: state.count + ". " + request.path };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with proxyAlways mode and complex predicates (path + method)
	proxyPort := 8103
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyAlways",
							"predicateGenerators": []map[string]interface{}{
								{
									"matches": map[string]interface{}{
										"path":   true,
										"method": true,
									},
									"caseSensitive": false,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make requests through proxy
	resp1, err := http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "1. /first" {
		t.Errorf("expected '1. /first', got %q", string(body1))
	}

	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d/second", proxyPort))
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "2. /second" {
		t.Errorf("expected '2. /second', got %q", string(body2))
	}

	resp3, err := http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if string(body3) != "3. /first" {
		t.Errorf("expected '3. /first', got %q", string(body3))
	}

	// Delete proxy and verify stubs
	delResp, delBody, err := del(fmt.Sprintf("/imposters/%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to delete proxy: %v", err)
	}
	if delResp.StatusCode != 200 {
		t.Fatalf("expected status 200 for delete, got %d", delResp.StatusCode)
	}

	// Verify stubs were recorded with correct structure
	stubs, ok := delBody["stubs"].([]interface{})
	if !ok {
		t.Fatal("stubs field is not an array")
	}

	// Should have 3 stubs
	if len(stubs) != 3 {
		t.Errorf("expected 3 stubs, got %d", len(stubs))
	}

	// Verify predicates include both path and method (case-insensitive)
	if len(stubs) > 1 {
		stub1 := stubs[1].(map[string]interface{})
		if preds, ok := stub1["predicates"].([]interface{}); ok && len(preds) > 0 {
			pred := preds[0].(map[string]interface{})
			// Should have matches predicate with path and method
			if matches, ok := pred["matches"].(map[string]interface{}); ok {
				if _, hasPath := matches["path"]; !hasPath {
					t.Error("predicate should include path")
				}
				if _, hasMethod := matches["method"]; !hasMethod {
					t.Error("predicate should include method")
				}
			}
		}
	}
}

// TestHTTPProxy_ReturnProxiedStatus tests that proxy returns correct status codes from origin
func TestHTTPProxy_ReturnProxiedStatus(t *testing.T) {
	defer cleanup(t)

	// Create origin server with non-200 status code
	originPort := 8104
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 400,
							"body":       "Bad Request from origin",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy
	proxyPort := 8105
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("http://localhost:%d", originPort),
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request through proxy
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Verify status code from origin is passed through
	if resp.StatusCode != 400 {
		t.Errorf("expected status code 400 from proxy, got %d", resp.StatusCode)
	}

	if string(body) != "Bad Request from origin" {
		t.Errorf("expected 'Bad Request from origin', got %q", string(body))
	}
}

// TestHTTPProxy_PersistBehaviors tests that behaviors on proxy responses are applied and persisted
func TestHTTPProxy_PersistBehaviors(t *testing.T) {
	defer cleanup(t)

	// Create origin server
	originPort := 8106
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "origin server",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with behaviors
	proxyPort := 8107
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("http://localhost:%d", originPort),
						},
						"_behaviors": map[string]interface{}{
							"decorate": `function(request, response) { response.headers['X-Test'] = 'decorated'; response.body += ' DECORATED'; }`,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Test origin server first
	originTest, _ := http.Get(fmt.Sprintf("http://localhost:%d/test", originPort))
	originBody, _ := io.ReadAll(originTest.Body)
	originTest.Body.Close()
	t.Logf("Origin response: status=%d, body=%q", originTest.StatusCode, string(originBody))

	// Make request through proxy - use same path as origin test
	resp1, err := http.Get(fmt.Sprintf("http://localhost:%d/test", proxyPort))
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	t.Logf("First response: status=%d, body=%q, X-Test=%q", resp1.StatusCode, string(body1), resp1.Header.Get("X-Test"))

	// Verify decorate behavior was applied
	if resp1.Header.Get("X-Test") != "decorated" {
		t.Errorf("expected X-Test header 'decorated', got %q", resp1.Header.Get("X-Test"))
	}

	// Verify body was decorated
	if string(body1) != "origin server DECORATED" {
		t.Errorf("expected 'origin server DECORATED', got %q", string(body1))
	}

	// Make second request - should also apply behaviors
	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d/test", proxyPort))
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	// Verify behaviors still applied
	if resp2.Header.Get("X-Test") != "decorated" {
		t.Errorf("expected X-Test header 'decorated' on second request, got %q", resp2.Header.Get("X-Test"))
	}

	if string(body2) != "origin server DECORATED" {
		t.Errorf("expected 'origin server DECORATED', got %q", string(body2))
	}
}

// TestHTTPProxy_AddWaitBehavior tests that addWaitBehavior captures response time
func TestHTTPProxy_AddWaitBehavior(t *testing.T) {
	defer cleanup(t)

	// Create origin server with delay
	originPort := 8108
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "origin server",
						},
						"_behaviors": map[string]interface{}{
							"wait": 100, // 100ms delay
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with addWaitBehavior
	proxyPort := 8109
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":              fmt.Sprintf("http://localhost:%d", originPort),
							"addWaitBehavior": true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if string(body) != "origin server" {
		t.Errorf("expected 'origin server', got %q", string(body))
	}

	// Get imposter to check recorded stub
	getResp, getBody, err := get(fmt.Sprintf("/imposters/%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	// Verify stub has wait behavior with captured response time
	stubs, ok := getBody["stubs"].([]interface{})
	if !ok || len(stubs) < 2 {
		t.Fatalf("expected at least 2 stubs, got %d", len(stubs))
	}

	// First recorded stub should have wait behavior (at index 0)
	recordedStub := stubs[0].(map[string]interface{})
	responses := recordedStub["responses"].([]interface{})
	if len(responses) < 1 {
		t.Fatal("expected at least 1 response")
	}

	response := responses[0].(map[string]interface{})
	// Check for behaviors array (mountebank uses "behaviors" key in recorded stubs)
	if behaviors, ok := response["behaviors"].([]interface{}); ok && len(behaviors) > 0 {
		behavior := behaviors[0].(map[string]interface{})
		if wait, ok := behavior["wait"].(float64); ok {
			// Should be at least 90ms (accounting for some variation)
			if wait < 90 {
				t.Errorf("expected wait >= 90ms, got %v", wait)
			}
		} else {
			t.Error("expected wait behavior field")
		}
	} else {
		t.Error("expected behaviors array with wait behavior")
	}
}

// TestHTTPProxy_AddDecorateBehavior tests that addDecorateBehavior adds decorator to recorded stub
func TestHTTPProxy_AddDecorateBehavior(t *testing.T) {
	defer cleanup(t)

	// Create origin server
	originPort := 8110
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "origin server",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with addDecorateBehavior
	proxyPort := 8111
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":                  fmt.Sprintf("http://localhost:%d", originPort),
							"addDecorateBehavior": "function(request, response) { response.body += ' decorated'; }",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// First request should proxy without decoration
	resp1, err := http.Get(fmt.Sprintf("http://localhost:%d/test", proxyPort))
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "origin server" {
		t.Errorf("expected 'origin server' on first request, got %q", string(body1))
	}

	// Second request should use recorded stub WITH decoration
	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d/test", proxyPort))
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "origin server decorated" {
		t.Errorf("expected 'origin server decorated' on second request, got %q", string(body2))
	}
}

// Phase 2: Predicate Generators

// TestHTTPProxy_PredicateGeneratorEntireObject tests matching entire object graphs (query: true)
func TestHTTPProxy_PredicateGeneratorEntireObject(t *testing.T) {
	defer cleanup(t)

	// Create origin server that returns request counter and query
	originPort := 8112
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							state.count = state.count || 0;
							state.count += 1;
							return { body: state.count + '. ' + JSON.stringify(request.query) };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with predicateGenerators matching entire query object
	proxyPort := 8113
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyOnce",
							"predicateGenerators": []map[string]interface{}{
								{
									"matches": map[string]interface{}{
										"query": true,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// First request: ?first=1&second=2
	resp1, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=1&second=2", proxyPort))
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	// JSON key order is not guaranteed
	body1Str := string(body1)
	if body1Str != `1. {"first":"1","second":"2"}` && body1Str != `1. {"second":"2","first":"1"}` {
		t.Errorf("first request: expected '1. {\"first\":\"1\",\"second\":\"2\"}' (any key order), got %q", body1Str)
	}

	// Second request: ?first=1 (different query)
	resp2, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=1", proxyPort))
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if string(body2) != `2. {"first":"1"}` {
		t.Errorf("second request: expected '2. {\"first\":\"1\"}', got %q", string(body2))
	}

	// Third request: ?first=2&second=2 (different query)
	resp3, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=2&second=2", proxyPort))
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	// JSON key order is not guaranteed, so check both possibilities
	body3Str := string(body3)
	if body3Str != `3. {"first":"2","second":"2"}` && body3Str != `3. {"second":"2","first":"2"}` {
		t.Errorf("third request: expected '3. {\"first\":\"2\",\"second\":\"2\"}' (any key order), got %q", body3Str)
	}

	// Fourth request: ?first=1&second=2 (matches first request - should use recorded stub)
	resp4, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=1&second=2", proxyPort))
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	// JSON key order is not guaranteed
	body4Str := string(body4)
	if body4Str != `1. {"first":"1","second":"2"}` && body4Str != `1. {"second":"2","first":"1"}` {
		t.Errorf("fourth request: expected '1. {\"first\":\"1\",\"second\":\"2\"}' (from stub, any key order), got %q", body4Str)
	}
}

// TestHTTPProxy_PredicateGeneratorSubObject tests matching sub-objects (query: { first: true })
func TestHTTPProxy_PredicateGeneratorSubObject(t *testing.T) {
	defer cleanup(t)

	// Create origin server
	originPort := 8114
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							state.count = state.count || 0;
							state.count += 1;
							return { body: state.count + '. ' + JSON.stringify(request.query) };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy matching only 'first' field in query
	proxyPort := 8115
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyOnce",
							"predicateGenerators": []map[string]interface{}{
								{
									"matches": map[string]interface{}{
										"query": map[string]interface{}{
											"first": true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// First: ?first=1&second=2
	resp1, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=1&second=2", proxyPort))
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	body1Str := string(body1)
	if body1Str != `1. {"first":"1","second":"2"}` && body1Str != `1. {"second":"2","first":"1"}` {
		t.Errorf("first request: expected '1. {\"first\":\"1\",\"second\":\"2\"}' (any key order), got %q", body1Str)
	}

	// Second: ?first=2&second=2 (first changed - new stub)
	resp2, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=2&second=2", proxyPort))
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	body2Str := string(body2)
	if body2Str != `2. {"first":"2","second":"2"}` && body2Str != `2. {"second":"2","first":"2"}` {
		t.Errorf("second request: expected '2. {\"first\":\"2\",\"second\":\"2\"}' (any key order), got %q", body2Str)
	}

	// Third: ?first=3&second=2 (first changed again - new stub)
	resp3, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=3&second=2", proxyPort))
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	body3Str := string(body3)
	if body3Str != `3. {"first":"3","second":"2"}` && body3Str != `3. {"second":"2","first":"3"}` {
		t.Errorf("third request: expected '3. {\"first\":\"3\",\"second\":\"2\"}' (any key order), got %q", body3Str)
	}

	// Fourth: ?first=1&second=2&third=3 (first=1 matches first request, other fields ignored)
	resp4, _ := http.Get(fmt.Sprintf("http://localhost:%d/?first=1&second=2&third=3", proxyPort))
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	body4Str := string(body4)
	if body4Str != `1. {"first":"1","second":"2"}` && body4Str != `1. {"second":"2","first":"1"}` {
		t.Errorf("fourth request: expected '1. {\"first\":\"1\",\"second\":\"2\"}' (from stub, any key order), got %q", body4Str)
	}
}

// TestHTTPProxy_PredicateGeneratorMultipleFields tests matching multiple fields
func TestHTTPProxy_PredicateGeneratorMultipleFields(t *testing.T) {
	defer cleanup(t)

	// Create origin server
	originPort := 8116
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							state.count = state.count || 0;
							state.count += 1;
							return { body: state.count + '. ' + request.method + ' ' + request.path };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy matching method and path
	proxyPort := 8117
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyOnce",
							"predicateGenerators": []map[string]interface{}{
								{
									"matches": map[string]interface{}{
										"method": true,
										"path":   true,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// GET /first
	resp1, _ := http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if string(body1) != "1. GET /first" {
		t.Errorf("first request: expected '1. GET /first', got %q", string(body1))
	}

	// DELETE /first (different method - new stub)
	req2, _ := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:%d/first", proxyPort), nil)
	resp2, _ := http.DefaultClient.Do(req2)
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if string(body2) != "2. DELETE /first" {
		t.Errorf("second request: expected '2. DELETE /first', got %q", string(body2))
	}

	// GET /second (different path - new stub)
	resp3, _ := http.Get(fmt.Sprintf("http://localhost:%d/second", proxyPort))
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if string(body3) != "3. GET /second" {
		t.Errorf("third request: expected '3. GET /second', got %q", string(body3))
	}

	// GET /first again (matches first request - uses stub)
	resp4, _ := http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	if string(body4) != "1. GET /first" {
		t.Errorf("fourth request: expected '1. GET /first' (from stub), got %q", string(body4))
	}

	// DELETE /first again (matches second request - uses stub)
	req5, _ := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:%d/first", proxyPort), nil)
	resp5, _ := http.DefaultClient.Do(req5)
	body5, _ := io.ReadAll(resp5.Body)
	resp5.Body.Close()
	if string(body5) != "2. DELETE /first" {
		t.Errorf("fifth request: expected '2. DELETE /first' (from stub), got %q", string(body5))
	}

	// GET /second again (matches third request - uses stub)
	resp6, _ := http.Get(fmt.Sprintf("http://localhost:%d/second", proxyPort))
	body6, _ := io.ReadAll(resp6.Body)
	resp6.Body.Close()
	if string(body6) != "3. GET /second" {
		t.Errorf("sixth request: expected '3. GET /second' (from stub), got %q", string(body6))
	}
}

// TestHTTPProxy_PredicateGeneratorCaseSensitive tests case-sensitive predicate matching
func TestHTTPProxy_PredicateGeneratorCaseSensitive(t *testing.T) {
	defer cleanup(t)

	// Create origin server
	originPort := 8118
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							state.count = state.count || 0;
							state.count += 1;
							return { body: state.count + '. ' + request.path };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with case-insensitive matching
	proxyPort := 8119
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyOnce",
							"predicateGenerators": []map[string]interface{}{
								{
									"matches": map[string]interface{}{
										"path": true,
									},
									"caseSensitive": false,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// GET /test
	resp1, _ := http.Get(fmt.Sprintf("http://localhost:%d/test", proxyPort))
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if string(body1) != "1. /test" {
		t.Errorf("first request: expected '1. /test', got %q", string(body1))
	}

	// GET /TEST (case-insensitive match - should use stub)
	resp2, _ := http.Get(fmt.Sprintf("http://localhost:%d/TEST", proxyPort))
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	// With case-insensitive matching, /TEST should match /test stub
	if string(body2) != "1. /test" {
		t.Errorf("second request: expected '1. /test' (case-insensitive match), got %q", string(body2))
	}
}

// Phase 3: Binary & Headers

// TestHTTPProxy_BinaryMIMETypes tests that binary MIME types are handled correctly through proxy
func TestHTTPProxy_BinaryMIMETypes(t *testing.T) {
	defer cleanup(t)

	binaryMIMETypes := []string{
		"application/octet-stream",
		"audio/mpeg",
		"audio/mp4",
		"image/gif",
		"image/jpeg",
		"video/avi",
		"video/mpeg",
	}

	for _, mimeType := range binaryMIMETypes {
		t.Run(mimeType, func(t *testing.T) {
			// Create origin server with binary response
			originPort := 8120
			buffer := []byte{0, 1, 2, 3}
			originResp, _, err := post("/imposters", map[string]interface{}{
				"protocol": "http",
				"port":     originPort,
				"name":     "origin",
				"stubs": []map[string]interface{}{
					{
						"responses": []map[string]interface{}{
							{
								"is": map[string]interface{}{
									"body": base64.StdEncoding.EncodeToString(buffer),
									"headers": map[string]interface{}{
										"Content-Type": mimeType,
									},
									"_mode": "binary",
								},
							},
						},
					},
				},
			})
			if err != nil {
				t.Fatalf("failed to create origin imposter: %v", err)
			}
			if originResp.StatusCode != 201 {
				t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
			}

			time.Sleep(100 * time.Millisecond)

			// Create proxy
			proxyPort := 8121
			proxyResp, _, err := post("/imposters", map[string]interface{}{
				"protocol": "http",
				"port":     proxyPort,
				"name":     "proxy",
				"stubs": []map[string]interface{}{
					{
						"responses": []map[string]interface{}{
							{
								"proxy": map[string]interface{}{
									"to": fmt.Sprintf("http://localhost:%d", originPort),
								},
							},
						},
					},
				},
			})
			if err != nil {
				t.Fatalf("failed to create proxy imposter: %v", err)
			}
			if proxyResp.StatusCode != 201 {
				t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
			}

			time.Sleep(100 * time.Millisecond)

			// Make request through proxy
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", proxyPort))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Binary content should be preserved
			if string(body) != string(buffer) {
				t.Errorf("expected binary content %v, got %v", buffer, []byte(body))
			}

			// Cleanup for next iteration
			del(fmt.Sprintf("/imposters/%d", originPort))
			del(fmt.Sprintf("/imposters/%d", proxyPort))
		})
	}
}

// TestHTTPProxy_ContentEncodingGzip tests that Content-Encoding: gzip triggers binary mode
func TestHTTPProxy_ContentEncodingGzip(t *testing.T) {
	defer cleanup(t)

	// Create origin server that returns text with Content-Encoding: gzip header
	// This simulates a server that would normally return gzipped content
	originPort := 8122
	textBody := "This is gzipped content"
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": textBody,
							"headers": map[string]interface{}{
								"Content-Encoding": "gzip",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with ProxyOnce mode
	proxyPort := 8123
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyOnce",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request through proxy (this will trigger recording)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	// Verify the recorded stub is in binary mode
	getResp, getBody, err := get(fmt.Sprintf("/imposters/%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	stubs, ok := getBody["stubs"].([]interface{})
	if !ok || len(stubs) < 1 {
		t.Fatalf("expected at least 1 stub, got %d stubs", len(stubs))
	}

	// The recorded stub should be the first one
	recordedStub := stubs[0].(map[string]interface{})
	responses := recordedStub["responses"].([]interface{})
	if len(responses) < 1 {
		t.Fatal("expected at least 1 response")
	}

	response := responses[0].(map[string]interface{})
	isResp := response["is"].(map[string]interface{})

	// Verify binary mode was triggered by Content-Encoding: gzip
	if mode, ok := isResp["_mode"].(string); !ok || mode != "binary" {
		t.Errorf("expected _mode: 'binary' due to Content-Encoding: gzip, got %v", isResp["_mode"])
	}

	// Verify body is base64-encoded
	if body, ok := isResp["body"].(string); ok {
		// Should be able to decode as base64
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			t.Errorf("expected base64-encoded body in binary mode, got decode error: %v", err)
		}
		// Decoded content should match original text
		if string(decoded) != textBody {
			t.Errorf("expected decoded body %q, got %q", textBody, string(decoded))
		}
	} else {
		t.Error("expected body field in recorded response")
	}
}

// TestHTTPProxy_HeaderCasePreservation tests that header case is preserved from origin server
func TestHTTPProxy_HeaderCasePreservation(t *testing.T) {
	defer cleanup(t)

	// Create origin server with custom headers (mixed case)
	originPort := 8124
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "test body",
							"headers": map[string]interface{}{
								"X-Custom-Header":  "value1",
								"X-Another-Header": "value2",
								"Content-Type":     "text/plain",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy
	proxyPort := 8125
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("http://localhost:%d", originPort),
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request through proxy
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	// Check that custom headers are present (case may vary due to HTTP canonicalization)
	if resp.Header.Get("X-Custom-Header") != "value1" {
		t.Errorf("expected X-Custom-Header: value1, got %q", resp.Header.Get("X-Custom-Header"))
	}
	if resp.Header.Get("X-Another-Header") != "value2" {
		t.Errorf("expected X-Another-Header: value2, got %q", resp.Header.Get("X-Another-Header"))
	}

	// Verify recorded stub preserves header case
	getResp, getBody, err := get(fmt.Sprintf("/imposters/%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	stubs, ok := getBody["stubs"].([]interface{})
	if !ok || len(stubs) < 1 {
		t.Fatalf("expected at least 1 stub")
	}

	recordedStub := stubs[0].(map[string]interface{})
	responses := recordedStub["responses"].([]interface{})
	if len(responses) < 1 {
		t.Fatal("expected at least 1 response")
	}

	response := responses[0].(map[string]interface{})
	isResp := response["is"].(map[string]interface{})
	headers := isResp["headers"].(map[string]interface{})

	// Verify headers are preserved
	if headers["X-Custom-Header"] != "value1" {
		t.Errorf("expected recorded header X-Custom-Header: value1, got %v", headers["X-Custom-Header"])
	}
	if headers["X-Another-Header"] != "value2" {
		t.Errorf("expected recorded header X-Another-Header: value2, got %v", headers["X-Another-Header"])
	}
}
