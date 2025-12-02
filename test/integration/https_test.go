package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// generateTestCert generates a self-signed certificate for testing
func generateTestCert(commonName string) (certPEM, keyPEM string, err error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Test Organization"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{commonName, "localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	// Encode certificate to PEM
	certPEMBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	// Encode private key to PEM
	keyPEMBlock := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return string(certPEMBlock), string(keyPEMBlock), nil
}

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

// TestHTTPS_CustomCertificate tests that a client-provided certificate is used
func TestHTTPS_CustomCertificate(t *testing.T) {
	defer cleanup(t)

	// Generate a certificate at runtime with a custom CN
	customCN := "custom-test-server.local"
	certPEM, keyPEM, err := generateTestCert(customCN)
	if err != nil {
		t.Fatalf("failed to generate test certificate: %v", err)
	}

	// Create HTTPS imposter with custom certificate
	resp, body, err := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6008,
		"cert":     certPEM,
		"key":      keyPEM,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "custom cert response"}},
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

	// Verify the commonName matches our custom certificate
	if body["commonName"] != customCN {
		t.Errorf("expected commonName '%s', got '%v'", customCN, body["commonName"])
	}

	// Verify private key is NOT returned
	if body["key"] != nil && body["key"] != "" {
		t.Error("private key should NOT be returned in API response")
	}

	// Verify certificate fingerprint is present
	if body["certificateFingerprint"] == nil || body["certificateFingerprint"] == "" {
		t.Error("expected certificateFingerprint in response")
	}

	time.Sleep(100 * time.Millisecond)

	// Make HTTPS request and verify the server uses our certificate
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	impResp, err := client.Get("https://localhost:6008/test")
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer impResp.Body.Close()

	// Verify the server certificate has our custom CN
	if impResp.TLS != nil && len(impResp.TLS.PeerCertificates) > 0 {
		serverCert := impResp.TLS.PeerCertificates[0]
		if serverCert.Subject.CommonName != customCN {
			t.Errorf("server certificate CN expected '%s', got '%s'", customCN, serverCert.Subject.CommonName)
		}
	} else {
		t.Error("expected TLS peer certificate information")
	}

	respBody, _ := io.ReadAll(impResp.Body)
	if string(respBody) != "custom cert response" {
		t.Errorf("expected 'custom cert response', got '%s'", string(respBody))
	}
}

// TestHTTPS_InvalidCertificate tests that invalid certificate returns an error
func TestHTTPS_InvalidCertificate(t *testing.T) {
	defer cleanup(t)

	// Try to create HTTPS imposter with invalid certificate
	resp, body, _ := post("/imposters", map[string]interface{}{
		"protocol": "https",
		"port":     6009,
		"cert":     "not a valid certificate",
		"key":      "not a valid key",
	})

	// Should return 400 Bad Request with an error
	if resp.StatusCode != 400 {
		t.Errorf("expected status 400 for invalid cert, got %d", resp.StatusCode)
	}

	// Check for error message
	if body["errors"] == nil {
		t.Error("expected errors in response for invalid certificate")
	}
}
