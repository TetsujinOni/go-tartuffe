package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/config"
)

// Tests for CLI commands (save, replay, stop)

func TestCLI_SaveCommand(t *testing.T) {
	defer cleanup(t)

	// Create an imposter via API
	imposter := map[string]interface{}{
		"port":     4580,
		"protocol": "http",
		"name":     "cli save test",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "test response"}},
				},
			},
		},
	}

	body, _ := json.Marshal(imposter)
	resp, err := http.Post(baseURL+"/imposters", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// Use save endpoint to get replayable config
	saveResp, err := http.Get(baseURL + "/imposters?replayable=true")
	if err != nil {
		t.Fatalf("failed to get imposters: %v", err)
	}
	defer saveResp.Body.Close()

	saveBody, _ := io.ReadAll(saveResp.Body)

	var saved map[string]interface{}
	if err := json.Unmarshal(saveBody, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	// Verify the saved config contains our imposter
	imposters, ok := saved["imposters"].([]interface{})
	if !ok {
		t.Fatalf("expected imposters array")
	}

	if len(imposters) != 1 {
		t.Fatalf("expected 1 imposter, got %d", len(imposters))
	}

	imp := imposters[0].(map[string]interface{})
	if imp["name"] != "cli save test" {
		t.Errorf("expected name 'cli save test', got %v", imp["name"])
	}
}

func TestCLI_SaveWithRemoveProxies(t *testing.T) {
	defer cleanup(t)

	// Create an imposter with proxy stub
	imposter := map[string]interface{}{
		"port":     4581,
		"protocol": "http",
		"name":     "proxy test",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"proxy": map[string]interface{}{"to": "http://example.com"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "non-proxy response"}},
				},
			},
		},
	}

	body, _ := json.Marshal(imposter)
	resp, err := http.Post(baseURL+"/imposters", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	resp.Body.Close()

	// Get with removeProxies
	saveResp, err := http.Get(baseURL + "/imposters?replayable=true&removeProxies=true")
	if err != nil {
		t.Fatalf("failed to get imposters: %v", err)
	}
	defer saveResp.Body.Close()

	saveBody, _ := io.ReadAll(saveResp.Body)

	var saved map[string]interface{}
	if err := json.Unmarshal(saveBody, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	imposters := saved["imposters"].([]interface{})
	imp := imposters[0].(map[string]interface{})
	stubs := imp["stubs"].([]interface{})

	// Should only have 1 stub (the non-proxy one)
	if len(stubs) != 1 {
		t.Errorf("expected 1 stub after removeProxies, got %d", len(stubs))
	}
}

func TestCLI_ReplayEndpoint(t *testing.T) {
	defer cleanup(t)

	// Create an imposter with proxy stub
	imposter := map[string]interface{}{
		"port":     4582,
		"protocol": "http",
		"name":     "replay test",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"proxy": map[string]interface{}{"to": "http://example.com"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "static response"}},
				},
			},
		},
	}

	body, _ := json.Marshal(imposter)
	resp, err := http.Post(baseURL+"/imposters", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	resp.Body.Close()

	// Get imposters without proxies (simulating replay)
	getResp, err := http.Get(baseURL + "/imposters?replayable=true&removeProxies=true")
	if err != nil {
		t.Fatalf("failed to get imposters: %v", err)
	}
	getBody, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()

	// PUT back (replay)
	client := &http.Client{}
	putReq, _ := http.NewRequest("PUT", baseURL+"/imposters", bytes.NewReader(getBody))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := client.Do(putReq)
	if err != nil {
		t.Fatalf("failed to PUT imposters: %v", err)
	}
	putResp.Body.Close()

	if putResp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", putResp.StatusCode)
	}

	// Verify the imposter now has no proxy stubs
	verifyResp, err := http.Get(baseURL + "/imposters/4582")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	defer verifyResp.Body.Close()

	verifyBody, _ := io.ReadAll(verifyResp.Body)
	var verified map[string]interface{}
	json.Unmarshal(verifyBody, &verified)

	stubs := verified["stubs"].([]interface{})
	if len(stubs) != 1 {
		t.Errorf("expected 1 stub after replay, got %d", len(stubs))
	}
}

func TestCLI_ConfigFileWithNewOptions(t *testing.T) {
	defer cleanup(t)

	// Verify config endpoint returns new options
	resp, err := http.Get(baseURL + "/config")
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	options := config["options"].(map[string]interface{})

	// Verify expected options exist
	if _, ok := options["port"]; !ok {
		t.Error("expected port in options")
	}
	if _, ok := options["allowInjection"]; !ok {
		t.Error("expected allowInjection in options")
	}
	if _, ok := options["debug"]; !ok {
		t.Error("expected debug in options")
	}
}

func TestCLI_BinaryExists(t *testing.T) {
	// Build the binary
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..", "..")
	binaryPath := filepath.Join(projectRoot, "tartuffe-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/tartuffe")
	cmd.Dir = projectRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	// Test version flag
	versionCmd := exec.Command(binaryPath, "--version")
	output, err := versionCmd.Output()
	if err != nil {
		t.Fatalf("failed to run version command: %v", err)
	}

	if !bytes.Contains(output, []byte("go-tartuffe")) {
		t.Errorf("expected version output to contain 'go-tartuffe', got: %s", output)
	}
}

func TestCLI_StartWithConfigFile(t *testing.T) {
	// This test verifies that the server can start with a config file
	// We'll use the test server that's already running and just verify
	// that config file loading works through the API

	defer cleanup(t)

	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "simple.json")

	// Load config and add to test server
	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := testServer.LoadImposters(cfg.Imposters); err != nil {
		t.Fatalf("failed to load imposters: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify the imposter is running
	resp, err := http.Get("http://localhost:4545/")
	if err != nil {
		t.Fatalf("failed to connect to imposter: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello from simple service" {
		t.Errorf("expected 'Hello from simple service', got '%s'", string(body))
	}
}
