package imposter

import (
	"testing"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/require"
	"github.com/stretchr/testify/assert"
)

func TestGojaNodeEngineBufferApi(t *testing.T) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)

	_, err := vm.RunString(`
	var b = Buffer.from('Hello, World', 'utf8');
	b.toString('base64');`)
	if err != nil {
		t.Errorf("Buffer.validation check failed: %v", err)
	}
}

func TestJSEngine_ExecuteResponse(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		script  string
		req     *models.Request
		want    *models.IsResponse
		wantErr bool
	}{
		{
			name:    "Ensure Buffer API is available",
			script:  "function(request, state, logger) { return { statusCode: 200, body: Buffer.from('Hello, World').toString('base64') }; }",
			req:     &models.Request{Method: "GET", Path: "/test"},
			want:    &models.IsResponse{StatusCode: 200, Body: "SGVsbG8sIFdvcmxk"},
			wantErr: false,
		},
		{
			name:   "Ensure console API is available",
			script: `function(request, state, logger) { console.log("Test log message"); return { statusCode: 200, body: "Hello." }; }`,
			req:    &models.Request{Method: "GET", Path: "/test"},
			want:   &models.IsResponse{StatusCode: 200, Body: "Hello."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewJSEngine()
			state := make(map[string]interface{}) // Imposter state for injection
			got, gotErr := e.ExecuteResponse(tt.script, tt.req, state)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ExecuteResponse() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ExecuteResponse() succeeded unexpectedly")
			}
			assert.Equal(got, tt.want, "ExecuteResponse() = %v, want %v", got, tt.want)
		})
	}
}

// TestJSEngine_ExecuteResponse_WithState tests that imposter state persists across calls
func TestJSEngine_ExecuteResponse_WithState(t *testing.T) {
	engine := NewJSEngine()
	state := make(map[string]interface{})
	req := &models.Request{Method: "GET", Path: "/"}

	// Script that increments counter in imposterState
	// Mountebank's old interface is: (config, injectState, logger, callback, imposterState)
	// where config has request fields flattened onto it
	script := `function(request, injectState, logger, callback, imposterState) {
		imposterState.counter = (imposterState.counter || 0) + 1;
		return { statusCode: 200, body: String(imposterState.counter) };
	}`

	resp1, err := engine.ExecuteResponse(script, req, state)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	resp2, err := engine.ExecuteResponse(script, req, state)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	if resp1.Body != "1" {
		t.Errorf("First response = %q, want %q", resp1.Body, "1")
	}
	if resp2.Body != "2" {
		t.Errorf("Second response = %q, want %q", resp2.Body, "2")
	}
}

// TestJSEngine_ExecutePredicate_WithState tests that imposter state persists across predicate calls
func TestJSEngine_ExecutePredicate_WithState(t *testing.T) {
	engine := NewJSEngine()
	state := make(map[string]interface{})
	req := &models.Request{Method: "GET", Path: "/"}

	// Script that tracks hits and returns false after 2 calls
	script := `function(request, logger, imposterState) {
		imposterState.hits = (imposterState.hits || 0) + 1;
		return imposterState.hits < 3;
	}`

	result1, err := engine.ExecutePredicate(script, req, state)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	result2, err := engine.ExecutePredicate(script, req, state)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	result3, err := engine.ExecutePredicate(script, req, state)
	if err != nil {
		t.Fatalf("Third call failed: %v", err)
	}

	if !result1 {
		t.Error("First call: expected true, got false")
	}
	if !result2 {
		t.Error("Second call: expected true, got false")
	}
	if result3 {
		t.Error("Third call: expected false, got true")
	}
}

// TestJSEngine_SharedState_PredicateAndResponse tests state sharing between predicate and response
func TestJSEngine_SharedState_PredicateAndResponse(t *testing.T) {
	engine := NewJSEngine()
	state := make(map[string]interface{})
	req := &models.Request{Method: "GET", Path: "/"}

	// Predicate sets a value in state
	// Mountebank predicate interface is: (config, logger, imposterState)
	predicateScript := `function(config, logger, imposterState) {
		imposterState.message = 'SET_BY_PREDICATE';
		return true;
	}`

	// Response reads the value set by predicate
	// Mountebank response interface is: (config, injectState, logger, callback, imposterState)
	responseScript := `function(config, injectState, logger, callback, imposterState) {
		return { statusCode: 200, body: imposterState.message || 'NOT_SET' };
	}`

	// Call predicate first
	_, err := engine.ExecutePredicate(predicateScript, req, state)
	if err != nil {
		t.Fatalf("Predicate failed: %v", err)
	}

	// Then call response - should see value set by predicate
	resp, err := engine.ExecuteResponse(responseScript, req, state)
	if err != nil {
		t.Fatalf("Response failed: %v", err)
	}

	if resp.Body != "SET_BY_PREDICATE" {
		t.Errorf("Response body = %q, want %q (state should flow from predicate to response)", resp.Body, "SET_BY_PREDICATE")
	}
}
