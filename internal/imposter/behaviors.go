package imposter

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/dop251/goja"
)

// BehaviorExecutor handles behavior execution
type BehaviorExecutor struct {
	jsEngine *JSEngine
}

// NewBehaviorExecutor creates a new behavior executor
func NewBehaviorExecutor(jsEngine *JSEngine) *BehaviorExecutor {
	return &BehaviorExecutor{
		jsEngine: jsEngine,
	}
}

// Execute applies all behaviors to a response
func (e *BehaviorExecutor) Execute(req *models.Request, resp *models.IsResponse, behaviors []models.Behavior) (*models.IsResponse, error) {
	if len(behaviors) == 0 {
		return resp, nil
	}

	result := resp

	for _, behavior := range behaviors {
		var err error

		// Handle wait behavior
		if behavior.Wait != nil {
			if err = e.executeWait(behavior.Wait); err != nil {
				return nil, fmt.Errorf("wait behavior error: %w", err)
			}
		}

		// Handle copy behavior
		if behavior.Copy != nil {
			result, err = e.executeCopy(req, result, behavior.Copy)
			if err != nil {
				return nil, fmt.Errorf("copy behavior error: %w", err)
			}
		}

		// Handle lookup behavior
		if behavior.Lookup != nil {
			result, err = e.executeLookup(req, result, behavior.Lookup)
			if err != nil {
				return nil, fmt.Errorf("lookup behavior error: %w", err)
			}
		}

		// Handle decorate behavior
		if behavior.Decorate != "" {
			result, err = e.executeDecorate(req, result, behavior.Decorate)
			if err != nil {
				return nil, fmt.Errorf("decorate behavior error: %w", err)
			}
		}

		// Handle shellTransform behavior
		if behavior.ShellTransform != "" {
			result, err = e.executeShellTransform(req, result, behavior.ShellTransform)
			if err != nil {
				return nil, fmt.Errorf("shellTransform behavior error: %w", err)
			}
		}
	}

	return result, nil
}

// executeWait adds latency to the response
func (e *BehaviorExecutor) executeWait(wait interface{}) error {
	var milliseconds int

	switch v := wait.(type) {
	case int:
		milliseconds = v
	case float64:
		milliseconds = int(v)
	case string:
		// Could be a number string or a JavaScript function
		if ms, err := strconv.Atoi(v); err == nil {
			milliseconds = ms
		} else {
			// Try to execute as JavaScript function
			ms, err := e.executeWaitFunction(v)
			if err != nil {
				return err
			}
			milliseconds = ms
		}
	default:
		return fmt.Errorf("invalid wait value type: %T", wait)
	}

	if milliseconds > 0 {
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)
	}

	return nil
}

// executeWaitFunction executes a JavaScript function to get wait time
func (e *BehaviorExecutor) executeWaitFunction(script string) (int, error) {
	vm := goja.New()
	jsLogger := NewJSLogger("behavior:wait")
	vm.Set("logger", jsLogger.createLoggerObject())

	// Wrap and execute the function
	wrappedScript := fmt.Sprintf(`(%s)()`, script)
	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return 0, formatJSError(err, script, "wait behavior")
	}

	// Convert result to int
	switch v := result.Export().(type) {
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("wait function must return a number, got %T", v)
	}
}

// executeCopy copies values from request to response
func (e *BehaviorExecutor) executeCopy(req *models.Request, resp *models.IsResponse, copyConfig *models.Copy) (*models.IsResponse, error) {
	// Get the source value from request
	fromValue := e.getFromValue(req, copyConfig.From)
	if fromValue == "" {
		return resp, nil
	}

	// Extract value using the specified method
	var values []string
	if copyConfig.Using != nil {
		var err error
		values, err = e.extractValues(fromValue, copyConfig.Using)
		if err != nil {
			return nil, err
		}
	} else {
		values = []string{fromValue}
	}

	// Replace tokens in response
	result := e.replaceTokens(resp, copyConfig.Into, values)
	return result, nil
}

// getFromValue extracts a value from the request based on the "from" configuration
func (e *BehaviorExecutor) getFromValue(req *models.Request, from interface{}) string {
	switch v := from.(type) {
	case string:
		return e.getRequestField(req, v)
	case map[string]interface{}:
		// Nested field access like {"query": "id"}
		for field, subfield := range v {
			// If it's a map field like query or headers
			if subfieldStr, ok := subfield.(string); ok {
				switch strings.ToLower(field) {
				case "query":
					if req.Query != nil {
						return req.Query[subfieldStr]
					}
				case "headers":
					if req.Headers != nil {
						// Case-insensitive header lookup
						for k, val := range req.Headers {
							if strings.EqualFold(k, subfieldStr) {
								return val
							}
						}
					}
				}
			}
		}
	}
	return ""
}

// getRequestField gets a field value from the request
func (e *BehaviorExecutor) getRequestField(req *models.Request, field string) string {
	switch strings.ToLower(field) {
	case "method":
		return req.Method
	case "path":
		return req.Path
	case "body":
		return req.Body
	default:
		return ""
	}
}

// extractValues extracts values using the specified method
func (e *BehaviorExecutor) extractValues(source string, using *models.Using) ([]string, error) {
	switch using.Method {
	case "regex":
		return e.extractRegex(source, using.Selector, using.Options)
	case "xpath":
		// XPath extraction - not implemented yet
		return []string{source}, nil
	case "jsonpath":
		return e.extractJSONPath(source, using.Selector)
	default:
		return []string{source}, nil
	}
}

// extractRegex extracts values using regex
func (e *BehaviorExecutor) extractRegex(source, pattern string, options *models.UsingOptions) ([]string, error) {
	flags := ""
	if options != nil {
		if options.IgnoreCase {
			flags += "(?i)"
		}
		if options.Multiline {
			flags += "(?m)"
		}
	}

	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matches := re.FindStringSubmatch(source)
	if len(matches) == 0 {
		return []string{}, nil
	}

	// Return all capture groups (or the whole match if no groups)
	if len(matches) > 1 {
		return matches[1:], nil
	}
	return matches, nil
}

// extractJSONPath extracts values using JSONPath
func (e *BehaviorExecutor) extractJSONPath(source, selector string) ([]string, error) {
	// Simple JSONPath implementation for common cases
	// Full JSONPath would require a library

	var data interface{}
	if err := json.Unmarshal([]byte(source), &data); err != nil {
		return []string{}, nil
	}

	// Handle simple paths like $.field or $..field
	selector = strings.TrimPrefix(selector, "$")
	selector = strings.TrimPrefix(selector, ".")

	result := e.navigateJSON(data, selector)
	if result != "" {
		return []string{result}, nil
	}

	return []string{}, nil
}

// navigateJSON navigates JSON structure with a simple path
func (e *BehaviorExecutor) navigateJSON(data interface{}, path string) string {
	if path == "" {
		return e.jsonToString(data)
	}

	parts := strings.SplitN(path, ".", 2)
	key := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	// Handle array index
	if idx := strings.Index(key, "["); idx != -1 {
		arrayKey := key[:idx]
		indexStr := strings.Trim(key[idx:], "[]")
		index, _ := strconv.Atoi(indexStr)

		if m, ok := data.(map[string]interface{}); ok {
			if arr, ok := m[arrayKey].([]interface{}); ok && index < len(arr) {
				return e.navigateJSON(arr[index], rest)
			}
		}
		return ""
	}

	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			return e.navigateJSON(val, rest)
		}
	}

	return ""
}

// jsonToString converts a JSON value to string
func (e *BehaviorExecutor) jsonToString(data interface{}) string {
	switch v := data.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// replaceTokens replaces tokens in the response
func (e *BehaviorExecutor) replaceTokens(resp *models.IsResponse, token string, values []string) *models.IsResponse {
	result := &models.IsResponse{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]interface{}),
		Mode:       resp.Mode,
	}

	// Copy headers
	for k, v := range resp.Headers {
		if str, ok := v.(string); ok {
			result.Headers[k] = e.replaceInString(str, token, values)
		} else {
			result.Headers[k] = v
		}
	}

	// Replace in body
	if resp.Body != nil {
		switch body := resp.Body.(type) {
		case string:
			result.Body = e.replaceInString(body, token, values)
		default:
			// For non-string bodies, try to marshal and replace
			if b, err := json.Marshal(body); err == nil {
				replaced := e.replaceInString(string(b), token, values)
				var parsed interface{}
				if json.Unmarshal([]byte(replaced), &parsed) == nil {
					result.Body = parsed
				} else {
					result.Body = replaced
				}
			} else {
				result.Body = body
			}
		}
	}

	return result
}

// replaceInString replaces tokens in a string
func (e *BehaviorExecutor) replaceInString(s, token string, values []string) string {
	result := s

	// Replace indexed tokens like ${TOKEN}[0], ${TOKEN}[1], etc.
	for i, value := range values {
		indexedToken := fmt.Sprintf("%s[%d]", token, i)
		result = strings.ReplaceAll(result, indexedToken, value)
	}

	// Replace the base token with the first value
	if len(values) > 0 {
		result = strings.ReplaceAll(result, token, values[0])
	}

	return result
}

// executeLookup looks up values from a data source
func (e *BehaviorExecutor) executeLookup(req *models.Request, resp *models.IsResponse, lookup *models.Lookup) (*models.IsResponse, error) {
	if lookup.FromDataSource == nil || lookup.FromDataSource.CSV == nil {
		return resp, nil
	}

	// Get the key value from request
	keyValue := ""
	if lookup.Key != nil {
		if keyMap, ok := lookup.Key.(map[string]interface{}); ok {
			if from, ok := keyMap["from"]; ok {
				keyValue = e.getFromValue(req, from)
			}
			if using, ok := keyMap["using"].(map[string]interface{}); ok {
				useConfig := e.parseUsing(using)
				if values, err := e.extractValues(keyValue, useConfig); err == nil && len(values) > 0 {
					keyValue = values[0]
				}
			}
		}
	}

	if keyValue == "" {
		return resp, nil
	}

	// Load CSV and find matching row
	row, err := e.lookupCSV(lookup.FromDataSource.CSV, keyValue)
	if err != nil {
		return resp, nil // Silently fail on lookup errors
	}

	// Replace tokens with row values
	result := e.replaceRowTokens(resp, lookup.Into, row)
	return result, nil
}

// parseUsing parses a using configuration from a map
func (e *BehaviorExecutor) parseUsing(m map[string]interface{}) *models.Using {
	using := &models.Using{}
	if method, ok := m["method"].(string); ok {
		using.Method = method
	}
	if selector, ok := m["selector"].(string); ok {
		using.Selector = selector
	}
	return using
}

// lookupCSV looks up a row in a CSV file
func (e *BehaviorExecutor) lookupCSV(csvConfig *models.CSVSource, keyValue string) (map[string]string, error) {
	file, err := os.Open(csvConfig.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	delimiter := ','
	if csvConfig.Delimiter != "" {
		delimiter = rune(csvConfig.Delimiter[0])
	}

	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = delimiter

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	// Find key column index
	keyIndex := -1
	for i, h := range headers {
		if h == csvConfig.KeyColumn {
			keyIndex = i
			break
		}
	}
	if keyIndex == -1 {
		return nil, fmt.Errorf("key column %s not found", csvConfig.KeyColumn)
	}

	// Find matching row
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if len(record) > keyIndex && record[keyIndex] == keyValue {
			// Build result map
			result := make(map[string]string)
			for i, h := range headers {
				if i < len(record) {
					result[h] = record[i]
				}
			}
			return result, nil
		}
	}

	return map[string]string{}, nil
}

// replaceRowTokens replaces tokens with row values
func (e *BehaviorExecutor) replaceRowTokens(resp *models.IsResponse, token string, row map[string]string) *models.IsResponse {
	result := &models.IsResponse{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]interface{}),
		Mode:       resp.Mode,
	}

	replacer := func(s string) string {
		for key, value := range row {
			// Replace ${TOKEN}["key"], ${TOKEN}['key'], and ${TOKEN}[key]
			s = strings.ReplaceAll(s, fmt.Sprintf(`%s["%s"]`, token, key), value)
			s = strings.ReplaceAll(s, fmt.Sprintf(`%s['%s']`, token, key), value)
			s = strings.ReplaceAll(s, fmt.Sprintf(`%s[%s]`, token, key), value)
		}
		return s
	}

	// Replace in headers
	for k, v := range resp.Headers {
		if str, ok := v.(string); ok {
			result.Headers[k] = replacer(str)
		} else {
			result.Headers[k] = v
		}
	}

	// Replace in body
	if resp.Body != nil {
		switch body := resp.Body.(type) {
		case string:
			result.Body = replacer(body)
		default:
			if b, err := json.Marshal(body); err == nil {
				replaced := replacer(string(b))
				var parsed interface{}
				if json.Unmarshal([]byte(replaced), &parsed) == nil {
					result.Body = parsed
				} else {
					result.Body = replaced
				}
			} else {
				result.Body = body
			}
		}
	}

	return result
}

// executeDecorate runs JavaScript to post-process the response
func (e *BehaviorExecutor) executeDecorate(req *models.Request, resp *models.IsResponse, script string) (*models.IsResponse, error) {
	vm := goja.New()
	jsLogger := NewJSLogger("behavior:decorate")

	// Ensure headers is not nil
	respHeaders := resp.Headers
	if respHeaders == nil {
		respHeaders = make(map[string]interface{})
	}

	// Create request object
	requestObj := map[string]interface{}{
		"method":  req.Method,
		"path":    req.Path,
		"query":   req.Query,
		"headers": req.Headers,
		"body":    req.Body,
	}

	// Create response object (mutable copy)
	responseObj := map[string]interface{}{
		"statusCode": resp.StatusCode,
		"headers":    copyHeadersInterface(respHeaders),
		"body":       resp.Body,
	}

	loggerObj := jsLogger.createLoggerObject()

	// Create config object (new interface)
	config := map[string]interface{}{
		"request":  requestObj,
		"response": responseObj,
		"logger":   loggerObj,
		"state":    map[string]interface{}{},
	}

	vm.Set("config", config)

	// Also set individual variables for old interface compatibility
	vm.Set("request", requestObj)
	vm.Set("response", responseObj)
	vm.Set("logger", loggerObj)

	// Execute the decorator
	// Support both old interface (request, response) and new interface (config)
	// Detect interface by checking function arity (fn.length)
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			var result;
			if (fn.length >= 2) {
				// Old interface: function(request, response, logger)
				result = fn(request, response, logger);
			} else {
				// New interface: function(config)
				result = fn(config);
			}
			return result || response;
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		return nil, formatJSError(err, script, formatRequestInfo(req))
	}

	// Convert result back to IsResponse
	return e.convertDecorateResult(result, resp)
}

// copyHeaders creates a mutable copy of headers
func copyHeaders(src map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range src {
		result[k] = v
	}
	return result
}

// copyHeadersInterface creates a mutable copy of headers from interface map
func copyHeadersInterface(src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range src {
		result[k] = v
	}
	return result
}

// convertDecorateResult converts the decorator result to IsResponse
func (e *BehaviorExecutor) convertDecorateResult(val goja.Value, original *models.IsResponse) (*models.IsResponse, error) {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return original, nil
	}

	exported := val.Export()
	respMap, ok := exported.(map[string]interface{})
	if !ok {
		return original, nil
	}

	result := &models.IsResponse{
		StatusCode: original.StatusCode,
		Headers:    make(map[string]interface{}),
		Mode:       original.Mode,
	}

	// Extract statusCode
	if sc, ok := respMap["statusCode"]; ok {
		switch v := sc.(type) {
		case int64:
			result.StatusCode = int(v)
		case float64:
			result.StatusCode = int(v)
		case int:
			result.StatusCode = v
		}
	}

	// Extract headers
	if h, ok := respMap["headers"]; ok {
		if headersMap, ok := h.(map[string]interface{}); ok {
			for k, v := range headersMap {
				result.Headers[k] = v
			}
		}
	}

	// Extract body
	if b, ok := respMap["body"]; ok {
		result.Body = b
	}

	return result, nil
}

// executeShellTransform transforms response using an external command
func (e *BehaviorExecutor) executeShellTransform(req *models.Request, resp *models.IsResponse, command string) (*models.IsResponse, error) {
	// Prepare request and response as JSON
	reqJSON, _ := json.Marshal(map[string]interface{}{
		"method":  req.Method,
		"path":    req.Path,
		"query":   req.Query,
		"headers": req.Headers,
		"body":    req.Body,
	})

	respJSON, _ := json.Marshal(map[string]interface{}{
		"statusCode": resp.StatusCode,
		"headers":    resp.Headers,
		"body":       resp.Body,
	})

	// Set environment variables
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = append(os.Environ(),
		"MB_REQUEST="+string(reqJSON),
		"MB_RESPONSE="+string(respJSON),
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("shell command failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse output as JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("shell command returned invalid JSON: %s", stdout.String())
	}

	// Convert to IsResponse
	newResp := &models.IsResponse{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]interface{}),
	}

	if sc, ok := result["statusCode"]; ok {
		switch v := sc.(type) {
		case float64:
			newResp.StatusCode = int(v)
		case int:
			newResp.StatusCode = v
		}
	}

	if h, ok := result["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			newResp.Headers[k] = v
		}
	}

	if b, ok := result["body"]; ok {
		newResp.Body = b
	}

	return newResp, nil
}
