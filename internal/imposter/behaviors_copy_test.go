package imposter

import (
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestCopyWithRegex tests copy behavior using regex selectors
func TestCopyWithRegex(t *testing.T) {

	tests := []struct {
		name         string
		copyBehavior models.Copy
		requestPath  string
		requestBody  string
		initialBody  string
		wantBody     string
	}{
		{
			name: "copy from path using regex",
			copyBehavior: models.Copy{
				From: map[string]interface{}{
					"path": "$PATH",
				},
				Into: "${userId}",
				Using: &models.Using{
					Method:   "regex",
					Selector: "/users/(\\d+)",
				},
			},
			requestPath: "/users/123",
			initialBody: "User ID: ${userId}",
			wantBody:    "User ID: 123",
		},
		{
			name: "copy multiple capture groups",
			copyBehavior: models.Copy{
				From: map[string]interface{}{
					"path": "$PATH",
				},
				Into: "${resource}",
				Using: &models.Using{
					Method:   "regex",
					Selector: "/api/(\\w+)/(\\d+)",
				},
			},
			requestPath: "/api/users/456",
			initialBody: "Resource: ${resource}",
			wantBody:    "Resource: users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsEngine := NewJSEngine()
			executor := NewBehaviorExecutor(jsEngine)

			req := &models.Request{
				Method: "GET",
				Path:   tt.requestPath,
				Body:   tt.requestBody,
			}

			resp := &models.IsResponse{
				StatusCode: 200,
				Headers:    make(map[string]interface{}),
				Body:       tt.initialBody,
			}

			behavior := models.Behavior{
				Copy: []models.Copy{tt.copyBehavior},
			}

			result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
			if err != nil {
				t.Fatalf("ApplyBehaviors() error = %v", err)
			}

			if bodyStr, ok := result.Body.(string); ok {
				if bodyStr != tt.wantBody {
					t.Errorf("Body = %q, want %q", bodyStr, tt.wantBody)
				}
			}
		})
	}
}

// TestCopyWithJSONPath tests copy behavior using jsonpath selectors
func TestCopyWithJSONPath(t *testing.T) {

	tests := []struct {
		name         string
		copyBehavior models.Copy
		requestBody  string
		initialBody  string
		wantBody     string
	}{
		{
			name: "copy from JSON body using jsonpath",
			copyBehavior: models.Copy{
				From: map[string]interface{}{
					"body": "$.user.name",
				},
				Into: "${name}",
				Using: &models.Using{
					Method:   "jsonpath",
					Selector: "$",
				},
			},
			requestBody: `{"user":{"name":"Jane","age":30}}`,
			initialBody: "Hello ${name}",
			wantBody:    "Hello Jane",
		},
		{
			name: "copy from nested JSON array",
			copyBehavior: models.Copy{
				From: map[string]interface{}{
					"body": "$.items[0].id",
				},
				Into: "${itemId}",
				Using: &models.Using{
					Method:   "jsonpath",
					Selector: "$",
				},
			},
			requestBody: `{"items":[{"id":"abc123","name":"Item1"}]}`,
			initialBody: "First item: ${itemId}",
			wantBody:    "First item: abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsEngine := NewJSEngine()
			executor := NewBehaviorExecutor(jsEngine)

			req := &models.Request{
				Method: "POST",
				Path:   "/test",
				Body:   tt.requestBody,
			}

			resp := &models.IsResponse{
				StatusCode: 200,
				Headers:    make(map[string]interface{}),
				Body:       tt.initialBody,
			}

			behavior := models.Behavior{
				Copy: []models.Copy{tt.copyBehavior},
			}

			result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
			if err != nil {
				t.Fatalf("ApplyBehaviors() error = %v", err)
			}

			if bodyStr, ok := result.Body.(string); ok {
				if bodyStr != tt.wantBody {
					t.Errorf("Body = %q, want %q", bodyStr, tt.wantBody)
				}
			}
		})
	}
}

// TestCopyIntoHeader tests copying into response headers
func TestCopyIntoHeader(t *testing.T) {

	jsEngine := NewJSEngine()
	executor := NewBehaviorExecutor(jsEngine)

	req := &models.Request{
		Method: "GET",
		Path:   "/users/789",
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers: map[string]interface{}{
			"X-User-ID": "${userId}",
		},
		Body: "test",
	}

	behavior := models.Behavior{
		Copy: []models.Copy{
			{
				From: map[string]interface{}{
					"path": "$PATH",
				},
				Into: "${userId}",
				Using: &models.Using{
					Method:   "regex",
					Selector: "/users/(\\d+)",
				},
			},
		},
	}

	result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
	if err != nil {
		t.Fatalf("ApplyBehaviors() error = %v", err)
	}

	if userID, ok := result.Headers["X-User-ID"]; !ok || userID != "789" {
		t.Errorf("X-User-ID header = %v, want %q", userID, "789")
	}
}

// TestCopyFromQuery tests copying from query parameters
func TestCopyFromQuery(t *testing.T) {

	jsEngine := NewJSEngine()
	executor := NewBehaviorExecutor(jsEngine)

	req := &models.Request{
		Method: "GET",
		Path:   "/search",
		Query: map[string]string{
			"q":    "golang",
			"page": "2",
		},
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers:    make(map[string]interface{}),
		Body:       "Search query: ${query}",
	}

	behavior := models.Behavior{
		Copy: []models.Copy{
			{
				From: map[string]interface{}{
					"query": "q",
				},
				Into: "${query}",
				Using: &models.Using{
					Method:   "regex",
					Selector: ".*",
				},
			},
		},
	}

	result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
	if err != nil {
		t.Fatalf("ApplyBehaviors() error = %v", err)
	}

	wantBody := "Search query: golang"
	if bodyStr, ok := result.Body.(string); ok {
		if bodyStr != wantBody {
			t.Errorf("Body = %q, want %q", bodyStr, wantBody)
		}
	}
}
