package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestHomeHTML tests that GET / returns HTML when Accept: text/html is set
func TestHomeHTML(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/", nil)
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	body := readBody(resp)
	if !strings.Contains(body, "<html>") {
		t.Error("Response should contain <html>")
	}
	if !strings.Contains(body, "Welcome, friend") {
		t.Error("Response should contain 'Welcome, friend'")
	}
}

// TestHomeJSON tests that GET / returns JSON when Accept is not text/html
func TestHomeJSON(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if _, ok := result["_links"]; !ok {
		t.Error("Response should contain _links")
	}
}

// TestImpostersHTML tests that GET /imposters returns HTML when Accept: text/html is set
func TestImpostersHTML(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/imposters", nil)
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	body := readBody(resp)
	if !strings.Contains(body, "<html>") {
		t.Error("Response should contain <html>")
	}
	if !strings.Contains(body, "Imposters") {
		t.Error("Response should contain 'Imposters'")
	}
}

// TestImpostersJSON tests that GET /imposters returns JSON when Accept is not text/html
func TestImpostersJSON(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/imposters", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if _, ok := result["imposters"]; !ok {
		t.Error("Response should contain imposters array")
	}
}

// TestLogsHTML tests that GET /logs returns HTML when Accept: text/html is set
func TestLogsHTML(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/logs", nil)
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	body := readBody(resp)
	if !strings.Contains(body, "<html>") {
		t.Error("Response should contain <html>")
	}
	if !strings.Contains(body, "Logs") {
		t.Error("Response should contain 'Logs'")
	}
	if !strings.Contains(body, "Follow log") {
		t.Error("Response should contain 'Follow log' button")
	}
}

// TestLogsJSON tests that GET /logs returns JSON when Accept is not text/html
func TestLogsJSON(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/logs", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if _, ok := result["logs"]; !ok {
		t.Error("Response should contain logs array")
	}
}

// TestConfigHTML tests that GET /config returns HTML when Accept: text/html is set
func TestConfigHTML(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/config", nil)
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	body := readBody(resp)
	if !strings.Contains(body, "<html>") {
		t.Error("Response should contain <html>")
	}
	if !strings.Contains(body, "Config") {
		t.Error("Response should contain 'Config'")
	}
	if !strings.Contains(body, "Process Information") {
		t.Error("Response should contain 'Process Information'")
	}
}

// TestConfigJSON tests that GET /config returns JSON when Accept is not text/html
func TestConfigJSON(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/config", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if _, ok := result["version"]; !ok {
		t.Error("Response should contain version")
	}
	if _, ok := result["options"]; !ok {
		t.Error("Response should contain options")
	}
	if _, ok := result["process"]; !ok {
		t.Error("Response should contain process")
	}
}

// TestStaticCSS tests that static CSS files are served correctly
func TestStaticCSS(t *testing.T) {
	resp, err := http.Get(baseURL + "/public/css/application.css")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/css") {
		t.Errorf("Expected Content-Type containing text/css, got %s", contentType)
	}

	body := readBody(resp)
	if !strings.Contains(body, "body") {
		t.Error("CSS file should contain body selector")
	}
}

// TestStaticJS tests that static JS files are served correctly
func TestStaticJS(t *testing.T) {
	resp, err := http.Get(baseURL + "/public/js/urlHashHandler.js")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := readBody(resp)
	if !strings.Contains(body, "toggleExpandedOnSection") {
		t.Error("JS file should contain toggleExpandedOnSection function")
	}
}

// TestStaticNotFound tests that non-existent static files return 404
func TestStaticNotFound(t *testing.T) {
	resp, err := http.Get(baseURL + "/public/nonexistent.txt")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// TestImposterHTMLWithData tests that GET /imposters/{id} returns HTML with imposter data
func TestImposterHTMLWithData(t *testing.T) {
	// Use a unique port for this test
	testPort := 54546

	// First create an imposter
	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     testPort,
		"name":     "html-test-imposter",
	}
	resp, _, err := post("/imposters", imposter)
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body := readBody(resp)
		t.Fatalf("Failed to create imposter: %d, body: %s", resp.StatusCode, body)
	}

	// Now get the imposter as HTML
	req, _ := http.NewRequest("GET", baseURL+"/imposters/54546", nil)
	req.Header.Set("Accept", "text/html")

	htmlResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer htmlResp.Body.Close()

	if htmlResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", htmlResp.StatusCode)
	}

	contentType := htmlResp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	body := readBody(htmlResp)
	if !strings.Contains(body, "http") {
		t.Error("Response should contain protocol")
	}
	if !strings.Contains(body, "54546") {
		t.Error("Response should contain port")
	}

	// Clean up
	del("/imposters/54546")
}

// readBody reads the response body as a string
func readBody(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}
