package imposter

import (
	"testing"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestWaitBasic tests basic wait behavior with static millisecond values
func TestWaitBasic(t *testing.T) {
	tests := []struct {
		name        string
		wait        interface{}
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:        "wait 100ms",
			wait:        100,
			minDuration: 90 * time.Millisecond,
			maxDuration: 150 * time.Millisecond,
		},
		{
			name:        "wait 0ms (no delay)",
			wait:        0,
			minDuration: 0,
			maxDuration: 10 * time.Millisecond,
		},
		{
			name:        "wait 50ms",
			wait:        50,
			minDuration: 40 * time.Millisecond,
			maxDuration: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsEngine := NewJSEngine()
			executor := NewBehaviorExecutor(jsEngine)

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
				Wait: tt.wait,
			}

			start := time.Now()
			_, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("ApplyBehaviors() error = %v", err)
			}

			if elapsed < tt.minDuration {
				t.Errorf("Wait duration = %v, want >= %v", elapsed, tt.minDuration)
			}
			if elapsed > tt.maxDuration {
				t.Errorf("Wait duration = %v, want <= %v", elapsed, tt.maxDuration)
			}
		})
	}
}

// TestWaitWithFunction tests wait behavior using JavaScript functions
func TestWaitWithFunction(t *testing.T) {
	tests := []struct {
		name        string
		waitFunc    string
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:        "function returns fixed delay",
			waitFunc:    "function() { return 100; }",
			minDuration: 90 * time.Millisecond,
			maxDuration: 150 * time.Millisecond,
		},
		{
			name:        "function with request access",
			waitFunc:    "function(request) { return request.path === '/test' ? 50 : 0; }",
			minDuration: 40 * time.Millisecond,
			maxDuration: 100 * time.Millisecond,
		},
		{
			name:        "function returns 0 (no delay)",
			waitFunc:    "function() { return 0; }",
			minDuration: 0,
			maxDuration: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsEngine := NewJSEngine()
			executor := NewBehaviorExecutor(jsEngine)

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
				Wait: tt.waitFunc,
			}

			start := time.Now()
			_, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("ApplyBehaviors() error = %v", err)
			}

			if elapsed < tt.minDuration {
				t.Errorf("Wait duration = %v, want >= %v", elapsed, tt.minDuration)
			}
			if elapsed > tt.maxDuration {
				t.Errorf("Wait duration = %v, want <= %v", elapsed, tt.maxDuration)
			}
		})
	}
}

// TestWaitMultipleBehaviors tests wait with other behaviors
func TestWaitMultipleBehaviors(t *testing.T) {
	jsEngine := NewJSEngine()
	executor := NewBehaviorExecutor(jsEngine)

	req := &models.Request{
		Method: "GET",
		Path:   "/test",
	}

	resp := &models.IsResponse{
		StatusCode: 200,
		Headers:    make(map[string]interface{}),
		Body:       "original",
	}

	behaviors := []models.Behavior{
		{Wait: 50},
		{
			Decorate: `function(req, res) {
				res.body = res.body + ' - decorated';
			}`,
		},
	}

	start := time.Now()
	result, err := executor.ApplyBehaviors(req, resp, behaviors)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ApplyBehaviors() error = %v", err)
	}

	// Check wait occurred
	if elapsed < 40*time.Millisecond {
		t.Errorf("Wait duration = %v, want >= 40ms", elapsed)
	}

	// Check decorate was applied
	if bodyStr, ok := result.Body.(string); ok {
		want := "original - decorated"
		if bodyStr != want {
			t.Errorf("Body = %q, want %q", bodyStr, want)
		}
	}
}

// TestWaitErrorHandling tests error cases in wait behavior
func TestWaitErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		wait      interface{}
		wantError bool
	}{
		{
			name:      "function throws error",
			wait:      "function() { throw new Error('test error'); }",
			wantError: true,
		},
		{
			name:      "function returns non-number",
			wait:      "function() { return 'invalid'; }",
			wantError: true,
		},
		{
			name:      "negative wait value",
			wait:      -100,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsEngine := NewJSEngine()
			executor := NewBehaviorExecutor(jsEngine)

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
				Wait: tt.wait,
			}

			_, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
			if (err != nil) != tt.wantError {
				t.Errorf("ApplyBehaviors() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
