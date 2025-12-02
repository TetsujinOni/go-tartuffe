package integration

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"testing"
	"time"
)

// Binary mode tests

func TestBinaryMode_ShouldDecodeBase64Response(t *testing.T) {
	defer cleanup(t)

	// Create binary data (PNG header bytes as example)
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	base64Data := base64.StdEncoding.EncodeToString(binaryData)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5600,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body":    base64Data,
							"_mode":   "binary",
							"headers": map[string]interface{}{"Content-Type": "image/png"},
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

	impResp, err := http.Get("http://localhost:5600/image")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	// Should receive the decoded binary data
	if !bytes.Equal(body, binaryData) {
		t.Errorf("expected binary data %v, got %v", binaryData, body)
	}

	if impResp.Header.Get("Content-Type") != "image/png" {
		t.Errorf("expected Content-Type 'image/png', got '%s'", impResp.Header.Get("Content-Type"))
	}
}

func TestBinaryMode_ShouldEncodeRequestBodyWhenBinary(t *testing.T) {
	defer cleanup(t)

	// Create imposter with recordRequests enabled
	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "http",
		"port":           5601,
		"recordRequests": true,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{"body": "received"},
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

	// Send binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}
	impResp, err := http.Post("http://localhost:5601/upload", "application/octet-stream", bytes.NewReader(binaryData))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	impResp.Body.Close()

	// Check the recorded request
	getResp, body, err := get("/imposters/5601")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	// The recorded request should have mode set to binary and body base64-encoded
	requests, ok := body["requests"].([]interface{})
	if !ok || len(requests) == 0 {
		t.Fatal("expected recorded requests")
	}

	firstRequest := requests[0].(map[string]interface{})
	mode, _ := firstRequest["_mode"].(string)
	if mode != "binary" {
		t.Errorf("expected mode 'binary', got '%s'", mode)
	}

	recordedBody, _ := firstRequest["body"].(string)
	expectedBase64 := base64.StdEncoding.EncodeToString(binaryData)
	if recordedBody != expectedBase64 {
		t.Errorf("expected base64 body '%s', got '%s'", expectedBase64, recordedBody)
	}
}

func TestBinaryMode_TextRequestShouldNotBeBinaryEncoded(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "http",
		"port":           5602,
		"recordRequests": true,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{"body": "received"},
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

	// Send text data
	textData := "Hello, World!"
	impResp, err := http.Post("http://localhost:5602/text", "text/plain", bytes.NewReader([]byte(textData)))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	impResp.Body.Close()

	// Check the recorded request
	getResp, body, err := get("/imposters/5602")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	requests, ok := body["requests"].([]interface{})
	if !ok || len(requests) == 0 {
		t.Fatal("expected recorded requests")
	}

	firstRequest := requests[0].(map[string]interface{})
	mode, _ := firstRequest["_mode"].(string)
	if mode == "binary" {
		t.Error("expected text request not to be marked as binary")
	}

	recordedBody, _ := firstRequest["body"].(string)
	if recordedBody != textData {
		t.Errorf("expected body '%s', got '%s'", textData, recordedBody)
	}
}

func TestBinaryMode_JSONRequestShouldNotBeBinaryEncoded(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "http",
		"port":           5603,
		"recordRequests": true,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{"body": "received"},
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

	// Send JSON data
	jsonData := `{"key": "value"}`
	impResp, err := http.Post("http://localhost:5603/json", "application/json", bytes.NewReader([]byte(jsonData)))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	impResp.Body.Close()

	// Check the recorded request
	getResp, body, err := get("/imposters/5603")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	requests, ok := body["requests"].([]interface{})
	if !ok || len(requests) == 0 {
		t.Fatal("expected recorded requests")
	}

	firstRequest := requests[0].(map[string]interface{})
	mode, _ := firstRequest["_mode"].(string)
	if mode == "binary" {
		t.Error("expected JSON request not to be marked as binary")
	}

	recordedBody, _ := firstRequest["body"].(string)
	if recordedBody != jsonData {
		t.Errorf("expected body '%s', got '%s'", jsonData, recordedBody)
	}
}

func TestBinaryMode_LargeImageResponse(t *testing.T) {
	defer cleanup(t)

	// Create a larger binary payload (1KB of pseudo-random data)
	binaryData := make([]byte, 1024)
	for i := range binaryData {
		binaryData[i] = byte((i * 17) % 256)
	}
	base64Data := base64.StdEncoding.EncodeToString(binaryData)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5604,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body":       base64Data,
							"_mode":      "binary",
							"statusCode": 200,
							"headers":    map[string]interface{}{"Content-Type": "application/octet-stream"},
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

	impResp, err := http.Get("http://localhost:5604/data")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if !bytes.Equal(body, binaryData) {
		t.Errorf("binary data mismatch: expected %d bytes, got %d bytes", len(binaryData), len(body))
	}
}

func TestBinaryMode_WithPredicate(t *testing.T) {
	defer cleanup(t)

	// PNG image header
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	pngBase64 := base64.StdEncoding.EncodeToString(pngData)

	// JPEG image header
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	jpegBase64 := base64.StdEncoding.EncodeToString(jpegData)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5605,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/png"}},
				},
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body":    pngBase64,
							"_mode":   "binary",
							"headers": map[string]interface{}{"Content-Type": "image/png"},
						},
					},
				},
			},
			{
				"predicates": []map[string]interface{}{
					{"equals": map[string]interface{}{"path": "/jpeg"}},
				},
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body":    jpegBase64,
							"_mode":   "binary",
							"headers": map[string]interface{}{"Content-Type": "image/jpeg"},
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

	// Test PNG endpoint
	pngResp, err := http.Get("http://localhost:5605/png")
	if err != nil {
		t.Fatalf("PNG request failed: %v", err)
	}
	pngBody, _ := io.ReadAll(pngResp.Body)
	pngResp.Body.Close()

	if !bytes.Equal(pngBody, pngData) {
		t.Errorf("PNG data mismatch")
	}

	// Test JPEG endpoint
	jpegResp, err := http.Get("http://localhost:5605/jpeg")
	if err != nil {
		t.Fatalf("JPEG request failed: %v", err)
	}
	jpegBody, _ := io.ReadAll(jpegResp.Body)
	jpegResp.Body.Close()

	if !bytes.Equal(jpegBody, jpegData) {
		t.Errorf("JPEG data mismatch")
	}
}
