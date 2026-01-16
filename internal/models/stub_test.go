package models

import (
	"encoding/json"
	"testing"
)

// TestBehaviorUnmarshalSingleObject tests that _behaviors can be unmarshaled from an object
// This is the root cause of behavior test failures - mountebank accepts both object and array formats
func TestBehaviorUnmarshalSingleObject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *Response)
	}{
		{
			name: "single wait behavior as object",
			input: `{
				"is": {"body": "test"},
				"_behaviors": {"wait": 1000}
			}`,
			wantErr: false,
			validate: func(t *testing.T, r *Response) {
				if len(r.Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(r.Behaviors))
				}
				if r.Behaviors[0].Wait == nil {
					t.Error("expected wait behavior to be set")
				}
				// Wait can be int or string (function)
				waitVal, ok := r.Behaviors[0].Wait.(float64)
				if !ok || waitVal != 1000 {
					t.Errorf("expected wait=1000, got %v (type %T)", r.Behaviors[0].Wait, r.Behaviors[0].Wait)
				}
			},
		},
		{
			name: "single decorate behavior as object",
			input: `{
				"is": {"body": "test"},
				"_behaviors": {"decorate": "function() {}"}
			}`,
			wantErr: false,
			validate: func(t *testing.T, r *Response) {
				if len(r.Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(r.Behaviors))
				}
				if r.Behaviors[0].Decorate != "function() {}" {
					t.Errorf("expected decorate to be set, got %q", r.Behaviors[0].Decorate)
				}
			},
		},
		{
			name: "multiple behaviors in object",
			input: `{
				"is": {"body": "test"},
				"_behaviors": {"wait": 500, "repeat": 2}
			}`,
			wantErr: false,
			validate: func(t *testing.T, r *Response) {
				if len(r.Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(r.Behaviors))
				}
				if r.Behaviors[0].Wait == nil {
					t.Error("expected wait to be set")
				}
				if r.Behaviors[0].Repeat != 2 {
					t.Errorf("expected repeat=2, got %d", r.Behaviors[0].Repeat)
				}
			},
		},
		{
			name: "behaviors as array (current working format)",
			input: `{
				"is": {"body": "test"},
				"_behaviors": [{"wait": 1000}]
			}`,
			wantErr: false,
			validate: func(t *testing.T, r *Response) {
				if len(r.Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(r.Behaviors))
				}
			},
		},
		{
			name: "copy behavior as object",
			input: `{
				"is": {"body": "test"},
				"_behaviors": {
					"copy": {
						"from": "path",
						"into": "${DEST}",
						"using": {"method": "regex", "selector": ".*"}
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, r *Response) {
				if len(r.Behaviors) != 1 {
					t.Errorf("expected 1 behavior, got %d", len(r.Behaviors))
				}
				if len(r.Behaviors[0].Copy) == 0 {
					t.Fatal("expected copy behavior to be set")
				}
				if r.Behaviors[0].Copy[0].Into != "${DEST}" {
					t.Errorf("expected copy.into=${DEST}, got %q", r.Behaviors[0].Copy[0].Into)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp Response
			err := json.Unmarshal([]byte(tt.input), &resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, &resp)
			}
		})
	}
}

// TestImposterWithBehaviors tests full imposter creation with behaviors
func TestImposterWithBehaviors(t *testing.T) {
	input := `{
		"protocol": "http",
		"port": 3000,
		"stubs": [
			{
				"responses": [
					{
						"is": {"statusCode": 200, "body": "OK"},
						"_behaviors": {"wait": 1000}
					}
				]
			}
		]
	}`

	var imp Imposter
	err := json.Unmarshal([]byte(input), &imp)
	if err != nil {
		t.Fatalf("Failed to unmarshal imposter with behaviors: %v", err)
	}

	if len(imp.Stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(imp.Stubs))
	}
	if len(imp.Stubs[0].Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(imp.Stubs[0].Responses))
	}
	if len(imp.Stubs[0].Responses[0].Behaviors) != 1 {
		t.Errorf("expected 1 behavior, got %d", len(imp.Stubs[0].Responses[0].Behaviors))
	}
}

// TestBehaviorMarshalRoundTrip tests that behaviors can be marshaled and unmarshaled
func TestBehaviorMarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "object format",
			input: `{"is":{"body":"test"},"_behaviors":{"wait":1000}}`,
		},
		{
			name:  "array format",
			input: `{"is":{"body":"test"},"_behaviors":[{"wait":1000}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp Response
			if err := json.Unmarshal([]byte(tt.input), &resp); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Marshal back
			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal again
			var resp2 Response
			if err := json.Unmarshal(data, &resp2); err != nil {
				t.Fatalf("Second unmarshal failed: %v", err)
			}

			// Verify behaviors match
			if len(resp.Behaviors) != len(resp2.Behaviors) {
				t.Errorf("behavior count mismatch: %d vs %d", len(resp.Behaviors), len(resp2.Behaviors))
			}
		})
	}
}
