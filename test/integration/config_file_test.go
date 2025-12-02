package integration

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/config"
)

// Tests converted from mbTest/cli/configFileTest.js

func TestConfigFile_LoadSimpleJSON(t *testing.T) {
	defer cleanup(t)

	// Get the fixtures path
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "simple.json")

	// Load the config
	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Imposters) != 1 {
		t.Fatalf("expected 1 imposter, got %d", len(cfg.Imposters))
	}

	imp := cfg.Imposters[0]
	if imp.Port != 4545 {
		t.Errorf("expected port 4545, got %d", imp.Port)
	}
	if imp.Protocol != "http" {
		t.Errorf("expected protocol 'http', got '%s'", imp.Protocol)
	}
	if imp.Name != "simple service" {
		t.Errorf("expected name 'simple service', got '%s'", imp.Name)
	}
}

func TestConfigFile_LoadEJSWithIncludes(t *testing.T) {
	defer cleanup(t)

	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "imposters.ejs")

	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Imposters) != 2 {
		t.Fatalf("expected 2 imposters, got %d", len(cfg.Imposters))
	}

	// Check orders service
	if cfg.Imposters[0].Port != 4546 {
		t.Errorf("expected first imposter port 4546, got %d", cfg.Imposters[0].Port)
	}
	if cfg.Imposters[0].Name != "order service" {
		t.Errorf("expected first imposter name 'order service', got '%s'", cfg.Imposters[0].Name)
	}

	// Check users service
	if cfg.Imposters[1].Port != 4547 {
		t.Errorf("expected second imposter port 4547, got %d", cfg.Imposters[1].Port)
	}
	if cfg.Imposters[1].Name != "user service" {
		t.Errorf("expected second imposter name 'user service', got '%s'", cfg.Imposters[1].Name)
	}
}

func TestConfigFile_LoadEJSWithStringify(t *testing.T) {
	defer cleanup(t)

	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "stringify-test.ejs")

	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Imposters) != 1 {
		t.Fatalf("expected 1 imposter, got %d", len(cfg.Imposters))
	}

	imp := cfg.Imposters[0]
	if len(imp.Stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(imp.Stubs))
	}

	// Check the body contains the file content (JSON-escaped)
	stub := imp.Stubs[0]
	if len(stub.Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(stub.Responses))
	}

	body, ok := stub.Responses[0].Is.Body.(string)
	if !ok {
		t.Fatalf("expected body to be string, got %T", stub.Responses[0].Is.Body)
	}

	// The content should contain the embedded file text
	expectedContent := "This is embedded content\nwith multiple lines\nand \"special\" characters"
	if body != expectedContent {
		t.Errorf("expected body '%s', got '%s'", expectedContent, body)
	}
}

func TestConfigFile_NoParse(t *testing.T) {
	defer cleanup(t)

	wd, _ := os.Getwd()
	// Create a file with EJS tags that should NOT be parsed
	testFile := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "noparse-test.json")
	content := `{
  "imposters": [
    {
      "port": 4549,
      "protocol": "http",
      "stubs": [
        {
          "responses": [
            {
              "is": {
                "body": "<%- this should not be parsed %>"
              }
            }
          ]
        }
      ]
    }
  ]
}`
	os.WriteFile(testFile, []byte(content), 0644)
	defer os.Remove(testFile)

	// Load with noParse=true
	cfg, err := config.LoadFile(testFile, true)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// The EJS tags should be preserved literally
	body, ok := cfg.Imposters[0].Stubs[0].Responses[0].Is.Body.(string)
	if !ok {
		t.Fatalf("expected body to be string")
	}

	if body != "<%- this should not be parsed %>" {
		t.Errorf("expected EJS tags to be preserved, got '%s'", body)
	}
}

func TestConfigFile_LoadAndStartImposters(t *testing.T) {
	defer cleanup(t)

	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "simple.json")

	// Load the config
	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Load imposters via API (simulating what main.go does)
	if err := testServer.LoadImposters(cfg.Imposters); err != nil {
		t.Fatalf("failed to load imposters: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Test the imposter is running
	resp, err := http.Get("http://localhost:4545/")
	if err != nil {
		t.Fatalf("failed to make request to imposter: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello from simple service" {
		t.Errorf("expected 'Hello from simple service', got '%s'", string(body))
	}
}

func TestConfigFile_LoadEJSAndStartImposters(t *testing.T) {
	defer cleanup(t)

	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "imposters.ejs")

	// Load the config
	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Load imposters
	if err := testServer.LoadImposters(cfg.Imposters); err != nil {
		t.Fatalf("failed to load imposters: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Test order service - POST to create order
	resp1, err := http.Post("http://localhost:4546/orders", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to make POST request: %v", err)
	}
	resp1.Body.Close()

	if resp1.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp1.StatusCode)
	}
	if resp1.Header.Get("Location") != "http://localhost:4546/orders/123" {
		t.Errorf("expected Location header, got '%s'", resp1.Header.Get("Location"))
	}

	// Test order service - GET order
	resp2, err := http.Get("http://localhost:4546/orders/123")
	if err != nil {
		t.Fatalf("failed to make GET request: %v", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	if string(body2) != "Order 123" {
		t.Errorf("expected 'Order 123', got '%s'", string(body2))
	}

	// Test users service
	resp3, err := http.Get("http://localhost:4547/users")
	if err != nil {
		t.Fatalf("failed to make GET request: %v", err)
	}
	defer resp3.Body.Close()

	body3, _ := io.ReadAll(resp3.Body)
	expectedUsers := `{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`
	if string(body3) != expectedUsers {
		t.Errorf("expected '%s', got '%s'", expectedUsers, string(body3))
	}
}

func TestConfigFile_LoadEJSWithStringifyDataInjection(t *testing.T) {
	defer cleanup(t)

	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "datatest", "imposter.ejs")

	// Load the config
	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Imposters) != 1 {
		t.Fatalf("expected 1 imposter, got %d", len(cfg.Imposters))
	}

	imp := cfg.Imposters[0]
	if len(imp.Stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(imp.Stubs))
	}

	if len(imp.Stubs[0].Responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(imp.Stubs[0].Responses))
	}

	// First response should have injected-value-1
	body1, ok := imp.Stubs[0].Responses[0].Is.Body.(string)
	if !ok {
		t.Fatalf("expected body to be string, got %T", imp.Stubs[0].Responses[0].Is.Body)
	}
	expectedBody1 := `{
  "success": true,
  "injectedValue": "injected-value-1"
}`
	if strings.TrimSpace(body1) != strings.TrimSpace(expectedBody1) {
		t.Errorf("expected body1:\n%s\ngot:\n%s", expectedBody1, body1)
	}

	// Second response should have injected-value-2
	body2, ok := imp.Stubs[0].Responses[1].Is.Body.(string)
	if !ok {
		t.Fatalf("expected body to be string, got %T", imp.Stubs[0].Responses[1].Is.Body)
	}
	expectedBody2 := `{
  "success": true,
  "injectedValue": "injected-value-2"
}`
	if strings.TrimSpace(body2) != strings.TrimSpace(expectedBody2) {
		t.Errorf("expected body2:\n%s\ngot:\n%s", expectedBody2, body2)
	}
}

func TestConfigFile_LoadEJSWithNestedStringify(t *testing.T) {
	defer cleanup(t)

	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "nestedtest", "imposter.ejs")

	// Load the config
	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Imposters) != 1 {
		t.Fatalf("expected 1 imposter, got %d", len(cfg.Imposters))
	}

	imp := cfg.Imposters[0]
	if imp.Port != 4570 {
		t.Errorf("expected port 4570, got %d", imp.Port)
	}

	if len(imp.Stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(imp.Stubs))
	}

	if len(imp.Stubs[0].Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(imp.Stubs[0].Responses))
	}

	// The body should contain nested JSON that was stringified twice
	body, ok := imp.Stubs[0].Responses[0].Is.Body.(string)
	if !ok {
		t.Fatalf("expected body to be string, got %T", imp.Stubs[0].Responses[0].Is.Body)
	}

	// The body should be valid JSON with nested content
	if !strings.Contains(body, `"nested": true`) {
		t.Errorf("expected body to contain nested: true, got: %s", body)
	}
	if !strings.Contains(body, `"content":`) {
		t.Errorf("expected body to contain content field, got: %s", body)
	}
	// The deeply nested content should be escaped
	if !strings.Contains(body, `deeply`) {
		t.Errorf("expected body to contain deeply nested content, got: %s", body)
	}
}
