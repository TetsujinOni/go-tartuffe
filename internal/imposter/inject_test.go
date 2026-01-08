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
			got, gotErr := e.ExecuteResponse(tt.script, tt.req)
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
