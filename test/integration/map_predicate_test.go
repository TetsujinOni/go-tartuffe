package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Tests for predicates operating on map fields (headers, query) with map patterns
// These test scenarios were identified as gaps in test coverage for the same pattern
// that was fixed in matchesPattern (commit 4ca6ae9)

// ============================================================================
// CONTAINS PREDICATE ON HEADERS (map[string]string actual, map[string]interface{} expected)
// ============================================================================

// TestContains_HeadersWithMapPattern tests contains predicate on headers field
func TestContains_HeadersWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7100,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"headers": map[string]interface{}{
								"Content-Type": "json",
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

	// Should match - Content-Type header contains "json"
	req, _ := http.NewRequest("POST", "http://localhost:7100/", strings.NewReader("test"))
	req.Header.Set("Content-Type", "application/json")
	impResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - Content-Type does not contain "json"
	req2, _ := http.NewRequest("POST", "http://localhost:7100/", strings.NewReader("test"))
	req2.Header.Set("Content-Type", "text/plain")
	impResp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - Content-Type does not contain 'json'")
	}
}

// TestContains_QueryWithMapPattern tests contains predicate on query field
func TestContains_QueryWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7101,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"query": map[string]interface{}{
								"search": "test",
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

	// Should match - search query contains "test"
	impResp, err := http.Get("http://localhost:7101/?search=testing123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - search query does not contain "test"
	impResp2, err := http.Get("http://localhost:7101/?search=hello")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - query does not contain 'test'")
	}
}

// ============================================================================
// STARTSWITH PREDICATE ON HEADERS/QUERY
// ============================================================================

// TestStartsWith_HeadersWithMapPattern tests startsWith predicate on headers field
func TestStartsWith_HeadersWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7102,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"startsWith": map[string]interface{}{
							"headers": map[string]interface{}{
								"Content-Type": "application/",
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

	// Should match - Content-Type starts with "application/"
	req, _ := http.NewRequest("POST", "http://localhost:7102/", strings.NewReader("test"))
	req.Header.Set("Content-Type", "application/json")
	impResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - Content-Type does not start with "application/"
	req2, _ := http.NewRequest("POST", "http://localhost:7102/", strings.NewReader("test"))
	req2.Header.Set("Content-Type", "text/plain")
	impResp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - Content-Type does not start with 'application/'")
	}
}

// TestStartsWith_QueryWithMapPattern tests startsWith predicate on query field
func TestStartsWith_QueryWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7103,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"startsWith": map[string]interface{}{
							"query": map[string]interface{}{
								"filter": "active_",
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

	// Should match - filter starts with "active_"
	impResp, err := http.Get("http://localhost:7103/?filter=active_users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - filter does not start with "active_"
	impResp2, err := http.Get("http://localhost:7103/?filter=inactive_users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - filter does not start with 'active_'")
	}
}

// ============================================================================
// ENDSWITH PREDICATE ON HEADERS/QUERY
// ============================================================================

// TestEndsWith_HeadersWithMapPattern tests endsWith predicate on headers field
func TestEndsWith_HeadersWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7104,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"endsWith": map[string]interface{}{
							"headers": map[string]interface{}{
								"Accept": "json",
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

	// Should match - Accept header ends with "json"
	req, _ := http.NewRequest("GET", "http://localhost:7104/", nil)
	req.Header.Set("Accept", "application/json")
	impResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - Accept does not end with "json"
	req2, _ := http.NewRequest("GET", "http://localhost:7104/", nil)
	req2.Header.Set("Accept", "text/xml")
	impResp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - Accept does not end with 'json'")
	}
}

// TestEndsWith_QueryWithMapPattern tests endsWith predicate on query field
func TestEndsWith_QueryWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7105,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"endsWith": map[string]interface{}{
							"query": map[string]interface{}{
								"filename": ".pdf",
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

	// Should match - filename ends with ".pdf"
	impResp, err := http.Get("http://localhost:7105/?filename=document.pdf")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - filename does not end with ".pdf"
	impResp2, err := http.Get("http://localhost:7105/?filename=document.txt")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - filename does not end with '.pdf'")
	}
}

// ============================================================================
// MATCHES PREDICATE ON HEADERS/QUERY (should already work after patch 4ca6ae9)
// ============================================================================

// TestMatches_HeadersWithMapPattern tests matches predicate on headers field
// This should work after the fix in commit 4ca6ae9
func TestMatches_HeadersWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7106,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"headers": map[string]interface{}{
								"Authorization": "^Bearer ",
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

	// Should match - Authorization header matches "^Bearer "
	req, _ := http.NewRequest("GET", "http://localhost:7106/", nil)
	req.Header.Set("Authorization", "Bearer abc123")
	impResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - Authorization does not match pattern
	req2, _ := http.NewRequest("GET", "http://localhost:7106/", nil)
	req2.Header.Set("Authorization", "Basic abc123")
	impResp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - Authorization does not match '^Bearer '")
	}
}

// TestMatches_QueryWithMapPattern tests matches predicate on query field
func TestMatches_QueryWithMapPattern(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7107,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"query": map[string]interface{}{
								"id": "^\\d+$",
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

	// Should match - id is all digits
	impResp, err := http.Get("http://localhost:7107/?id=12345")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - id contains non-digits
	impResp2, err := http.Get("http://localhost:7107/?id=abc123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - id is not all digits")
	}
}

// ============================================================================
// CASE SENSITIVITY TESTS FOR MAP PREDICATES
// ============================================================================

// TestContains_HeadersWithMapPattern_CaseSensitive tests case-sensitive contains on headers
func TestContains_HeadersWithMapPattern_CaseSensitive(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7108,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"headers": map[string]interface{}{
								"X-Custom": "TEST",
							},
						},
						"caseSensitive": true,
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

	// Should match - exact case
	req, _ := http.NewRequest("GET", "http://localhost:7108/", nil)
	req.Header.Set("X-Custom", "THIS IS A TEST VALUE")
	impResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body))
	}

	// Should NOT match - wrong case
	req2, _ := http.NewRequest("GET", "http://localhost:7108/", nil)
	req2.Header.Set("X-Custom", "this is a test value")
	impResp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched - case does not match with caseSensitive: true")
	}
}

// TestStartsWith_QueryWithMapPattern_CaseInsensitive tests default case-insensitive startsWith
func TestStartsWith_QueryWithMapPattern_CaseInsensitive(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     7109,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"startsWith": map[string]interface{}{
							"query": map[string]interface{}{
								"status": "ACTIVE",
							},
						},
						// Default: caseSensitive is false
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

	// Should match - case-insensitive "active" starts with "ACTIVE"
	impResp, err := http.Get("http://localhost:7109/?status=active_users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "matched" {
		t.Errorf("expected 'matched' (case-insensitive), got '%s'", string(body))
	}
}
