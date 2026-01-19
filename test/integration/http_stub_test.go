package integration

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Tests converted from mbTest/api/http/httpStubTest.js

func TestHttpStub_ShouldReturnStubbedResponse(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5000,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 400,
							"headers":    map[string]interface{}{"X-Test": "test header"},
							"body":       "test body",
						},
					},
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

	impResp, err := http.Get("http://localhost:5000/test?key=true")
	if err != nil {
		t.Fatalf("request to imposter failed: %v", err)
	}
	defer impResp.Body.Close()

	if impResp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", impResp.StatusCode)
	}

	body, _ := io.ReadAll(impResp.Body)
	if string(body) != "test body" {
		t.Errorf("expected 'test body', got '%s'", string(body))
	}

	if impResp.Header.Get("X-Test") != "test header" {
		t.Errorf("expected header 'test header', got '%s'", impResp.Header.Get("X-Test"))
	}
}

func TestHttpStub_ShouldAllowSequenceAsCircularBuffer(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5001,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"statusCode": 400}},
					{"is": map[string]interface{}{"statusCode": 405}},
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

	expected := []int{400, 405, 400, 405}
	for i, exp := range expected {
		impResp, err := http.Get("http://localhost:5001/test")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		impResp.Body.Close()

		if impResp.StatusCode != exp {
			t.Errorf("request %d: expected status %d, got %d", i, exp, impResp.StatusCode)
		}
	}
}

func TestHttpStub_ShouldOnlyMatchComplexPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5002,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"statusCode": 400}},
				},
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/test", "method": "POST"}},
					{"startsWith": map[string]interface{}{"body": "T"}},
					{"contains": map[string]interface{}{"body": "ES"}},
					{"endsWith": map[string]interface{}{"body": "T"}},
					{"matches": map[string]interface{}{"body": "^TEST$"}},
					{"equals": map[string]interface{}{"body": "TEST"}},
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

	// Should not match - wrong path
	resp1, err := http.Post("http://localhost:5002/", "text/plain", strings.NewReader("TEST"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != 200 {
		t.Errorf("wrong path: expected 200, got %d", resp1.StatusCode)
	}

	// Should not match - wrong method
	resp2, err := http.Get("http://localhost:5002/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("wrong method: expected 200, got %d", resp2.StatusCode)
	}

	// Should not match - wrong body (doesn't end with T)
	resp3, err := http.Post("http://localhost:5002/test", "text/plain", strings.NewReader("TESTing"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != 200 {
		t.Errorf("wrong body: expected 200, got %d", resp3.StatusCode)
	}

	// Should match
	resp4, err := http.Post("http://localhost:5002/test", "text/plain", strings.NewReader("TEST"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp4.Body.Close()
	if resp4.StatusCode != 400 {
		t.Errorf("should have matched: expected 400, got %d", resp4.StatusCode)
	}
}

func TestHttpStub_ShouldSupportJSONBodies(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5003,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": map[string]interface{}{
								"key": "value",
								"sub": map[string]interface{}{
									"string-key": "value",
								},
								"arr": []int{1, 2},
							},
						},
					},
					{
						"is": map[string]interface{}{
							"body": map[string]interface{}{
								"key": "second request",
							},
						},
					},
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
	resp1, err := http.Get("http://localhost:5003/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	var result1 map[string]interface{}
	if err := json.Unmarshal(body1, &result1); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if result1["key"] != "value" {
		t.Errorf("expected key='value', got '%v'", result1["key"])
	}

	// Second request
	resp2, err := http.Get("http://localhost:5003/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	var result2 map[string]interface{}
	if err := json.Unmarshal(body2, &result2); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if result2["key"] != "second request" {
		t.Errorf("expected key='second request', got '%v'", result2["key"])
	}
}

func TestHttpStub_ShouldSupportDefaultResponse(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5004,
		"defaultResponse": map[string]interface{}{
			"is": map[string]interface{}{
				"statusCode": 404,
				"body":       "Not found",
			},
		},
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Wrong address"}},
					{"is": map[string]interface{}{"statusCode": 500}},
				},
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/"}},
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

	// First request to / - should return stub response with default statusCode
	resp1, err := http.Get("http://localhost:5004/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "Wrong address" {
		t.Errorf("expected 'Wrong address', got '%s'", string(body1))
	}

	// Second request to / - should return stub statusCode 500
	resp2, err := http.Get("http://localhost:5004/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp2.Body.Close()

	if resp2.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp2.StatusCode)
	}

	// Request to /differentStub - should get default response
	resp3, err := http.Get("http://localhost:5004/differentStub")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if resp3.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp3.StatusCode)
	}
	if string(body3) != "Not found" {
		t.Errorf("expected 'Not found', got '%s'", string(body3))
	}
}

func TestHttpStub_ShouldSupportOverwritingStubs(t *testing.T) {
	defer cleanup(t)

	// Create imposter with initial stub
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5005,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "ORIGINAL"}},
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

	// Overwrite stubs
	putResp, putBody, err := put("/imposters/5005/stubs", map[string]interface{}{
		"stubs": []map[string]interface{}{
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "FIRST"}}}},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "ORIGINAL"}}}},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "THIRD"}}}},
		},
	})
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}

	if putResp.StatusCode != 200 {
		t.Errorf("expected PUT status 200, got %d", putResp.StatusCode)
	}

	stubs := putBody["stubs"].([]interface{})
	if len(stubs) != 3 {
		t.Errorf("expected 3 stubs, got %d", len(stubs))
	}

	// Verify imposter now returns FIRST
	impResp, err := http.Get("http://localhost:5005/")
	if err != nil {
		t.Fatalf("request to imposter failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "FIRST" {
		t.Errorf("expected 'FIRST', got '%s'", string(body))
	}
}

func TestHttpStub_ShouldSupportDeletingSingleStub(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5006,
		"stubs": []map[string]interface{}{
			{
				"responses":  []map[string]interface{}{{"is": map[string]interface{}{"body": "first"}}},
				"predicates": []map[string]interface{}{{"equals": map[string]interface{}{"path": "/first"}}},
			},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SECOND"}}}},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "third"}}}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Delete stub at index 1
	delResp, delBody, err := del("/imposters/5006/stubs/1")
	if err != nil {
		t.Fatalf("DELETE request failed: %v", err)
	}

	if delResp.StatusCode != 200 {
		t.Errorf("expected DELETE status 200, got %d", delResp.StatusCode)
	}

	stubs := delBody["stubs"].([]interface{})
	if len(stubs) != 2 {
		t.Errorf("expected 2 stubs after delete, got %d", len(stubs))
	}

	// Verify imposter now returns 'third' for unmatched requests
	impResp, err := http.Get("http://localhost:5006/")
	if err != nil {
		t.Fatalf("request to imposter failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "third" {
		t.Errorf("expected 'third', got '%s'", string(body))
	}
}

func TestHttpStub_ShouldSupportAddingSingleStub(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5007,
		"stubs": []map[string]interface{}{
			{
				"responses":  []map[string]interface{}{{"is": map[string]interface{}{"body": "first"}}},
				"predicates": []map[string]interface{}{{"equals": map[string]interface{}{"path": "/first"}}},
			},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "third"}}}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Add stub at index 1
	postResp, postBody, err := post("/imposters/5007/stubs", map[string]interface{}{
		"index": 1,
		"stub": map[string]interface{}{
			"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SECOND"}}},
		},
	})
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	if postResp.StatusCode != 200 {
		t.Errorf("expected POST status 200, got %d", postResp.StatusCode)
	}

	stubs := postBody["stubs"].([]interface{})
	if len(stubs) != 3 {
		t.Errorf("expected 3 stubs after add, got %d", len(stubs))
	}

	// Verify imposter now returns 'SECOND' for unmatched requests
	impResp, err := http.Get("http://localhost:5007/")
	if err != nil {
		t.Fatalf("request to imposter failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "SECOND" {
		t.Errorf("expected 'SECOND', got '%s'", string(body))
	}
}

func TestHttpStub_ShouldSupportOverwritingSingleStub(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5008,
		"stubs": []map[string]interface{}{
			{
				"responses":  []map[string]interface{}{{"is": map[string]interface{}{"body": "first"}}},
				"predicates": []map[string]interface{}{{"equals": map[string]interface{}{"path": "/first"}}},
			},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SECOND"}}}},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "third"}}}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Replace stub at index 1
	putResp, putBody, err := put("/imposters/5008/stubs/1", map[string]interface{}{
		"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "CHANGED"}}},
	})
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}

	if putResp.StatusCode != 200 {
		t.Errorf("expected PUT status 200, got %d", putResp.StatusCode)
	}

	stubs := putBody["stubs"].([]interface{})
	if len(stubs) != 3 {
		t.Errorf("expected 3 stubs, got %d", len(stubs))
	}

	// Verify imposter now returns 'CHANGED' for unmatched requests
	impResp, err := http.Get("http://localhost:5008/")
	if err != nil {
		t.Fatalf("request to imposter failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "CHANGED" {
		t.Errorf("expected 'CHANGED', got '%s'", string(body))
	}
}

func TestHttpStub_ShouldRequireProtocol(t *testing.T) {
	defer cleanup(t)

	// Try to create imposter without protocol
	postResp, postBody, err := post("/imposters", map[string]interface{}{
		"port": 5009,
	})
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	if postResp.StatusCode != 400 {
		t.Errorf("expected POST status 400, got %d", postResp.StatusCode)
	}

	errors, ok := postBody["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Error("expected errors array")
		return
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "bad data" {
		t.Errorf("expected error code 'bad data', got '%v'", firstError["code"])
	}
}

func TestHttpStub_ShouldAddStubAtEndWithoutIndex(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5010,
		"stubs": []map[string]interface{}{
			{
				"responses":  []map[string]interface{}{{"is": map[string]interface{}{"body": "first"}}},
				"predicates": []map[string]interface{}{{"equals": map[string]interface{}{"path": "/first"}}},
			},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "third"}}}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Add stub without index (should add at end)
	postResp, postBody, err := post("/imposters/5010/stubs", map[string]interface{}{
		"stub": map[string]interface{}{
			"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "LAST"}}},
		},
	})
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	if postResp.StatusCode != 200 {
		t.Errorf("expected POST status 200, got %d", postResp.StatusCode)
	}

	stubs := postBody["stubs"].([]interface{})
	if len(stubs) != 3 {
		t.Errorf("expected 3 stubs after add, got %d", len(stubs))
	}

	// Verify imposter returns 'third' for unmatched (LAST is at the end)
	impResp, err := http.Get("http://localhost:5010/")
	if err != nil {
		t.Fatalf("request to imposter failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "third" {
		t.Errorf("expected 'third', got '%s'", string(body))
	}
}

// TestHttpStub_DeepEqualsObjectPredicates tests deepEquals with query params containing predicate keywords
// mountebank httpStubTest.js: "should correctly handle deepEquals object predicates"
func TestHttpStub_DeepEqualsObjectPredicates(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5011,
		"stubs": []map[string]interface{}{
			{
				"responses":  []map[string]interface{}{{"is": map[string]interface{}{"body": "first stub"}}},
				"predicates": []map[string]interface{}{{"deepEquals": map[string]interface{}{"query": map[string]interface{}{}}}},
			},
			{
				"responses":  []map[string]interface{}{{"is": map[string]interface{}{"body": "second stub"}}},
				"predicates": []map[string]interface{}{{"deepEquals": map[string]interface{}{"query": map[string]interface{}{"equals": "1"}}}},
			},
			{
				"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "third stub"}}},
				"predicates": []map[string]interface{}{
					{"deepEquals": map[string]interface{}{"query": map[string]interface{}{"equals": "true", "contains": "false"}}},
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

	// First request - empty query
	resp1, err := http.Get("http://localhost:5011/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if string(body1) != "first stub" {
		t.Errorf("empty query: expected 'first stub', got '%s'", string(body1))
	}

	// Second request - equals=something (should not match)
	resp2, err := http.Get("http://localhost:5011/?equals=something")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if string(body2) != "" {
		t.Errorf("equals=something: expected empty response, got '%s'", string(body2))
	}

	// Third request - equals=1 (should match second stub)
	resp3, err := http.Get("http://localhost:5011/?equals=1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if string(body3) != "second stub" {
		t.Errorf("equals=1: expected 'second stub', got '%s'", string(body3))
	}

	// Fourth request - contains=false&equals=true (should match third stub)
	resp4, err := http.Get("http://localhost:5011/?contains=false&equals=true")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	if string(body4) != "third stub" {
		t.Errorf("contains=false&equals=true: expected 'third stub', got '%s'", string(body4))
	}

	// Fifth request - extra parameter (should not match)
	resp5, err := http.Get("http://localhost:5011/?contains=false&equals=true&matches=yes")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body5, _ := io.ReadAll(resp5.Body)
	resp5.Body.Close()
	if string(body5) != "" {
		t.Errorf("extra parameter: expected empty response, got '%s'", string(body5))
	}
}

// TestHttpStub_JSONBodyPredicateMatching tests treating body as JSON for predicate matching
// mountebank httpStubTest.js: "should support treating the body as a JSON object for predicate matching"
func TestHttpStub_JSONBodyPredicateMatching(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5012,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SUCCESS"}}},
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"body": map[string]interface{}{"key": "value"}}},
					{"equals": map[string]interface{}{"body": map[string]interface{}{"arr": float64(3)}}},
					{"deepEquals": map[string]interface{}{"body": map[string]interface{}{"key": "value", "arr": []interface{}{float64(2), float64(1), float64(3)}}}},
					{"matches": map[string]interface{}{"body": map[string]interface{}{"key": "^v"}}},
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

	// Send JSON body that should match all predicates
	impResp, err := http.Post("http://localhost:5012/", "application/json",
		strings.NewReader(`{"key": "value", "arr": [3,2,1]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "SUCCESS" {
		t.Errorf("expected 'SUCCESS', got '%s'", string(body))
	}
}

// TestHttpStub_JSONNullValues tests handling JSON null values in responses
// mountebank httpStubTest.js: "should handle JSON null values"
func TestHttpStub_JSONNullValues(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5013,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": map[string]interface{}{"name": "test", "type": nil}}},
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

	impResp, err := http.Get("http://localhost:5013/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("expected name='test', got '%v'", result["name"])
	}
	if result["type"] != nil {
		t.Errorf("expected type=null, got '%v'", result["type"])
	}
}

// TestHttpStub_DeepEqualsNullPredicate tests null values in deepEquals predicate
// mountebank httpStubTest.js: "should handle null values in deepEquals predicate (issue #229)"
func TestHttpStub_DeepEqualsNullPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5014,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"deepEquals": map[string]interface{}{"body": map[string]interface{}{"field": nil}}},
				},
				"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SUCCESS"}}},
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

	impResp, err := http.Post("http://localhost:5014/", "application/json",
		strings.NewReader(`{"field": null}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "SUCCESS" {
		t.Errorf("expected 'SUCCESS', got '%s'", string(body))
	}
}

// TestHttpStub_EqualsNullPredicate tests null values in equals predicate
// mountebank httpStubTest.js: "should support predicate matching with null value (issue #262)"
func TestHttpStub_EqualsNullPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5015,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"body": map[string]interface{}{"version": nil}}},
				},
				"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SUCCESS"}}},
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

	impResp, err := http.Post("http://localhost:5015/", "application/json",
		strings.NewReader(`{"version": null}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "SUCCESS" {
		t.Errorf("expected 'SUCCESS', got '%s'", string(body))
	}
}

// TestHttpStub_XPathArrayPredicates tests array predicates with XPath selector
// mountebank httpStubTest.js: "should support array predicates with xpath"
func TestHttpStub_XPathArrayPredicates(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5016,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SUCCESS"}}},
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{"body": []string{"first", "third", "second"}},
						"xpath":  map[string]interface{}{"selector": "//value"},
					},
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

	xml := `<values><value>first</value><value>second</value><value>third</value></values>`
	impResp, err := http.Post("http://localhost:5016/", "application/xml",
		strings.NewReader(xml))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "SUCCESS" {
		t.Errorf("expected 'SUCCESS', got '%s'", string(body))
	}
}

// TestHttpStub_GzipRequestPredicates tests predicate matching on gzipped requests
// mountebank httpStubTest.js: "should support predicate from gzipped request (issue #499)"
func TestHttpStub_GzipRequestPredicates(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5017,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SUCCESS"}}},
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"body": map[string]interface{}{"key": "value"}}},
					{"equals": map[string]interface{}{"body": map[string]interface{}{"arr": float64(3)}}},
					{"deepEquals": map[string]interface{}{"body": map[string]interface{}{"key": "value", "arr": []interface{}{float64(2), float64(1), float64(3)}}}},
					{"matches": map[string]interface{}{"body": map[string]interface{}{"key": "^v"}}},
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

	// Create gzipped JSON payload
	jsonData := []byte(`{"key": "value", "arr": [3,2,1]}`)
	var buf strings.Builder
	gzipWriter := gzip.NewWriter(&buf)
	gzipWriter.Write(jsonData)
	gzipWriter.Close()

	// Send gzipped request
	req, err := http.NewRequest("POST", "http://localhost:5017/", strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Encoding", "gzip")

	client := &http.Client{}
	impResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "SUCCESS" {
		t.Errorf("expected 'SUCCESS' for gzipped request, got '%s'", string(body))
	}
}

// TestHttpStub_ValidationErrors tests stub validation error reporting
// mountebank httpStubTest.js: "should provide a good error message when adding stub with missing information"
func TestHttpStub_ValidationErrors(t *testing.T) {
	defer cleanup(t)

	// Create initial imposter
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5018,
		"stubs": []map[string]interface{}{
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "first"}}}},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Try to add stub with missing 'stub' field (using 'STUBS' instead)
	newStub := map[string]interface{}{
		"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "SECOND"}}},
	}
	errorBody := map[string]interface{}{
		"index": 1,
		"STUBS": newStub, // Wrong field name
	}

	postResp, postBody, err := post("/imposters/5018/stubs", errorBody)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	if postResp.StatusCode != 400 {
		t.Errorf("expected status 400 for validation error, got %d", postResp.StatusCode)
	}

	errors, ok := postBody["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Error("expected errors array in response")
		return
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "bad data" {
		t.Errorf("expected error code 'bad data', got '%v'", firstError["code"])
	}

	message := firstError["message"].(string)
	if !contains(message, "stub") {
		t.Errorf("expected error message to mention 'stub', got '%s'", message)
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
