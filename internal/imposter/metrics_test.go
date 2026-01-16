package imposter

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestHTTPMetricsRequestCount tests that numberOfRequests is tracked correctly
func TestHTTPMetricsRequestCount(t *testing.T) {
	tests := []struct {
		name            string
		requestCount    int
		wantNumRequests int
	}{
		{
			name:            "single request tracked",
			requestCount:    1,
			wantNumRequests: 1,
		},
		{
			name:            "multiple requests tracked",
			requestCount:    5,
			wantNumRequests: 5,
		},
		{
			name:            "ten requests tracked",
			requestCount:    10,
			wantNumRequests: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := 9500 + (len(tt.name) % 100)

			// Create imposter with simple stub
			imp := &models.Imposter{
				Protocol: "http",
				Port:     port,
				Stubs: []models.Stub{
					{
						Responses: []models.Response{
							{Is: &models.IsResponse{
								StatusCode: 200,
								Body:       "OK",
							}},
						},
					},
				},
				RecordRequests: true,
			}

			manager := NewManager()
			if err := manager.Start(imp); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			defer manager.Stop(port)

			time.Sleep(50 * time.Millisecond)

			// Make requests
			client := &http.Client{Timeout: 2 * time.Second}
			for i := 0; i < tt.requestCount; i++ {
				resp, err := client.Get(fmt.Sprintf("http://localhost:%d/test", port))
				if err != nil {
					t.Fatalf("GET request %d failed: %v", i+1, err)
				}
				resp.Body.Close()
			}

			// Get imposter to check numberOfRequests
			server := manager.GetServer(port)
			if server == nil {
				t.Fatal("Server not found")
			}

			imposter := server.GetImposter()
			if imposter.NumberOfRequests != tt.wantNumRequests {
				t.Errorf("NumberOfRequests = %d, want %d", imposter.NumberOfRequests, tt.wantNumRequests)
			}
		})
	}
}

// TestHTTPMetricsWithoutRecordRequests tests that numberOfRequests is counted even without recordRequests
func TestHTTPMetricsWithoutRecordRequests(t *testing.T) {
	port := 9600

	imp := &models.Imposter{
		Protocol: "http",
		Port:     port,
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{Is: &models.IsResponse{
						StatusCode: 200,
						Body:       "OK",
					}},
				},
			},
		},
		RecordRequests: false, // Explicitly disable recording
	}

	manager := NewManager()
	if err := manager.Start(imp); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer manager.Stop(port)

	time.Sleep(50 * time.Millisecond)

	// Make multiple requests
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 3; i++ {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		resp.Body.Close()
	}

	// Verify numberOfRequests is still tracked
	server := manager.GetServer(port)
	if server == nil {
		t.Fatal("Server not found")
	}

	imposter := server.GetImposter()
	if imposter.NumberOfRequests != 3 {
		t.Errorf("NumberOfRequests = %d, want 3 (should track even without recordRequests)", imposter.NumberOfRequests)
	}

	// Verify requests array is empty (since recordRequests=false)
	if len(imposter.Requests) != 0 {
		t.Errorf("Requests array should be empty when recordRequests=false, got %d requests", len(imposter.Requests))
	}
}

// TestHTTPMetricsResponseTime tests that response times can be measured
func TestHTTPMetricsResponseTime(t *testing.T) {
	port := 9601

	// Create imposter with wait behavior to ensure measurable response time
	imp := &models.Imposter{
		Protocol: "http",
		Port:     port,
		Stubs: []models.Stub{
			{
				Responses: []models.Response{
					{
						Is: &models.IsResponse{
							StatusCode: 200,
							Body:       "Delayed",
						},
						Behaviors: []models.Behavior{
							{Wait: 100}, // 100ms delay
						},
					},
				},
			},
		},
	}

	manager := NewManager()
	if err := manager.Start(imp); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer manager.Stop(port)

	time.Sleep(50 * time.Millisecond)

	// Make request and measure time
	client := &http.Client{Timeout: 2 * time.Second}
	start := time.Now()
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/", port))
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify response was delayed
	if elapsed < 100*time.Millisecond {
		t.Errorf("Response time = %v, expected >= 100ms due to wait behavior", elapsed)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Delayed" {
		t.Errorf("Body = %q, want %q", string(body), "Delayed")
	}

	// Verify request was counted
	server := manager.GetServer(port)
	imposter := server.GetImposter()
	if imposter.NumberOfRequests != 1 {
		t.Errorf("NumberOfRequests = %d, want 1", imposter.NumberOfRequests)
	}
}

// Note: TCP and SMTP numberOfRequests tracking is already tested in:
// - tcp_test.go (TestTCPBinaryMode verifies request recording)
// - smtp_test.go (TestSMTPRequestFormat verifies request recording)
// Those tests validate that NumberOfRequests is incremented correctly for their protocols
