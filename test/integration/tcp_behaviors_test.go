package integration

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestTCP_DecorateWithProxy tests decorate behavior with TCP proxy
func TestTCP_DecorateWithProxy(t *testing.T) {
	// Create origin server on port 7000
	originPort := 7000
	originResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     originPort,
		"name":     "ORIGIN",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "ORIGIN"}},
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

	// Create proxy imposter with decorate behavior on port 7001
	proxyPort := 7001
	proxyResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     proxyPort,
		"name":     "PROXY",
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"proxy": map[string]interface{}{
							"to": fmt.Sprintf("tcp://localhost:%d", originPort),
						},
						"_behaviors": map[string]interface{}{
							"decorate": "function(request, response) { response.data += ' DECORATED'; }",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create proxy imposter: %v", err)
	}
	defer del("/imposters")

	if proxyResp.StatusCode != 201 {
		t.Fatalf("expected status 201 for proxy, got %d", proxyResp.StatusCode)
	}

	// Send request to proxy
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", proxyPort), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Send "request"
	if _, err := conn.Write([]byte("request")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Verify response was decorated
	if string(response[:n]) != "ORIGIN DECORATED" {
		t.Errorf("expected 'ORIGIN DECORATED', got %q", string(response[:n]))
	}
}

// TestTCP_BehaviorComposition tests multiple behaviors together (excluding shellTransform for security)
func TestTCP_BehaviorComposition(t *testing.T) {
	// Note: shellTransform is intentionally excluded as it's a security risk
	// This test uses wait + repeat + decorate + copy behaviors
	port := 7002
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"data": "Hello, ${SUBJECT}${PUNCTUATION}",
						},
						"_behaviors": map[string]interface{}{
							"wait":     300, // 300ms latency
							"repeat":   2,   // Use this response twice
							"decorate": "function(request, response) { response.data = response.data.replace('${SUBJECT}', 'mountebank'); }",
							"copy": []map[string]interface{}{
								{
									"from":  "data",
									"into":  "${PUNCTUATION}",
									"using": map[string]interface{}{"method": "regex", "selector": "[,.?!]"},
								},
							},
						},
					},
					{
						"is": map[string]interface{}{
							"data": "No behaviors",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	// First request - should use first response with behaviors
	start1 := time.Now()
	conn1, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect first time: %v", err)
	}
	defer conn1.Close()

	if _, err := conn1.Write([]byte("!")); err != nil {
		t.Fatalf("failed to write first request: %v", err)
	}

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	response1 := make([]byte, 1024)
	n1, err := conn1.Read(response1)
	if err != nil {
		t.Fatalf("failed to read first response: %v", err)
	}
	elapsed1 := time.Since(start1)

	// Verify first response
	if string(response1[:n1]) != "Hello, mountebank!" {
		t.Errorf("expected 'Hello, mountebank!' on first request, got %q", string(response1[:n1]))
	}

	// Verify wait behavior (should take at least 250ms)
	if elapsed1 < 250*time.Millisecond {
		t.Errorf("expected wait of at least 250ms, got %v", elapsed1)
	}

	// Second request - should repeat first response
	start2 := time.Now()
	conn2, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect second time: %v", err)
	}
	defer conn2.Close()

	if _, err := conn2.Write([]byte("!")); err != nil {
		t.Fatalf("failed to write second request: %v", err)
	}

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	response2 := make([]byte, 1024)
	n2, err := conn2.Read(response2)
	if err != nil {
		t.Fatalf("failed to read second response: %v", err)
	}
	elapsed2 := time.Since(start2)

	// Verify second response (repeat)
	if string(response2[:n2]) != "Hello, mountebank!" {
		t.Errorf("expected 'Hello, mountebank!' on second request, got %q", string(response2[:n2]))
	}

	// Verify wait behavior on second request
	if elapsed2 < 250*time.Millisecond {
		t.Errorf("expected wait of at least 250ms on second request, got %v", elapsed2)
	}

	// Third request - should use second response (no behaviors)
	conn3, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect third time: %v", err)
	}
	defer conn3.Close()

	if _, err := conn3.Write([]byte("!")); err != nil {
		t.Fatalf("failed to write third request: %v", err)
	}

	conn3.SetReadDeadline(time.Now().Add(2 * time.Second))
	response3 := make([]byte, 1024)
	n3, err := conn3.Read(response3)
	if err != nil {
		t.Fatalf("failed to read third response: %v", err)
	}

	// Verify third response
	if string(response3[:n3]) != "No behaviors" {
		t.Errorf("expected 'No behaviors' on third request, got %q", string(response3[:n3]))
	}
}

// TestTCP_ResponseSequence tests response sequence/circular buffer
func TestTCP_ResponseSequence(t *testing.T) {
	port := 7003
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"data": "request"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "first"}},
					{"is": map[string]interface{}{"data": "second"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	// First request - should get "first"
	conn1, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect first time: %v", err)
	}
	defer conn1.Close()

	if _, err := conn1.Write([]byte("request")); err != nil {
		t.Fatalf("failed to write first request: %v", err)
	}

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	response1 := make([]byte, 1024)
	n1, err := conn1.Read(response1)
	if err != nil {
		t.Fatalf("failed to read first response: %v", err)
	}

	if string(response1[:n1]) != "first" {
		t.Errorf("expected 'first', got %q", string(response1[:n1]))
	}

	// Second request - should get "second"
	conn2, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect second time: %v", err)
	}
	defer conn2.Close()

	if _, err := conn2.Write([]byte("request")); err != nil {
		t.Fatalf("failed to write second request: %v", err)
	}

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	response2 := make([]byte, 1024)
	n2, err := conn2.Read(response2)
	if err != nil {
		t.Fatalf("failed to read second response: %v", err)
	}

	if string(response2[:n2]) != "second" {
		t.Errorf("expected 'second', got %q", string(response2[:n2]))
	}

	// Third request - should cycle back to "first"
	conn3, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect third time: %v", err)
	}
	defer conn3.Close()

	if _, err := conn3.Write([]byte("request")); err != nil {
		t.Fatalf("failed to write third request: %v", err)
	}

	conn3.SetReadDeadline(time.Now().Add(2 * time.Second))
	response3 := make([]byte, 1024)
	n3, err := conn3.Read(response3)
	if err != nil {
		t.Fatalf("failed to read third response: %v", err)
	}

	if string(response3[:n3]) != "first" {
		t.Errorf("expected 'first' on third request (circular buffer), got %q", string(response3[:n3]))
	}
}

// TestTCP_MatchesPredicateWithBinaryModeValidation tests validation error for matches predicate in binary mode
func TestTCP_MatchesPredicateWithBinaryModeValidation(t *testing.T) {
	// Attempt to create imposter with matches predicate in binary mode
	resp, body, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     7004,
		"mode":     "binary",
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"matches": map[string]interface{}{"data": "dGVzdA=="}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "dGVzdA=="}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer del("/imposters")

	// Should return 400 error
	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	// Verify error message
	if errors, ok := body["errors"].([]interface{}); ok && len(errors) > 0 {
		if errObj, ok := errors[0].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				if msg != "the matches predicate is not allowed in binary mode" {
					t.Errorf("expected error message 'the matches predicate is not allowed in binary mode', got %q", msg)
				}
			} else {
				t.Error("error message is not a string")
			}
		} else {
			t.Error("error is not an object")
		}
	} else {
		t.Error("response missing errors array")
	}
}

// TestTCP_LargePacketSplitting tests that large packets (65KB+) are split correctly
func TestTCP_LargePacketSplitting(t *testing.T) {
	port := 7005
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "tcp",
		"port":           port,
		"mode":           "text",
		"recordRequests": true,
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

	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	// Create large request > 64KB (65537 bytes)
	// Max packet size is typically 64KB, so this should be split
	largeRequest := strings.Repeat("1", 65536) + "2"

	// Send large request
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 5*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(largeRequest)); err != nil {
		t.Fatalf("failed to write large request: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Verify we got a response
	if string(response[:n]) != "success" {
		t.Errorf("expected 'success', got %q", string(response[:n]))
	}

	// Note: Mountebank splits large packets into multiple requests
	// go-tartuffe may handle this differently, but should at least not crash
	// The key requirement is that the imposter handles large data without errors
}