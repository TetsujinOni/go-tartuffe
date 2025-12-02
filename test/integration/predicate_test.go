package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Predicate tests converted from mountebank tests

func TestPredicate_Equals(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5200,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/test"}},
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

	// Should match
	resp1, err := http.Get("http://localhost:5200/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match
	resp2, err := http.Get("http://localhost:5200/other")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched /other path")
	}
}

func TestPredicate_Contains(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5201,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"contains": map[string]interface{}{"path": "users"}},
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

	// Should match - contains "users"
	resp1, err := http.Get("http://localhost:5201/api/users/123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match
	resp2, err := http.Get("http://localhost:5201/api/orders")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched path without 'users'")
	}
}

func TestPredicate_StartsWith(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5202,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"startsWith": map[string]interface{}{"path": "/api"}},
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

	// Should match
	resp1, err := http.Get("http://localhost:5202/api/users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match
	resp2, err := http.Get("http://localhost:5202/v1/api")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched path not starting with '/api'")
	}
}

func TestPredicate_EndsWith(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5203,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"endsWith": map[string]interface{}{"path": ".json"}},
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

	// Should match
	resp1, err := http.Get("http://localhost:5203/data.json")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match
	resp2, err := http.Get("http://localhost:5203/data.xml")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched path not ending with '.json'")
	}
}

func TestPredicate_Matches(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5204,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"matches": map[string]interface{}{"path": "^/users/\\d+$"}},
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

	// Should match
	resp1, err := http.Get("http://localhost:5204/users/123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match - has non-digit
	resp2, err := http.Get("http://localhost:5204/users/abc")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched path with non-digit")
	}
}

func TestPredicate_Exists(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5205,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"exists": map[string]interface{}{"headers.X-Custom": true}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "header exists"}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"exists": map[string]interface{}{"headers.X-Custom": false}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "header missing"}},
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

	// With header
	req1, _ := http.NewRequest("GET", "http://localhost:5205/", nil)
	req1.Header.Set("X-Custom", "value")
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "header exists" {
		t.Errorf("expected 'header exists', got '%s'", string(body1))
	}

	// Without header
	resp2, err := http.Get("http://localhost:5205/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "header missing" {
		t.Errorf("expected 'header missing', got '%s'", string(body2))
	}
}

func TestPredicate_And(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5206,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"and": []map[string]interface{}{
							{"equals": map[string]interface{}{"method": "POST"}},
							{"startsWith": map[string]interface{}{"path": "/api"}},
						},
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

	// Should match - POST to /api
	resp1, err := http.Post("http://localhost:5206/api/users", "text/plain", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match - GET to /api
	resp2, err := http.Get("http://localhost:5206/api/users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched GET request")
	}

	// Should not match - POST to /other
	resp3, err := http.Post("http://localhost:5206/other", "text/plain", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if string(body3) == "matched" {
		t.Error("should not have matched POST to wrong path")
	}
}

func TestPredicate_Or(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5207,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"or": []map[string]interface{}{
							{"equals": map[string]interface{}{"path": "/one"}},
							{"equals": map[string]interface{}{"path": "/two"}},
						},
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

	// Should match /one
	resp1, err := http.Get("http://localhost:5207/one")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched' for /one, got '%s'", string(body1))
	}

	// Should match /two
	resp2, err := http.Get("http://localhost:5207/two")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "matched" {
		t.Errorf("expected 'matched' for /two, got '%s'", string(body2))
	}

	// Should not match /three
	resp3, err := http.Get("http://localhost:5207/three")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if string(body3) == "matched" {
		t.Error("should not have matched /three")
	}
}

func TestPredicate_Not(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5208,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"not": map[string]interface{}{
							"equals": map[string]interface{}{"path": "/excluded"},
						},
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

	// Should match - not /excluded
	resp1, err := http.Get("http://localhost:5208/anything")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match - /excluded
	resp2, err := http.Get("http://localhost:5208/excluded")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched /excluded")
	}
}

func TestPredicate_CaseSensitive(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5209,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals":        map[string]interface{}{"body": "TEST"},
						"caseSensitive": true,
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "case sensitive match"}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"body": "TEST"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "case insensitive match"}},
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

	// Exact case - should match case sensitive
	resp1, err := http.Post("http://localhost:5209/", "text/plain", strings.NewReader("TEST"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "case sensitive match" {
		t.Errorf("expected 'case sensitive match', got '%s'", string(body1))
	}

	// Different case - should match case insensitive
	resp2, err := http.Post("http://localhost:5209/", "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "case insensitive match" {
		t.Errorf("expected 'case insensitive match', got '%s'", string(body2))
	}
}

func TestPredicate_MultipleFieldsInEquals(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5210,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"method": "POST",
							"path":   "/api/users",
						},
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

	// Should match - POST to /api/users
	resp1, err := http.Post("http://localhost:5210/api/users", "text/plain", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "matched" {
		t.Errorf("expected 'matched', got '%s'", string(body1))
	}

	// Should not match - GET to /api/users
	resp2, err := http.Get("http://localhost:5210/api/users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "matched" {
		t.Error("should not have matched GET")
	}

	// Should not match - POST to /api/orders
	resp3, err := http.Post("http://localhost:5210/api/orders", "text/plain", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()

	if string(body3) == "matched" {
		t.Error("should not have matched wrong path")
	}
}

func TestPredicate_BodyMatching(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5211,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"contains": map[string]interface{}{"body": "search-term"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "found it"}},
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

	// Should match
	resp1, err := http.Post("http://localhost:5211/", "text/plain", strings.NewReader("this contains search-term in it"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "found it" {
		t.Errorf("expected 'found it', got '%s'", string(body1))
	}

	// Should not match
	resp2, err := http.Post("http://localhost:5211/", "text/plain", strings.NewReader("no matching content"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) == "found it" {
		t.Error("should not have matched")
	}
}
