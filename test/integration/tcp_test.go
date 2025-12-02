package integration

import (
	"bufio"
	"encoding/base64"
	"net"
	"testing"
	"time"
)

// TCP protocol tests

func TestTCP_BasicResponse(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5700,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"data": "Hello from TCP!",
						},
					},
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

	time.Sleep(100 * time.Millisecond)

	// Connect to TCP server
	conn, err := net.Dial("tcp", "localhost:5700")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send some data
	conn.Write([]byte("test"))

	// Read response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	response := string(buffer[:n])
	if response != "Hello from TCP!" {
		t.Errorf("expected 'Hello from TCP!', got '%s'", response)
	}
}

func TestTCP_PredicateEquals(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5701,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"data": "PING"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "PONG"}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"data": "HELLO"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "WORLD"}},
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

	time.Sleep(100 * time.Millisecond)

	// Test PING -> PONG
	conn1, err := net.Dial("tcp", "localhost:5701")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn1.Write([]byte("PING"))
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, _ := conn1.Read(buffer)
	conn1.Close()

	if string(buffer[:n]) != "PONG" {
		t.Errorf("expected 'PONG', got '%s'", string(buffer[:n]))
	}

	// Test HELLO -> WORLD
	conn2, err := net.Dial("tcp", "localhost:5701")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn2.Write([]byte("HELLO"))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn2.Read(buffer)
	conn2.Close()

	if string(buffer[:n]) != "WORLD" {
		t.Errorf("expected 'WORLD', got '%s'", string(buffer[:n]))
	}
}

func TestTCP_PredicateContains(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5702,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"contains": map[string]interface{}{"data": "error"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "ERROR_RESPONSE"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "OK_RESPONSE"}},
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

	time.Sleep(100 * time.Millisecond)

	// Test with "error" in message
	conn1, err := net.Dial("tcp", "localhost:5702")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn1.Write([]byte("this is an error message"))
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, _ := conn1.Read(buffer)
	conn1.Close()

	if string(buffer[:n]) != "ERROR_RESPONSE" {
		t.Errorf("expected 'ERROR_RESPONSE', got '%s'", string(buffer[:n]))
	}

	// Test without "error"
	conn2, err := net.Dial("tcp", "localhost:5702")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn2.Write([]byte("normal message"))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn2.Read(buffer)
	conn2.Close()

	if string(buffer[:n]) != "OK_RESPONSE" {
		t.Errorf("expected 'OK_RESPONSE', got '%s'", string(buffer[:n]))
	}
}

func TestTCP_PredicateMatches(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5703,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"matches": map[string]interface{}{"data": "^GET .*"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "HTTP/1.1 200 OK"}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"matches": map[string]interface{}{"data": "^POST .*"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "HTTP/1.1 201 Created"}},
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

	time.Sleep(100 * time.Millisecond)

	// Test GET request
	conn1, err := net.Dial("tcp", "localhost:5703")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn1.Write([]byte("GET /index.html HTTP/1.1"))
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, _ := conn1.Read(buffer)
	conn1.Close()

	if string(buffer[:n]) != "HTTP/1.1 200 OK" {
		t.Errorf("expected 'HTTP/1.1 200 OK', got '%s'", string(buffer[:n]))
	}

	// Test POST request
	conn2, err := net.Dial("tcp", "localhost:5703")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn2.Write([]byte("POST /submit HTTP/1.1"))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn2.Read(buffer)
	conn2.Close()

	if string(buffer[:n]) != "HTTP/1.1 201 Created" {
		t.Errorf("expected 'HTTP/1.1 201 Created', got '%s'", string(buffer[:n]))
	}
}

func TestTCP_RecordRequests(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "tcp",
		"port":           5704,
		"recordRequests": true,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "ACK"}},
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

	time.Sleep(100 * time.Millisecond)

	// Send some requests
	for _, msg := range []string{"Message1", "Message2", "Message3"} {
		conn, err := net.Dial("tcp", "localhost:5704")
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		conn.Write([]byte(msg))
		buffer := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.Read(buffer)
		conn.Close()
	}

	// Check recorded requests
	getResp, body, err := get("/imposters/5704")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	tcpRequests, ok := body["tcpRequests"].([]interface{})
	if !ok {
		t.Fatal("expected tcpRequests array")
	}

	if len(tcpRequests) != 3 {
		t.Errorf("expected 3 recorded requests, got %d", len(tcpRequests))
	}

	// Verify first request
	if len(tcpRequests) > 0 {
		firstReq := tcpRequests[0].(map[string]interface{})
		if firstReq["data"] != "Message1" {
			t.Errorf("expected first request data 'Message1', got '%v'", firstReq["data"])
		}
	}
}

func TestTCP_BinaryMode(t *testing.T) {
	defer cleanup(t)

	// Binary data to send (raw bytes)
	binaryData := []byte{0x01, 0x02, 0x03, 0x04, 0xFF}
	binaryResponse := []byte{0xAA, 0xBB, 0xCC}

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5705,
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
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Connect and send binary data
	conn, err := net.Dial("tcp", "localhost:5705")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	conn.Write(binaryData)

	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Verify we got the binary response back
	response := buffer[:n]
	if len(response) != len(binaryResponse) {
		t.Errorf("expected %d bytes, got %d", len(binaryResponse), len(response))
	}
	for i := range binaryResponse {
		if i < len(response) && response[i] != binaryResponse[i] {
			t.Errorf("byte %d mismatch: expected 0x%02X, got 0x%02X", i, binaryResponse[i], response[i])
		}
	}
}

func TestTCP_PredicateStartsWith(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5706,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"startsWith": map[string]interface{}{"data": "CMD:"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "COMMAND_RECEIVED"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "UNKNOWN"}},
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

	time.Sleep(100 * time.Millisecond)

	// Test command message
	conn1, err := net.Dial("tcp", "localhost:5706")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn1.Write([]byte("CMD:START"))
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, _ := conn1.Read(buffer)
	conn1.Close()

	if string(buffer[:n]) != "COMMAND_RECEIVED" {
		t.Errorf("expected 'COMMAND_RECEIVED', got '%s'", string(buffer[:n]))
	}

	// Test non-command message
	conn2, err := net.Dial("tcp", "localhost:5706")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn2.Write([]byte("DATA:something"))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn2.Read(buffer)
	conn2.Close()

	if string(buffer[:n]) != "UNKNOWN" {
		t.Errorf("expected 'UNKNOWN', got '%s'", string(buffer[:n]))
	}
}

func TestTCP_DefaultResponse(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5707,
		"defaultResponse": map[string]interface{}{
			"is": map[string]interface{}{
				"data": "DEFAULT",
			},
		},
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"data": "SPECIAL"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "SPECIAL_RESPONSE"}},
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

	time.Sleep(100 * time.Millisecond)

	// Test matching stub
	conn1, err := net.Dial("tcp", "localhost:5707")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn1.Write([]byte("SPECIAL"))
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, _ := conn1.Read(buffer)
	conn1.Close()

	if string(buffer[:n]) != "SPECIAL_RESPONSE" {
		t.Errorf("expected 'SPECIAL_RESPONSE', got '%s'", string(buffer[:n]))
	}

	// Test default response
	conn2, err := net.Dial("tcp", "localhost:5707")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn2.Write([]byte("ANYTHING_ELSE"))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn2.Read(buffer)
	conn2.Close()

	if string(buffer[:n]) != "DEFAULT" {
		t.Errorf("expected 'DEFAULT', got '%s'", string(buffer[:n]))
	}
}

func TestTCP_MultiplePorts(t *testing.T) {
	defer cleanup(t)

	// Create two TCP imposters on different ports
	resp1, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5708,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "SERVER1"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter 1: %v", err)
	}
	if resp1.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp1.StatusCode)
	}

	resp2, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5709,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "SERVER2"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter 2: %v", err)
	}
	if resp2.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp2.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Test server 1
	conn1, err := net.Dial("tcp", "localhost:5708")
	if err != nil {
		t.Fatalf("failed to connect to server 1: %v", err)
	}
	conn1.Write([]byte("test"))
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, _ := conn1.Read(buffer)
	conn1.Close()

	if string(buffer[:n]) != "SERVER1" {
		t.Errorf("expected 'SERVER1', got '%s'", string(buffer[:n]))
	}

	// Test server 2
	conn2, err := net.Dial("tcp", "localhost:5709")
	if err != nil {
		t.Fatalf("failed to connect to server 2: %v", err)
	}
	conn2.Write([]byte("test"))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn2.Read(buffer)
	conn2.Close()

	if string(buffer[:n]) != "SERVER2" {
		t.Errorf("expected 'SERVER2', got '%s'", string(buffer[:n]))
	}
}

func TestTCP_LineProtocol(t *testing.T) {
	defer cleanup(t)

	// Simulate a line-based protocol like SMTP or Redis
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     5710,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"startsWith": map[string]interface{}{"data": "EHLO"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "250 Hello\r\n"}},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"startsWith": map[string]interface{}{"data": "QUIT"}},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "221 Bye\r\n"}},
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

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:5710")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Send EHLO
	conn.Write([]byte("EHLO localhost\r\n"))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, _ := reader.ReadString('\n')

	if line != "250 Hello\r\n" {
		t.Errorf("expected '250 Hello\\r\\n', got '%q'", line)
	}
}

// Helper function to suppress unused import error
var _ = bufio.NewReader
