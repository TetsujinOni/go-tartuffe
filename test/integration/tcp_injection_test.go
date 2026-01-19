package integration

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// TestTCP_PredicateInjectionOldInterface tests JavaScript injection in predicates (old interface)
func TestTCP_PredicateInjectionOldInterface(t *testing.T) {
	// Create imposter with injected predicate (old interface: function(request))
	port := 6100
	imposterResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"inject": "function(request) { return request.data.toString() === 'test'; }",
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "MATCHED"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	// Verify imposter created
	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", imposterResp.StatusCode)
	}

	// Send matching request
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send "test" - should match
	if _, err := conn.Write([]byte("test")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if string(response[:n]) != "MATCHED" {
		t.Errorf("expected 'MATCHED', got %q", string(response[:n]))
	}
}

// TestTCP_PredicateInjectionNewInterface tests JavaScript injection in predicates (new interface)
func TestTCP_PredicateInjectionNewInterface(t *testing.T) {
	// Create imposter with injected predicate (new interface: function(config))
	port := 6101
	imposterResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"inject": "function(config) { return config.request.data.toString() === 'test'; }",
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "MATCHED"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	// Verify imposter created
	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", imposterResp.StatusCode)
	}

	// Send matching request
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send "test" - should match
	if _, err := conn.Write([]byte("test")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if string(response[:n]) != "MATCHED" {
		t.Errorf("expected 'MATCHED', got %q", string(response[:n]))
	}
}

// TestTCP_ResponseInjectionOldInterface tests JavaScript injection in responses (old interface)
func TestTCP_ResponseInjectionOldInterface(t *testing.T) {
	// Create imposter with injected response (old interface: function(request))
	port := 6102
	imposterResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": "function(request) { return { data: request.data + ' INJECTED' }; }",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	// Verify imposter created
	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", imposterResp.StatusCode)
	}

	// Send request
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
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

	if string(response[:n]) != "request INJECTED" {
		t.Errorf("expected 'request INJECTED', got %q", string(response[:n]))
	}
}

// TestTCP_ResponseInjectionNewInterface tests JavaScript injection in responses (new interface)
func TestTCP_ResponseInjectionNewInterface(t *testing.T) {
	// Create imposter with injected response (new interface: function(config))
	port := 6103
	imposterResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": "function(config) { return { data: config.request.data + ' INJECTED' }; }",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	// Verify imposter created
	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", imposterResp.StatusCode)
	}

	// Send request
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
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

	if string(response[:n]) != "request INJECTED" {
		t.Errorf("expected 'request INJECTED', got %q", string(response[:n]))
	}
}

// TestTCP_StatefulInjectionOldInterface tests JavaScript injection with state persistence (old interface)
func TestTCP_StatefulInjectionOldInterface(t *testing.T) {
	// Create imposter with stateful injected response (old interface: function(request, state))
	port := 6104
	imposterResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(request, state) {
							if (!state.calls) { state.calls = 0; }
							state.calls += 1;
							return { data: state.calls.toString() };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	// Verify imposter created
	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", imposterResp.StatusCode)
	}

	// First request - should return "1"
	conn1, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn1.Close()

	if _, err := conn1.Write([]byte("request")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	response1 := make([]byte, 1024)
	n1, err := conn1.Read(response1)
	if err != nil {
		t.Fatalf("failed to read first response: %v", err)
	}

	if string(response1[:n1]) != "1" {
		t.Errorf("expected '1' on first request, got %q", string(response1[:n1]))
	}

	// Second request - should return "2"
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

	if string(response2[:n2]) != "2" {
		t.Errorf("expected '2' on second request, got %q", string(response2[:n2]))
	}
}

// TestTCP_StatefulInjectionNewInterface tests JavaScript injection with state persistence (new interface)
func TestTCP_StatefulInjectionNewInterface(t *testing.T) {
	// Create imposter with stateful injected response (new interface: function(config))
	port := 6105
	imposterResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"inject": `function(config) {
							if (!config.state.calls) { config.state.calls = 0; }
							config.state.calls += 1;
							return { data: config.state.calls.toString() };
						}`,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	// Verify imposter created
	if imposterResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", imposterResp.StatusCode)
	}

	// Verify imposter was created successfully (state tracking happens internally)

	// First request - should return "1"
	conn1, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn1.Close()

	if _, err := conn1.Write([]byte("request")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	response1 := make([]byte, 1024)
	n1, err := conn1.Read(response1)
	if err != nil {
		t.Fatalf("failed to read first response: %v", err)
	}

	if string(response1[:n1]) != "1" {
		t.Errorf("expected '1' on first request, got %q", string(response1[:n1]))
	}

	// Second request - should return "2"
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

	if string(response2[:n2]) != "2" {
		t.Errorf("expected '2' on second request, got %q", string(response2[:n2]))
	}
}
