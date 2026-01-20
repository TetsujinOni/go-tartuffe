package integration

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TCP Proxy tests

// TestTCP_ProxyBasicForwarding tests basic TCP proxy forwarding
// mountebank tcpProxyTest.js: "should send same request information to proxied socket"
func TestTCP_ProxyBasicForwarding(t *testing.T) {
	defer cleanup(t)

	// Create origin server on port 6000
	originPort := 6000
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     originPort,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "origin server"}},
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

	// Create proxy imposter on port 6001
	proxyPort := 6001
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("tcp://localhost:%d", originPort),
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

	// Connect to proxy and verify it forwards to origin
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Send request through proxy
	_, err = conn.Write([]byte("test message"))
	if err != nil {
		t.Fatalf("failed to write to proxy: %v", err)
	}

	// Read response from origin via proxy
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("failed to read from proxy: %v", err)
	}

	response := string(buffer[:n])
	if response != "origin server" {
		t.Errorf("expected 'origin server', got '%s'", response)
	}
}

// TestTCP_ProxyBinaryData tests proxying binary data
// mountebank tcpProxyTest.js: "should proxy binary data"
func TestTCP_ProxyBinaryData(t *testing.T) {
	defer cleanup(t)

	// Binary response from origin
	binaryResponse := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	// Create origin server
	originPort := 6002
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     originPort,
		"mode":     "binary",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"data": base64.StdEncoding.EncodeToString(binaryResponse),
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

	// Create proxy imposter
	proxyPort := 6003
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"mode":     "binary",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("tcp://localhost:%d", originPort),
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

	// Connect to proxy
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Send binary request
	binaryRequest := []byte{0x01, 0x02, 0x03, 0x04}
	_, err = conn.Write(binaryRequest)
	if err != nil {
		t.Fatalf("failed to write to proxy: %v", err)
	}

	// Read binary response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("failed to read from proxy: %v", err)
	}

	response := buffer[:n]
	if len(response) != len(binaryResponse) {
		t.Fatalf("expected %d bytes, got %d", len(binaryResponse), len(response))
	}
	for i := range binaryResponse {
		if response[i] != binaryResponse[i] {
			t.Errorf("byte %d mismatch: expected 0x%02X, got 0x%02X", i, binaryResponse[i], response[i])
		}
	}
}

// TestTCP_ProxyConnectionRefused tests proxy handling of connection refused
// mountebank tcpProxyTest.js: "should gracefully deal with non listening ports"
func TestTCP_ProxyConnectionRefused(t *testing.T) {
	defer cleanup(t)

	// Create proxy pointing to non-listening port
	proxyPort := 6004
	nonListeningPort := 7999 // Hopefully nothing is listening here
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("tcp://localhost:%d", nonListeningPort),
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

	// Connect to proxy
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Send request
	_, err = conn.Write([]byte("test"))
	if err != nil {
		t.Fatalf("failed to write to proxy: %v", err)
	}

	// Should receive JSON error response for connection refused
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buffer)

	if err != nil {
		t.Fatalf("expected error response, got read error: %v", err)
	}
	if n == 0 {
		t.Fatal("expected error response, got empty response")
	}

	// Parse and validate the error response
	response := string(buffer[:n])
	var errorResp struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(buffer[:n], &errorResp); err != nil {
		t.Fatalf("failed to parse error response: %v, response was: %s", err, response)
	}

	if len(errorResp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errorResp.Errors))
	}
	if errorResp.Errors[0].Code != "invalid proxy" {
		t.Errorf("expected error code 'invalid proxy', got %q", errorResp.Errors[0].Code)
	}
	if !strings.Contains(errorResp.Errors[0].Message, "Unable to connect") {
		t.Errorf("expected message to contain 'Unable to connect', got %q", errorResp.Errors[0].Message)
	}
}

// TestTCP_ProxyKeepalive tests keepalive proxy connections
// mountebank tcpStubTest.js: "should allow keepalive proxies"
func TestTCP_ProxyKeepalive(t *testing.T) {
	defer cleanup(t)

	// Create origin server
	originPort := 6005
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     originPort,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "response"}},
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

	// Create proxy with keepalive
	proxyPort := 6006
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to":        fmt.Sprintf("tcp://localhost:%d", originPort),
							"keepalive": true,
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

	// Make multiple requests through proxy to verify keepalive
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
		if err != nil {
			t.Fatalf("request %d: failed to connect to proxy: %v", i+1, err)
		}

		conn.Write([]byte(fmt.Sprintf("request%d", i+1)))
		buffer := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := conn.Read(buffer)
		conn.Close()

		if err != nil {
			t.Fatalf("request %d: failed to read from proxy: %v", i+1, err)
		}

		response := string(buffer[:n])
		if response != "response" {
			t.Errorf("request %d: expected 'response', got '%s'", i+1, response)
		}
	}
}

// TestTCP_ProxyEndOfRequestResolver tests proxy with custom request boundary detection
// mountebank tcpProxyTest.js: "should obey endOfRequestResolver"
func TestTCP_ProxyEndOfRequestResolver(t *testing.T) {
	defer cleanup(t)

	// Create origin server that echoes back the request
	originPort := 6007
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     originPort,
		"mode":     "binary",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"data": base64.StdEncoding.EncodeToString([]byte("RESPONSE")),
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

	// Create proxy with endOfRequestResolver
	// The resolver reads a 4-byte length header, then that many bytes
	resolverFn := `function(requestData) {
		if (requestData.length < 4) {
			return false; // Need at least 4 bytes for length header
		}
		// Read 4-byte little-endian length
		var length = requestData.readUInt32LE(0);
		// Check if we have the full message (4 byte header + length bytes)
		return requestData.length >= 4 + length;
	}`

	proxyPort := 6008
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"mode":     "binary",
		"endOfRequestResolver": map[string]interface{}{
			"inject": resolverFn,
		},
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("tcp://localhost:%d", originPort),
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

	// Connect to proxy
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Send a message with length header
	message := []byte("TEST MESSAGE")
	lengthHeader := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthHeader, uint32(len(message)))

	// Write length header
	_, err = conn.Write(lengthHeader)
	if err != nil {
		t.Fatalf("failed to write length header: %v", err)
	}

	// Write message
	_, err = conn.Write(message)
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// Read response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("failed to read from proxy: %v", err)
	}

	response := string(buffer[:n])
	if response != "RESPONSE" {
		t.Errorf("expected 'RESPONSE', got '%s'", response)
	}
}

// TestTCP_ProxyProtocolValidation tests that proxy rejects non-TCP protocols
// mountebank tcpProxyTest.js: "should reject non-tcp protocols"
func TestTCP_ProxyProtocolValidation(t *testing.T) {
	defer cleanup(t)

	// Create TCP proxy pointing to HTTP endpoint (non-TCP protocol)
	// The imposter is created successfully, but returns error at runtime
	proxyPort := 6009
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": "http://localhost:8080",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", proxyResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Connect and send a request - should get error response at runtime
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("test"))
	if err != nil {
		t.Fatalf("failed to write to proxy: %v", err)
	}

	// Should receive JSON error response for non-TCP protocol
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buffer)

	if err != nil {
		t.Fatalf("expected error response, got read error: %v", err)
	}
	if n == 0 {
		t.Fatal("expected error response, got empty response")
	}

	// Parse and validate the error response
	response := string(buffer[:n])
	var errorResp struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(buffer[:n], &errorResp); err != nil {
		t.Fatalf("failed to parse error response: %v, response was: %s", err, response)
	}

	if len(errorResp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errorResp.Errors))
	}
	if errorResp.Errors[0].Code != "invalid proxy" {
		t.Errorf("expected error code 'invalid proxy', got %q", errorResp.Errors[0].Code)
	}
	if !strings.Contains(errorResp.Errors[0].Message, "Unable to proxy to any protocol other than tcp") {
		t.Errorf("expected message about non-tcp protocol, got %q", errorResp.Errors[0].Message)
	}
}

// TestTCP_ProxyDNSError tests proxy handling of DNS resolution failures
// mountebank tcpProxyTest.js: "should gracefully deal with DNS errors"
// Note: This test is conditional on airplane mode in mountebank
func TestTCP_ProxyDNSError(t *testing.T) {
	defer cleanup(t)

	// Create proxy pointing to invalid hostname
	proxyPort := 6010
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": "tcp://invalid.domain.that.does.not.exist:9999",
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

	// Connect to proxy
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Send request
	_, err = conn.Write([]byte("test"))
	if err != nil {
		t.Fatalf("failed to write to proxy: %v", err)
	}

	// Should receive JSON error response for DNS failure
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buffer)

	if err != nil {
		t.Fatalf("expected error response, got read error: %v", err)
	}
	if n == 0 {
		t.Fatal("expected error response, got empty response")
	}

	// Parse and validate the error response
	response := string(buffer[:n])
	var errorResp struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(buffer[:n], &errorResp); err != nil {
		t.Fatalf("failed to parse error response: %v, response was: %s", err, response)
	}

	if len(errorResp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errorResp.Errors))
	}
	if errorResp.Errors[0].Code != "invalid proxy" {
		t.Errorf("expected error code 'invalid proxy', got %q", errorResp.Errors[0].Code)
	}
	if !strings.Contains(errorResp.Errors[0].Message, "Cannot resolve") {
		t.Errorf("expected message to contain 'Cannot resolve', got %q", errorResp.Errors[0].Message)
	}
}
