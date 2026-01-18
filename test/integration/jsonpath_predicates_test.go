package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// JSONPath Predicates tests - based on mountebank's jsonpathTest.js
// Tests for extracting values using JSONPath before predicate comparison

// ============================================================================
// EQUALS WITH JSONPATH
// ============================================================================

// TestJSONPathEquals_NotJSON tests that JSONPath fails when body is not JSON
// mountebank jsonpathTest.js: "should be false if field is not JSON"
func TestJSONPathEquals_NotJSON(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6200,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "VALUE",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "no match"}},
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

	time.Sleep(100 * time.Millisecond)

	// Non-JSON body should fail
	impResp, err := http.Post("http://localhost:6200/", "text/plain",
		strings.NewReader("VALUE"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("JSONPath should not match non-JSON body")
	}
}

// TestJSONPathEquals_RecursiveDescent tests $..title recursive descent
// mountebank jsonpathTest.js: "should be true if value in provided json"
func TestJSONPathEquals_RecursiveDescent(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6201,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "VALUE",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	time.Sleep(100 * time.Millisecond)

	impResp, err := http.Post("http://localhost:6201/", "application/json",
		strings.NewReader(`{ "title": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for recursive descent, got '%s'", string(body))
	}
}

// TestJSONPathEquals_CaseInsensitiveSelector tests default case-insensitive selector
// mountebank jsonpathTest.js: "should use case-insensitive json selector by default"
func TestJSONPathEquals_CaseInsensitiveSelector(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6202,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "VALUE",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..Title", // Capital T in selector
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "no match"}},
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

	time.Sleep(100 * time.Millisecond)

	// Lowercase 'title' in JSON should match selector '$..Title'
	impResp, err := http.Post("http://localhost:6202/", "application/json",
		strings.NewReader(`{ "title": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for case-insensitive selector, got '%s'", string(body))
	}
}

// TestJSONPathEquals_CaseSensitiveSelector tests caseSensitive option for selectors
// mountebank jsonpathTest.js: "should not equal if case-sensitive json selector does not match"
func TestJSONPathEquals_CaseSensitiveSelector(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6203,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "value",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title", // lowercase t
						},
						"caseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "no match"}},
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

	time.Sleep(100 * time.Millisecond)

	// TITLE (uppercase) in JSON should NOT match selector '$..title' with caseSensitive
	impResp, err := http.Post("http://localhost:6203/", "application/json",
		strings.NewReader(`{ "TITLE": "value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match with caseSensitive when key case differs")
	}
}

// ============================================================================
// DEEP EQUALS WITH JSONPATH
// ============================================================================

// TestJSONPathDeepEquals_NestedAttribute tests $.title..attribute selector
// mountebank jsonpathTest.js: "should be false if value in provided jsonpath attribute expression does not equal"
func TestJSONPathDeepEquals_NestedAttribute(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6210,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": "value",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.title.attribute",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "no match"}},
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

	time.Sleep(100 * time.Millisecond)

	// Should match
	impResp1, err := http.Post("http://localhost:6210/", "application/json",
		strings.NewReader(`{ "title": { "attribute": "value" } }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' for nested attribute, got '%s'", string(body1))
	}

	// Should not match - wrong value
	impResp2, err := http.Post("http://localhost:6210/", "application/json",
		strings.NewReader(`{ "title": { "attribute": "wrong" } }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match with wrong value")
	}
}

// TestJSONPathDeepEquals_DoublyNested tests $.title.attribute.test selector
// mountebank jsonpathTest.js: "should be true if doubly embedded value in provided jsonpath attribute expression does equal"
func TestJSONPathDeepEquals_DoublyNested(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6211,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": "value",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.title.attribute.test",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	time.Sleep(100 * time.Millisecond)

	impResp, err := http.Post("http://localhost:6211/", "application/json",
		strings.NewReader(`{ "title": { "attribute": { "test": "value" } } }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for doubly nested path, got '%s'", string(body))
	}
}

// TestJSONPathDeepEquals_ArrayIndex tests $..title[0].attribute selector
// mountebank jsonpathTest.js: "should return a string if looking at an index of 1 item"
func TestJSONPathDeepEquals_ArrayIndex(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6212,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": "value",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title[0].attribute",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	time.Sleep(100 * time.Millisecond)

	impResp, err := http.Post("http://localhost:6212/", "application/json",
		strings.NewReader(`{ "title": [{ "attribute": "value" }, { "attribute": "other value" }] }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for array index access, got '%s'", string(body))
	}
}

// TestJSONPathDeepEquals_BooleanValue tests matching boolean values
// mountebank jsonpathTest.js: "should be true if boolean value matches"
func TestJSONPathDeepEquals_BooleanValue(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6213,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": false,
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..active",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	time.Sleep(100 * time.Millisecond)

	impResp, err := http.Post("http://localhost:6213/", "application/json",
		strings.NewReader(`{ "active": false }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for boolean value, got '%s'", string(body))
	}
}

// ============================================================================
// CONTAINS, STARTSWITH WITH JSONPATH
// ============================================================================

// TestJSONPathContains tests contains predicate with JSONPath
// mountebank jsonpathTest.js: "should be true if direct text value contains predicate"
func TestJSONPathContains(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6220,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"body": "value",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	time.Sleep(100 * time.Millisecond)

	impResp, err := http.Post("http://localhost:6220/", "application/json",
		strings.NewReader(`{ "title": "this is a value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for contains with JSONPath, got '%s'", string(body))
	}
}

// TestJSONPathStartsWith tests startsWith predicate with JSONPath
// mountebank jsonpathTest.js: "should be true if direct namespaced jsonpath selection starts with value"
func TestJSONPathStartsWith(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6221,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"startsWith": map[string]interface{}{
							"body": "this",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "no match"}},
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

	time.Sleep(100 * time.Millisecond)

	// Should match - value starts with 'this'
	impResp1, err := http.Post("http://localhost:6221/", "application/json",
		strings.NewReader(`{ "title": "this is a value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' for startsWith, got '%s'", string(body1))
	}

	// Should NOT match - value doesn't start with 'this'
	impResp2, err := http.Post("http://localhost:6221/", "application/json",
		strings.NewReader(`{ "title": "if this is a value, it is a value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match when value doesn't start with prefix")
	}
}

// ============================================================================
// EXISTS WITH JSONPATH
// ============================================================================

// TestJSONPathExists_HasResult tests exists predicate when JSONPath finds a match
// mountebank jsonpathTest.js: "should be true if jsonpath selector has at least one result"
func TestJSONPathExists_HasResult(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6230,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body": true,
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "no match"}},
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

	time.Sleep(100 * time.Millisecond)

	// Should match - title exists
	impResp1, err := http.Post("http://localhost:6230/", "application/json",
		strings.NewReader(`{ "title": "value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' when JSONPath finds result, got '%s'", string(body1))
	}

	// Should NOT match - title doesn't exist
	impResp2, err := http.Post("http://localhost:6230/", "application/json",
		strings.NewReader(`{ "newTitle": "if this is a value, it is a value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match when JSONPath finds no result")
	}
}

// ============================================================================
// MATCHES WITH JSONPATH
// ============================================================================

// TestJSONPathMatches tests matches predicate with JSONPath
// mountebank jsonpathTest.js: "should be true if selected value matches regex"
func TestJSONPathMatches(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6240,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": "^v",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$..title",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "no match"}},
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

	time.Sleep(100 * time.Millisecond)

	// Should match - 'value' starts with 'v'
	impResp1, err := http.Post("http://localhost:6240/", "application/json",
		strings.NewReader(`{ "title": "value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' for regex match, got '%s'", string(body1))
	}

	// Should NOT match - 'value' doesn't end with 'v'
	impResp2, err := http.Post("http://localhost:6240/", "application/json",
		strings.NewReader(`{ "title": "test" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match when regex doesn't match")
	}
}

// TestJSONPathMatches_Issue361 tests that selector formatting is preserved
// mountebank jsonpathTest.js: "should maintain selector to match JSON (issue #361)"
func TestJSONPathMatches_Issue361(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6241,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": `111\.222\.333\.*`,
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.ipAddress",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	time.Sleep(100 * time.Millisecond)

	impResp, err := http.Post("http://localhost:6241/", "application/json",
		strings.NewReader(`{ "ipAddress": "111.222.333.456" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for IP address regex (issue #361), got '%s'", string(body))
	}
}
