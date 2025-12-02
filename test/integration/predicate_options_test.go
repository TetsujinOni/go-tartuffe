package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestPredicateOption_Except tests the except option that strips regex patterns
func TestPredicateOption_Except(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5900,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "This is a test",
						},
						"except": "\\d+", // Strip all digits
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched with except"}},
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

	// Request body has digits that should be stripped
	impResp, err := http.Post("http://localhost:5900/test", "text/plain",
		strings.NewReader("1This is 3a 2test"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched with except" {
		t.Errorf("expected 'matched with except', got '%s'", string(body))
	}
}

// TestPredicateOption_Except_CaseInsensitive tests except with case-insensitive matching
func TestPredicateOption_Except_CaseInsensitive(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5901,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "is is a test",
						},
						"except": "^tH", // Should match 'Th' case-insensitively
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	impResp, err := http.Post("http://localhost:5901/test", "text/plain",
		strings.NewReader("This is a test"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestPredicateOption_Except_CaseSensitive tests except with case-sensitive matching
func TestPredicateOption_Except_CaseSensitive(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5902,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "This is a test", // Pattern won't match because of case
						},
						"except":        "^t",   // Lowercase 't', won't match 'T'
						"caseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// Should match because except pattern doesn't match (case-sensitive)
	impResp, err := http.Post("http://localhost:5902/test", "text/plain",
		strings.NewReader("This is a test"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestPredicateOption_Except_Contains tests except with contains predicate
func TestPredicateOption_Except_Contains(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5903,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"body": "hello world",
						},
						"except": " \\d{4}-\\d{2}-\\d{2}", // Strip date patterns
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// Body contains hello world with a date that gets stripped
	impResp, err := http.Post("http://localhost:5903/test", "text/plain",
		strings.NewReader("hello 2024-01-15 world"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestPredicateOption_Except_Matches tests except with matches predicate
func TestPredicateOption_Except_Matches(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5904,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": "^\\d{1,10}$", // Only digits
						},
						"except": "\\D", // Strip all non-digits
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// '1+2' becomes '12' after stripping non-digits
	impResp, err := http.Post("http://localhost:5904/test", "text/plain",
		strings.NewReader("1+2"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestPredicateOption_KeyCaseSensitive_Query tests keyCaseSensitive for query parameters
func TestPredicateOption_KeyCaseSensitive_Query(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5905,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"query": map[string]interface{}{
								"MyParam": "value",
							},
						},
						"keyCaseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "exact key match"}},
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

	// Test with exact case
	impResp, err := http.Get("http://localhost:5905/test?MyParam=value")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "exact key match" {
		t.Errorf("expected 'exact key match', got '%s'", string(body))
	}
}

// TestPredicateOption_KeyCaseSensitive_Query_Mismatch tests keyCaseSensitive blocks wrong case
func TestPredicateOption_KeyCaseSensitive_Query_Mismatch(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5906,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"query": map[string]interface{}{
								"MyParam": "value",
							},
						},
						"keyCaseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// Test with different case - should NOT match
	impResp, err := http.Get("http://localhost:5906/test?myparam=value")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	// Should get default response (empty body), not the stub's response
	if string(body) == "matched" {
		t.Error("should not have matched with wrong case key when keyCaseSensitive is true")
	}
}

// TestPredicateOption_KeyCaseSensitive_Query_Default tests default case-insensitive key matching
func TestPredicateOption_KeyCaseSensitive_Query_Default(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5907,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"query": map[string]interface{}{
								"MyParam": "value",
							},
						},
						// Default: keyCaseSensitive is false
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// Test with different case - should match with default keyCaseSensitive: false
	impResp, err := http.Get("http://localhost:5907/test?myparam=value")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestPredicateOption_Exists_KeyCaseSensitive tests exists predicate with keyCaseSensitive
func TestPredicateOption_Exists_KeyCaseSensitive(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5908,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"query.MyParam": true,
						},
						"keyCaseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "param exists"}},
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

	// Test with exact case - should find param
	impResp, err := http.Get("http://localhost:5908/test?MyParam=value")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "param exists" {
		t.Errorf("expected 'param exists', got '%s'", string(body))
	}
}

// TestPredicateOption_Exists_KeyCaseSensitive_Mismatch tests exists with wrong case
func TestPredicateOption_Exists_KeyCaseSensitive_Mismatch(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5909,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"exists": map[string]interface{}{
							"query.MyParam": true,
						},
						"keyCaseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "param exists"}},
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

	// Test with wrong case - should NOT find param
	impResp, err := http.Get("http://localhost:5909/test?myparam=value")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) == "param exists" {
		t.Error("should not have found param with wrong case when keyCaseSensitive is true")
	}
}

// TestPredicateOption_Except_JSONPath tests except with JSONPath selector
func TestPredicateOption_Except_JSONPath(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5910,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "VE",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.key",
						},
						"except":        "ALU",
						"caseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// JSONPath extracts "VALUE", except strips "ALU", leaves "VE"
	impResp, err := http.Post("http://localhost:5910/test", "application/json",
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

// TestPredicateOption_StartsWith_Except tests except with startsWith predicate
func TestPredicateOption_StartsWith_Except(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5911,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"startsWith": map[string]interface{}{
							"body": "Hello",
						},
						"except": "^\\[\\d+\\] ", // Strip [123] prefix
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// "[123] Hello World" -> "Hello World" after except
	impResp, err := http.Post("http://localhost:5911/test", "text/plain",
		strings.NewReader("[123] Hello World"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}

// TestPredicateOption_EndsWith_Except tests except with endsWith predicate
func TestPredicateOption_EndsWith_Except(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5912,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"endsWith": map[string]interface{}{
							"body": "World",
						},
						"except": "!+$", // Strip trailing exclamation marks
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "matched"}},
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

	// "Hello World!!!" -> "Hello World" after except
	impResp, err := http.Post("http://localhost:5912/test", "text/plain",
		strings.NewReader("Hello World!!!"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}
}
