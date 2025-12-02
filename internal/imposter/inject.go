package imposter

import (
	"encoding/json"
	"fmt"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/dop251/goja"
)

// JSEngine handles JavaScript injection execution
type JSEngine struct{}

// NewJSEngine creates a new JavaScript engine
func NewJSEngine() *JSEngine {
	return &JSEngine{}
}

// ExecuteResponse executes an inject script and returns the response
func (e *JSEngine) ExecuteResponse(script string, req *models.Request) (*models.IsResponse, error) {
	vm := goja.New()

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

	// Set up logger (mock for compatibility)
	vm.Set("logger", map[string]interface{}{
		"debug": func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"info":  func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"warn":  func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"error": func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
	})

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
		return nil, fmt.Errorf("inject script error: %w", err)
	}

	// Convert result to IsResponse
	return e.convertToResponse(result)
}

// ExecutePredicate executes an inject predicate script
func (e *JSEngine) ExecutePredicate(script string, req *models.Request) (bool, error) {
	vm := goja.New()

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

	// Set up logger
	vm.Set("logger", map[string]interface{}{
		"debug": func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"info":  func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"warn":  func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"error": func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
	})

	// Wrap the script in a function call
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			return fn(request, logger);
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return false, fmt.Errorf("inject predicate error: %w", err)
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

	// Set up the request data
	vm.Set("requestData", requestData)

	// Set up logger (mock for compatibility)
	vm.Set("logger", map[string]interface{}{
		"debug": func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"info":  func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"warn":  func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
		"error": func(call goja.FunctionCall) goja.Value { return goja.Undefined() },
	})

	// Wrap the script in a function call
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			return fn(requestData, logger);
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return false, fmt.Errorf("endOfRequestResolver error: %w", err)
	}

	return result.ToBoolean(), nil
}
