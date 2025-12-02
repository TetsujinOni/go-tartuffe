package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/api"
)

var (
	baseURL    = "http://localhost:2525"
	client     = &http.Client{Timeout: 5 * time.Second}
	testServer *api.Server
)

// TestMain sets up a single server for all tests
func TestMain(m *testing.M) {
	fmt.Println("Starting integration test server...")

	testServer = api.NewServer(api.ServerConfig{
		Port:           2525,
		AllowInjection: true,
		LocalOnly:      false,
		IPWhitelist:    "*",
	})

	go func() {
		if err := testServer.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("server error: %v\n", err)
		}
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Run tests
	code := m.Run()

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	testServer.Shutdown(ctx)

	os.Exit(code)
}

// Helper functions
func doRequest(method, path string, body interface{}) (*http.Response, map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if len(respBody) > 0 {
		json.Unmarshal(respBody, &result)
	}

	return resp, result, nil
}

func get(path string) (*http.Response, map[string]interface{}, error) {
	return doRequest("GET", path, nil)
}

func post(path string, body interface{}) (*http.Response, map[string]interface{}, error) {
	return doRequest("POST", path, body)
}

func put(path string, body interface{}) (*http.Response, map[string]interface{}, error) {
	return doRequest("PUT", path, body)
}

func del(path string) (*http.Response, map[string]interface{}, error) {
	return doRequest("DELETE", path, nil)
}

func cleanup(t *testing.T) {
	_, _, err := del("/imposters")
	if err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}

// Tests from impostersControllerTest.js

func TestPostImposters_CreateWithConsistentHypermedia(t *testing.T) {
	defer cleanup(t)

	resp, body, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     3000,
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	links := body["_links"].(map[string]interface{})
	selfLink := links["self"].(map[string]interface{})["href"].(string)

	if location != selfLink {
		t.Errorf("Location header %q != self link %q", location, selfLink)
	}

	// Verify GET returns same body
	_, getBody, _ := get(location)
	if getBody["port"] != body["port"] {
		t.Errorf("GET response differs from POST response")
	}
}

func TestPostImposters_Return400OnInvalidInput(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestPostImposters_Return400OnPortConflict(t *testing.T) {
	defer cleanup(t)

	// Try to create imposter on the API port itself
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     2525,
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestPostImposters_Return400OnInvalidJSON(t *testing.T) {
	defer cleanup(t)

	req, _ := http.NewRequest("POST", baseURL+"/imposters", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	errors := result["errors"].([]interface{})
	if len(errors) == 0 {
		t.Error("expected errors array")
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "invalid JSON" {
		t.Errorf("expected code 'invalid JSON', got %v", firstError["code"])
	}
}

func TestDeleteImposters_ReturnsEmptyArrayIfNoImposters(t *testing.T) {
	// Ensure clean state
	cleanup(t)

	resp, body, err := del("/imposters")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	imposters := body["imposters"].([]interface{})
	if len(imposters) != 0 {
		t.Errorf("expected empty imposters array, got %d items", len(imposters))
	}
}

func TestDeleteImposters_DeletesAllImposters(t *testing.T) {
	defer cleanup(t)

	// Create two imposters
	post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     3000,
		"name":     "imposter 1",
	})
	post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     3001,
		"name":     "imposter 2",
	})

	// Delete all
	resp, body, err := del("/imposters")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	imposters := body["imposters"].([]interface{})
	if len(imposters) != 2 {
		t.Errorf("expected 2 imposters in response, got %d", len(imposters))
	}

	// Verify they're deleted
	_, getBody, _ := get("/imposters")
	impostersAfter := getBody["imposters"].([]interface{})
	if len(impostersAfter) != 0 {
		t.Errorf("expected 0 imposters after delete, got %d", len(impostersAfter))
	}
}

func TestPutImposters_CreatesAllImposters(t *testing.T) {
	defer cleanup(t)

	resp, body, err := put("/imposters", map[string]interface{}{
		"imposters": []map[string]interface{}{
			{"protocol": "http", "port": 3000, "name": "imposter 1"},
			{"protocol": "http", "port": 3001, "name": "imposter 2"},
			{"protocol": "http", "port": 3002, "name": "imposter 3"},
		},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	imposters := body["imposters"].([]interface{})
	if len(imposters) != 3 {
		t.Errorf("expected 3 imposters, got %d", len(imposters))
	}
}

func TestPutImposters_OverwritesPreviousImposters(t *testing.T) {
	defer cleanup(t)

	// Create initial imposter
	post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     3000,
	})

	// Overwrite with PUT
	resp, body, err := put("/imposters", map[string]interface{}{
		"imposters": []map[string]interface{}{
			{"protocol": "http", "port": 4000, "name": "new imposter"},
		},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	imposters := body["imposters"].([]interface{})
	if len(imposters) != 1 {
		t.Errorf("expected 1 imposter, got %d", len(imposters))
	}

	// Verify old imposter is gone
	_, getBody, _ := get("/imposters")
	allImposters := getBody["imposters"].([]interface{})
	if len(allImposters) != 1 {
		t.Errorf("expected 1 imposter total, got %d", len(allImposters))
	}

	firstImposter := allImposters[0].(map[string]interface{})
	if firstImposter["port"].(float64) != 4000 {
		t.Errorf("expected port 4000, got %v", firstImposter["port"])
	}
}

// Tests from homeControllerTest.js

func TestHome_ReturnsHypermediaLinks(t *testing.T) {
	resp, body, err := get("/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	links := body["_links"].(map[string]interface{})

	impostersLink := links["imposters"].(map[string]interface{})["href"].(string)
	if impostersLink != "/imposters" {
		t.Errorf("expected /imposters, got %s", impostersLink)
	}

	configLink := links["config"].(map[string]interface{})["href"].(string)
	if configLink != "/config" {
		t.Errorf("expected /config, got %s", configLink)
	}

	logsLink := links["logs"].(map[string]interface{})["href"].(string)
	if logsLink != "/logs" {
		t.Errorf("expected /logs, got %s", logsLink)
	}
}

// Additional stub management tests

func TestStubs_ReplaceAllStubs(t *testing.T) {
	defer cleanup(t)

	// Create imposter
	post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     3000,
	})

	// Replace stubs
	resp, body, err := put("/imposters/3000/stubs", map[string]interface{}{
		"stubs": []map[string]interface{}{
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "hello"}}}},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "world"}}}},
		},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	stubs := body["stubs"].([]interface{})
	if len(stubs) != 2 {
		t.Errorf("expected 2 stubs, got %d", len(stubs))
	}
}

func TestStubs_AddStub(t *testing.T) {
	defer cleanup(t)

	// Create imposter with one stub
	post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     3000,
		"stubs": []map[string]interface{}{
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "first"}}}},
		},
	})

	// Add another stub
	resp, body, err := post("/imposters/3000/stubs", map[string]interface{}{
		"stub": map[string]interface{}{
			"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "second"}}},
		},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	stubs := body["stubs"].([]interface{})
	if len(stubs) != 2 {
		t.Errorf("expected 2 stubs, got %d", len(stubs))
	}
}

func TestStubs_DeleteStub(t *testing.T) {
	defer cleanup(t)

	// Create imposter with two stubs
	post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     3000,
		"stubs": []map[string]interface{}{
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "first"}}}},
			{"responses": []map[string]interface{}{{"is": map[string]interface{}{"body": "second"}}}},
		},
	})

	// Delete first stub
	resp, body, err := del("/imposters/3000/stubs/0")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	stubs := body["stubs"].([]interface{})
	if len(stubs) != 1 {
		t.Errorf("expected 1 stub, got %d", len(stubs))
	}
}

func TestConfig_ReturnsServerConfig(t *testing.T) {
	resp, body, err := get("/config")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if body["version"] == nil {
		t.Error("expected version field")
	}

	options := body["options"].(map[string]interface{})
	if options["port"].(float64) != 2525 {
		t.Errorf("expected port 2525, got %v", options["port"])
	}

	process := body["process"].(map[string]interface{})
	if process["goVersion"] == nil {
		t.Error("expected goVersion field")
	}
}

func TestLogs_ReturnsLogs(t *testing.T) {
	resp, body, err := get("/logs")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if body["logs"] == nil {
		t.Error("expected logs field")
	}
}

func TestGetImposter_Returns404ForNonExistent(t *testing.T) {
	resp, body, err := get("/imposters/9999")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	errors := body["errors"].([]interface{})
	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "no such resource" {
		t.Errorf("expected code 'no such resource', got %v", firstError["code"])
	}
}

func TestDeleteImposter_Returns200ForNonExistent(t *testing.T) {
	// DELETE is idempotent - should return 200 even for non-existent imposters
	resp, _, err := del("/imposters/9999")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// Imposter listener tests - verify imposters actually respond to HTTP requests

func TestImposterListener_RespondsToRequests(t *testing.T) {
	defer cleanup(t)

	// Create imposter with a stub
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4545,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Hello, World!", "statusCode": 200}},
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

	// Wait for imposter to start
	time.Sleep(100 * time.Millisecond)

	// Make request to imposter
	imposterResp, err := http.Get("http://localhost:4545/")
	if err != nil {
		t.Fatalf("failed to make request to imposter: %v", err)
	}
	defer imposterResp.Body.Close()

	if imposterResp.StatusCode != 200 {
		t.Errorf("expected imposter status 200, got %d", imposterResp.StatusCode)
	}

	body, _ := io.ReadAll(imposterResp.Body)
	if string(body) != "Hello, World!" {
		t.Errorf("expected body 'Hello, World!', got '%s'", string(body))
	}
}

func TestImposterListener_MatchesPredicates(t *testing.T) {
	defer cleanup(t)

	// Create imposter with predicates
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4546,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/api/users"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "users response", "statusCode": 200}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/api/orders"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "orders response", "statusCode": 200}},
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

	// Test /api/users
	usersResp, err := http.Get("http://localhost:4546/api/users")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer usersResp.Body.Close()

	usersBody, _ := io.ReadAll(usersResp.Body)
	if string(usersBody) != "users response" {
		t.Errorf("expected 'users response', got '%s'", string(usersBody))
	}

	// Test /api/orders
	ordersResp, err := http.Get("http://localhost:4546/api/orders")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer ordersResp.Body.Close()

	ordersBody, _ := io.ReadAll(ordersResp.Body)
	if string(ordersBody) != "orders response" {
		t.Errorf("expected 'orders response', got '%s'", string(ordersBody))
	}
}

func TestImposterListener_ReturnsHeaders(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4547,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{
						"statusCode": 201,
						"headers":    map[string]interface{}{"X-Custom-Header": "custom-value"},
						"body":       "created",
					}},
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

	imposterResp, err := http.Get("http://localhost:4547/")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer imposterResp.Body.Close()

	if imposterResp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", imposterResp.StatusCode)
	}

	if imposterResp.Header.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header 'custom-value', got '%s'", imposterResp.Header.Get("X-Custom-Header"))
	}
}

func TestImposterListener_CyclesThroughResponses(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4548,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "first"}},
					{"is": map[string]interface{}{"body": "second"}},
					{"is": map[string]interface{}{"body": "third"}},
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

	expected := []string{"first", "second", "third", "first", "second"}
	for i, exp := range expected {
		impResp, err := http.Get("http://localhost:4548/")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		body, _ := io.ReadAll(impResp.Body)
		impResp.Body.Close()

		if string(body) != exp {
			t.Errorf("request %d: expected '%s', got '%s'", i, exp, string(body))
		}
	}
}

func TestImposterListener_MatchesMethod(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4549,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"method": "POST"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "POST response"}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"method": "GET"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "GET response"}},
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

	// Test GET
	getResp, err := http.Get("http://localhost:4549/")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	getBody, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()

	if string(getBody) != "GET response" {
		t.Errorf("expected 'GET response', got '%s'", string(getBody))
	}

	// Test POST
	postResp, err := http.Post("http://localhost:4549/", "application/json", nil)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	postBody, _ := io.ReadAll(postResp.Body)
	postResp.Body.Close()

	if string(postBody) != "POST response" {
		t.Errorf("expected 'POST response', got '%s'", string(postBody))
	}
}

func TestImposterListener_ContainsPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4550,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"contains": map[string]interface{}{"path": "users"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "contains users"}},
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

	// Should match paths containing "users"
	impResp, err := http.Get("http://localhost:4550/api/users/123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "contains users" {
		t.Errorf("expected 'contains users', got '%s'", string(body))
	}
}

func TestImposterListener_StopsOnDelete(t *testing.T) {
	defer cleanup(t)

	// Create imposter
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4551,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "test"}},
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

	// Verify it works
	impResp, err := http.Get("http://localhost:4551/")
	if err != nil {
		t.Fatalf("request to imposter failed: %v", err)
	}
	impResp.Body.Close()

	// Delete the imposter
	del("/imposters/4551")

	time.Sleep(100 * time.Millisecond)

	// Verify it's stopped - request should fail
	_, err = http.Get("http://localhost:4551/")
	if err == nil {
		t.Error("expected request to fail after imposter deleted")
	}
}

func TestImposterListener_DefaultResponse(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     4552,
		"defaultResponse": map[string]interface{}{
			"is": map[string]interface{}{
				"statusCode": 404,
				"body":       "not found",
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

	// Request should get default response since no stubs match
	impResp, err := http.Get("http://localhost:4552/any/path")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer impResp.Body.Close()

	if impResp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", impResp.StatusCode)
	}

	body, _ := io.ReadAll(impResp.Body)
	if string(body) != "not found" {
		t.Errorf("expected 'not found', got '%s'", string(body))
	}
}
