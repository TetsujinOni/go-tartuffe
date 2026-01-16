package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/config"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// Tests converted from mbTest/cli/saveTest.js

func TestSave_ToDefaultFile(t *testing.T) {
	defer cleanup(t)

	// Create some imposters
	imposter := models.Imposter{
		Port:     4550,
		Protocol: "http",
		Name:     "save test",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Body: "test response"}},
				},
			},
		},
	}

	// Save to default file
	saveFile := filepath.Join(os.TempDir(), "mb.json")
	defer os.Remove(saveFile)

	err := config.SaveFile([]*models.Imposter{&imposter}, saveFile, false, true)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Fatalf("save file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(saveFile)
	if err != nil {
		t.Fatalf("failed to read save file: %v", err)
	}

	var saved config.Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	if len(saved.Imposters) != 1 {
		t.Fatalf("expected 1 imposter, got %d", len(saved.Imposters))
	}

	if saved.Imposters[0].Port != 4550 {
		t.Errorf("expected port 4550, got %d", saved.Imposters[0].Port)
	}
	if saved.Imposters[0].Name != "save test" {
		t.Errorf("expected name 'save test', got '%s'", saved.Imposters[0].Name)
	}
}

func TestSave_ToCustomFile(t *testing.T) {
	defer cleanup(t)

	imposter := models.Imposter{
		Port:     4551,
		Protocol: "http",
		Name:     "custom save test",
	}

	// Save to custom file
	customFile := filepath.Join(os.TempDir(), "custom-save.json")
	defer os.Remove(customFile)

	err := config.SaveFile([]*models.Imposter{&imposter}, customFile, false, true)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(customFile); os.IsNotExist(err) {
		t.Fatalf("custom save file was not created")
	}

	// Read and verify
	data, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatalf("failed to read save file: %v", err)
	}

	var saved config.Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	if saved.Imposters[0].Name != "custom save test" {
		t.Errorf("expected name 'custom save test', got '%s'", saved.Imposters[0].Name)
	}
}

func TestSave_WithRemoveProxies(t *testing.T) {
	defer cleanup(t)

	// Create imposter with proxy and non-proxy responses
	imposter := models.Imposter{
		Port:     4552,
		Protocol: "http",
		Name:     "proxy test",
		Stubs: []models.Stub{
			{
				// Stub with proxy response only - should be removed
				Responses: []models.Response{
					{Proxy: &models.ProxyResponse{To: "http://example.com"}},
				},
			},
			{
				// Stub with non-proxy response - should be kept
				Responses: []models.Response{
					{Is: &models.IsResponse{Body: "non-proxy response"}},
				},
			},
			{
				// Stub with mixed responses - proxy should be filtered
				Responses: []models.Response{
					{Is: &models.IsResponse{Body: "mixed stub"}},
					{Proxy: &models.ProxyResponse{To: "http://other.com"}},
				},
			},
		},
	}

	saveFile := filepath.Join(os.TempDir(), "proxy-test.json")
	defer os.Remove(saveFile)

	// Save with removeProxies=true
	err := config.SaveFile([]*models.Imposter{&imposter}, saveFile, true, true)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(saveFile)
	if err != nil {
		t.Fatalf("failed to read save file: %v", err)
	}

	var saved config.Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	// Should have 2 stubs (proxy-only stub removed)
	if len(saved.Imposters[0].Stubs) != 2 {
		t.Fatalf("expected 2 stubs after removing proxies, got %d", len(saved.Imposters[0].Stubs))
	}

	// First stub should have 1 response
	if len(saved.Imposters[0].Stubs[0].Responses) != 1 {
		t.Errorf("expected 1 response in first stub, got %d", len(saved.Imposters[0].Stubs[0].Responses))
	}

	// Second stub should have 1 response (proxy filtered out)
	if len(saved.Imposters[0].Stubs[1].Responses) != 1 {
		t.Errorf("expected 1 response in second stub, got %d", len(saved.Imposters[0].Stubs[1].Responses))
	}

	// Verify no proxy responses exist
	for _, stub := range saved.Imposters[0].Stubs {
		for _, resp := range stub.Responses {
			if resp.Proxy != nil {
				t.Errorf("found proxy response after removeProxies")
			}
		}
	}
}

func TestSave_Replayable(t *testing.T) {
	defer cleanup(t)

	// Create imposter with requests (simulating recorded requests)
	request := models.Request{
		Path:   "/test",
		Method: "GET",
	}

	numRequests := 1
	imposter := models.Imposter{
		Port:             4553,
		Protocol:         "http",
		Name:             "replayable test",
		Requests:         []models.Request{request},
		NumberOfRequests: &numRequests,
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Body: "test"}},
				},
			},
		},
	}

	saveFile := filepath.Join(os.TempDir(), "replayable-test.json")
	defer os.Remove(saveFile)

	// Save with replayable=true
	err := config.SaveFile([]*models.Imposter{&imposter}, saveFile, false, true)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	data, err := os.ReadFile(saveFile)
	if err != nil {
		t.Fatalf("failed to read save file: %v", err)
	}

	var saved config.Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	// Requests should be excluded in replayable mode
	if len(saved.Imposters[0].Requests) != 0 {
		t.Errorf("expected no requests in replayable mode, got %d", len(saved.Imposters[0].Requests))
	}

	if saved.Imposters[0].NumberOfRequests != nil {
		t.Errorf("expected numberOfRequests to be nil in replayable mode, got %d", *saved.Imposters[0].NumberOfRequests)
	}
}

func TestSave_ExcludesLinks(t *testing.T) {
	defer cleanup(t)

	// Create imposter with links (as returned by API)
	imposter := models.Imposter{
		Port:     4554,
		Protocol: "http",
		Name:     "links test",
		Links: &models.Links{
			Self: &models.Link{Href: "http://localhost:2525/imposters/4554"},
		},
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Body: "test"}},
				},
				Links: &models.StubLinks{
					Self: &models.Link{Href: "http://localhost:2525/imposters/4554/stubs/0"},
				},
			},
		},
	}

	saveFile := filepath.Join(os.TempDir(), "links-test.json")
	defer os.Remove(saveFile)

	err := config.SaveFile([]*models.Imposter{&imposter}, saveFile, false, true)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	data, err := os.ReadFile(saveFile)
	if err != nil {
		t.Fatalf("failed to read save file: %v", err)
	}

	// Verify no _links in output
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	imposters := raw["imposters"].([]interface{})
	imp := imposters[0].(map[string]interface{})

	if _, hasLinks := imp["_links"]; hasLinks {
		t.Errorf("imposter should not have _links in saved file")
	}

	stubs := imp["stubs"].([]interface{})
	stub := stubs[0].(map[string]interface{})

	if _, hasLinks := stub["_links"]; hasLinks {
		t.Errorf("stub should not have _links in saved file")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	defer cleanup(t)

	// Load config from EJS file
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "..", "..", "test", "fixtures", "imposters", "imposters.ejs")

	cfg, err := config.LoadFile(configPath, false)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Convert to pointer slice for Save
	impPtrs := make([]*models.Imposter, len(cfg.Imposters))
	for i := range cfg.Imposters {
		impPtrs[i] = &cfg.Imposters[i]
	}

	// Save to file
	saveFile := filepath.Join(os.TempDir(), "roundtrip-test.json")
	defer os.Remove(saveFile)

	err = config.SaveFile(impPtrs, saveFile, false, true)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load the saved file
	reloaded, err := config.LoadFile(saveFile, false)
	if err != nil {
		t.Fatalf("failed to reload saved config: %v", err)
	}

	// Verify round-trip
	if len(reloaded.Imposters) != len(cfg.Imposters) {
		t.Fatalf("expected %d imposters after round-trip, got %d", len(cfg.Imposters), len(reloaded.Imposters))
	}

	for i := range cfg.Imposters {
		if reloaded.Imposters[i].Port != cfg.Imposters[i].Port {
			t.Errorf("imposter %d: port mismatch", i)
		}
		if reloaded.Imposters[i].Protocol != cfg.Imposters[i].Protocol {
			t.Errorf("imposter %d: protocol mismatch", i)
		}
		if reloaded.Imposters[i].Name != cfg.Imposters[i].Name {
			t.Errorf("imposter %d: name mismatch", i)
		}
	}
}
