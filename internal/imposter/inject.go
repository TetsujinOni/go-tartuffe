package imposter

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

// scriptPreviewLength is the max length of script shown in error messages
const scriptPreviewLength = 100

// quoteJSString quotes a string for use in JavaScript, escaping special characters
func quoteJSString(s string) string {
	// Use JSON encoding which properly escapes for JavaScript
	b, _ := json.Marshal(s)
	return string(b)
}

// JSLogger provides logging functions to JavaScript code
type JSLogger struct {
	context string // e.g., "inject:response", "inject:predicate"
}

// NewJSLogger creates a logger with the given context
func NewJSLogger(context string) *JSLogger {
	return &JSLogger{context: context}
}

// createLoggerObject creates a logger object for the Goja VM
func (l *JSLogger) createLoggerObject() map[string]interface{} {
	return map[string]interface{}{
		"debug": func(call goja.FunctionCall) goja.Value {
			l.log("DEBUG", call.Arguments)
			return goja.Undefined()
		},
		"info": func(call goja.FunctionCall) goja.Value {
			l.log("INFO", call.Arguments)
			return goja.Undefined()
		},
		"warn": func(call goja.FunctionCall) goja.Value {
			l.log("WARN", call.Arguments)
			return goja.Undefined()
		},
		"error": func(call goja.FunctionCall) goja.Value {
			l.log("ERROR", call.Arguments)
			return goja.Undefined()
		},
	}
}

// log formats and outputs a log message
func (l *JSLogger) log(level string, args []goja.Value) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%v", arg.Export()))
	}
	msg := strings.Join(parts, " ")
	log.Printf("[%s] [%s] %s", level, l.context, msg)
}

// scriptPreview returns a truncated preview of a script for error messages
func scriptPreview(script string) string {
	// Normalize whitespace
	script = strings.Join(strings.Fields(script), " ")
	if len(script) > scriptPreviewLength {
		return script[:scriptPreviewLength] + "..."
	}
	return script
}

// formatJSError formats a JavaScript error with stack trace if available
func formatJSError(err error, script string, reqInfo string) error {
	preview := scriptPreview(script)

	// Check if it's a Goja exception with stack trace
	if exception, ok := err.(*goja.Exception); ok {
		return fmt.Errorf("JavaScript error: %s\n  Script: %s\n  Request: %s\n  Stack: %s",
			exception.Value().String(), preview, reqInfo, exception.String())
	}

	return fmt.Errorf("JavaScript error: %v\n  Script: %s\n  Request: %s", err, preview, reqInfo)
}

// formatRequestInfo creates a brief request description for error messages
func formatRequestInfo(req *models.Request) string {
	if req == nil {
		return "<nil request>"
	}
	return fmt.Sprintf("%s %s", req.Method, req.Path)
}

// JSEngine handles JavaScript injection execution
type JSEngine struct {
	registry *require.Registry // Node.js module registry for Goja Runtimes
}

// NewJSEngine creates a new JavaScript engine
func NewJSEngine() *JSEngine {
	reg := require.NewRegistry()
	return &JSEngine{
		registry: reg,
	}
}

// ExecuteResponse executes an inject script and returns the response
func (e *JSEngine) ExecuteResponse(script string, req *models.Request) (*models.IsResponse, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)
	console.Enable(vm)
	jsLogger := NewJSLogger("inject:response")

	// Set up the request object
	reqObj := map[string]interface{}{
		"method":      req.Method,
		"path":        req.Path,
		"query":       req.Query,
		"headers":     req.Headers,
		"body":        req.Body,
		"requestFrom": req.RequestFrom,
	}

	vm.Set("request", reqObj)
	vm.Set("logger", jsLogger.createLoggerObject())

	// Set up state object (empty but available)
	vm.Set("state", map[string]interface{}{})

	// Wrap the script in a function call
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			return fn(request, state, logger);
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return nil, formatJSError(err, script, formatRequestInfo(req))
	}

	// Convert result to IsResponse
	return e.convertToResponse(result)
}

// ExecutePredicate executes an inject predicate script
func (e *JSEngine) ExecutePredicate(script string, req *models.Request) (bool, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)

	jsLogger := NewJSLogger("inject:predicate")

	// Set up the request object
	reqObj := map[string]interface{}{
		"method":      req.Method,
		"path":        req.Path,
		"query":       req.Query,
		"headers":     req.Headers,
		"body":        req.Body,
		"requestFrom": req.RequestFrom,
	}

	vm.Set("request", reqObj)
	vm.Set("logger", jsLogger.createLoggerObject())

	// Wrap the script in a function call
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			return fn(request, logger);
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return false, formatJSError(err, script, formatRequestInfo(req))
	}

	return result.ToBoolean(), nil
}

// convertToResponse converts a goja value to an IsResponse
func (e *JSEngine) convertToResponse(val goja.Value) (*models.IsResponse, error) {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return &models.IsResponse{StatusCode: 200}, nil
	}

	// Export to Go value
	exported := val.Export()

	// Try to convert to map
	respMap, ok := exported.(map[string]interface{})
	if !ok {
		// If it's a string, use as body
		if str, ok := exported.(string); ok {
			return &models.IsResponse{
				StatusCode: 200,
				Body:       str,
			}, nil
		}
		return nil, fmt.Errorf("inject must return an object or string, got %T", exported)
	}

	resp := &models.IsResponse{}

	// Extract statusCode
	if sc, ok := respMap["statusCode"]; ok {
		switch v := sc.(type) {
		case int64:
			resp.StatusCode = int(v)
		case float64:
			resp.StatusCode = int(v)
		case int:
			resp.StatusCode = v
		}
	}
	if resp.StatusCode == 0 {
		resp.StatusCode = 200
	}

	// Extract headers
	if h, ok := respMap["headers"]; ok {
		if headersMap, ok := h.(map[string]interface{}); ok {
			resp.Headers = make(map[string]interface{})
			for k, v := range headersMap {
				resp.Headers[k] = v
			}
		}
	}

	// Extract body
	if b, ok := respMap["body"]; ok {
		switch body := b.(type) {
		case string:
			resp.Body = body
		case map[string]interface{}, []interface{}:
			// JSON encode objects/arrays
			jsonBytes, err := json.Marshal(body)
			if err == nil {
				resp.Body = string(jsonBytes)
			}
		default:
			resp.Body = fmt.Sprintf("%v", body)
		}
	}

	// Extract mode
	if m, ok := respMap["_mode"]; ok {
		if mode, ok := m.(string); ok {
			resp.Mode = mode
		}
	}

	return resp, nil
}

// ExecuteEndOfRequestResolver executes the resolver script to determine if request is complete
// Returns true if the accumulated data represents a complete request
func (e *JSEngine) ExecuteEndOfRequestResolver(script string, requestData string) (bool, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)

	jsLogger := NewJSLogger("inject:endOfRequestResolver")

	// Set up the request data
	vm.Set("requestData", requestData)
	vm.Set("logger", jsLogger.createLoggerObject())

	// Wrap the script in a function call
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			return fn(requestData, logger);
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		// Include preview of request data for debugging
		dataPreview := requestData
		if len(dataPreview) > 50 {
			dataPreview = dataPreview[:50] + "..."
		}
		return false, formatJSError(err, script, fmt.Sprintf("requestData: %q", dataPreview))
	}

	return result.ToBoolean(), nil
}

// ExecuteTCPPredicate executes an inject predicate script for TCP protocol
// Supports both old interface (request, logger) and new interface (config)
func (e *JSEngine) ExecuteTCPPredicate(script string, requestData string) (bool, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)

	jsLogger := NewJSLogger("inject:tcp-predicate")

	// Set up logger object
	vm.Set("logger", jsLogger.createLoggerObject())

	// Create combined script that sets up request, config and executes the function
	// This ensures all variables are in the same scope
	wrappedScript := fmt.Sprintf(`
		(function() {
			// Set up request with Buffer data
			var request = { data: Buffer.from(%s, 'utf8') };
			var config = { request: request, logger: logger };

			var fn = %s;
			// Try new interface first (single config parameter)
			try {
				var result = fn(config);
				// If result is not undefined/null, return it
				if (result !== undefined && result !== null) {
					return result;
				}
			} catch (e) {
				// New interface failed, will try old interface
			}

			// Try old interface (request, logger)
			return fn(request, logger);
		})()
	`, quoteJSString(requestData), script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		// Include preview of request data for debugging
		dataPreview := requestData
		if len(dataPreview) > 50 {
			dataPreview = dataPreview[:50] + "..."
		}
		return false, formatJSError(err, script, fmt.Sprintf("TCP data: %q", dataPreview))
	}

	return result.ToBoolean(), nil
}

// ExecuteTCPResponse executes an inject script for TCP response
// Supports both old interface (request, state, logger) and new interface (config)
func (e *JSEngine) ExecuteTCPResponse(script string, requestData string, state map[string]interface{}) (string, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)
	console.Enable(vm)
	jsLogger := NewJSLogger("inject:tcp-response")

	// Ensure state is not nil
	if state == nil {
		state = make(map[string]interface{})
	}

	// Set state and logger in VM
	vm.Set("state", state)
	vm.Set("logger", jsLogger.createLoggerObject())

	// Create combined script that sets up request, config and executes the function
	// This ensures all variables are in the same scope
	wrappedScript := fmt.Sprintf(`
		(function() {
			// Set up request with Buffer data
			var request = { data: Buffer.from(%s, 'utf8') };
			var config = { request: request, state: state, logger: logger };

			var fn = %s;
			// Try new interface first (single config parameter)
			try {
				var result = fn(config);
				// If result is not undefined/null, return it
				if (result !== undefined && result !== null) {
					return result;
				}
			} catch (e) {
				// New interface failed, will try old interface
			}

			// Try old interface (request, state, logger)
			return fn(request, state, logger);
		})()
	`, quoteJSString(requestData), script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		// Include preview of request data for debugging
		dataPreview := requestData
		if len(dataPreview) > 50 {
			dataPreview = dataPreview[:50] + "..."
		}
		return "", formatJSError(err, script, fmt.Sprintf("TCP data: %q", dataPreview))
	}

	// Extract the data field from the returned object
	exported := result.Export()
	if respMap, ok := exported.(map[string]interface{}); ok {
		if data, ok := respMap["data"]; ok {
			return fmt.Sprintf("%v", data), nil
		}
	}

	// If not an object with data field, convert directly to string
	return fmt.Sprintf("%v", exported), nil
}
