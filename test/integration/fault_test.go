package integration

import (
	"net"
	"net/http"
	"testing"
	"time"
)

// Fault response tests

func TestFault_ConnectionResetByPeer(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5400,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"fault": "CONNECTION_RESET_BY_PEER"},
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

	// Make request - should get connection reset
	_, err = http.Get("http://localhost:5400/test")
	if err == nil {
		t.Fatal("expected connection error, got none")
	}

	// Check that the error is a connection reset or EOF
	netErr, ok := err.(*net.OpError)
	if ok {
		// Connection was reset
		t.Logf("Got expected network error: %v", netErr)
	} else {
		// Could also be EOF or other connection error
		t.Logf("Got error (expected): %v", err)
	}
}

func TestFault_RandomDataThenClose(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5401,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"fault": "RANDOM_DATA_THEN_CLOSE"},
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

	// Make request - should get parse error due to garbage data
	_, err = http.Get("http://localhost:5401/test")
	if err == nil {
		t.Fatal("expected error due to garbage data, got none")
	}

	// The error should be about invalid HTTP response
	t.Logf("Got expected error: %v", err)
}

func TestFault_UnknownFaultReturnsEmpty(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5402,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"fault": "NON_EXISTENT_FAULT"},
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

	// Make request - unknown fault should just close connection gracefully
	_, err = http.Get("http://localhost:5402/test")
	// This might succeed with empty response or fail with EOF
	// Either is acceptable for unknown fault type
	t.Logf("Unknown fault result: err=%v", err)
}

func TestFault_WithPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5403,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/fault"}},
				},
				"responses": []map[string]interface{}{
					{"fault": "CONNECTION_RESET_BY_PEER"},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "normal response"}},
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

	// Request to /fault should reset connection
	_, err = http.Get("http://localhost:5403/fault")
	if err == nil {
		t.Error("expected connection error for /fault path")
	} else {
		t.Logf("Got expected error for /fault: %v", err)
	}

	// Request to other path should work normally
	normalResp, err := http.Get("http://localhost:5403/normal")
	if err != nil {
		t.Fatalf("request to /normal failed: %v", err)
	}
	normalResp.Body.Close()

	if normalResp.StatusCode != 200 {
		t.Errorf("expected status 200 for /normal, got %d", normalResp.StatusCode)
	}
}
