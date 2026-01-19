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

// TestTCPBinaryMode tests TCP server in binary mode
func TestTCPBinaryMode(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		port         int
		requestData  []byte
		responseData string
		wantResponse []byte
	}{
		{
			name:         "binary mode with base64 encoded response",
			mode:         "binary",
			port:         9001,
			requestData:  []byte{0x01, 0x02, 0x03, 0x04},
			responseData: base64.StdEncoding.EncodeToString([]byte{0x05, 0x06, 0x07, 0x08}),
			wantResponse: []byte{0x05, 0x06, 0x07, 0x08},
		},
		{
			name:         "text mode with plain text",
			mode:         "text",
			port:         9002,
			requestData:  []byte("HELLO"),
			responseData: "WORLD",
			wantResponse: []byte("WORLD"),
		},
		{
			name:         "binary mode request stores base64",
			mode:         "binary",
			port:         9003,
			requestData:  []byte{0xFF, 0xFE, 0xFD},
			responseData: base64.StdEncoding.EncodeToString([]byte{0xAA, 0xBB}),
			wantResponse: []byte{0xAA, 0xBB},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create imposter with binary/text mode
			imp := &models.Imposter{
				Protocol: "tcp",
				Port:     tt.port,
				Mode:     tt.mode,
				Stubs: []models.Stub{
					{
						Responses: []models.Response{
							{Is: &models.IsResponse{Data: tt.responseData}},
						},
					},
				},
				RecordRequests: true,
			}

			// Start server
			srv, err := NewTCPServer(imp)
			if err != nil {
				t.Fatalf("NewTCPServer() error = %v", err)
			}

			if err := srv.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			defer srv.Stop(context.Background())

			// Give server time to start
			time.Sleep(50 * time.Millisecond)

			// Connect and send request
			addr := fmt.Sprintf("localhost:%d", imp.Port)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			// Send request
			if _, err := conn.Write(tt.requestData); err != nil {
				t.Fatalf("failed to write: %v", err)
			}

			// Read response
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			response := make([]byte, 1024)
			n, err := conn.Read(response)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}

			// Verify response
			got := response[:n]
			if string(got) != string(tt.wantResponse) {
				t.Errorf("response mismatch:\ngot:  %v (%s)\nwant: %v (%s)",
					got, string(got), tt.wantResponse, string(tt.wantResponse))
			}

			// Verify request was recorded correctly
			time.Sleep(10 * time.Millisecond) // Give time for recording
			storedImp := srv.GetImposter()
			if len(storedImp.TCPRequests) != 1 {
				t.Fatalf("expected 1 recorded request, got %d", len(storedImp.TCPRequests))
			}

			recorded := storedImp.TCPRequests[0].Data
			if tt.mode == "binary" {
				// In binary mode, data should be base64 encoded
				expected := base64.StdEncoding.EncodeToString(tt.requestData)
				if recorded != expected {
					t.Errorf("recorded request data mismatch in binary mode:\ngot:  %s\nwant: %s",
						recorded, expected)
				}
			} else {
				// In text mode, data should be plain text
				if recorded != string(tt.requestData) {
					t.Errorf("recorded request data mismatch in text mode:\ngot:  %s\nwant: %s",
						recorded, string(tt.requestData))
				}
			}
		})
	}
}

// TestTCPPredicateMatching tests predicate matching with TCP data
func TestTCPPredicateMatching(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		predicates   []models.Predicate
		requestData  []byte
		shouldMatch  bool
		responseData string
	}{
		{
			name: "equals predicate in text mode",
			mode: "text",
			predicates: []models.Predicate{
				{Equals: map[string]interface{}{"data": "HELLO"}},
			},
			requestData:  []byte("HELLO"),
			shouldMatch:  true,
			responseData: "MATCHED",
		},
		{
			name: "equals predicate no match",
			mode: "text",
			predicates: []models.Predicate{
				{Equals: map[string]interface{}{"data": "HELLO"}},
			},
			requestData:  []byte("GOODBYE"),
			shouldMatch:  false,
			responseData: "",
		},
		{
			name: "contains predicate in text mode",
			mode: "text",
			predicates: []models.Predicate{
				{Contains: map[string]interface{}{"data": "WORLD"}},
			},
			requestData:  []byte("HELLO WORLD"),
			shouldMatch:  true,
			responseData: "CONTAINS",
		},
		{
			name: "startsWith predicate",
			mode: "text",
			predicates: []models.Predicate{
				{StartsWith: map[string]interface{}{"data": "GET"}},
			},
			requestData:  []byte("GET /path"),
			shouldMatch:  true,
			responseData: "STARTS",
		},
		{
			name: "endsWith predicate",
			mode: "text",
			predicates: []models.Predicate{
				{EndsWith: map[string]interface{}{"data": "END"}},
			},
			requestData:  []byte("MESSAGE END"),
			shouldMatch:  true,
			responseData: "ENDS",
		},
		{
			name: "matches regex predicate",
			mode: "text",
			predicates: []models.Predicate{
				{Matches: map[string]interface{}{"data": "^[0-9]+$"}},
			},
			requestData:  []byte("12345"),
			shouldMatch:  true,
			responseData: "REGEX",
		},
		{
			name: "case insensitive equals",
			mode: "text",
			predicates: []models.Predicate{
				{
					Equals:        map[string]interface{}{"data": "hello"},
					CaseSensitive: false,
				},
			},
			requestData:  []byte("HELLO"),
			shouldMatch:  true,
			responseData: "CASE_INSENSITIVE",
		},
		{
			name: "binary mode equals with base64",
			mode: "binary",
			predicates: []models.Predicate{
				{Equals: map[string]interface{}{"data": base64.StdEncoding.EncodeToString([]byte{0x01, 0x02})}},
			},
			requestData:  []byte{0x01, 0x02},
			shouldMatch:  true,
			responseData: base64.StdEncoding.EncodeToString([]byte("BINARY_MATCH")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := 9100 + (len(tt.name) % 100)

			// Create imposter with predicates
			imp := &models.Imposter{
				Protocol: "tcp",
				Port:     port,
				Mode:     tt.mode,
				Stubs: []models.Stub{
					{
						Predicates: tt.predicates,
						Responses: []models.Response{
							{Is: &models.IsResponse{Data: tt.responseData}},
						},
					},
					{
						// Default stub with no predicates
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

			conn.Write(tt.requestData)

			// Read response
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			response := make([]byte, 1024)
			n, _ := conn.Read(response)

			got := response[:n]

			if tt.shouldMatch {
				// Should get the matched response
				var expected []byte
				if tt.mode == "binary" {
					expected, _ = base64.StdEncoding.DecodeString(tt.responseData)
				} else {
					expected = []byte(tt.responseData)
				}

				if string(got) != string(expected) {
					t.Errorf("expected match response:\ngot:  %s\nwant: %s", string(got), string(expected))
				}
			} else {
				// Should get default response
				if string(got) != "DEFAULT" {
					t.Errorf("expected default response, got: %s", string(got))
				}
			}
		})
	}
}

// TestTCPEndOfRequestResolver tests the end-of-request resolver functionality
func TestTCPEndOfRequestResolver(t *testing.T) {
	tests := []struct {
		name     string
		resolver string
		chunks   []string
		wantData string
	}{
		{
			name:     "simple newline delimiter",
			resolver: `function(requestData, logger) { return requestData.indexOf('\n') > -1; }`,
			chunks:   []string{"HELLO", "\n"},
			wantData: "HELLO\n",
		},
		{
			name:     "END marker",
			resolver: `function(requestData, logger) { return requestData.indexOf('END') > -1; }`,
			chunks:   []string{"START", "MIDDLE", "END"},
			wantData: "STARTMIDDLEEND",
		},
		{
			name:     "double newline (HTTP-like)",
			resolver: `function(requestData, logger) { return requestData.indexOf('\n\n') > -1; }`,
			chunks:   []string{"Header1\n", "Header2\n", "\n"},
			wantData: "Header1\nHeader2\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := 9200 + (len(tt.name) % 100)

			imp := &models.Imposter{
				Protocol: "tcp",
				Port:     port,
				Mode:     "text",
				EndOfRequestResolver: &models.EndOfRequestResolver{
					Inject: tt.resolver,
				},
				Stubs: []models.Stub{
					{
						Responses: []models.Response{
							{Is: &models.IsResponse{Data: "RESPONSE"}},
						},
					},
				},
				RecordRequests: true,
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

			// Connect
			addr := fmt.Sprintf("localhost:%d", port)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer conn.Close()

			// Send chunks with small delays
			for _, chunk := range tt.chunks {
				conn.Write([]byte(chunk))
				time.Sleep(10 * time.Millisecond)
			}

			// Read response
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			response := make([]byte, 1024)
			n, _ := conn.Read(response)

			if string(response[:n]) != "RESPONSE" {
				t.Errorf("expected RESPONSE, got: %s", string(response[:n]))
			}

			// Verify the full request was accumulated correctly
			time.Sleep(50 * time.Millisecond)
			storedImp := srv.GetImposter()
			if len(storedImp.TCPRequests) != 1 {
				t.Fatalf("expected 1 recorded request, got %d", len(storedImp.TCPRequests))
			}

			if storedImp.TCPRequests[0].Data != tt.wantData {
				t.Errorf("accumulated data mismatch:\ngot:  %q\nwant: %q",
					storedImp.TCPRequests[0].Data, tt.wantData)
			}
		})
	}
}

// TestTCPBehaviorDecorate tests decorate behavior with TCP responses
func TestTCPBehaviorDecorate(t *testing.T) {
	port := 9300

	imp := &models.Imposter{
		Protocol: "tcp",
		Port:     port,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Is: &models.IsResponse{Data: "BASE"},
						Behaviors: []models.Behavior{
							{
								Decorate: `function(request, response) {
									response.data = response.data + '-DECORATED';
								}`,
							},
						},
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

	// Connect and send request
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

	got := string(response[:n])
	want := "BASE-DECORATED"
	if got != want {
		t.Errorf("Response = %q, want %q (should be decorated)", got, want)
	}
}

// TestTCPMultipleBehaviors tests multiple behaviors composition for TCP
func TestTCPMultipleBehaviors(t *testing.T) {
	port := 9301

	imp := &models.Imposter{
		Protocol: "tcp",
		Port:     port,
		Mode:     "text",
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Is: &models.IsResponse{Data: "ORIGINAL"},
						Behaviors: []models.Behavior{
							{
								Wait: 50, // 50ms delay
							},
							{
								Decorate: `function(request, response) {
									response.data = response.data + '-STEP1';
								}`,
							},
							{
								Decorate: `function(request, response) {
									response.data = response.data + '-STEP2';
								}`,
							},
						},
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

	// Connect and send request
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Measure time for wait behavior
	start := time.Now()
	conn.Write([]byte("TEST\n"))

	// Read response
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	response := make([]byte, 1024)
	n, _ := conn.Read(response)
	elapsed := time.Since(start)

	got := string(response[:n])
	want := "ORIGINAL-STEP1-STEP2"
	if got != want {
		t.Errorf("Response = %q, want %q (should apply all decorations)", got, want)
	}

	// Verify wait behavior was applied (should be >= 50ms)
	if elapsed < 50*time.Millisecond {
		t.Errorf("Response time = %v, expected >= 50ms due to wait behavior", elapsed)
	}
}
