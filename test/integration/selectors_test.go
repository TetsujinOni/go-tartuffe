package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// JSONPath selector tests

func TestJSONPath_SimpleFieldExtraction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5800,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "John",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.name",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Hello John!"}},
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

	// Send request with JSON body
	impResp, err := http.Post("http://localhost:5800/test", "application/json",
		strings.NewReader(`{"name": "John", "age": 30}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Hello John!" {
		t.Errorf("expected 'Hello John!', got '%s'", string(body))
	}
}

func TestJSONPath_NestedFieldExtraction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5801,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "San Francisco",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.user.address.city",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Welcome from SF!"}},
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

	jsonBody := `{
		"user": {
			"name": "Alice",
			"address": {
				"city": "San Francisco",
				"zip": "94102"
			}
		}
	}`

	impResp, err := http.Post("http://localhost:5801/test", "application/json",
		strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Welcome from SF!" {
		t.Errorf("expected 'Welcome from SF!', got '%s'", string(body))
	}
}

func TestJSONPath_ArrayIndexExtraction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5802,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "first",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.items[0]",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "First item matched!"}},
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

	impResp, err := http.Post("http://localhost:5802/test", "application/json",
		strings.NewReader(`{"items": ["first", "second", "third"]}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "First item matched!" {
		t.Errorf("expected 'First item matched!', got '%s'", string(body))
	}
}

func TestJSONPath_ArrayObjectFieldExtraction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5803,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "widget",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.products[0].name",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Widget found!"}},
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

	jsonBody := `{
		"products": [
			{"name": "widget", "price": 10},
			{"name": "gadget", "price": 20}
		]
	}`

	impResp, err := http.Post("http://localhost:5803/test", "application/json",
		strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Widget found!" {
		t.Errorf("expected 'Widget found!', got '%s'", string(body))
	}
}

func TestJSONPath_ContainsPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5804,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"body": "admin",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.user.role",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Admin access granted"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Regular user"}},
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

	// Test admin user
	impResp1, err := http.Post("http://localhost:5804/test", "application/json",
		strings.NewReader(`{"user": {"role": "super_admin"}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "Admin access granted" {
		t.Errorf("expected 'Admin access granted', got '%s'", string(body1))
	}

	// Test regular user
	impResp2, err := http.Post("http://localhost:5804/test", "application/json",
		strings.NewReader(`{"user": {"role": "guest"}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) != "Regular user" {
		t.Errorf("expected 'Regular user', got '%s'", string(body2))
	}
}

func TestJSONPath_MatchesPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5805,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"matches": map[string]interface{}{
							"body": "^[a-z]+@[a-z]+\\.[a-z]+$",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.email",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Valid email"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Invalid email"}},
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

	// Test valid email
	impResp1, err := http.Post("http://localhost:5805/test", "application/json",
		strings.NewReader(`{"email": "test@example.com"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "Valid email" {
		t.Errorf("expected 'Valid email', got '%s'", string(body1))
	}

	// Test invalid email
	impResp2, err := http.Post("http://localhost:5805/test", "application/json",
		strings.NewReader(`{"email": "not-an-email"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) != "Invalid email" {
		t.Errorf("expected 'Invalid email', got '%s'", string(body2))
	}
}

// XPath selector tests

func TestXPath_SimpleElementExtraction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5806,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "Test Book",
						},
						"xpath": map[string]interface{}{
							"selector": "//title",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Book found!"}},
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

	xmlBody := `<?xml version="1.0"?>
	<book>
		<title>Test Book</title>
		<author>John Doe</author>
	</book>`

	impResp, err := http.Post("http://localhost:5806/test", "application/xml",
		strings.NewReader(xmlBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Book found!" {
		t.Errorf("expected 'Book found!', got '%s'", string(body))
	}
}

func TestXPath_NestedElementExtraction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5807,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "Alice",
						},
						"xpath": map[string]interface{}{
							"selector": "/order/customer/name",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Order for Alice received"}},
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

	xmlBody := `<?xml version="1.0"?>
	<order>
		<customer>
			<name>Alice</name>
			<email>alice@example.com</email>
		</customer>
		<items>
			<item>Widget</item>
		</items>
	</order>`

	impResp, err := http.Post("http://localhost:5807/test", "application/xml",
		strings.NewReader(xmlBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Order for Alice received" {
		t.Errorf("expected 'Order for Alice received', got '%s'", string(body))
	}
}

func TestXPath_AttributeExtraction(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5808,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "123",
						},
						"xpath": map[string]interface{}{
							"selector": "//product/@id",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Product 123 found"}},
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

	xmlBody := `<?xml version="1.0"?>
	<catalog>
		<product id="123">
			<name>Widget</name>
		</product>
	</catalog>`

	impResp, err := http.Post("http://localhost:5808/test", "application/xml",
		strings.NewReader(xmlBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Product 123 found" {
		t.Errorf("expected 'Product 123 found', got '%s'", string(body))
	}
}

func TestXPath_ContainsPredicate(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5809,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"contains": map[string]interface{}{
							"body": "urgent",
						},
						"xpath": map[string]interface{}{
							"selector": "//message/priority",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Urgent message received"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Normal message received"}},
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

	// Test urgent message
	urgentXML := `<message><priority>urgent-high</priority><text>Help!</text></message>`
	impResp1, err := http.Post("http://localhost:5809/test", "application/xml",
		strings.NewReader(urgentXML))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "Urgent message received" {
		t.Errorf("expected 'Urgent message received', got '%s'", string(body1))
	}

	// Test normal message
	normalXML := `<message><priority>low</priority><text>FYI</text></message>`
	impResp2, err := http.Post("http://localhost:5809/test", "application/xml",
		strings.NewReader(normalXML))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) != "Normal message received" {
		t.Errorf("expected 'Normal message received', got '%s'", string(body2))
	}
}

func TestSelector_MultiplePredicates(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5810,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "premium",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.subscription.type",
						},
					},
					{
						"equals": map[string]interface{}{
							"method": "POST",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Premium user POST"}},
				},
			},
			{
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Other request"}},
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

	// Test premium POST
	impResp1, err := http.Post("http://localhost:5810/test", "application/json",
		strings.NewReader(`{"subscription": {"type": "premium"}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body1, _ := io.ReadAll(impResp1.Body)
	impResp1.Body.Close()

	if string(body1) != "Premium user POST" {
		t.Errorf("expected 'Premium user POST', got '%s'", string(body1))
	}

	// Test premium GET (should not match)
	impResp2, err := http.Get("http://localhost:5810/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(impResp2.Body)
	impResp2.Body.Close()

	if string(body2) != "Other request" {
		t.Errorf("expected 'Other request', got '%s'", string(body2))
	}
}

func TestJSONPath_NumericValue(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "http",
		"port":     5811,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"body": "100",
						},
						"jsonpath": map[string]interface{}{
							"selector": "$.order.total",
						},
					},
				},
				"responses": []map[string]interface{}{
					{"is": map[string]interface{}{"body": "Order total is $100"}},
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

	impResp, err := http.Post("http://localhost:5811/test", "application/json",
		strings.NewReader(`{"order": {"total": 100}}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(impResp.Body)
	impResp.Body.Close()

	if string(body) != "Order total is $100" {
		t.Errorf("expected 'Order total is $100', got '%s'", string(body))
	}
}
