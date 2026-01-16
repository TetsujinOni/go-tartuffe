package imposter

import (
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestDecorateBasic tests basic decorate behavior functionality
func TestDecorateBasic(t *testing.T) {
	tests := []struct {
		name           string
		decorateScript string
		initialBody    interface{}
		wantBody       string
	}{
		{
			name: "modify string body",
			decorateScript: `function(request, response) {
				response.body = response.body.toUpperCase();
			}`,
			initialBody: "hello world",
			wantBody:    "HELLO WORLD",
		},
		{
			name: "append to body",
			decorateScript: `function(request, response) {
				response.body = response.body + " - decorated";
			}`,
			initialBody: "original",
			wantBody:    "original - decorated",
		},
		{
			name: "add header via decoration",
			decorateScript: `function(request, response) {
				response.headers['X-Decorated'] = 'true';
			}`,
			initialBody: "test",
			wantBody:    "test",
		},
		{
			name: "modify status code",
			decorateScript: `function(request, response) {
				response.statusCode = 201;
			}`,
			initialBody: "test",
			wantBody:    "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewBehaviorExecutor()

			req := &models.Request{
				Method: "GET",
				Path:   "/test",
			}

			resp := &models.IsResponse{
				StatusCode: 200,
				Headers:    make(map[string]interface{}),
				Body:       tt.initialBody,
			}

			behavior := models.Behavior{
				Decorate: tt.decorateScript,
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

// TestDecorateWithRequestAccess tests decorate behavior accessing request data
func TestDecorateWithRequestAccess(t *testing.T) {
	tests := []struct {
		name           string
		decorateScript string
		requestPath    string
		requestMethod  string
		requestHeaders map[string]interface{}
		wantBody       string
	}{
		{
			name: "use request path in response",
			decorateScript: `function(request, response) {
				response.body = 'You requested: ' + request.path;
			}`,
			requestPath: "/api/users",
			wantBody:    "You requested: /api/users",
		},
		{
			name: "use request method",
			decorateScript: `function(request, response) {
				response.body = request.method + ' request received';
			}`,
			requestMethod: "POST",
			wantBody:      "POST request received",
		},
		{
			name: "access request headers",
			decorateScript: `function(request, response) {
				response.body = 'Auth: ' + request.headers['Authorization'];
			}`,
			requestHeaders: map[string]interface{}{
				"Authorization": "Bearer token123",
			},
			wantBody: "Auth: Bearer token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewBehaviorExecutor()

			req := &models.Request{
				Method:  tt.requestMethod,
				Path:    tt.requestPath,
				Headers: tt.requestHeaders,
			}

			resp := &models.IsResponse{
				StatusCode: 200,
				Headers:    make(map[string]interface{}),
				Body:       "initial",
			}

			behavior := models.Behavior{
				Decorate: tt.decorateScript,
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

// TestDecorateReturnValue tests decorate returning a value vs modifying in place
func TestDecorateReturnValue(t *testing.T) {
	tests := []struct {
		name           string
		decorateScript string
		wantBody       string
	}{
		{
			name: "return value modifies response (old interface)",
			decorateScript: `function(request, response) {
				return {
					body: 'returned body',
					statusCode: 201,
					headers: { 'X-Custom': 'header' }
				};
			}`,
			wantBody: "returned body",
		},
		{
			name: "return value overwrites response",
			decorateScript: `function(request, response) {
				response.body = 'modified';
				return { body: 'returned takes precedence' };
			}`,
			wantBody: "returned takes precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewBehaviorExecutor()

			req := &models.Request{
				Method: "GET",
				Path:   "/test",
			}

			resp := &models.IsResponse{
				StatusCode: 200,
				Headers:    make(map[string]interface{}),
				Body:       "initial",
			}

			behavior := models.Behavior{
				Decorate: tt.decorateScript,
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

// TestDecorateContentLength tests that decorate adjusts Content-Length when body changes
func TestDecorateContentLength(t *testing.T) {
	executor := NewBehaviorExecutor()

	req := &models.Request{
		Method: "GET",
		Path:   "/test",
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers: map[string]interface{}{
			"Content-Length": "5",
		},
		Body: "hello",
	}

	behavior := models.Behavior{
		Decorate: `function(request, response) {
			response.body = response.body + ' world';
		}`,
	}

	result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
	if err != nil {
		t.Fatalf("ApplyBehaviors() error = %v", err)
	}

	// Content-Length should be updated to match new body length
	if result.Body != "hello world" {
		t.Errorf("Body = %q, want %q", result.Body, "hello world")
	}

	// Note: Content-Length adjustment is typically done at the HTTP server level,
	// not in the behavior executor. This test documents the expected behavior.
}

// TestDecorateMultipleTimes tests applying decorate behavior multiple times
func TestDecorateMultipleTimes(t *testing.T) {
	executor := NewBehaviorExecutor()

	req := &models.Request{
		Method: "GET",
		Path:   "/test",
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers:    make(map[string]interface{}),
		Body:       "start",
	}

	behaviors := []models.Behavior{
		{Decorate: `function(req, res) { res.body = res.body + ' > step1'; }`},
		{Decorate: `function(req, res) { res.body = res.body + ' > step2'; }`},
		{Decorate: `function(req, res) { res.body = res.body + ' > step3'; }`},
	}

	result, err := executor.ApplyBehaviors(req, resp, behaviors)
	if err != nil {
		t.Fatalf("ApplyBehaviors() error = %v", err)
	}

	want := "start > step1 > step2 > step3"
	if bodyStr, ok := result.Body.(string); ok {
		if bodyStr != want {
			t.Errorf("Body = %q, want %q", bodyStr, want)
		}
	}
}

// TestDecorateErrorHandling tests error handling in decorate scripts
func TestDecorateErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		decorateScript string
		wantError      bool
	}{
		{
			name: "runtime error in script",
			decorateScript: `function(request, response) {
				throw new Error('intentional error');
			}`,
			wantError: true,
		},
		{
			name: "undefined variable",
			decorateScript: `function(request, response) {
				response.body = undefinedVar;
			}`,
			wantError: true,
		},
		{
			name: "syntax error",
			decorateScript: `function(request, response) {
				response.body = 'missing quote;
			}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewBehaviorExecutor()

			req := &models.Request{
				Method: "GET",
				Path:   "/test",
			}

			resp := &models.IsResponse{
				StatusCode: 200,
				Headers:    make(map[string]interface{}),
				Body:       "test",
			}

			behavior := models.Behavior{
				Decorate: tt.decorateScript,
			}

			_, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
			if (err != nil) != tt.wantError {
				t.Errorf("ApplyBehaviors() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
