package integration

import (
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Test certificate and key for testing
const testCert = `-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQDU+pQ4pHiQODANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjMwMTAxMDAwMDAwWhcNMjQwMTAxMDAwMDAwWjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC9
hZwN3eGhgZXnz3TfOu7xGQNi+L9M3kYUiXjF4dYMWvHrHNYxLw3TqVPKJMKaEOGj
PaXR6gQJeq9Q9s8BqVKq3S6R2vY4MnVVq9Sh7kHQ9gqXZ3jLHKfZHXeVR1LBqH3e
vqo7UYdwj7V4LPDqR8x5Nh7HBhYQ3kLYq8Y5HQjXvT7lGc3qvqS7qZYR7P3q3Xvb
cDQwNLQ7kX0uPYmVN6kPvECvYlL3OTBQ7XL3k9DYhR7OqN5rqSE6QfbJwAQ0KLMA
F7UQpoFXCpXR7yVG8DLqwN7Xf3q3vRGjG0Mq3bqV5k4Q3gF4L7vR7Oa3bNq9P3e9
EQAgwAABBCNMF3U5Qa3bAgMBAAEwDQYJKoZIhvcNAQELBQADggEBALiHBvOYYkfF
bWN3TqXL3SVLK6PjmGcpxlqT8kXLNFq7Q9lLp7Jq1M0YYvP8PfJRRBnF9k6K3x7f
Bvh9nRYMNvpR7W4hXHJy8OqMX9o3LYdD5CqGSbCBP5nJLGVoGzHQ7kXKLMXQQpLA
fBJfBTZQChlVQSqGZ3TvNQ3CnLNLwJY3Y3q9c3JqV7C3XLJR4n5qX3g0zGl7Dq8X
lKQZ3QM7kJfV3KQljZ8Qy8TfWDQnCz3Jn3kQX8C3tQPQ3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q
3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q3Q7Q
3Q7Q3Q7Q3Q4=
-----END CERTIFICATE-----`

const testKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAvYWcDd3hoYGV58903zru8RkDYvi/TN5GFIl4xeHWDFrx6xzW
MS8N06lTyiTCmhDhoz2l0eoECXqvUPbPAalSqt0ukdr2ODJ1VavUoe5B0PYKl2d4
yxyn2R13lUdSwah93r6qO1GHcI+1eCzw6kfMeTYexwYWEN5C2KvGOR0I170+5RnN
6r6ku6mWEez96t1723A0MDS0O5F9Lj2JlTepD7xAr2JS9zkwUO1y95PQ2IUezqje
a6khOkH2ycAENCizABe1EKaBVwqV0e8lRvAy6sDe1396t70RoxtDKt26leZOEN4B
eC+70ezmty zavT93vREAIMAAAQQjTBd1OUGt2wIDAQABAoIBAFqBMaYXzsdO3c2O
f3rR7qKx3N3kLmYlD8vF0sKn5fXrXcYfBBnJK3fOFdT7CRfFr0qV1PQGF3g8R7QB
N8vP7K3hL8FJ8LqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7k
LqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7b
Qv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvR
qQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U
5FvRqQECgYEA5e3Q7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5Fv
RqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5
U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQECgYEA
0vRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLq
P5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv
7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLsCgYEA2Q7b
Qv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvR
qQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U
5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQECgYBQv7kLqP5U5FvRqQ7b
Qv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvR
qQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U
5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQwKBgDqP5U5FvRqQ7bQv7kLqP5
U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7k
LqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7b
Qv7kLqP5U5FvRqQ7bQv7kLqP5U5FvRqQ7bQ=
-----END RSA PRIVATE KEY-----`

// TestHTTPS_CreateImposter_WithSelfSignedCert tests creating an HTTPS imposter without providing certs
func TestHTTPS_CreateImposter_WithSelfSignedCert(t *testing.T) {
	defer cleanup(t)

	// Create HTTPS imposter without providing cert/key (should auto-generate)
	resp, body, err := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6000,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Hello from HTTPS!"}},
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

	// Verify certificate metadata is present
	if body["certificateFingerprint"] == nil || body["certificateFingerprint"] == "" {
		t.Error("expected certificateFingerprint in response")
	}
	if body["commonName"] == nil || body["commonName"] == "" {
		t.Error("expected commonName in response")
	}
	if body["validFrom"] == nil || body["validFrom"] == "" {
		t.Error("expected validFrom in response")
	}
	if body["validTo"] == nil || body["validTo"] == "" {
		t.Error("expected validTo in response")
	}

	// Verify private key is NOT returned
	if body["key"] != nil && body["key"] != "" {
		t.Error("private key should NOT be returned in API response")
	}

	time.Sleep(100 * time.Millisecond)

	// Make HTTPS request to imposter
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip verification for self-signed cert
			},
		},
	}

	impResp, err := client.Get("https://localhost:6000/test")
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	respBody, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(respBody) != "Hello from HTTPS!" {
		t.Errorf("expected 'Hello from HTTPS!', got '%s'", string(respBody))
	}
}

// TestHTTPS_GetImposter_NoPrivateKey tests that GET /imposters doesn't return private key
func TestHTTPS_GetImposter_NoPrivateKey(t *testing.T) {
	defer cleanup(t)

	// Create HTTPS imposter
	_, _, err := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6001,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Get the imposter
	resp, body, err := get("/imposters/6001")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify private key is NOT returned
	if body["key"] != nil && body["key"] != "" {
		t.Error("private key should NOT be returned in GET response")
	}

	// Verify certificate metadata IS present
	if body["certificateFingerprint"] == nil {
		t.Error("expected certificateFingerprint in response")
	}
}

// TestHTTPS_GetAllImposters_NoPrivateKey tests that GET /imposters doesn't return private keys
func TestHTTPS_GetAllImposters_NoPrivateKey(t *testing.T) {
	defer cleanup(t)

	// Create HTTPS imposter
	_, _, err := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6002,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Get all imposters
	resp, body, err := get("/imposters")
	if err != nil {
		t.Fatalf("failed to get imposters: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Check imposters array
	imposters, ok := body["imposters"].([]interface{})
	if !ok {
		t.Fatal("expected imposters array in response")
	}

	for _, imp := range imposters {
		impMap, ok := imp.(map[string]interface{})
		if !ok {
			continue
		}
		if impMap["protocol"] == "https" {
			if impMap["key"] != nil && impMap["key"] != "" {
				t.Error("private key should NOT be returned for HTTPS imposter in GET /imposters")
			}
		}
	}
}

// TestHTTPS_DeleteImposters_NoPrivateKey tests that DELETE /imposters doesn't return private key
func TestHTTPS_DeleteImposters_NoPrivateKey(t *testing.T) {
	// Create HTTPS imposter
	_, _, err := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6003,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Delete all imposters
	resp, body, err := del("/imposters")
	if err != nil {
		t.Fatalf("failed to delete imposters: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Check imposters array
	imposters, ok := body["imposters"].([]interface{})
	if !ok {
		t.Fatal("expected imposters array in response")
	}

	for _, imp := range imposters {
		impMap, ok := imp.(map[string]interface{})
		if !ok {
			continue
		}
		if impMap["protocol"] == "https" {
			if impMap["key"] != nil && impMap["key"] != "" {
				t.Error("private key should NOT be returned for HTTPS imposter in DELETE response")
			}
		}
	}
}

// TestHTTPS_StubMatching tests that HTTPS imposter matches stubs correctly
func TestHTTPS_StubMatching(t *testing.T) {
	defer cleanup(t)

	// Create HTTPS imposter with path matching
	_, _, err := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6004,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/hello"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Hello!"}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/world"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "World!"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Test /hello
	resp1, err := client.Get("https://localhost:6004/hello")
	if err != nil {
		t.Fatalf("request to /hello failed: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if string(body1) != "Hello!" {
		t.Errorf("expected 'Hello!', got '%s'", string(body1))
	}

	// Test /world
	resp2, err := client.Get("https://localhost:6004/world")
	if err != nil {
		t.Fatalf("request to /world failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if string(body2) != "World!" {
		t.Errorf("expected 'World!', got '%s'", string(body2))
	}
}

// TestHTTPS_RequestRecording tests that HTTPS imposter records requests
func TestHTTPS_RequestRecording(t *testing.T) {
	defer cleanup(t)

	// Create HTTPS imposter with request recording
	_, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "https",
		"port":           6005,
		"recordRequests": true,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "recorded"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Make a request
	req, _ := http.NewRequest("POST", "https://localhost:6005/test", strings.NewReader("test body"))
	req.Header.Set("X-Custom-Header", "custom-value")
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Get the imposter and check recorded requests
	_, body, err := get("/imposters/6005")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}

	requests, ok := body["requests"].([]interface{})
	if !ok || len(requests) == 0 {
		t.Error("expected recorded requests")
	} else {
		reqMap := requests[0].(map[string]interface{})
		if reqMap["path"] != "/test" {
			t.Errorf("expected path '/test', got '%v'", reqMap["path"])
		}
		if reqMap["method"] != "POST" {
			t.Errorf("expected method 'POST', got '%v'", reqMap["method"])
		}
	}
}

// TestHTTPS_CertificateMetadata tests that certificate metadata is correctly extracted
func TestHTTPS_CertificateMetadata(t *testing.T) {
	defer cleanup(t)

	// Create HTTPS imposter
	resp, body, err := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6006,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	// Check certificate fingerprint format (should be uppercase hex)
	fingerprint, ok := body["certificateFingerprint"].(string)
	if !ok || fingerprint == "" {
		t.Error("expected non-empty certificateFingerprint")
	} else {
		// SHA-256 fingerprint should be 64 hex characters
		if len(fingerprint) != 64 {
			t.Errorf("expected 64 character fingerprint, got %d", len(fingerprint))
		}
	}

	// Check common name
	commonName, ok := body["commonName"].(string)
	if !ok || commonName == "" {
		t.Error("expected non-empty commonName")
	}

	// Check validity dates are in RFC3339 format
	validFrom, ok := body["validFrom"].(string)
	if !ok || validFrom == "" {
		t.Error("expected non-empty validFrom")
	}
	validTo, ok := body["validTo"].(string)
	if !ok || validTo == "" {
		t.Error("expected non-empty validTo")
	}
}

// TestHTTPS_MutualAuth tests mutual TLS configuration
func TestHTTPS_MutualAuth(t *testing.T) {
	defer cleanup(t)

	// Create HTTPS imposter with mutualAuth enabled
	resp, body, err := post("/imposters", map[string]interface{}{
		"protocol":   "https",
		"port":       6007,
		"mutualAuth": true,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "mutual auth"}},
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

	// Verify mutualAuth is in response
	if body["mutualAuth"] != true {
		t.Error("expected mutualAuth to be true in response")
	}

	time.Sleep(100 * time.Millisecond)

	// Make request without client cert (should still work as rejectUnauthorized is false by default)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	impResp, err := client.Get("https://localhost:6007/test")
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	respBody, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(respBody) != "mutual auth" {
		t.Errorf("expected 'mutual auth', got '%s'", string(respBody))
	}
}
