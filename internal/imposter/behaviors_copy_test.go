package imposter

import (
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestCopyWithRegex tests copy behavior using regex selectors
func TestCopyWithRegex(t *testing.T) {
	t.Skip("Copy behavior not yet implemented - test created to guide implementation")

	tests := []struct {
		name         string
		copyBehavior models.Copy
		requestPath  string
		requestBody  interface{}
		initialBody  string
		wantBody     string
	}{
		{
			name: "copy from path using regex",
			copyBehavior: models.Copy{
				From: map[string]interface{}{
					"path": "$PATH",
				},
				Into: "${body}",
				Using: &models.Using{
					Method:   "regex",
					Selector: "/users/(\\d+)",
				},
			},
			requestPath: "/users/123",
			initialBody: "User ID: ",
			wantBody:    "User ID: 123",
		},
		{
			name: "copy multiple capture groups",
			copyBehavior: models.Copy{
				From: map[string]interface{}{
					"path": "$PATH",
				},
				Into: "${body}",
				Using: &models.Using{
					Method:   "regex",
					Selector: "/api/(\\w+)/(\\d+)",
				},
			},
			requestPath: "/api/users/456",
			initialBody: "Resource: ",
			wantBody:    "Resource: users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewBehaviorExecutor()

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
				Copy: &tt.copyBehavior,
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
	t.Skip("Copy behavior not yet implemented - test created to guide implementation")

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
				Into: "${body}",
				Using: &models.Using{
					Method:   "jsonpath",
					Selector: "$",
				},
			},
			requestBody: `{"user":{"name":"Jane","age":30}}`,
			initialBody: "Hello ",
			wantBody:    "Hello Jane",
		},
		{
			name: "copy from nested JSON array",
			copyBehavior: models.Copy{
				From: map[string]interface{}{
					"body": "$.items[0].id",
				},
				Into: "${body}",
				Using: &models.Using{
					Method:   "jsonpath",
					Selector: "$",
				},
			},
			requestBody: `{"items":[{"id":"abc123","name":"Item1"}]}`,
			initialBody: "First item: ",
			wantBody:    "First item: abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewBehaviorExecutor()

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
				Copy: &tt.copyBehavior,
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
	t.Skip("Copy behavior not yet implemented - test created to guide implementation")

	executor := NewBehaviorExecutor()

	req := &models.Request{
		Method: "GET",
		Path:   "/users/789",
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers:    make(map[string]interface{}),
		Body:       "test",
	}

	behavior := models.Behavior{
		Copy: &models.Copy{
			From: map[string]interface{}{
				"path": "$PATH",
			},
			Into: "${headers}['X-User-ID']",
			Using: &models.Using{
				Method:   "regex",
				Selector: "/users/(\\d+)",
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
	t.Skip("Copy behavior not yet implemented - test created to guide implementation")

	executor := NewBehaviorExecutor()

	req := &models.Request{
		Method: "GET",
		Path:   "/search",
		Query: map[string]interface{}{
			"q":    "golang",
			"page": "2",
		},
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers:    make(map[string]interface{}),
		Body:       "Search query: ",
	}

	behavior := models.Behavior{
		Copy: &models.Copy{
			From: map[string]interface{}{
				"query": "q",
			},
			Into: "${body}",
			Using: &models.Using{
				Method:   "regex",
				Selector: ".*",
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
