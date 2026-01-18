package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestHTTPProxy_InvalidDomainError tests that proxy to invalid domain returns proper error
func TestHTTPProxy_InvalidDomainError(t *testing.T) {
	defer cleanup(t)

	// Create proxy to invalid domain
	proxyPort := 8130
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": "http://invalid.domain",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}
	defer del("/imposters")

	time.Sleep(100 * time.Millisecond)

	// Make request through proxy to invalid domain
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Should return 5xx error (500 or 502 are both acceptable)
	if resp.StatusCode < 500 || resp.StatusCode >= 600 {
		t.Errorf("expected 5xx status code for invalid domain, got %d", resp.StatusCode)
	}

	// Parse error response
	var errorResp map[string]interface{}
	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errors, ok := errorResp["errors"].([]interface{}); ok && len(errors) > 0 {
			if errObj, ok := errors[0].(map[string]interface{}); ok {
				// Verify error code
				if code, ok := errObj["code"].(string); ok {
					if code != "invalid proxy" {
						t.Errorf("expected error code 'invalid proxy', got %q", code)
					}
				}
				// Verify error message contains the invalid domain
				if msg, ok := errObj["message"].(string); ok {
					if !strings.Contains(msg, "invalid.domain") {
						t.Errorf("expected error message to mention 'invalid.domain', got %q", msg)
					}
				}
			}
		}
	}
}

// TestHTTPProxy_QueryStringFidelity tests query string preservation (issue #410)
func TestHTTPProxy_QueryStringFidelity(t *testing.T) {
	defer cleanup(t)

	// Create a raw HTTP server (not a mountebank imposter) that echoes the request URL
	// This matches mountebank's test which uses http.createServer
	originPort := 8131
	originMux := http.NewServeMux()
	originMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Echo the full request URL (path + query string)
		w.Write([]byte(r.URL.RequestURI()))
	})
	originServer := &http.Server{Addr: fmt.Sprintf(":%d", originPort), Handler: originMux}
	go originServer.ListenAndServe()
	defer originServer.Close()

	time.Sleep(100 * time.Millisecond)

	// Create proxy
	proxyPort := 8132
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyAlways",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Test 1: Query key without = (e.g., ?WSDL)
	resp1, err := http.Get(fmt.Sprintf("http://localhost:%d/path?WSDL", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if string(body1) != "/path?WSDL" {
		t.Errorf("expected '/path?WSDL', got %q", string(body1))
	}

	// Test 2: Query key with = but no value (e.g., ?WSDL=)
	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d/path?WSDL=", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if string(body2) != "/path?WSDL=" {
		t.Errorf("expected '/path?WSDL=', got %q", string(body2))
	}
}

// TestHTTPProxy_JSONBodyStorage tests that JSON bodies are saved as objects (issue #656)
func TestHTTPProxy_JSONBodyStorage(t *testing.T) {
	defer cleanup(t)

	// Create origin that returns JSON body
	originPort := 8133
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": map[string]interface{}{
								"json": true,
							},
							"headers": map[string]interface{}{
								"Content-Type": "application/json",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with ProxyOnce to record
	proxyPort := 8134
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyOnce",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Test origin server directly first
	originTest, err := http.Get(fmt.Sprintf("http://localhost:%d/", originPort))
	if err != nil {
		t.Fatalf("origin request failed: %v", err)
	}
	originBody, _ := io.ReadAll(originTest.Body)
	originTest.Body.Close()
	t.Logf("Origin server response: status=%d, body=%q, Content-Type=%q",
		originTest.StatusCode, string(originBody), originTest.Header.Get("Content-Type"))

	// Make request through proxy to trigger recording
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", proxyPort))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	t.Logf("Proxy response: status=%d, body=%q, Content-Type=%q, Content-Length=%q",
		resp.StatusCode, string(body), resp.Header.Get("Content-Type"), resp.Header.Get("Content-Length"))

	// Verify response contains JSON (pretty-printed to match mountebank format)
	expectedJSON := `{
    "json": true
}`
	actualJSON := string(body)
	if actualJSON != expectedJSON {
		t.Errorf("JSON format mismatch:\nExpected:\n%q\nGot:\n%q", expectedJSON, actualJSON)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify recorded stub has body as object, not string
	getResp, getBody, err := get(fmt.Sprintf("/imposters/%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	stubs, ok := getBody["stubs"].([]interface{})
	if !ok || len(stubs) < 1 {
		t.Fatalf("expected at least 1 stub")
	}

	recordedStub := stubs[0].(map[string]interface{})
	responses := recordedStub["responses"].([]interface{})
	if len(responses) < 1 {
		t.Fatal("expected at least 1 response")
	}

	response := responses[0].(map[string]interface{})
	isResp := response["is"].(map[string]interface{})

	// Body should be an object, not a string
	bodyField := isResp["body"]
	if bodyObj, ok := bodyField.(map[string]interface{}); ok {
		if jsonVal, ok := bodyObj["json"].(bool); !ok || !jsonVal {
			t.Errorf("expected body to be {json: true} object, got %v", bodyObj)
		}
	} else {
		t.Errorf("expected body to be object, got type %T: %v", bodyField, bodyField)
	}
}

// TestHTTPProxy_ContentLengthPreservation tests that Content-Length is preserved (issue #132)
func TestHTTPProxy_ContentLengthPreservation(t *testing.T) {
	defer cleanup(t)

	// Create origin that echoes the transfer encoding
	originPort := 8135
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request) {
							var encoding = "";
							var headers = request.headers || {};

							// Check for Transfer-Encoding header
							var hasTransferEncoding = Object.keys(headers).some(function(key) {
								return key.toLowerCase() === "transfer-encoding";
							});

							// Check for Content-Length header
							var hasContentLength = Object.keys(headers).some(function(key) {
								return key.toLowerCase() === "content-length";
							});

							if (hasTransferEncoding) {
								encoding = "chunked";
							} else if (hasContentLength) {
								encoding = "content-length";
							}

							return {
								statusCode: 200,
								body: "Encoding: " + encoding
							};
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy
	proxyPort := 8136
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("http://localhost:%d", originPort),
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make PUT request with Content-Length header
	client := &http.Client{}
	bodyStr := "TEST"
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://localhost:%d/", proxyPort), strings.NewReader(bodyStr))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyStr)))
	req.ContentLength = int64(len(bodyStr))

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Origin should receive Content-Length, not chunked encoding
	if string(body) != "Encoding: content-length" {
		t.Errorf("expected 'Encoding: content-length', got %q", string(body))
	}
}

// TestHTTPProxy_RemoveProxies tests removeProxies query parameter
func TestHTTPProxy_RemoveProxies(t *testing.T) {
	defer cleanup(t)

	// Create origin with counter
	originPort := 8137
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     originPort,
		"name":     "origin",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							state.counter = state.counter || 0;
							state.counter++;
							return {
								statusCode: 200,
								body: state.counter + ". " + request.path
							};
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create origin imposter: %v", err)
	}
	if originResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for origin, got %d", originResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Create proxy with ProxyAlways mode
	proxyPort := 8138
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     proxyPort,
		"name":     "proxy",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":   fmt.Sprintf("http://localhost:%d", originPort),
							"mode": "proxyAlways",
							"predicateGenerators": []map[string]interface{}{
								{
									"matches": map[string]interface{}{
										"path": true,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Make requests to record responses
	http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	time.Sleep(50 * time.Millisecond)
	http.Get(fmt.Sprintf("http://localhost:%d/second", proxyPort))
	time.Sleep(50 * time.Millisecond)
	http.Get(fmt.Sprintf("http://localhost:%d/first", proxyPort))
	time.Sleep(100 * time.Millisecond)

	// Get imposters with removeProxies=true
	getResp, getBody, err := get("/imposters?removeProxies=true")
	if err != nil {
		t.Fatalf("failed to get imposters: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	// Parse response
	imposters, ok := getBody["imposters"].([]interface{})
	if !ok {
		t.Fatal("expected imposters array")
	}

	// Find proxy imposter
	var proxyImposter map[string]interface{}
	for _, imp := range imposters {
		impObj := imp.(map[string]interface{})
		if portNum, ok := impObj["port"].(float64); ok && int(portNum) == proxyPort {
			proxyImposter = impObj
			break
		}
	}

	if proxyImposter == nil {
		t.Fatal("proxy imposter not found")
	}

	// Verify proxy responses have been replaced with "is" responses
	stubs, ok := proxyImposter["stubs"].([]interface{})
	if !ok || len(stubs) < 1 {
		t.Fatalf("expected at least 1 stub")
	}

	// Check that stubs contain "is" responses, not "proxy" responses
	for i, stub := range stubs {
		stubObj := stub.(map[string]interface{})
		responses := stubObj["responses"].([]interface{})

		for j, resp := range responses {
			respObj := resp.(map[string]interface{})

			// Should have "is" field
			if _, ok := respObj["is"]; !ok {
				t.Errorf("stub %d response %d: expected 'is' field", i, j)
			}

			// Should NOT have "proxy" field
			if _, ok := respObj["proxy"]; ok {
				t.Errorf("stub %d response %d: unexpected 'proxy' field (should be removed)", i, j)
			}
		}
	}
}
