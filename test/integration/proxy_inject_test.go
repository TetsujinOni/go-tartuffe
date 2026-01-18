package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Proxy and Inject tests

func TestProxy_ShouldProxyToTarget(t *testing.T) {
	defer cleanup(t)

	// Create a target imposter
	targetResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5300,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{
						"statusCode": 200,
						"headers":    map[string]interface{}{"X-Target": "true"},
						"body":       "response from target",
					}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create target imposter: %v", err)
	}
	if targetResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", targetResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy imposter
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5301,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"proxy": map[string]interface{}{
						"to":   "http://localhost:5300",
						"mode": "proxyTransparent",
					}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request to proxy
	resp, err := http.Get("http://localhost:5301/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if string(body) != "response from target" {
		t.Errorf("expected 'response from target', got '%s'", string(body))
	}

	if resp.Header.Get("X-Target") != "true" {
		t.Error("expected X-Target header from target")
	}
}

func TestProxy_ProxyOnce_ShouldRecordAndReplay(t *testing.T) {
	defer cleanup(t)

	// Create a target imposter that returns different responses
	targetResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5302,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "first"}},
					{"is": map[string]interface{}{"body": "second"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create target imposter: %v", err)
	}
	if targetResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", targetResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxyOnce imposter
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5303,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"proxy": map[string]interface{}{
						"to":   "http://localhost:5302",
						"mode": "proxyOnce",
					}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// First request - should proxy and record
	resp1, err := http.Get("http://localhost:5303/test")
	if err != nil {
		t.Fatalf("request 1 failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "first" {
		t.Errorf("expected 'first', got '%s'", string(body1))
	}

	time.Sleep(50 * time.Millisecond)

	// Second request - should replay recorded stub
	resp2, err := http.Get("http://localhost:5303/test")
	if err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	// Should still be "first" because it's replaying the recorded stub
	if string(body2) != "first" {
		t.Errorf("expected 'first' (replay), got '%s'", string(body2))
	}
}

func TestProxy_WithPredicateGenerators(t *testing.T) {
	defer cleanup(t)

	// Create target
	targetResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5304,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "from target"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create target imposter: %v", err)
	}
	if targetResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", targetResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with predicate generators
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5305,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"proxy": map[string]interface{}{
						"to":   "http://localhost:5304",
						"mode": "proxyOnce",
						"predicateGenerators": []map[string]interface{}{
							{
								"matches": map[string]interface{}{
									"method": true,
									"path":   true,
								},
							},
						},
					}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request to record
	resp, err := http.Get("http://localhost:5305/api/users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if string(body) != "from target" {
		t.Errorf("expected 'from target', got '%s'", string(body))
	}

	time.Sleep(50 * time.Millisecond)

	// Verify stub was recorded with correct predicates
	getResp, imposter, err := get("/imposters/5305")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	stubs, ok := imposter["stubs"].([]interface{})
	if !ok || len(stubs) < 2 {
		t.Fatalf("expected at least 2 stubs (recorded + proxy), got %v", imposter["stubs"])
	}
}

func TestInject_Response_ShouldExecuteJavaScript(t *testing.T) {
	defer cleanup(t)

	injectScript := `function(request, state, logger) {
		return {
			statusCode: 201,
			headers: { "X-Custom": "injected" },
			body: "Hello from inject! Path was: " + request.path
		};
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5306,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"inject": injectScript},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request
	impResp, err := http.Get("http://localhost:5306/mypath")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if impResp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", impResp.StatusCode)
	}

	if impResp.Header.Get("X-Custom") != "injected" {
		t.Errorf("expected X-Custom header 'injected', got '%s'", impResp.Header.Get("X-Custom"))
	}

	expectedBody := "Hello from inject! Path was: /mypath"
	if string(body) != expectedBody {
		t.Errorf("expected '%s', got '%s'", expectedBody, string(body))
	}
}

func TestInject_Response_ShouldAccessRequestBody(t *testing.T) {
	defer cleanup(t)

	injectScript := `function(request, state, logger) {
		var data = JSON.parse(request.body);
		return {
			statusCode: 200,
			body: JSON.stringify({ received: data.message })
		};
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5307,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"inject": injectScript},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make POST request with JSON body
	reqBody := strings.NewReader(`{"message": "hello world"}`)
	impResp, err := http.Post("http://localhost:5307/echo", "application/json", reqBody)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response: %v (body: %s)", err, string(body))
	}

	if result["received"] != "hello world" {
		t.Errorf("expected received='hello world', got %v", result["received"])
	}
}

func TestInject_Predicate_ShouldMatchBasedOnScript(t *testing.T) {
	defer cleanup(t)

	injectScript := `function(request, logger) {
		return request.path === "/secret";
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5308,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"inject": injectScript},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched secret"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "default"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Should match inject predicate
	resp1, err := http.Get("http://localhost:5308/secret")
	if err != nil {
		t.Fatalf("request 1 failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched secret" {
		t.Errorf("expected 'matched secret', got '%s'", string(body1))
	}

	// Should not match, fall through to default
	resp2, err := http.Get("http://localhost:5308/other")
	if err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "default" {
		t.Errorf("expected 'default', got '%s'", string(body2))
	}
}

func TestInject_Response_ShouldReturnJSONObject(t *testing.T) {
	defer cleanup(t)

	injectScript := `function(request, state, logger) {
		return {
			statusCode: 200,
			headers: { "Content-Type": "application/json" },
			body: { id: 123, name: "test" }
		};
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5309,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"inject": injectScript},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request
	impResp, err := http.Get("http://localhost:5309/data")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response JSON: %v (body: %s)", err, string(body))
	}

	if result["id"] != float64(123) {
		t.Errorf("expected id=123, got %v", result["id"])
	}
	if result["name"] != "test" {
		t.Errorf("expected name='test', got %v", result["name"])
	}
}

func TestProxy_InjectHeaders(t *testing.T) {
	defer cleanup(t)

	// Create target that echoes headers
	// Headers are stored with canonical case (Go Title-Case)
	targetScript := `function(request, state, logger) {
		return {
			statusCode: 200,
			body: JSON.stringify({
				authorization: request.headers["Authorization"],
				custom: request.headers["X-Custom-Header"]
			})
		};
	}`

	targetResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5310,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"inject": targetScript},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create target imposter: %v", err)
	}
	if targetResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", targetResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with injected headers
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5311,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"proxy": map[string]interface{}{
						"to":   "http://localhost:5310",
						"mode": "proxyTransparent",
						"injectHeaders": map[string]interface{}{
							"Authorization":   "Bearer test-token",
							"X-Custom-Header": "custom-value",
						},
					}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make request through proxy
	resp, err := http.Get("http://localhost:5311/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response: %v (body: %s)", err, string(body))
	}

	if result["authorization"] != "Bearer test-token" {
		t.Errorf("expected authorization='Bearer test-token', got %v", result["authorization"])
	}
	if result["custom"] != "custom-value" {
		t.Errorf("expected custom='custom-value', got %v", result["custom"])
	}
}

// TestInject_NewConfigInterface tests the new config interface (config.request, config.state, config.logger)
// mountebank test: "should allow javascript predicate for matching" (test #21)
func TestInject_NewConfigInterface_Predicate(t *testing.T) {
	defer cleanup(t)

	// New interface: function takes single config parameter with request, state, logger
	injectScript := `config => config.request.path === '/test'`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5320,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"inject": injectScript},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "MATCHED"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "DEFAULT"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Test matching
	impResp, err := http.Get("http://localhost:5320/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "MATCHED" {
		t.Errorf("expected 'MATCHED', got '%s'", string(body))
	}
}

// TestInject_StateInPredicateInjection tests state modification in predicate injection
// mountebank test: "should allow changing state in predicate injection (issue #495)" (test #22)
func TestInject_StateInPredicateInjection(t *testing.T) {
	defer cleanup(t)

	// Predicate that modifies state and matches on second request
	injectScript := `config => {
		config.state.requests = config.state.requests || 0;
		config.state.requests += 1;
		return config.state.requests === 2;
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5321,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"inject": injectScript},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "MATCHED"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "UNMATCHED"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// First request - state.requests becomes 1, should not match
	resp1, _ := http.Get("http://localhost:5321/")
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "UNMATCHED" {
		t.Errorf("first request: expected 'UNMATCHED', got '%s'", string(body1))
	}

	// Second request - state.requests becomes 2, should match
	resp2, _ := http.Get("http://localhost:5321/")
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "MATCHED" {
		t.Errorf("second request: expected 'MATCHED', got '%s'", string(body2))
	}
}

// TestInject_StatePersistenceAcrossRequests tests state persistence in response injection
// mountebank test: "should allow javascript injection to keep state between requests" (test #25-26)
func TestInject_StatePersistenceAcrossRequests(t *testing.T) {
	defer cleanup(t)

	// Response injection that increments a counter
	injectScript := `config => {
		if (!config.state.calls) { config.state.calls = 0; }
		config.state.calls += 1;
		return { body: config.state.calls.toString() };
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5322,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"inject": injectScript},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// First request
	resp1, _ := http.Get("http://localhost:5322/")
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "1" {
		t.Errorf("first request: expected '1', got '%s'", string(body1))
	}

	// Second request
	resp2, _ := http.Get("http://localhost:5322/")
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "2" {
		t.Errorf("second request: expected '2', got '%s'", string(body2))
	}
}

// TestInject_SharedStateBetweenPredicateAndResponse tests state sharing
// mountebank test: "should share state with predicate and response injection" (test #27-28)
func TestInject_SharedStateBetweenPredicateAndResponse(t *testing.T) {
	defer cleanup(t)

	// Response injection that increments calls counter
	responseFn := `config => {
		config.state.calls = config.state.calls || 0;
		config.state.calls += 1;
		return { body: 'INJECT' };
	}`

	// Predicate injection that checks if calls > 1
	predicateFn := `config => {
		var numCalls = config.state.calls || 0;
		return numCalls > 1;
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5323,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"and": []map[string]interface{}{
						{"inject": predicateFn},
						{"equals": map[string]interface{}{"path": "/"}},
					}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "IS"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"inject": responseFn},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// First request - predicate checks calls=0, false; falls to inject which sets calls=1
	resp1, _ := http.Get("http://localhost:5323/")
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "INJECT" {
		t.Errorf("first request: expected 'INJECT', got '%s'", string(body1))
	}

	// Second request - predicate checks calls=1, false (1 > 1 is false); falls to inject which sets calls=2
	resp2, _ := http.Get("http://localhost:5323/")
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "INJECT" {
		t.Errorf("second request: expected 'INJECT', got '%s'", string(body2))
	}

	// Third request - predicate checks calls=2, true (2 > 1); returns IS
	resp3, _ := http.Get("http://localhost:5323/")
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if string(body3) != "IS" {
		t.Errorf("third request: expected 'IS', got '%s'", string(body3))
	}
}
