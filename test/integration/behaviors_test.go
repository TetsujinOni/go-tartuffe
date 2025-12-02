package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Behavior tests

func TestBehavior_Wait_ShouldAddLatency(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5500,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{"body": "delayed"},
						"_behaviors": []map[string]interface{}{
							{"wait": 500},
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

	// Make request and measure time
	start := time.Now()
	impResp, err := http.Get("http://localhost:5500/test")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "delayed" {
		t.Errorf("expected 'delayed', got '%s'", string(body))
	}

	// Should have waited at least 450ms (allowing some tolerance)
	if elapsed < 450*time.Millisecond {
		t.Errorf("expected at least 450ms delay, got %v", elapsed)
	}
}

func TestBehavior_Wait_ShouldSupportFunction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5501,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{"body": "delayed by function"},
						"_behaviors": []map[string]interface{}{
							{"wait": "function() { return 300; }"},
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

	start := time.Now()
	impResp, err := http.Get("http://localhost:5501/test")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	impResp.Body.Close()

	// Should have waited at least 250ms
	if elapsed < 250*time.Millisecond {
		t.Errorf("expected at least 250ms delay, got %v", elapsed)
	}
}

func TestBehavior_Copy_ShouldCopyRequestValueToResponse(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5502,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "Hello ${NAME}!",
						},
						"_behaviors": []map[string]interface{}{
							{
								"copy": map[string]interface{}{
									"from": map[string]interface{}{
										"query": "name",
									},
									"into": "${NAME}",
								},
							},
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

	impResp, err := http.Get("http://localhost:5502/greet?name=World")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Hello World!" {
		t.Errorf("expected 'Hello World!', got '%s'", string(body))
	}
}

func TestBehavior_Copy_WithRegex(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5503,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "ID is ${ID}",
						},
						"_behaviors": []map[string]interface{}{
							{
								"copy": map[string]interface{}{
									"from": "path",
									"into": "${ID}",
									"using": map[string]interface{}{
										"method":   "regex",
										"selector": "/users/(\\d+)",
									},
								},
							},
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

	impResp, err := http.Get("http://localhost:5503/users/12345")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "ID is 12345" {
		t.Errorf("expected 'ID is 12345', got '%s'", string(body))
	}
}

func TestBehavior_Decorate_ShouldPostProcessResponse(t *testing.T) {
	defer cleanup(t)

	decoratorScript := `function(config) {
		config.response.body = config.response.body.replace('${PATH}', config.request.path);
		return config.response;
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5504,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "You requested ${PATH}",
						},
						"_behaviors": []map[string]interface{}{
							{"decorate": decoratorScript},
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

	impResp, err := http.Get("http://localhost:5504/mypath")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "You requested /mypath" {
		t.Errorf("expected 'You requested /mypath', got '%s'", string(body))
	}
}

func TestBehavior_Decorate_OldInterface(t *testing.T) {
	defer cleanup(t)

	// Old interface: (request, response)
	decoratorScript := `function(request, response) {
		response.body = "Method was " + request.method;
		return response;
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5505,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "original",
						},
						"_behaviors": []map[string]interface{}{
							{"decorate": decoratorScript},
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

	impResp, err := http.Get("http://localhost:5505/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Method was GET" {
		t.Errorf("expected 'Method was GET', got '%s'", string(body))
	}
}

func TestBehavior_Decorate_CanModifyStatusCode(t *testing.T) {
	defer cleanup(t)

	decoratorScript := `function(config) {
		config.response.statusCode = 201;
		config.response.headers = config.response.headers || {};
		config.response.headers["X-Custom"] = "decorated";
		return config.response;
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5506,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "test",
						},
						"_behaviors": []map[string]interface{}{
							{"decorate": decoratorScript},
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

	impResp, err := http.Get("http://localhost:5506/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	impResp.Body.Close()

	if impResp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", impResp.StatusCode)
	}

	if impResp.Header.Get("X-Custom") != "decorated" {
		t.Errorf("expected X-Custom header 'decorated', got '%s'", impResp.Header.Get("X-Custom"))
	}
}

func TestBehavior_Lookup_CSV(t *testing.T) {
	defer cleanup(t)

	// Create a temporary CSV file
	csvContent := `id,name,email
1,Alice,alice@example.com
2,Bob,bob@example.com
3,Charlie,charlie@example.com`

	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "users.csv")
	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create CSV file: %v", err)
	}

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5507,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": `{"name": "${ROW}[name]", "email": "${ROW}[email]"}`,
							"headers": map[string]interface{}{
								"Content-Type": "application/json",
							},
						},
						"_behaviors": []map[string]interface{}{
							{
								"lookup": map[string]interface{}{
									"key": map[string]interface{}{
										"from": map[string]interface{}{
											"query": "id",
										},
										"using": map[string]interface{}{
											"method":   "regex",
											"selector": ".*",
										},
									},
									"fromDataSource": map[string]interface{}{
										"csv": map[string]interface{}{
											"path":      csvPath,
											"keyColumn": "id",
										},
									},
									"into": "${ROW}",
								},
							},
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

	impResp, err := http.Get("http://localhost:5507/user?id=2")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response: %v (body: %s)", err, string(body))
	}

	if result["name"] != "Bob" {
		t.Errorf("expected name='Bob', got %v", result["name"])
	}
	if result["email"] != "bob@example.com" {
		t.Errorf("expected email='bob@example.com', got %v", result["email"])
	}
}

func TestBehavior_MultipleBehaviors(t *testing.T) {
	defer cleanup(t)

	decoratorScript := `function(config) {
		config.response.body = config.response.body + " - decorated";
		return config.response;
	}`

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5508,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "Hello ${NAME}",
						},
						"_behaviors": []map[string]interface{}{
							{
								"copy": map[string]interface{}{
									"from": map[string]interface{}{
										"query": "name",
									},
									"into": "${NAME}",
								},
							},
							{"decorate": decoratorScript},
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

	impResp, err := http.Get("http://localhost:5508/test?name=World")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	expected := "Hello World - decorated"
	if string(body) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(body))
	}
}

func TestBehavior_Copy_FromBody(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5509,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"body": "Received: ${DATA}",
						},
						"_behaviors": []map[string]interface{}{
							{
								"copy": map[string]interface{}{
									"from": "body",
									"into": "${DATA}",
									"using": map[string]interface{}{
										"method":   "regex",
										"selector": "message=(\\w+)",
									},
								},
							},
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

	impResp, err := http.Post("http://localhost:5509/echo", "text/plain", strings.NewReader("message=HelloWorld"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Received: HelloWorld" {
		t.Errorf("expected 'Received: HelloWorld', got '%s'", string(body))
	}
}
