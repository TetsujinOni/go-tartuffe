package imposter

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestTCPProxyBasicForwarding tests basic TCP proxy forwarding
func TestTCPProxyBasicForwarding(t *testing.T) {
	// Start origin TCP server
	originPort := 9700
	originImp := &models.Imposter{
		Protocol: "tcp",
		Port:     originPort,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "ORIGIN_RESPONSE"}},
				},
			},
		},
	}

	originSrv, err := NewTCPServer(originImp)
	if err != nil {
		t.Fatalf("NewTCPServer(origin) error = %v", err)
	}

	if err := originSrv.Start(); err != nil {
		t.Fatalf("origin Start() error = %v", err)
	}
	defer originSrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Create proxy imposter
	proxyPort := 9701
	proxyImp := &models.Imposter{
		Protocol: "tcp",
		Port:     proxyPort,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Proxy: &models.ProxyResponse{
							To: fmt.Sprintf("tcp://localhost:%d", originPort),
						},
					},
				},
			},
		},
	}

	proxySrv, err := NewTCPServer(proxyImp)
	if err != nil {
		t.Fatalf("NewTCPServer(proxy) error = %v", err)
	}

	if err := proxySrv.Start(); err != nil {
		t.Fatalf("proxy Start() error = %v", err)
	}
	defer proxySrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Connect to proxy and send request
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("TEST\n"))

	// Read response from proxy (should be from origin)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)

	got := string(response[:n])
	want := "ORIGIN_RESPONSE"
	if got != want {
		t.Errorf("Proxy response = %q, want %q (should forward from origin)", got, want)
	}
}

// TestTCPProxyBinaryData tests TCP proxy with binary data
func TestTCPProxyBinaryData(t *testing.T) {
	// Start origin TCP server in binary mode
	originPort := 9702
	originResponse := []byte{0x05, 0x06, 0x07, 0x08}
	originImp := &models.Imposter{
		Protocol: "tcp",
		Port:     originPort,
		Mode:     "binary",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{
						Data: base64.StdEncoding.EncodeToString(originResponse),
					}},
				},
			},
		},
	}

	originSrv, err := NewTCPServer(originImp)
	if err != nil {
		t.Fatalf("NewTCPServer(origin) error = %v", err)
	}

	if err := originSrv.Start(); err != nil {
		t.Fatalf("origin Start() error = %v", err)
	}
	defer originSrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Create proxy imposter in binary mode
	proxyPort := 9703
	proxyImp := &models.Imposter{
		Protocol: "tcp",
		Port:     proxyPort,
		Mode:     "binary",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Proxy: &models.ProxyResponse{
							To: fmt.Sprintf("tcp://localhost:%d", originPort),
						},
					},
				},
			},
		},
	}

	proxySrv, err := NewTCPServer(proxyImp)
	if err != nil {
		t.Fatalf("NewTCPServer(proxy) error = %v", err)
	}

	if err := proxySrv.Start(); err != nil {
		t.Fatalf("proxy Start() error = %v", err)
	}
	defer proxySrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Connect to proxy and send binary data
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte{0x01, 0x02, 0x03, 0x04})

	// Read binary response
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)

	got := response[:n]
	if len(got) != len(originResponse) {
		t.Fatalf("Response length = %d, want %d", len(got), len(originResponse))
	}
	for i := range originResponse {
		if got[i] != originResponse[i] {
			t.Errorf("Response[%d] = 0x%02x, want 0x%02x", i, got[i], originResponse[i])
		}
	}
}

// TestTCPProxyWithPredicateMatching tests proxy with predicate matching
func TestTCPProxyWithPredicateMatching(t *testing.T) {
	// Start origin server
	originPort := 9704
	originImp := &models.Imposter{
		Protocol: "tcp",
		Port:     originPort,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "PROXIED"}},
				},
			},
		},
	}

	originSrv, err := NewTCPServer(originImp)
	if err != nil {
		t.Fatalf("NewTCPServer(origin) error = %v", err)
	}

	if err := originSrv.Start(); err != nil {
		t.Fatalf("origin Start() error = %v", err)
	}
	defer originSrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Create proxy with predicate matching
	proxyPort := 9705
	proxyImp := &models.Imposter{
		Protocol: "tcp",
		Port:     proxyPort,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				// Only proxy if request contains "PROXY"
				Predicates: []models.Predicate{
					{Contains: map[string]interface{}{"data": "PROXY"}},
				},
				Responses: []models.Response{
					{
						Proxy: &models.ProxyResponse{
							To: fmt.Sprintf("tcp://localhost:%d", originPort),
						},
					},
				},
			},
			{
				// Default response if no match
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "LOCAL"}},
				},
			},
		},
	}

	proxySrv, err := NewTCPServer(proxyImp)
	if err != nil {
		t.Fatalf("NewTCPServer(proxy) error = %v", err)
	}

	if err := proxySrv.Start(); err != nil {
		t.Fatalf("proxy Start() error = %v", err)
	}
	defer proxySrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	tests := []struct {
		name     string
		request  string
		wantResp string
	}{
		{"matches predicate - proxied", "PROXY_REQUEST", "PROXIED"},
		{"no match - local response", "OTHER_REQUEST", "LOCAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
			if err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			conn.Write([]byte(tt.request + "\n"))

			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			response := make([]byte, 1024)
			n, _ := conn.Read(response)

			got := string(response[:n])
			if got != tt.wantResp {
				t.Errorf("Response = %q, want %q", got, tt.wantResp)
			}
		})
	}
}

// TestTCPProxyWithEndOfRequestResolver tests proxy with custom request resolver
func TestTCPProxyWithEndOfRequestResolver(t *testing.T) {
	// Start origin server
	originPort := 9706
	originImp := &models.Imposter{
		Protocol: "tcp",
		Port:     originPort,
		Mode:     "text",
		EndOfRequestResolver: &models.EndOfRequestResolver{
			Inject: `function(requestData, logger) { return requestData.indexOf('\n') >= 0; }`,
		},
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "COMPLETE"}},
				},
			},
		},
	}

	originSrv, err := NewTCPServer(originImp)
	if err != nil {
		t.Fatalf("NewTCPServer(origin) error = %v", err)
	}

	if err := originSrv.Start(); err != nil {
		t.Fatalf("origin Start() error = %v", err)
	}
	defer originSrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Create proxy with same resolver
	proxyPort := 9707
	proxyImp := &models.Imposter{
		Protocol: "tcp",
		Port:     proxyPort,
		Mode:     "text",
		EndOfRequestResolver: &models.EndOfRequestResolver{
			Inject: `function(requestData, logger) { return requestData.indexOf('\n') >= 0; }`,
		},
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Proxy: &models.ProxyResponse{
							To: fmt.Sprintf("tcp://localhost:%d", originPort),
						},
					},
				},
			},
		},
	}

	proxySrv, err := NewTCPServer(proxyImp)
	if err != nil {
		t.Fatalf("NewTCPServer(proxy) error = %v", err)
	}

	if err := proxySrv.Start(); err != nil {
		t.Fatalf("proxy Start() error = %v", err)
	}
	defer proxySrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Send multi-part request
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send in chunks - resolver should wait for newline
	conn.Write([]byte("MULTI"))
	time.Sleep(10 * time.Millisecond)
	conn.Write([]byte("PART"))
	time.Sleep(10 * time.Millisecond)
	conn.Write([]byte("\n"))

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)

	got := string(response[:n])
	want := "COMPLETE"
	if got != want {
		t.Errorf("Response = %q, want %q", got, want)
	}
}

// TestTCPProxyDNSError tests proxy behavior with DNS resolution error
func TestTCPProxyDNSError(t *testing.T) {
	// Create proxy pointing to non-existent host
	proxyPort := 9708
	proxyImp := &models.Imposter{
		Protocol: "tcp",
		Port:     proxyPort,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Proxy: &models.ProxyResponse{
							To: "tcp://nonexistent.invalid.host:9999",
						},
					},
				},
			},
		},
	}

	proxySrv, err := NewTCPServer(proxyImp)
	if err != nil {
		t.Fatalf("NewTCPServer(proxy) error = %v", err)
	}

	if err := proxySrv.Start(); err != nil {
		t.Fatalf("proxy Start() error = %v", err)
	}
	defer proxySrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Connect and send request - should handle DNS error gracefully
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("TEST\n"))

	// Should either close connection or return error response
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)

	// Connection should close (n == 0) or return empty response - both are acceptable
	// The key is that it handles the error gracefully without crashing
	if n > 0 {
		t.Logf("Got response despite DNS error: %q (acceptable - error handled gracefully)", string(response[:n]))
	} else {
		t.Logf("Connection closed on DNS error (acceptable - error handled gracefully)")
	}
}

// TestTCPProxyConnectionRefused tests proxy behavior when origin refuses connection
func TestTCPProxyConnectionRefused(t *testing.T) {
	// Create proxy pointing to port that's not listening
	proxyPort := 9709
	closedPort := 9710 // No server on this port

	proxyImp := &models.Imposter{
		Protocol: "tcp",
		Port:     proxyPort,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Proxy: &models.ProxyResponse{
							To: fmt.Sprintf("tcp://localhost:%d", closedPort),
						},
					},
				},
			},
		},
	}

	proxySrv, err := NewTCPServer(proxyImp)
	if err != nil {
		t.Fatalf("NewTCPServer(proxy) error = %v", err)
	}

	if err := proxySrv.Start(); err != nil {
		t.Fatalf("proxy Start() error = %v", err)
	}
	defer proxySrv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Connect and send request - should handle connection refused gracefully
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxyPort))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("TEST\n"))

	// Should handle connection refused gracefully
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)

	// Connection should close (n == 0) or return empty response - both are acceptable
	// The key is that it handles the error gracefully without crashing
	if n > 0 {
		t.Logf("Got response despite connection refused: %q (acceptable - error handled gracefully)", string(response[:n]))
	} else {
		t.Logf("Connection closed on connection refused (acceptable - error handled gracefully)")
	}
}
