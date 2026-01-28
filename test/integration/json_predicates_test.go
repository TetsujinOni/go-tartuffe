package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// JSON Predicates tests - based on mountebank's jsonTest.js
// Tests for treating strings as JSON in predicates

// ============================================================================
// EQUALS PREDICATE WITH JSON
// ============================================================================

// TestJSONEquals_NotJSON tests that equals fails when field is not valid JSON
// mountebank jsonTest.js: "should be false if field does not equal given value"
func TestJSONEquals_NotJSON(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6100,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "VALUE"},
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

	// Send non-JSON body
	impResp, err := http.Post("http://localhost:6100/", "text/plain",
		strings.NewReader("KEY: VALUE"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match non-JSON body against JSON predicate")
	}
}

// TestJSONEquals_NoMatch tests that equals fails when JSON doesn't match
// mountebank jsonTest.js: "should be false if JSON string value does not equal JSON predicate"
func TestJSONEquals_NoMatch(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6100,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "VALUE"},
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

	impResp, err := http.Post("http://localhost:6100/", "application/json",
		strings.NewReader("Not value"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match when JSON value differs")
	}
}

// TestJSONEquals_Match tests that equals matches JSON string against JSON predicate
// mountebank jsonTest.js: "should be true if JSON string value equals JSON predicate"
func TestJSONEquals_Match(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6101,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "VALUE"},
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

	impResp, err := http.Post("http://localhost:6101/", "application/json",
		strings.NewReader(`{ "key": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestJSONEquals_CaseInsensitive tests default case-insensitive JSON matching
// mountebank jsonTest.js: "should be true if JSON string value equals JSON predicate except for case"
func TestJSONEquals_CaseInsensitive(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6102,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"KEY": "value"},
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

	// Different case in both key and value
	impResp, err := http.Post("http://localhost:6102/", "application/json",
		strings.NewReader(`{ "key": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' (case-insensitive), got '%s'", string(body))
	}
}

// TestJSONEquals_CaseSensitive tests caseSensitive option blocks different case
// mountebank jsonTest.js: "should not be true if JSON string value case different and caseSensitive is true"
func TestJSONEquals_CaseSensitive(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6103,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"KEY": "value"},
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

	// Different case should NOT match with caseSensitive
	impResp, err := http.Post("http://localhost:6103/", "application/json",
		strings.NewReader(`{ "key": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match with case-sensitive when case differs")
	}
}

// TestJSONEquals_WithExcept tests except option strips pattern before comparison
// mountebank jsonTest.js: "should equal if case-sensitive predicate matches, stripping out the exception"
func TestJSONEquals_WithExcept(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6107,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "VE"},
						},
						"caseSensitive": true,
						"except":        "ALU",
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

	// "VALUE" with "ALU" removed = "VE" (matches)
	impResp, err := http.Post("http://localhost:6107/", "application/json",
		strings.NewReader(`{ "key": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' with except stripping, got '%s'", string(body))
	}
}

// TestJSONEquals_ExceptMismatch tests except option when stripped values don't match
// mountebank jsonTest.js: "should not equal if case-sensitive predicate matches, but stripped values differ"
func TestJSONEquals_ExceptMismatch(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6108,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "V"},
						},
						"caseSensitive": true,
						"except":        "ALU",
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

	// "VALUE" with "ALU" removed = "VE" (not "V")
	impResp, err := http.Post("http://localhost:6108/", "application/json",
		strings.NewReader(`{ "key": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match when stripped values differ")
	}
}

// TestJSONEquals_NoArrayMatch tests when no array elements match
// mountebank jsonTest.js: "should be false if no array elements match the predicate value"
func TestJSONEquals_NoArrayMatch(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6109,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "Second"},
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

	// Array without matching element
	impResp, err := http.Post("http://localhost:6109/", "application/json",
		strings.NewReader(`{"key": ["first", "third"]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match when no array elements match (case-insensitive)")
	}
}

// TestJSONEquals_AllKeysNoMatch tests when object array keys don't all match
// mountebank jsonTest.js: "should be false if all keys in an array do not match"
func TestJSONEquals_AllKeysNoMatch(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6110,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": true},
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

	// Not all objects have matching key value
	impResp, err := http.Post("http://localhost:6110/", "application/json",
		strings.NewReader(`[{ "key": "first" }, { "different": true }, { "key": "third" }]`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match when not all objects have matching key value")
	}
}

// TestJSONEquals_ArrayElement tests matching any element in JSON array
// mountebank jsonTest.js: "should be true if any array element equals the predicate value"
func TestJSONEquals_ArrayElement(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6104,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "Second"},
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

	// Array with matching element
	impResp, err := http.Post("http://localhost:6104/", "application/json",
		strings.NewReader(`{"key": ["First", "Second", "Third"]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' when array contains element, got '%s'", string(body))
	}
}

// TestJSONEquals_NullValue tests matching null values in JSON
// mountebank jsonTest.js: "should be true if null value for key matches"
func TestJSONEquals_NullValue(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6105,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": nil},
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

	impResp, err := http.Post("http://localhost:6105/", "application/json",
		strings.NewReader(`{ "key": null }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for null value, got '%s'", string(body))
	}
}

// TestJSONEquals_ObjectInArray tests matching object in array
// mountebank jsonTest.js: "should be true if matches key for any object in array"
func TestJSONEquals_ObjectInArray(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6106,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": map[string]interface{}{"key": "third"},
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

	// Array of objects, one with matching key
	impResp, err := http.Post("http://localhost:6106/", "application/json",
		strings.NewReader(`[{ "key": "first" }, { "different": true }, { "key": "third" }]`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' when object in array matches, got '%s'", string(body))
	}
}

// ============================================================================
// DEEP EQUALS PREDICATE WITH JSON
// ============================================================================

// TestJSONDeepEquals_NotJSON tests that deepEquals fails when field is not valid JSON
// mountebank jsonTest.js: "should be false if field is not JSON and JSON predicate used"
func TestJSONDeepEquals_NotJSON(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6150,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{"key": "VALUE"},
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

	// Send invalid JSON
	impResp, err := http.Post("http://localhost:6150/", "text/plain",
		strings.NewReader(`"key": "VALUE"`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match non-JSON against JSON predicate with deepEquals")
	}
}

// TestJSONDeepEquals_Match tests basic deepEquals with JSON
// mountebank jsonTest.js: "should equal value in provided JSON attribute"
func TestJSONDeepEquals_Match(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6110,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{"key": "VALUE"},
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

	impResp, err := http.Post("http://localhost:6110/", "application/json",
		strings.NewReader(`{"key": "VALUE"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestJSONDeepEquals_NoMatch tests that deepEquals fails when values don't match
// mountebank jsonTest.js: "should be false if value in provided JSON predicate does not equal"
func TestJSONDeepEquals_NoMatch(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6151,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{"key": "test"},
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

	impResp, err := http.Post("http://localhost:6151/", "application/json",
		strings.NewReader(`{ "key": "VALUE"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match when values differ (case-insensitive test)")
	}
}

// TestJSONDeepEquals_NestedObject tests deepEquals with nested objects
// mountebank jsonTest.js: "should be true if all values in a JSON predicate match are present"
func TestJSONDeepEquals_NestedObject(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6111,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{
								"key":   "value",
								"outer": map[string]interface{}{"inner": "value"},
							},
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

	impResp, err := http.Post("http://localhost:6111/", "application/json",
		strings.NewReader(`{"key": "VALUE", "outer": { "inner": "value" } }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for nested objects, got '%s'", string(body))
	}
}

// TestJSONDeepEquals_MissingFields tests that deepEquals fails when request is missing fields
// mountebank jsonTest.js: "should be false if some values in a multi-value JSON predicate match are missing"
func TestJSONDeepEquals_MissingFields(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6152,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{
								"key":   "value",
								"outer": map[string]interface{}{"inner": "value"},
							},
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

	// Missing "key" field
	impResp, err := http.Post("http://localhost:6152/", "application/json",
		strings.NewReader(`{"outer": { "inner": "value" } }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("deepEquals should not match when request is missing fields")
	}
}

// TestJSONDeepEquals_ArrayOrderInsensitive tests that array elements match regardless of order
// mountebank jsonTest.js: "should be true if all array values in a JSON predicate match are present regardless of order"
func TestJSONDeepEquals_ArrayOrderInsensitive(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6112,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{"key": []interface{}{float64(2), float64(1), float64(3)}},
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

	// Different order should match
	impResp, err := http.Post("http://localhost:6112/", "application/json",
		strings.NewReader(`{"key": [3, 1, 2] }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for array with different order, got '%s'", string(body))
	}
}

// TestJSONDeepEquals_ExtraFields tests that extra fields fail deepEquals
// mountebank jsonTest.js: "should be false if some values in a multi-value JSON predicate match are missing"
func TestJSONDeepEquals_ExtraFields(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6113,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{
								"key": "value",
							},
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

	// Extra field should NOT match deepEquals
	impResp, err := http.Post("http://localhost:6113/", "application/json",
		strings.NewReader(`{"key": "VALUE", "extra": "field"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("deepEquals should not match with extra fields")
	}
}

// TestJSONDeepEquals_ObjectArray tests deepEquals with array of objects
// mountebank jsonTest.js: "should be true if all objects in an array have fields equaling predicate"
func TestJSONDeepEquals_ObjectArray(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6114,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": []interface{}{
								map[string]interface{}{"key": "first"},
								map[string]interface{}{"key": "second"},
							},
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

	impResp, err := http.Post("http://localhost:6114/", "application/json",
		strings.NewReader(`[{ "key": "first" }, { "key": "second" }]`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for matching object array, got '%s'", string(body))
	}
}

// TestJSONDeepEquals_MissingInArray tests that deepEquals fails with extra object in array
// mountebank jsonTest.js: "should be false if missing an object in an array in request"
func TestJSONDeepEquals_MissingInArray(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6153,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": []interface{}{
								map[string]interface{}{"key": "first"},
								map[string]interface{}{"key": "second"},
							},
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

	// Extra object in array - should not match deepEquals
	impResp, err := http.Post("http://localhost:6153/", "application/json",
		strings.NewReader(`[{ "key": "first" }, { "different": true }, { "key": "second" }]`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("deepEquals should not match when array has extra object")
	}
}

// TestJSONDeepEquals_ObjectArrayOrderInsensitive tests object array matches regardless of order
// mountebank jsonTest.js: "should be true if all objects in an array have fields equaling predicate regardless of order"
func TestJSONDeepEquals_ObjectArrayOrderInsensitive(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6115,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": []interface{}{
								map[string]interface{}{"key": "second"},
								map[string]interface{}{"key": "first"},
							},
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

	// Different order should match
	impResp, err := http.Post("http://localhost:6115/", "application/json",
		strings.NewReader(`[{ "key": "first" }, { "key": "second" }]`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for object array with different order, got '%s'", string(body))
	}
}

// ============================================================================
// CONTAINS, STARTSWITH, ENDSWITH PREDICATES WITH JSON
// ============================================================================

// TestJSONContains tests contains predicate with JSON
// mountebank jsonTest.js: "should be true if JSON value contains predicate"
func TestJSONContains(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6120,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"body": map[string]interface{}{"key": "alu"},
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

	impResp, err := http.Post("http://localhost:6120/", "application/json",
		strings.NewReader(`{ "key": "VALUE" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for contains, got '%s'", string(body))
	}
}

// TestJSONContains_CaseSensitive tests contains with case-sensitive option
// mountebank jsonTest.js: "should be false if JSON value does not contain predicate"
func TestJSONContains_CaseSensitive(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6160,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"body": map[string]interface{}{"key": "VALUE"},
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

	// Case-sensitive should NOT match
	impResp, err := http.Post("http://localhost:6160/", "application/json",
		strings.NewReader(`{"key": "test"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match with case-sensitive contains when case differs")
	}
}

// TestJSONStartsWith tests startsWith predicate with JSON
// mountebank jsonTest.js: "should be true if JSON field starts with value"
func TestJSONStartsWith(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6121,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"startsWith": map[string]interface{}{
							"body": map[string]interface{}{"key": "Harry"},
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
	impResp1, err := http.Post("http://localhost:6121/", "application/json",
		strings.NewReader(`{"key": "Harry Potter"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' for startsWith, got '%s'", string(body1))
	}

	// Should NOT match
	impResp2, err := http.Post("http://localhost:6121/", "application/json",
		strings.NewReader(`{"key": "Ron Weasley"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match when value doesn't start with prefix")
	}
}

// TestJSONEndsWith tests endsWith predicate with JSON
// mountebank jsonTest.js: "should be true if JSON field ends with predicate"
func TestJSONEndsWith(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6122,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"endsWith": map[string]interface{}{
							"body": map[string]interface{}{"key": "Potter"},
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
	impResp1, err := http.Post("http://localhost:6122/", "application/json",
		strings.NewReader(`{"key": "Harry Potter"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' for endsWith, got '%s'", string(body1))
	}

	// Should NOT match
	impResp2, err := http.Post("http://localhost:6122/", "application/json",
		strings.NewReader(`{"key": "Harry"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match when value doesn't end with suffix")
	}
}

// ============================================================================
// MATCHES PREDICATE WITH JSON
// ============================================================================

// TestJSONMatches_NotJSON tests that matches fails when field is not valid JSON
// mountebank jsonTest.js: "should be false if field is not JSON"
func TestJSONMatches_NotJSON(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6170,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": map[string]interface{}{"key": "VALUE"},
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

	// Send invalid JSON
	impResp, err := http.Post("http://localhost:6170/", "text/plain",
		strings.NewReader(`"key": "value"`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match non-JSON against JSON predicate with matches")
	}
}

// TestJSONMatches tests matches predicate with JSON
// mountebank jsonTest.js: "should be true if selected JSON value matches regex"
func TestJSONMatches(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6130,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": map[string]interface{}{"key": "^v"},
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

	// Should match - value starts with v (case-insensitive)
	impResp1, err := http.Post("http://localhost:6130/", "application/json",
		strings.NewReader(`{"key": "Value"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' for regex match, got '%s'", string(body1))
	}

	// Should NOT match - v$ regex doesn't match
	impResp2, err := http.Post("http://localhost:6130/", "application/json",
		strings.NewReader(`{"key": "test"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match when regex doesn't match")
	}
}

// TestJSONMatches_CaseInsensitiveKey tests case-insensitive key matching in matches
// mountebank jsonTest.js: "should support case-insensitive key matching in JSON body"
func TestJSONMatches_CaseInsensitiveKey(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6131,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": map[string]interface{}{"KEY": "^Value"},
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

	// Predicate has KEY, request has Key (different case)
	impResp, err := http.Post("http://localhost:6131/", "application/json",
		strings.NewReader(`{ "Key": "Value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' with case-insensitive key, got '%s'", string(body))
	}
}

// TestJSONMatches_UppercaseKey tests uppercase key matching (issue #228)
// mountebank jsonTest.js: "should support upper case object key in JSON body (issue #228)"
func TestJSONMatches_UppercaseKey(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6171,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": map[string]interface{}{"Key": "^Value"},
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

	impResp, err := http.Post("http://localhost:6171/", "application/json",
		strings.NewReader(`{ "Key": "Value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' with uppercase key, got '%s'", string(body))
	}
}

// TestJSONMatches_CaseSensitiveKey tests case-sensitive key matching
// mountebank jsonTest.js: "should support case sensitive key matching in JSON body if case sensitive configured"
func TestJSONMatches_CaseSensitiveKey(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6172,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": map[string]interface{}{"KEY": "^Value"},
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

	// KEY (predicate) vs Key (request) with caseSensitive should NOT match
	impResp, err := http.Post("http://localhost:6172/", "application/json",
		strings.NewReader(`{ "Key": "Value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match with case-sensitive key when case differs")
	}
}

// ============================================================================
// DEEP EQUALS CASE SENSITIVITY
// ============================================================================

// TestJSONDeepEquals_CaseInsensitiveKey tests case-insensitive key matching in deepEquals
// mountebank jsonTest.js: "should support case-insensitive key in JSON body"
func TestJSONDeepEquals_CaseInsensitiveKey(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6173,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{"KEY": "Value"},
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

	// KEY (predicate) matches Key (request) case-insensitively
	impResp, err := http.Post("http://localhost:6173/", "application/json",
		strings.NewReader(`{ "Key": "value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' with case-insensitive key (deepEquals), got '%s'", string(body))
	}
}

// TestJSONDeepEquals_CaseSensitiveKey tests case-sensitive key matching in deepEquals
// mountebank jsonTest.js: "should support case-sensitive key in JSON body if case sensitive configured"
func TestJSONDeepEquals_CaseSensitiveKey(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6174,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"deepEquals": map[string]interface{}{
							"body": map[string]interface{}{"KEY": "Value"},
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

	// KEY (predicate) vs Key (request) with caseSensitive should NOT match
	impResp, err := http.Post("http://localhost:6174/", "application/json",
		strings.NewReader(`{ "Key": "value" }`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "matched" {
		t.Error("should not match with case-sensitive key (deepEquals) when case differs")
	}
}

// ============================================================================
// EXISTS PREDICATE WITH JSON
// ============================================================================

// TestJSONExists_KeyExists tests exists predicate with JSON key
// mountebank jsonTest.js: "should be true if JSON key exists"
func TestJSONExists_KeyExists(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6140,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body": map[string]interface{}{"key": true},
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

	// Key exists
	impResp1, err := http.Post("http://localhost:6140/", "application/json",
		strings.NewReader(`{"key":"exists"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' when key exists, got '%s'", string(body1))
	}

	// Key doesn't exist
	impResp2, err := http.Post("http://localhost:6140/", "application/json",
		strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not match when key doesn't exist")
	}
}

// TestJSONExists_EmptyArray tests that empty array counts as existing
// mountebank jsonTest.js: "should be true if JSON array key exists"
func TestJSONExists_EmptyArray(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6141,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body": map[string]interface{}{"key": true},
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

	// Key with empty array counts as existing
	impResp, err := http.Post("http://localhost:6141/", "application/json",
		strings.NewReader(`{"key": []}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for empty array (key exists), got '%s'", string(body))
	}
}

// TestJSONExists_NestedDotNotation tests exists with dot notation for nested JSON
func TestJSONExists_NestedDotNotation(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6150,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body.user.address.city": true,
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

	// Should match - nested path exists
	impResp, err := http.Post("http://localhost:6150/", "application/json",
		strings.NewReader(`{"user": {"address": {"city": "NYC"}}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for nested path exists, got '%s'", string(body))
	}

	// Should NOT match - intermediate path missing
	impResp2, err := http.Post("http://localhost:6150/", "application/json",
		strings.NewReader(`{"user": {}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Errorf("should NOT match when intermediate path is missing")
	}
}

// TestJSONExists_ArrayIndex tests exists with array index notation
func TestJSONExists_ArrayIndex(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6151,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body.items[0].name": true,
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

	// Should match - array element has name
	impResp, err := http.Post("http://localhost:6151/", "application/json",
		strings.NewReader(`{"items": [{"name": "item1"}]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for array index path exists, got '%s'", string(body))
	}

	// Should NOT match - empty array
	impResp2, err := http.Post("http://localhost:6151/", "application/json",
		strings.NewReader(`{"items": []}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Errorf("should NOT match when array is empty")
	}
}

// TestJSONExists_ArrayWildcard tests exists with array wildcard
func TestJSONExists_ArrayWildcard(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6152,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body.items[*].id": true,
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

	// Should match - at least one element has id
	impResp, err := http.Post("http://localhost:6152/", "application/json",
		strings.NewReader(`{"items": [{"id": 1}, {"id": 2}]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for wildcard path exists, got '%s'", string(body))
	}

	// Should NOT match - no element has id
	impResp2, err := http.Post("http://localhost:6152/", "application/json",
		strings.NewReader(`{"items": [{"name": "no-id"}]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Errorf("should NOT match when no element has the property")
	}
}

// TestJSONExists_NestedMapDeep tests deeply nested map specification
func TestJSONExists_NestedMapDeep(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6153,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body": map[string]interface{}{
								"user": map[string]interface{}{
									"address": map[string]interface{}{
										"city": true,
									},
								},
							},
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

	// Should match - deeply nested path exists
	impResp, err := http.Post("http://localhost:6153/", "application/json",
		strings.NewReader(`{"user": {"address": {"city": "NYC"}}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for deeply nested map, got '%s'", string(body))
	}

	// Should NOT match - intermediate missing
	impResp2, err := http.Post("http://localhost:6153/", "application/json",
		strings.NewReader(`{"user": {"name": "John"}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Errorf("should NOT match when nested path doesn't exist")
	}
}

// TestJSONExists_NullValue tests that null values count as existing
func TestJSONExists_NullValue(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6154,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body.value": true,
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

	// Key with null value counts as existing
	impResp, err := http.Post("http://localhost:6154/", "application/json",
		strings.NewReader(`{"value": null}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for null value (key exists), got '%s'", string(body))
	}
}

// TestJSONExists_MissingIntermediatePath tests exists: false with missing intermediate paths
func TestJSONExists_MissingIntermediatePath(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6155,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body.a.b.c": false,
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

	// Should match - path doesn't exist
	impResp, err := http.Post("http://localhost:6155/", "application/json",
		strings.NewReader(`{"a": {}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for exists:false with missing path, got '%s'", string(body))
	}

	// Should match - root object is empty
	impResp2, err := http.Post("http://localhost:6155/", "application/json",
		strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) != "matched" {
		t.Errorf("expected 'matched' for exists:false with empty object, got '%s'", string(body2))
	}
}

// TestJSONExists_NegativeArrayIndex tests negative array index
func TestJSONExists_NegativeArrayIndex(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     6156,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"body.items[-1].name": true,
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

	// Should match - last element has name
	impResp, err := http.Post("http://localhost:6156/", "application/json",
		strings.NewReader(`{"items": [{"id": 1}, {"name": "last"}]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' for negative index path exists, got '%s'", string(body))
	}

	// Should NOT match - last element doesn't have name
	impResp2, err := http.Post("http://localhost:6156/", "application/json",
		strings.NewReader(`{"items": [{"name": "first"}, {"id": 2}]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Errorf("should NOT match when last element doesn't have the property")
	}
}
