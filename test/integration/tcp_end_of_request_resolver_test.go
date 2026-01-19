package integration

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestTCP_EndOfRequestResolver_Binary tests that endOfRequestResolver works with binary mode
// for requests that span multiple TCP packets. Uses a length-prefixed protocol where
// the first 4 bytes indicate the length of the payload.
func TestTCP_EndOfRequestResolver_Binary(t *testing.T) {
	port := 6200

	// Create a large request: 4-byte length prefix + payload
	payloadSize := 100000 // 100KB payload
	request := make([]byte, 4+payloadSize)
	binary.LittleEndian.PutUint32(request[0:4], uint32(payloadSize))
	// Fill payload with zeros (already done by make)

	// Response to send
	responseData := []byte{0, 1, 2, 3}

	// JavaScript resolver that checks if we have the complete message
	// For binary mode, requestData is a Buffer object
	resolver := `function(requestData) {
		var messageLength = requestData.readUInt32LE(0);
		return requestData.length === messageLength + 4;
	}`

	imposterResp, body, err := post("/imposters", map[string]interface{}{
		"protocol":       "tcp",
		"port":           port,
		"mode":           "binary",
		"recordRequests": true,
		"endOfRequestResolver": map[string]interface{}{
			"inject": resolver,
		},
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": base64.StdEncoding.EncodeToString(responseData)}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d: %v", imposterResp.StatusCode, body)
	}

	// Connect and send the large request
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 5*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send the entire request
	if _, err := conn.Write(request); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Verify response
	if n != len(responseData) {
		t.Errorf("expected response length %d, got %d", len(responseData), n)
	}
	for i := 0; i < n && i < len(responseData); i++ {
		if response[i] != responseData[i] {
			t.Errorf("response byte %d: expected %d, got %d", i, responseData[i], response[i])
		}
	}

	// Give server a moment to record the request
	time.Sleep(100 * time.Millisecond)

	// Verify the request was recorded as a single request
	resp, imposter, err := get(fmt.Sprintf("/imposters/%d", port))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	requests, ok := imposter["requests"].([]interface{})
	if !ok {
		t.Fatalf("expected requests array, got %T", imposter["requests"])
	}

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	// Verify the request data is the entire request (base64 encoded)
	req := requests[0].(map[string]interface{})
	recordedData := req["data"].(string)
	expectedData := base64.StdEncoding.EncodeToString(request)
	if recordedData != expectedData {
		t.Errorf("recorded request data doesn't match expected\nrecorded length: %d\nexpected length: %d",
			len(recordedData), len(expectedData))
	}
}

// TestTCP_EndOfRequestResolver_Text tests that endOfRequestResolver works with text mode
// for requests that span multiple TCP packets. Uses HTTP-like Content-Length protocol.
func TestTCP_EndOfRequestResolver_Text(t *testing.T) {
	port := 6201

	// Create a large request: Content-Length header + body
	bodySize := 100000
	body := strings.Repeat("x", bodySize)
	request := fmt.Sprintf("Content-Length: %d\n\n%s", bodySize, body)

	// JavaScript resolver that checks if we have the complete message
	// For text mode, requestData is a string
	resolver := `function(requestData) {
		var match = /Content-Length: (\d+)/.exec(requestData);
		if (!match) return false;
		var bodyLength = parseInt(match[1]);
		var bodyMatch = /\n\n(.*)/.exec(requestData);
		if (!bodyMatch) return false;
		return bodyMatch[1].length === bodyLength;
	}`

	imposterResp, respBody, err := post("/imposters", map[string]interface{}{
		"protocol":       "tcp",
		"port":           port,
		"mode":           "text",
		"recordRequests": true,
		"endOfRequestResolver": map[string]interface{}{
			"inject": resolver,
		},
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "success"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d: %v", imposterResp.StatusCode, respBody)
	}

	// Connect and send the large request
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 5*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send the entire request
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Verify response
	if string(response[:n]) != "success" {
		t.Errorf("expected 'success', got %q", string(response[:n]))
	}

	// Give server a moment to record the request
	time.Sleep(100 * time.Millisecond)

	// Verify the request was recorded as a single request
	resp, imposter, err := get(fmt.Sprintf("/imposters/%d", port))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	requests, ok := imposter["requests"].([]interface{})
	if !ok {
		t.Fatalf("expected requests array, got %T", imposter["requests"])
	}

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	// Verify the request data is the entire request
	req := requests[0].(map[string]interface{})
	recordedData := req["data"].(string)
	if recordedData != request {
		t.Errorf("recorded request data doesn't match expected\nrecorded length: %d\nexpected length: %d",
			len(recordedData), len(request))
	}
}
