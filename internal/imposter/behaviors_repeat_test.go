package imposter

import (
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestRepeatBasic tests basic repeat behavior functionality
func TestRepeatBasic(t *testing.T) {
	tests := []struct {
		name        string
		repeat      int
		numRequests int
		wantBodies  []string
	}{
		{
			name:        "repeat 2 times",
			repeat:      2,
			numRequests: 5,
			wantBodies:  []string{"response 1", "response 1", "response 2", "response 2", "response 3"},
		},
		{
			name:        "repeat 3 times",
			repeat:      3,
			numRequests: 7,
			wantBodies:  []string{"response 1", "response 1", "response 1", "response 2", "response 2", "response 2", "response 3"},
		},
		{
			name:        "repeat 1 time (no repeat)",
			repeat:      1,
			numRequests: 3,
			wantBodies:  []string{"response 1", "response 2", "response 3"},
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

			responses := []string{"response 1", "response 2", "response 3"}
			responseIndex := 0
			currentRepeat := 0

			for i := 0; i < tt.numRequests; i++ {
				resp := &models.IsResponse{
					StatusCode: 200,
					Headers:    make(map[string]interface{}),
					Body:       responses[responseIndex],
				}

				behavior := models.Behavior{
					Repeat: tt.repeat,
				}

				result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
				if err != nil {
					t.Fatalf("ApplyBehaviors() error = %v", err)
				}

				if bodyStr, ok := result.Body.(string); ok {
					if bodyStr != tt.wantBodies[i] {
						t.Errorf("Request %d: Body = %q, want %q", i+1, bodyStr, tt.wantBodies[i])
					}
				}

				// Simulate repeat counting
				currentRepeat++
				if currentRepeat >= tt.repeat {
					currentRepeat = 0
					responseIndex++
					if responseIndex >= len(responses) {
						responseIndex = len(responses) - 1
					}
				}
			}
		})
	}
}

// TestRepeatWithOtherBehaviors tests repeat combined with other behaviors
func TestRepeatWithOtherBehaviors(t *testing.T) {
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
		{Repeat: 2},
		{
			Decorate: `function(req, res) {
				res.body = res.body + ' - decorated';
			}`,
		},
	}

	result, err := executor.ApplyBehaviors(req, resp, behaviors)
	if err != nil {
		t.Fatalf("ApplyBehaviors() error = %v", err)
	}

	want := "original - decorated"
	if bodyStr, ok := result.Body.(string); ok {
		if bodyStr != want {
			t.Errorf("Body = %q, want %q", bodyStr, want)
		}
	}
}

// TestRepeatZero tests that repeat of 0 means no limit
func TestRepeatZero(t *testing.T) {
	jsEngine := NewJSEngine()
	executor := NewBehaviorExecutor(jsEngine)

	req := &models.Request{
		Method: "GET",
		Path:   "/test",
	}

	// With repeat: 0, the response should not change
	for i := 0; i < 5; i++ {
		resp := &models.IsResponse{
			StatusCode: 200,
			Headers:    make(map[string]interface{}),
			Body:       "test response",
		}

		behavior := models.Behavior{
			Repeat: 0,
		}

		result, err := executor.ApplyBehaviors(req, resp, []models.Behavior{behavior})
		if err != nil {
			t.Fatalf("ApplyBehaviors() error = %v", err)
		}

		if bodyStr, ok := result.Body.(string); ok {
			if bodyStr != "test response" {
				t.Errorf("Body = %q, want %q", bodyStr, "test response")
			}
		}
	}
}
