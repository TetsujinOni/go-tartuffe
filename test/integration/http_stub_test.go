package integration

import (
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
