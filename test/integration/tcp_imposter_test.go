package integration

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// TestTCP_AutoAssignPort tests that port=0 or missing port auto-assigns a port
func TestTCP_AutoAssignPort(t *testing.T) {
	// Create imposter without specifying port (defaults to 0)
	resp, body, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	// Verify imposter created
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	// Verify port was auto-assigned (should be > 0)
	portVal, ok := body["port"]
	if !ok {
		t.Fatal("response missing 'port' field")
	}

	port, ok := portVal.(float64)
	if !ok {
		t.Fatalf("port is not a number: %v", portVal)
	}

	if port <= 0 {
		t.Errorf("expected auto-assigned port > 0, got %v", port)
	}

	// Verify we can connect to the auto-assigned port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", int(port)), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to auto-assigned port %d: %v", int(port), err)
	}
	conn.Close()
}

// TestTCP_StubListRetrieval tests that GET /imposters/:id returns stubs array
func TestTCP_StubListRetrieval(t *testing.T) {
	port := 6200
	// Create imposter with multiple stubs
	createResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "1"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"data": "2"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	if createResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", createResp.StatusCode)
	}

	// Retrieve imposter
	getResp, imposter, err := get(fmt.Sprintf("/imposters/%d", port))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}

	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	// Verify stubs array exists
	stubsVal, ok := imposter["stubs"]
	if !ok {
		t.Fatal("response missing 'stubs' field")
	}

	stubs, ok := stubsVal.([]interface{})
	if !ok {
		t.Fatalf("stubs is not an array: %v", stubsVal)
	}

	if len(stubs) != 2 {
		t.Fatalf("expected 2 stubs, got %d", len(stubs))
	}

	// Verify first stub has responses
	stub1 := stubs[0].(map[string]interface{})
	responses1 := stub1["responses"].([]interface{})
	if len(responses1) != 1 {
		t.Errorf("expected 1 response in first stub, got %d", len(responses1))
	}

	resp1 := responses1[0].(map[string]interface{})
	is1 := resp1["is"].(map[string]interface{})
	if is1["data"] != "1" {
		t.Errorf("expected first stub data '1', got %v", is1["data"])
	}

	// Verify second stub has responses
	stub2 := stubs[1].(map[string]interface{})
	responses2 := stub2["responses"].([]interface{})
	if len(responses2) != 1 {
		t.Errorf("expected 1 response in second stub, got %d", len(responses2))
	}

	resp2 := responses2[0].(map[string]interface{})
	is2 := resp2["is"].(map[string]interface{})
	if is2["data"] != "2" {
		t.Errorf("expected second stub data '2', got %v", is2["data"])
	}

	// Note: _links verification is optional - mountebank includes them but not critical
	// for go-tartuffe compatibility. Stubs array content is the key requirement.
}

// TestTCP_ModeFieldInResponse tests that GET /imposters/:id returns mode field
func TestTCP_ModeFieldInResponse(t *testing.T) {
	port := 6201
	// Create imposter with name
	createResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"name":     "test-imposter",
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	if createResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", createResp.StatusCode)
	}

	// Retrieve imposter
	getResp, imposter, err := get(fmt.Sprintf("/imposters/%d", port))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}

	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	// Verify mode field exists and defaults to "text"
	modeVal, ok := imposter["mode"]
	if !ok {
		t.Error("response missing 'mode' field")
	} else {
		mode, ok := modeVal.(string)
		if !ok {
			t.Errorf("mode is not a string: %v", modeVal)
		} else if mode != "text" {
			t.Errorf("expected mode 'text', got %q", mode)
		}
	}

	// Verify other standard fields
	if imposter["protocol"] != "tcp" {
		t.Errorf("expected protocol 'tcp', got %v", imposter["protocol"])
	}
	if imposter["port"].(float64) != float64(port) {
		t.Errorf("expected port %d, got %v", port, imposter["port"])
	}
	if imposter["name"] != "test-imposter" {
		t.Errorf("expected name 'test-imposter', got %v", imposter["name"])
	}
	if imposter["recordRequests"] != false {
		t.Errorf("expected recordRequests false, got %v", imposter["recordRequests"])
	}
	if imposter["numberOfRequests"].(float64) != 0 {
		t.Errorf("expected numberOfRequests 0, got %v", imposter["numberOfRequests"])
	}

	// Verify requests and stubs arrays exist
	if _, ok := imposter["requests"]; !ok {
		t.Error("response missing 'requests' field")
	}
	if _, ok := imposter["stubs"]; !ok {
		t.Error("response missing 'stubs' field")
	}
}

// TestTCP_EndOfRequestResolverRetrieval tests that endOfRequestResolver is returned in API
func TestTCP_EndOfRequestResolverRetrieval(t *testing.T) {
	port := 6202
	resolverCode := "function(config) { return config.request.data.length >= 100; }"

	// Create imposter with endOfRequestResolver
	createResp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "tcp",
		"port":     port,
		"endOfRequestResolver": map[string]interface{}{
			"inject": resolverCode,
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	defer del("/imposters")

	if createResp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", createResp.StatusCode)
	}

	// Retrieve imposter
	getResp, imposter, err := get(fmt.Sprintf("/imposters/%d", port))
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}

	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	// Verify endOfRequestResolver is returned
	resolverVal, ok := imposter["endOfRequestResolver"]
	if !ok {
		t.Fatal("response missing 'endOfRequestResolver' field")
	}

	resolver, ok := resolverVal.(map[string]interface{})
	if !ok {
		t.Fatalf("endOfRequestResolver is not an object: %v", resolverVal)
	}

	// Verify inject code is returned
	injectVal, ok := resolver["inject"]
	if !ok {
		t.Fatal("endOfRequestResolver missing 'inject' field")
	}

	inject, ok := injectVal.(string)
	if !ok {
		t.Fatalf("inject is not a string: %v", injectVal)
	}

	if inject != resolverCode {
		t.Errorf("expected inject code %q, got %q", resolverCode, inject)
	}
}
