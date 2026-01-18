package imposter

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

// scriptPreviewLength is the max length of script shown in error messages
const scriptPreviewLength = 100

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

// createSortedQueryObject creates a JavaScript object from query parameters with sorted keys
// This ensures JSON.stringify() produces consistent output regardless of Go map iteration order
func createSortedQueryObject(vm *goja.Runtime, query map[string]string) goja.Value {
	if len(query) == 0 {
		return vm.NewObject()
	}

	// Sort keys for deterministic ordering
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build JavaScript code to create object with sorted keys
	// This ensures the object has a specific key order that JSON.stringify will preserve
	var jsCode strings.Builder
	jsCode.WriteString("(function() { var obj = {}; ")
	for _, k := range keys {
		// Escape the key and value for JavaScript
		keyJSON, _ := json.Marshal(k)
		valueJSON, _ := json.Marshal(query[k])
		jsCode.WriteString(fmt.Sprintf("obj[%s] = %s; ", string(keyJSON), string(valueJSON)))
	}
	jsCode.WriteString("return obj; })()")

	result, err := vm.RunString(jsCode.String())
	if err != nil {
		// Fallback to regular object if script fails
		obj := vm.NewObject()
		for k, v := range query {
			obj.Set(k, v)
		}
		return obj
	}

	return result
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
// imposterState is the global state shared across all requests to this imposter
// Supports both interfaces:
// - NEW config interface: function(config) where config = { request, state, logger }
// - OLD multi-param interface: function(request, state, logger, callback, imposterState)
//
// For backwards compatibility, the config object has all request fields flattened onto it
// (config.method, config.path, etc.) in addition to config.request.method, etc.
// This allows old interface code like `request => request.method` to work because
// the first parameter is actually config which has method directly on it.
func (e *JSEngine) ExecuteResponse(script string, req *models.Request, imposterState map[string]interface{}) (*models.IsResponse, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)
	console.Enable(vm)
	jsLogger := NewJSLogger("inject:response")

	// Create sorted query object for deterministic JSON.stringify() output
	sortedQuery := createSortedQueryObject(vm, req.Query)

	// Set up the request object
	reqObj := vm.NewObject()
	reqObj.Set("method", req.Method)
	reqObj.Set("path", req.Path)
	reqObj.Set("query", sortedQuery)
	reqObj.Set("headers", req.Headers)
	reqObj.Set("body", req.Body)
	reqObj.Set("requestFrom", req.RequestFrom)

	vm.Set("request", reqObj)
	vm.Set("logger", jsLogger.createLoggerObject())

	// Ensure imposterState is not nil
	if imposterState == nil {
		imposterState = make(map[string]interface{})
	}

	// Set up state - use imposterState as the shared state for new config interface
	vm.Set("state", imposterState)
	vm.Set("imposterState", imposterState)

	// Wrap the script in a function call
	// Mountebank's injection works by creating a config object with:
	// - request: the request object
	// - state: the imposter state
	// - logger: the logger object
	// - callback: a done callback (for async, we don't support this)
	// AND all request fields flattened onto config (config.method, config.path, etc.)
	// This allows old interface `request => request.method` to work because
	// the first param is actually config which has method directly on it.
	//
	// The function is called as: fn(config, injectState, logger, done, imposterState)
	// where injectState is deprecated (same as imposterState for backwards compat)
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			// Create config object with request fields flattened onto it
			// This provides backwards compatibility - old interface code
			// like "request => request.method" works because the first
			// param (config) has method directly on it
			var config = {
				request: request,
				state: state,
				logger: logger,
				callback: function(response) { return response; }
			};
			// Flatten request fields onto config for backwards compatibility
			// (downcastInjectionConfig equivalent)
			Object.keys(request).forEach(function(key) {
				config[key] = request[key];
			});

			// Call function with all parameters for maximum compatibility
			// Mountebank calls: fn(config, injectState, logger, done, imposterState)
			// injectState is deprecated, we use state (same as imposterState)
			return fn(config, state, logger, config.callback, imposterState);
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
// imposterState is the global state shared across all requests to this imposter
// Supports both interfaces:
// - NEW config interface: function(config) where config = { request, state, logger }
// - OLD multi-param interface: function(request, logger, imposterState)
//
// For backwards compatibility, the config object has all request fields flattened onto it
// (config.method, config.path, etc.) in addition to config.request.method, etc.
// This allows old interface code like `request => request.path` to work because
// the first parameter is actually config which has path directly on it.
func (e *JSEngine) ExecutePredicate(script string, req *models.Request, imposterState map[string]interface{}) (bool, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)

	jsLogger := NewJSLogger("inject:predicate")

	// Create sorted query object for deterministic JSON.stringify() output
	sortedQuery := createSortedQueryObject(vm, req.Query)

	// Set up the request object
	reqObj := vm.NewObject()
	reqObj.Set("method", req.Method)
	reqObj.Set("path", req.Path)
	reqObj.Set("query", sortedQuery)
	reqObj.Set("headers", req.Headers)
	reqObj.Set("body", req.Body)
	reqObj.Set("requestFrom", req.RequestFrom)

	vm.Set("request", reqObj)
	vm.Set("logger", jsLogger.createLoggerObject())

	// Ensure imposterState is not nil
	if imposterState == nil {
		imposterState = make(map[string]interface{})
	}
	// Use imposterState as the shared state for new config interface
	vm.Set("state", imposterState)
	vm.Set("imposterState", imposterState)

	// Wrap the script in a function call
	// Mountebank's injection works by creating a config object with:
	// - request: the request object
	// - state: the imposter state
	// - logger: the logger object
	// AND all request fields flattened onto config (config.method, config.path, etc.)
	// This allows old interface `request => request.path` to work because
	// the first param is actually config which has path directly on it.
	//
	// The function is called as: fn(config, logger, imposterState)
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			// Create config object with request fields flattened onto it
			// This provides backwards compatibility - old interface code
			// like "request => request.path === '/test'" works because the first
			// param (config) has path directly on it
			var config = {
				request: request,
				state: state,
				logger: logger
			};
			// Flatten request fields onto config for backwards compatibility
			// (downcastInjectionConfig equivalent)
			Object.keys(request).forEach(function(key) {
				config[key] = request[key];
			});

			// Call function with all parameters for maximum compatibility
			// Mountebank calls: fn(config, logger, imposterState)
			return fn(config, logger, imposterState);
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

// ExecutePredicateGenerator executes a predicate generator script
// Returns an array of predicates generated from the request
// The script should be a function that takes a config object and returns an array of predicates
func (e *JSEngine) ExecutePredicateGenerator(script string, req *models.Request) (interface{}, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)
	console.Enable(vm)
	jsLogger := NewJSLogger("inject:predicateGenerator")

	// Create sorted query object for deterministic JSON.stringify() output
	sortedQuery := createSortedQueryObject(vm, req.Query)

	// Set up the request object
	reqObj := vm.NewObject()
	reqObj.Set("method", req.Method)
	reqObj.Set("path", req.Path)
	reqObj.Set("query", sortedQuery)
	reqObj.Set("headers", req.Headers)
	reqObj.Set("body", req.Body)
	reqObj.Set("requestFrom", req.RequestFrom)

	vm.Set("request", reqObj)
	vm.Set("logger", jsLogger.createLoggerObject())

	// Create config object with request fields flattened
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			var config = {
				request: request,
				logger: logger
			};
			// Flatten request fields onto config for backwards compatibility
			Object.keys(request).forEach(function(key) {
				config[key] = request[key];
			});

			// Call the predicate generator function
			return fn(config);
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return nil, formatJSError(err, script, formatRequestInfo(req))
	}

	// Export the result as a Go value
	return result.Export(), nil
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
// For backwards compatibility, the config object has all request fields flattened onto it
// (config.data, etc.) in addition to config.request.data, etc.
func (e *JSEngine) ExecuteTCPPredicate(script string, requestData string) (bool, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	buffer.Enable(vm)

	jsLogger := NewJSLogger("inject:tcp-predicate")

	// Set up logger object
	vm.Set("logger", jsLogger.createLoggerObject())

	// Create combined script that sets up request, config and executes the function
	// This ensures all variables are in the same scope
	vm.Set("requestData", requestData)

	// TCP predicates use same flattening pattern as HTTP
	// Config has request fields flattened onto it (config.data, etc.)
	wrappedScript := fmt.Sprintf(`
		(function() {
			// Set up request with Buffer data
			var request = { data: Buffer.from(requestData, 'utf8') };

			// Create config object with request fields flattened onto it
			var config = {
				request: request,
				logger: logger,
				state: {}
			};
			// Flatten request fields onto config for backwards compatibility
			Object.keys(request).forEach(function(key) {
				config[key] = request[key];
			});

			var fn = %s;
			// Call function with all parameters for maximum compatibility
			// Mountebank calls: fn(config, logger, imposterState)
			return fn(config, logger, {});
		})()
	`, script)

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
// For backwards compatibility, the config object has all request fields flattened onto it
// (config.data, etc.) in addition to config.request.data, etc.
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
	vm.Set("requestData", requestData)

	// TCP response injection uses same flattening pattern as HTTP
	// Config has request fields flattened onto it (config.data, etc.)
	// Old interface is: (request, state, logger, callback)
	wrappedScript := fmt.Sprintf(`
		(function() {
			// Set up request with Buffer data
			var request = { data: Buffer.from(requestData, 'utf8') };

			// Create config object with request fields flattened onto it
			var config = {
				request: request,
				state: state,
				logger: logger,
				callback: function(response) { return response; }
			};
			// Flatten request fields onto config for backwards compatibility
			Object.keys(request).forEach(function(key) {
				config[key] = request[key];
			});

			var fn = %s;
			// Call function with all parameters for maximum compatibility
			// TCP injection calls: fn(config, state, logger, callback)
			return fn(config, state, logger, config.callback);
		})()
	`, script)

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
