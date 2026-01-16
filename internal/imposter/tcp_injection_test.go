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

// TestTCPPredicateInjection tests JavaScript injection in TCP predicates
func TestTCPPredicateInjection(t *testing.T) {
	tests := []struct {
		name         string
		inject       string
		requestData  string
		shouldMatch  bool
		responseData string
	}{
		{
			name: "simple indexOf injection",
			inject: `function(request, logger) {
				return request.data.indexOf('HELLO') >= 0;
			}`,
			requestData:  "HELLO WORLD",
			shouldMatch:  true,
			responseData: "MATCHED",
		},
		{
			name: "indexOf injection no match",
			inject: `function(request, logger) {
				return request.data.indexOf('GOODBYE') >= 0;
			}`,
			requestData:  "HELLO WORLD",
			shouldMatch:  false,
			responseData: "", // No match, no response
		},
		{
			name: "regex injection",
			inject: `function(request, logger) {
				// Trim newline before testing
				var trimmed = request.data.replace(/\n$/, '');
				return /^[A-Z]+$/.test(trimmed);
			}`,
			requestData:  "ALLCAPS",
			shouldMatch:  true,
			responseData: "CAPS_MATCH",
		},
		{
			name: "logger usage in injection",
			inject: `function(request, logger) {
				logger.info('Checking request: ' + request.data);
				// Trim newline for comparison
				return request.data.replace(/\n$/, '') === 'TEST';
			}`,
			requestData:  "TEST",
			shouldMatch:  true,
			responseData: "LOGGED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := 9700 + (len(tt.name) % 100)

			imp := &models.Imposter{
				Protocol: "tcp",
				Port:     port,
				Mode:     "text",
				Stubs: []models.Stub{
					{
						Predicates: []models.Predicate{
							{Inject: tt.inject},
						},
						Responses: []models.Response{
							{Is: &models.IsResponse{Data: tt.responseData}},
						},
					},
					{
						// Default response if no match
						Responses: []models.Response{
							{Is: &models.IsResponse{Data: "DEFAULT"}},
						},
					},
				},
			}

			srv, err := NewTCPServer(imp)
			if err != nil {
				t.Fatalf("NewTCPServer() error = %v", err)
			}

			if err := srv.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			defer srv.Stop(context.Background())

			time.Sleep(50 * time.Millisecond)

			// Connect and send request
			addr := fmt.Sprintf("localhost:%d", port)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			conn.Write([]byte(tt.requestData + "\n"))

			// Read response
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			response := make([]byte, 1024)
			n, _ := conn.Read(response)

			got := string(response[:n])
			if tt.shouldMatch {
				if got != tt.responseData {
					t.Errorf("Response = %q, want %q (injection should have matched)", got, tt.responseData)
				}
			} else {
				if got != "DEFAULT" {
					t.Errorf("Response = %q, want %q (injection should not have matched, using default)", got, "DEFAULT")
				}
			}
		})
	}
}

// TestTCPResponseInjection tests JavaScript injection in TCP responses
func TestTCPResponseInjection(t *testing.T) {
	tests := []struct {
		name         string
		inject       string
		requestData  string
		wantResponse string
	}{
		{
			name: "echo request data",
			inject: `function(request, state, logger) {
				// Trim newline from request data
				var trimmed = request.data.replace(/\n$/, '');
				return { data: 'ECHO: ' + trimmed };
			}`,
			requestData:  "TEST",
			wantResponse: "ECHO: TEST",
		},
		{
			name: "transform request",
			inject: `function(request, state, logger) {
				// Trim newline before transforming
				var trimmed = request.data.replace(/\n$/, '');
				return { data: trimmed.toUpperCase() };
			}`,
			requestData:  "lowercase",
			wantResponse: "LOWERCASE",
		},
		{
			name: "state management",
			inject: `function(request, state, logger) {
				state.counter = (state.counter || 0) + 1;
				return { data: 'COUNT:' + state.counter };
			}`,
			requestData:  "TEST",
			wantResponse: "COUNT:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := 9800 + (len(tt.name) % 100)

			imp := &models.Imposter{
				Protocol: "tcp",
				Port:     port,
				Mode:     "text",
				Stubs: []models.Stub{
					{
						Responses: []models.Response{
							{Inject: tt.inject},
						},
					},
				},
			}

			srv, err := NewTCPServer(imp)
			if err != nil {
				t.Fatalf("NewTCPServer() error = %v", err)
			}

			if err := srv.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			defer srv.Stop(context.Background())

			time.Sleep(50 * time.Millisecond)

			// Connect and send request
			addr := fmt.Sprintf("localhost:%d", port)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			conn.Write([]byte(tt.requestData + "\n"))

			// Read response
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			response := make([]byte, 1024)
			n, _ := conn.Read(response)

			got := string(response[:n])
			if got != tt.wantResponse {
				t.Errorf("Response = %q, want %q", got, tt.wantResponse)
			}
		})
	}
}

// TestTCPInjectionWithBinaryMode tests injection in binary mode
func TestTCPInjectionWithBinaryMode(t *testing.T) {
	port := 9900

	// Predicate injection that checks for specific binary data
	imp := &models.Imposter{
		Protocol: "tcp",
		Port:     port,
		Mode:     "binary",
		Stubs: []models.Stub{
			{
				Predicates: []models.Predicate{
					{
						Inject: `function(request, logger) {
							// In binary mode, request.data is base64 encoded
							// Decode and check for specific bytes
							return request.data.indexOf('AQI=') >= 0; // Base64 for [0x01, 0x02]
						}`,
					},
				},
				Responses: []models.Response{
					{Is: &models.IsResponse{
						Data: base64.StdEncoding.EncodeToString([]byte{0x03, 0x04}),
					}},
				},
			},
		},
	}

	srv, err := NewTCPServer(imp)
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Send binary data
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte{0x01, 0x02})

	// Read response
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)

	want := []byte{0x03, 0x04}
	got := response[:n]
	if len(got) != len(want) {
		t.Fatalf("Response length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Response[%d] = 0x%02x, want 0x%02x", i, got[i], want[i])
		}
	}
}

// TestTCPInjectionStateAcrossRequests tests that state persists across multiple requests
func TestTCPInjectionStateAcrossRequests(t *testing.T) {
	port := 9901

	imp := &models.Imposter{
		Protocol: "tcp",
		Port:     port,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Inject: `function(request, state, logger) {
							state.counter = (state.counter || 0) + 1;
							return { data: String(state.counter) };
						}`,
					},
				},
			},
		},
	}

	srv, err := NewTCPServer(imp)
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	addr := fmt.Sprintf("localhost:%d", port)

	// Make 3 requests, each should increment counter
	for i := 1; i <= 3; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("connection %d failed: %v", i, err)
		}

		conn.Write([]byte("TEST\n"))

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		response := make([]byte, 1024)
		n, _ := conn.Read(response)

		got := string(response[:n])
		want := fmt.Sprintf("%d", i)
		if got != want {
			t.Errorf("Request %d: Response = %q, want %q (state should persist)", i, got, want)
		}

		conn.Close()
	}
}

// TestTCPInjectionErrorHandling tests error handling in injection functions
func TestTCPInjectionErrorHandling(t *testing.T) {
	port := 9902

	imp := &models.Imposter{
		Protocol: "tcp",
		Port:     port,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Predicates: []models.Predicate{
					{
						Inject: `function(request, logger) {
							// Syntax error - missing closing brace
							return request.data === 'TEST'
						`,
					},
				},
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "MATCH"}},
				},
			},
			{
				// Default stub
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "DEFAULT"}},
				},
			},
		},
	}

	srv, err := NewTCPServer(imp)
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Send request
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("TEST\n"))

	// Read response
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)

	// On error, should fall through to default stub
	got := string(response[:n])
	if got != "DEFAULT" {
		t.Errorf("Response = %q, want %q (should use default on injection error)", got, "DEFAULT")
	}
}

// TestTCPMultipleInjectionsInStub tests multiple injection predicates in a single stub
func TestTCPMultipleInjectionsInStub(t *testing.T) {
	port := 9903

	imp := &models.Imposter{
		Protocol: "tcp",
		Port:     port,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Predicates: []models.Predicate{
					{
						// First predicate checks length
						Inject: `function(request, logger) {
							return request.data.length > 5;
						}`,
					},
					{
						// Second predicate checks content
						Inject: `function(request, logger) {
							return request.data.indexOf('TEST') >= 0;
						}`,
					},
				},
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "BOTH_MATCHED"}},
				},
			},
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{Data: "NO_MATCH"}},
				},
			},
		},
	}

	srv, err := NewTCPServer(imp)
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(50 * time.Millisecond)

	tests := []struct {
		name     string
		request  string
		wantResp string
	}{
		{"both match", "TEST DATA", "BOTH_MATCHED"},
		{"length fails", "TEST", "NO_MATCH"},
		{"content fails", "LONG STRING", "NO_MATCH"},
	}

	addr := fmt.Sprintf("localhost:%d", port)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := net.Dial("tcp", addr)
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
