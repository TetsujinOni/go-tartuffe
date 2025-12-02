package integration

import (
	"net/http"
	"strings"
	"testing"
)

// Tests for CORS (Cross-Origin Resource Sharing) support
// Matching mountebank's httpImposterTest.js CORS tests

// TestCORSDisabledByDefault tests that CORS is disabled by default
func TestCORSDisabledByDefault(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol": "http",
		"port":     4800,
		// No allowCORS option
	}
	post("/imposters", imposter)

	// Make OPTIONS preflight request
	req, _ := http.NewRequest("OPTIONS", "http://localhost:4800/", nil)
	req.Header.Set("Access-Control-Request-Method", "PUT")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Header")
	req.Header.Set("Origin", "localhost:8080")

	resp, _ := client.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Should NOT have CORS headers
	headers := make(map[string]bool)
	for k := range resp.Header {
		headers[strings.ToLower(k)] = true
	}

	if headers["access-control-allow-headers"] {
		t.Error("should not have access-control-allow-headers when allowCORS is disabled")
	}
	if headers["access-control-allow-methods"] {
		t.Error("should not have access-control-allow-methods when allowCORS is disabled")
	}
	if headers["access-control-allow-origin"] {
		t.Error("should not have access-control-allow-origin when allowCORS is disabled")
	}
}

// TestCORSEnabledWithAllowCORS tests that CORS works when allowCORS is enabled
func TestCORSEnabledWithAllowCORS(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol":  "http",
		"port":      4801,
		"allowCORS": true,
	}
	post("/imposters", imposter)

	// Make OPTIONS preflight request
	req, _ := http.NewRequest("OPTIONS", "http://localhost:4801/", nil)
	req.Header.Set("Access-Control-Request-Method", "PUT")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Header")
	req.Header.Set("Origin", "localhost:8080")

	resp, _ := client.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Should have CORS headers
	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders != "X-Custom-Header" {
		t.Errorf("expected Access-Control-Allow-Headers: X-Custom-Header, got %s", allowHeaders)
	}

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if allowMethods != "PUT" {
		t.Errorf("expected Access-Control-Allow-Methods: PUT, got %s", allowMethods)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "localhost:8080" {
		t.Errorf("expected Access-Control-Allow-Origin: localhost:8080, got %s", allowOrigin)
	}
}

// TestCORSNotHandledWithoutPreflightHeaders tests that non-preflight OPTIONS are not handled
func TestCORSNotHandledWithoutPreflightHeaders(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol":  "http",
		"port":      4802,
		"allowCORS": true,
	}
	post("/imposters", imposter)

	// Make OPTIONS request WITHOUT preflight headers
	req, _ := http.NewRequest("OPTIONS", "http://localhost:4802/", nil)
	// Missing Access-Control-Request-Method, Access-Control-Request-Headers, Origin

	resp, _ := client.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Should NOT have CORS headers since this isn't a valid preflight
	headers := make(map[string]bool)
	for k := range resp.Header {
		headers[strings.ToLower(k)] = true
	}

	if headers["access-control-allow-headers"] {
		t.Error("should not have access-control-allow-headers for non-preflight OPTIONS")
	}
	if headers["access-control-allow-methods"] {
		t.Error("should not have access-control-allow-methods for non-preflight OPTIONS")
	}
	if headers["access-control-allow-origin"] {
		t.Error("should not have access-control-allow-origin for non-preflight OPTIONS")
	}
}

// TestCORSWithMultipleHeaders tests preflight with multiple headers
func TestCORSWithMultipleHeaders(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol":  "http",
		"port":      4803,
		"allowCORS": true,
	}
	post("/imposters", imposter)

	req, _ := http.NewRequest("OPTIONS", "http://localhost:4803/", nil)
	req.Header.Set("Access-Control-Request-Method", "DELETE")
	req.Header.Set("Access-Control-Request-Headers", "X-Header-One, X-Header-Two, Content-Type")
	req.Header.Set("Origin", "https://example.com")

	resp, _ := client.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders != "X-Header-One, X-Header-Two, Content-Type" {
		t.Errorf("expected all requested headers, got %s", allowHeaders)
	}

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if allowMethods != "DELETE" {
		t.Errorf("expected DELETE, got %s", allowMethods)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", allowOrigin)
	}
}

// TestCORSWithStubMatch tests that CORS preflight doesn't interfere with stubs
func TestCORSWithStubMatch(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol":  "http",
		"port":      4804,
		"allowCORS": true,
		"stubs": []interface{}{
			map[string]interface{}{
				"predicates": []interface{}{
					map[string]interface{}{
						"equals": map[string]interface{}{"method": "PUT"},
					},
				},
				"responses": []interface{}{
					map[string]interface{}{
						"is": map[string]interface{}{"body": "PUT handled"},
					},
				},
			},
		},
	}
	post("/imposters", imposter)

	// First do preflight
	preflightReq, _ := http.NewRequest("OPTIONS", "http://localhost:4804/", nil)
	preflightReq.Header.Set("Access-Control-Request-Method", "PUT")
	preflightReq.Header.Set("Origin", "http://example.com")

	preflightResp, _ := client.Do(preflightReq)
	preflightResp.Body.Close()

	// Then do actual PUT - should match stub
	putReq, _ := http.NewRequest("PUT", "http://localhost:4804/", nil)
	putReq.Header.Set("Origin", "http://example.com")

	putResp, _ := client.Do(putReq)
	defer putResp.Body.Close()

	if putResp.StatusCode != 200 {
		t.Errorf("expected 200 from PUT, got %d", putResp.StatusCode)
	}
}

// TestCORSHttpsProtocol tests CORS with HTTPS protocol
func TestCORSHttpsProtocol(t *testing.T) {
	defer cleanup(t)

	imposter := map[string]interface{}{
		"protocol":  "https",
		"port":      4805,
		"allowCORS": true,
	}
	_, body, _ := post("/imposters", imposter)

	// Get the actual port (might be auto-assigned)
	port := int(body["port"].(float64))

	// Skip HTTPS test if port creation failed
	if port == 0 {
		t.Skip("HTTPS imposter creation failed")
	}

	// HTTPS CORS tests would need an insecure client for self-signed certs
	// This is tested in https_test.go
	t.Skip("HTTPS CORS tested via https_test.go with proper TLS setup")
}
