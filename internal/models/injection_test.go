package models

import (
	"encoding/json"
	"testing"
)

// TestPredicateInjectionUnmarshal tests that predicates with inject field can be unmarshaled
func TestPredicateInjectionUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *Stub)
	}{
		{
			name: "predicate with inject",
			input: `{
				"predicates": [{"inject": "function(request) { return true; }"}],
				"responses": [{"is": {"body": "test"}}]
			}`,
			wantErr: false,
			validate: func(t *testing.T, s *Stub) {
				if len(s.Predicates) != 1 {
					t.Fatalf("expected 1 predicate, got %d", len(s.Predicates))
				}
				if s.Predicates[0].Inject == "" {
					t.Error("expected inject to be set")
				}
				if s.Predicates[0].Inject != "function(request) { return true; }" {
					t.Errorf("inject mismatch: got %q", s.Predicates[0].Inject)
				}
			},
		},
		{
			name: "response with inject",
			input: `{
				"responses": [{"inject": "function(request) { return {body: 'test'}; }"}]
			}`,
			wantErr: false,
			validate: func(t *testing.T, s *Stub) {
				if len(s.Responses) != 1 {
					t.Fatalf("expected 1 response, got %d", len(s.Responses))
				}
				if s.Responses[0].Inject == "" {
					t.Error("expected inject to be set")
				}
			},
		},
		{
			name: "combined predicates and response injection",
			input: `{
				"predicates": [{"inject": "config => config.request.path === '/test'"}],
				"responses": [{"inject": "config => ({body: config.request.method})"}]
			}`,
			wantErr: false,
			validate: func(t *testing.T, s *Stub) {
				if len(s.Predicates) != 1 {
					t.Fatalf("expected 1 predicate, got %d", len(s.Predicates))
				}
				if len(s.Responses) != 1 {
					t.Fatalf("expected 1 response, got %d", len(s.Responses))
				}
				if s.Predicates[0].Inject == "" {
					t.Error("expected predicate inject to be set")
				}
				if s.Responses[0].Inject == "" {
					t.Error("expected response inject to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stub Stub
			err := json.Unmarshal([]byte(tt.input), &stub)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, &stub)
			}
		})
	}
}

// TestImposterWithInjection tests full imposter creation with injection
func TestImposterWithInjection(t *testing.T) {
	input := `{
		"protocol": "http",
		"port": 3000,
		"stubs": [
			{
				"predicates": [{"inject": "request => request.path === '/test'"}],
				"responses": [{"inject": "request => ({body: 'INJECTED'})"}]
			}
		]
	}`

	var imp Imposter
	err := json.Unmarshal([]byte(input), &imp)
	if err != nil {
		t.Fatalf("Failed to unmarshal imposter with injection: %v", err)
	}

	if len(imp.Stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(imp.Stubs))
	}
	if len(imp.Stubs[0].Predicates) != 1 {
		t.Fatalf("expected 1 predicate, got %d", len(imp.Stubs[0].Predicates))
	}
	if imp.Stubs[0].Predicates[0].Inject == "" {
		t.Error("expected predicate inject to be set")
	}
	if len(imp.Stubs[0].Responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(imp.Stubs[0].Responses))
	}
	if imp.Stubs[0].Responses[0].Inject == "" {
		t.Error("expected response inject to be set")
	}
}

// TestEndOfRequestResolverInjection tests TCP end-of-request resolver injection
func TestEndOfRequestResolverInjection(t *testing.T) {
	input := `{
		"protocol": "tcp",
		"port": 3000,
		"mode": "text",
		"endOfRequestResolver": {
			"inject": "function(requestData, logger) { return requestData.indexOf('END') > -1; }"
		}
	}`

	var imp Imposter
	err := json.Unmarshal([]byte(input), &imp)
	if err != nil {
		t.Fatalf("Failed to unmarshal imposter with endOfRequestResolver: %v", err)
	}

	if imp.EndOfRequestResolver == nil {
		t.Fatal("expected endOfRequestResolver to be set")
	}
	if imp.EndOfRequestResolver.Inject == "" {
		t.Error("expected endOfRequestResolver inject to be set")
	}
}
