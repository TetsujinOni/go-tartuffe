package imposter

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// MatchResult contains the result of matching a request against stubs
type MatchResult struct {
	// The response to return (for "is" responses)
	Response *models.IsResponse

	// Proxy configuration (for "proxy" responses)
	Proxy *models.ProxyResponse

	// Inject script (for "inject" responses)
	Inject string

	// Fault type (for "fault" responses)
	Fault string

	// Behaviors to apply to the response
	Behaviors []models.Behavior

	// The matched stub (for recording purposes)
	Stub *models.Stub

	// Index of the matched stub
	StubIndex int
}

// Matcher handles request matching against stubs
type Matcher struct {
	imposter      *models.Imposter
	jsEngine      *JSEngine              // Shared JS engine
	imposterState map[string]interface{} // Shared state across all requests
}

// NewMatcher creates a new matcher for an imposter
func NewMatcher(imp *models.Imposter) *Matcher {
	return &Matcher{
		imposter:      imp,
		jsEngine:      NewJSEngine(),
		imposterState: make(map[string]interface{}),
	}
}

// SetState sets the imposter state reference (for sharing with Server)
func (m *Matcher) SetState(state map[string]interface{}) {
	m.imposterState = state
}

// GetState returns the imposter state reference
func (m *Matcher) GetState() map[string]interface{} {
	return m.imposterState
}

// GetJSEngine returns the JS engine
func (m *Matcher) GetJSEngine() *JSEngine {
	return m.jsEngine
}

// GetResponse finds a matching stub and returns the response
func (m *Matcher) GetResponse(req *models.Request) *models.IsResponse {
	result := m.Match(req)
	return result.Response
}

// Match finds a matching stub and returns a comprehensive result
func (m *Matcher) Match(req *models.Request) *MatchResult {
	for i := range m.imposter.Stubs {
		stub := &m.imposter.Stubs[i]
		if m.matchesAllPredicates(stub, req) {
			return m.getMatchResult(stub, i)
		}
	}

	// No match - return default response or empty 200
	// Use StubIndex of -1 to signal no match
	if m.imposter.DefaultResponse != nil {
		if m.imposter.DefaultResponse.Is != nil {
			return &MatchResult{Response: m.imposter.DefaultResponse.Is, StubIndex: -1}
		}
		if m.imposter.DefaultResponse.Proxy != nil {
			return &MatchResult{Proxy: m.imposter.DefaultResponse.Proxy, StubIndex: -1}
		}
		if m.imposter.DefaultResponse.Inject != "" {
			return &MatchResult{Inject: m.imposter.DefaultResponse.Inject, StubIndex: -1}
		}
	}

	return &MatchResult{Response: &models.IsResponse{StatusCode: 200}, StubIndex: -1}
}

// getMatchResult creates a MatchResult from a stub
func (m *Matcher) getMatchResult(stub *models.Stub, index int) *MatchResult {
	if len(stub.Responses) == 0 {
		return &MatchResult{
			Response:  &models.IsResponse{StatusCode: 200},
			Stub:      stub,
			StubIndex: index,
		}
	}

	resp := stub.NextResponse()
	if resp == nil {
		return &MatchResult{
			Response:  &models.IsResponse{StatusCode: 200},
			Stub:      stub,
			StubIndex: index,
		}
	}

	result := &MatchResult{
		Stub:      stub,
		StubIndex: index,
		Behaviors: resp.Behaviors,
	}

	if resp.Is != nil {
		result.Response = m.normalizeResponse(resp.Is)
	} else if resp.Proxy != nil {
		result.Proxy = resp.Proxy
	} else if resp.Inject != "" {
		result.Inject = resp.Inject
	} else if resp.Fault != "" {
		result.Fault = resp.Fault
	} else {
		result.Response = &models.IsResponse{StatusCode: 200}
	}

	return result
}

// normalizeResponse normalizes a response body, converting objects to JSON strings
// This matches mountebank's behavior of converting object bodies to JSON before processing
func (m *Matcher) normalizeResponse(resp *models.IsResponse) *models.IsResponse {
	// Make a copy to avoid modifying the original
	normalized := *resp

	// If body is an object (not a string), convert it to JSON
	if normalized.Body != nil {
		switch normalized.Body.(type) {
		case string, []byte:
			// Already a string or bytes, no conversion needed
		default:
			// It's an object - convert to JSON string (pretty-printed to match mountebank)
			if jsonBytes, err := models.MarshalBody(normalized.Body); err == nil {
				normalized.Body = string(jsonBytes)
			}
		}
	}

	return &normalized
}

// matchesAllPredicates checks if a request matches all predicates in a stub
func (m *Matcher) matchesAllPredicates(stub *models.Stub, req *models.Request) bool {
	// Empty predicates array matches everything
	if len(stub.Predicates) == 0 {
		return true
	}

	for _, pred := range stub.Predicates {
		if !m.evaluatePredicate(&pred, req) {
			return false
		}
	}

	return true
}

// predicateOptions holds the options that affect predicate evaluation
type predicateOptions struct {
	caseSensitive    bool
	keyCaseSensitive bool
	except           string
}

// evaluatePredicate evaluates a single predicate against a request
func (m *Matcher) evaluatePredicate(pred *models.Predicate, req *models.Request) bool {
	// Handle logical operators
	if pred.And != nil {
		for _, p := range pred.And {
			if !m.evaluatePredicate(&p, req) {
				return false
			}
		}
		return true
	}

	if pred.Or != nil {
		for _, p := range pred.Or {
			if m.evaluatePredicate(&p, req) {
				return true
			}
		}
		return false
	}

	if pred.Not != nil {
		return !m.evaluatePredicate(pred.Not, req)
	}

	// Build predicate options
	// If caseSensitive is true, it affects both values and keys
	keyCaseSensitive := pred.KeyCaseSensitive
	if pred.CaseSensitive {
		keyCaseSensitive = true
	}
	opts := predicateOptions{
		caseSensitive:    pred.CaseSensitive,
		keyCaseSensitive: keyCaseSensitive,
		except:           pred.Except,
	}

	// Apply selector to extract value from body if specified
	effectiveReq := req
	if pred.JSONPath != nil || pred.XPath != nil {
		extracted := m.applySelector(req, pred, keyCaseSensitive)
		if extracted == nil {
			// Selector extraction failed (e.g., invalid JSON for JSONPath)
			// Predicate does not match
			return false
		}
		effectiveReq = extracted
	}

	// Handle comparison operators
	if pred.Equals != nil {
		return m.evaluateEquals(pred.Equals, effectiveReq, opts)
	}

	if pred.DeepEquals != nil {
		return m.evaluateDeepEquals(pred.DeepEquals, effectiveReq, opts)
	}

	if pred.Contains != nil {
		return m.evaluateContains(pred.Contains, effectiveReq, opts)
	}

	if pred.StartsWith != nil {
		return m.evaluateStartsWith(pred.StartsWith, effectiveReq, opts)
	}

	if pred.EndsWith != nil {
		return m.evaluateEndsWith(pred.EndsWith, effectiveReq, opts)
	}

	if pred.Matches != nil {
		return m.evaluateMatches(pred.Matches, effectiveReq, opts)
	}

	if pred.Exists != nil {
		return m.evaluateExists(pred.Exists, effectiveReq, opts)
	}

	if pred.Inject != "" {
		return m.evaluateInject(pred.Inject, req)
	}

	// Default: no predicate matches
	return true
}

// applySelector applies JSONPath or XPath selector to extract value from body
func (m *Matcher) applySelector(req *models.Request, pred *models.Predicate, keyCaseSensitive bool) *models.Request {
	evaluator := NewSelectorEvaluator()

	var extractedValue string
	var err error

	if pred.JSONPath != nil {
		extractedValue, err = evaluator.ApplySelectorWithOptions(req.Body, pred.JSONPath, "jsonpath", keyCaseSensitive)
	} else if pred.XPath != nil {
		extractedValue, err = evaluator.ApplySelector(req.Body, pred.XPath, "xpath")
	}

	if err != nil {
		// If extraction fails (e.g., invalid JSON for JSONPath), return nil
		// to signal that the predicate should not match
		return nil
	}

	// Create a modified request with extracted value as body
	modifiedReq := *req
	modifiedReq.Body = extractedValue
	return &modifiedReq
}

// evaluateInject executes a JavaScript predicate
// Uses shared JS engine and imposter state for state persistence
func (m *Matcher) evaluateInject(script string, req *models.Request) bool {
	result, err := m.jsEngine.ExecutePredicate(script, req, m.imposterState)
	if err != nil {
		log.Printf("[ERROR] inject predicate failed: %v", err)
		return false
	}
	return result
}

// applyExcept strips matching pattern from a string value
func (m *Matcher) applyExcept(value string, except string, caseSensitive bool) string {
	if except == "" {
		return value
	}

	// Build regex with appropriate flags
	pattern := except
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return value
	}

	return re.ReplaceAllString(value, "")
}

// evaluateEquals checks if request fields equal the predicate values
func (m *Matcher) evaluateEquals(value interface{}, req *models.Request, opts predicateOptions) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual := m.getRequestField(req, field, opts.keyCaseSensitive)
		if !m.compareValues(actual, expected, opts) {
			return false
		}
	}

	return true
}

// evaluateDeepEquals checks deep equality (for nested objects)
// Unlike equals, deepEquals requires EXACT match - no extra fields allowed
func (m *Matcher) evaluateDeepEquals(value interface{}, req *models.Request, opts predicateOptions) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual := m.getRequestField(req, field, opts.keyCaseSensitive)
		if !m.deepCompareValues(actual, expected, opts) {
			return false
		}
	}

	return true
}

// forceToString recursively converts values to strings, matching mountebank's forceStrings behavior
// This is used in deepEquals to ensure type-insensitive comparison (e.g., 1 matches "1")
func (m *Matcher) forceToString(value interface{}) interface{} {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}:
		// Recursively convert map values to strings
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = m.forceToString(val)
		}
		return result
	case map[string]string:
		// Already strings, return as-is
		return v
	case []interface{}:
		// Recursively convert array elements to strings
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = m.forceToString(val)
		}
		return result
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		// For any other type, use fmt.Sprintf
		return fmt.Sprintf("%v", v)
	}
}

// deepCompareValues compares two values for deep equality
// This is stricter than compareValues - it requires exact match including no extra fields
func (m *Matcher) deepCompareValues(actual, expected interface{}, opts predicateOptions) bool {
	// Handle nil/null values
	if expected == nil {
		return actual == nil || actual == ""
	}

	// Mountebank forces all values to strings for deepEquals comparison
	// This allows { query: { equals: 1 } } to match ?equals=1
	actualForced := m.forceToString(actual)
	expectedForced := m.forceToString(expected)

	// Handle string comparison
	actualStr, actualIsStr := toString(actualForced)
	expectedStr, expectedIsStr := toString(expectedForced)

	if actualIsStr && expectedIsStr {
		actualStr = m.applyExcept(actualStr, opts.except, opts.caseSensitive)
		if opts.caseSensitive {
			return actualStr == expectedStr
		}
		return strings.EqualFold(actualStr, expectedStr)
	}

	// Handle body as JSON - try to parse both and compare deeply
	if actualIsStr && !expectedIsStr {
		// actual is string (body), expected is a map/array - try to parse actual as JSON
		var actualParsed interface{}
		if err := json.Unmarshal([]byte(actualStr), &actualParsed); err == nil {
			return m.deepEqualJSON(actualParsed, expected, opts)
		}
	}

	// Handle maps (like query or headers) with strict equality
	// Use the forced versions which have all values converted to strings
	if actualMap, ok := actualForced.(map[string]string); ok {
		if expectedMap, ok := expectedForced.(map[string]interface{}); ok {
			// For deepEquals, the maps must have the same keys
			if len(actualMap) != len(expectedMap) {
				return false
			}
			for k, v := range expectedMap {
				actualVal, exists := actualMap[k]
				if !exists && !opts.keyCaseSensitive {
					for ak, av := range actualMap {
						if strings.EqualFold(ak, k) {
							actualVal = av
							exists = true
							break
						}
					}
				}
				if !exists {
					return false
				}
				actualVal = m.applyExcept(actualVal, opts.except, opts.caseSensitive)
				expectedStr, _ := toString(v)
				if opts.caseSensitive {
					if actualVal != expectedStr {
						return false
					}
				} else {
					if !strings.EqualFold(actualVal, expectedStr) {
						return false
					}
				}
			}
			return true
		}
		// Expected is empty map/nil - actual must also be empty
		if expectedMap, ok := expectedForced.(map[string]interface{}); ok && len(expectedMap) == 0 {
			return len(actualMap) == 0
		}
	}

	// Handle nil expected with map actual
	if expectedForced == nil {
		if actualMap, ok := actualForced.(map[string]string); ok {
			return len(actualMap) == 0
		}
	}

	return reflect.DeepEqual(actualForced, expectedForced)
}

// deepEqualJSON compares two JSON values deeply with strict equality
func (m *Matcher) deepEqualJSON(actual, expected interface{}, opts predicateOptions) bool {
	// Handle nil
	if expected == nil {
		return actual == nil
	}
	if actual == nil {
		return expected == nil
	}

	// Handle maps
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		// For deepEquals, must have same number of keys
		if len(actualMap) != len(expectedMap) {
			return false
		}
		for k, ev := range expectedMap {
			av, exists := actualMap[k]
			if !exists {
				// Try case-insensitive if needed
				if !opts.keyCaseSensitive {
					for ak, akv := range actualMap {
						if strings.EqualFold(ak, k) {
							av = akv
							exists = true
							break
						}
					}
				}
			}
			if !exists {
				return false
			}
			if !m.deepEqualJSON(av, ev, opts) {
				return false
			}
		}
		return true
	}

	// Handle arrays - mountebank allows order-insensitive matching
	if expectedArr, ok := expected.([]interface{}); ok {
		actualArr, ok := actual.([]interface{})
		if !ok {
			return false
		}
		if len(actualArr) != len(expectedArr) {
			return false
		}

		// Check if every expected element exists in actual array (order-insensitive)
		// Track which actual elements have been matched
		matched := make([]bool, len(actualArr))
		for _, ev := range expectedArr {
			found := false
			for i, av := range actualArr {
				if !matched[i] && m.deepEqualJSON(av, ev, opts) {
					matched[i] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	// Handle strings with case sensitivity
	if expectedStr, ok := expected.(string); ok {
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		if opts.caseSensitive {
			return actualStr == expectedStr
		}
		return strings.EqualFold(actualStr, expectedStr)
	}

	// For other primitives (numbers, bools), use reflect.DeepEqual
	return reflect.DeepEqual(actual, expected)
}

// evaluateContains checks if request fields contain the predicate values
func (m *Matcher) evaluateContains(value interface{}, req *models.Request, opts predicateOptions) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual := m.getRequestField(req, field, opts.keyCaseSensitive)
		if !m.containsValue(actual, expected, opts) {
			return false
		}
	}

	return true
}

// evaluateStartsWith checks if request fields start with the predicate values
func (m *Matcher) evaluateStartsWith(value interface{}, req *models.Request, opts predicateOptions) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual := m.getRequestField(req, field, opts.keyCaseSensitive)
		if !m.startsWithValue(actual, expected, opts) {
			return false
		}
	}

	return true
}

// evaluateEndsWith checks if request fields end with the predicate values
func (m *Matcher) evaluateEndsWith(value interface{}, req *models.Request, opts predicateOptions) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual := m.getRequestField(req, field, opts.keyCaseSensitive)
		if !m.endsWithValue(actual, expected, opts) {
			return false
		}
	}

	return true
}

// evaluateMatches checks if request fields match the regex patterns
func (m *Matcher) evaluateMatches(value interface{}, req *models.Request, opts predicateOptions) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, pattern := range predMap {
		actual := m.getRequestField(req, field, opts.keyCaseSensitive)
		if !m.matchesPattern(actual, pattern, opts) {
			return false
		}
	}

	return true
}

// evaluateExists checks if request fields exist
func (m *Matcher) evaluateExists(value interface{}, req *models.Request, opts predicateOptions) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, shouldExist := range predMap {
		// Handle nested maps like {"headers": {"X-One": true, "X-Three": false}}
		if nestedMap, ok := shouldExist.(map[string]interface{}); ok {
			// Special handling for body JSON key existence checks
			if strings.ToLower(field) == "body" {
				// Parse body as JSON and check if keys exist
				bodyStr := fmt.Sprintf("%v", req.Body)
				var bodyParsed map[string]interface{}
				if err := json.Unmarshal([]byte(bodyStr), &bodyParsed); err == nil {
					for jsonKey, jsonShouldExist := range nestedMap {
						expected, _ := jsonShouldExist.(bool)
						_, exists := bodyParsed[jsonKey]

						// Try case-insensitive if needed
						if !exists && !opts.keyCaseSensitive {
							for k := range bodyParsed {
								if strings.EqualFold(k, jsonKey) {
									exists = true
									break
								}
							}
						}

						if exists != expected {
							return false
						}
					}
					continue
				}
			}

			// Regular nested field handling (headers, query, etc.)
			for nestedKey, nestedShouldExist := range nestedMap {
				fullField := field + "." + nestedKey
				exists := m.fieldExists(req, fullField, opts.keyCaseSensitive)
				expected, _ := nestedShouldExist.(bool)
				if exists != expected {
					return false
				}
			}
		} else {
			// Direct field check like {"path": true}
			exists := m.fieldExists(req, field, opts.keyCaseSensitive)
			expected, _ := shouldExist.(bool)
			if exists != expected {
				return false
			}
		}
	}

	return true
}

// getRequestField retrieves a field value from the request
func (m *Matcher) getRequestField(req *models.Request, field string, keyCaseSensitive bool) interface{} {
	switch strings.ToLower(field) {
	case "method":
		return req.Method
	case "path":
		return req.Path
	case "body":
		return req.Body
	case "query":
		return req.Query
	case "headers":
		return req.Headers
	case "form":
		return req.Form
	default:
		// Check if it's a nested field like "headers.Content-Type"
		parts := strings.SplitN(field, ".", 2)
		if len(parts) == 2 {
			parent := strings.ToLower(parts[0])
			key := parts[1]
			switch parent {
			case "query":
				if req.Query != nil {
					if keyCaseSensitive {
						// Exact key match
						return req.Query[key]
					}
					// Case-insensitive key match
					for k, v := range req.Query {
						if strings.EqualFold(k, key) {
							return v
						}
					}
				}
			case "headers":
				if req.Headers != nil {
					if keyCaseSensitive {
						// Exact key match
						return req.Headers[key]
					}
					// Case-insensitive key match (default for headers)
					for k, v := range req.Headers {
						if strings.EqualFold(k, key) {
							return v
						}
					}
				}
			case "form":
				if req.Form != nil {
					if keyCaseSensitive {
						// Exact key match
						return req.Form[key]
					}
					// Case-insensitive key match
					for k, v := range req.Form {
						if strings.EqualFold(k, key) {
							return v
						}
					}
				}
			}
		}
		return nil
	}
}

// fieldExists checks if a field exists in the request
func (m *Matcher) fieldExists(req *models.Request, field string, keyCaseSensitive bool) bool {
	parts := strings.SplitN(field, ".", 2)
	if len(parts) == 2 {
		parent := strings.ToLower(parts[0])
		key := parts[1]
		switch parent {
		case "query":
			if req.Query != nil {
				if keyCaseSensitive {
					_, exists := req.Query[key]
					return exists
				}
				// Case-insensitive key match
				for k := range req.Query {
					if strings.EqualFold(k, key) {
						return true
					}
				}
			}
		case "headers":
			if req.Headers != nil {
				if keyCaseSensitive {
					_, exists := req.Headers[key]
					return exists
				}
				// Case-insensitive key match
				for k := range req.Headers {
					if strings.EqualFold(k, key) {
						return true
					}
				}
			}
		case "form":
			if req.Form != nil {
				if keyCaseSensitive {
					_, exists := req.Form[key]
					return exists
				}
				// Case-insensitive key match
				for k := range req.Form {
					if strings.EqualFold(k, key) {
						return true
					}
				}
			}
		}
		return false
	}

	switch strings.ToLower(field) {
	case "method":
		return req.Method != ""
	case "path":
		return req.Path != ""
	case "body":
		return req.Body != ""
	case "query":
		return len(req.Query) > 0
	case "headers":
		return len(req.Headers) > 0
	case "form":
		return len(req.Form) > 0
	default:
		return false
	}
}

// compareValues compares two values for equality
func (m *Matcher) compareValues(actual, expected interface{}, opts predicateOptions) bool {
	// Handle nil/null values
	if expected == nil {
		return actual == nil || actual == ""
	}

	actualStr, actualIsStr := toString(actual)
	expectedStr, expectedIsStr := toString(expected)

	if actualIsStr && expectedIsStr {
		// Apply except pattern to strip matching portions
		actualStr = m.applyExcept(actualStr, opts.except, opts.caseSensitive)

		if opts.caseSensitive {
			return actualStr == expectedStr
		}
		return strings.EqualFold(actualStr, expectedStr)
	}

	// Handle body as JSON - try to parse and compare
	if actualIsStr && !expectedIsStr {
		// actual is string (body), expected is a map/array - try to parse actual as JSON
		var actualParsed interface{}
		if err := json.Unmarshal([]byte(actualStr), &actualParsed); err == nil {
			return m.jsonContains(actualParsed, expected, opts)
		}
	}

	// For maps (like query or headers)
	if actualMap, ok := actual.(map[string]string); ok {
		// Handle map[string]interface{} expected value
		if expectedMap, ok := expected.(map[string]interface{}); ok {
			for k, v := range expectedMap {
				actualVal, exists := actualMap[k]
				if !exists && !opts.keyCaseSensitive {
					// Try case-insensitive match for keys
					for ak, av := range actualMap {
						if strings.EqualFold(ak, k) {
							actualVal = av
							exists = true
							break
						}
					}
				}
				if !exists {
					return false
				}
				// Apply except pattern
				actualVal = m.applyExcept(actualVal, opts.except, opts.caseSensitive)

				expectedStr, _ := toString(v)
				if opts.caseSensitive {
					if actualVal != expectedStr {
						return false
					}
				} else {
					if !strings.EqualFold(actualVal, expectedStr) {
						return false
					}
				}
			}
			return true
		}

		// Handle map[string]string expected value
		if expectedMap, ok := expected.(map[string]string); ok {
			// Check lengths match for exact equality
			if len(actualMap) != len(expectedMap) {
				return false
			}

			for k, expectedVal := range expectedMap {
				actualVal, exists := actualMap[k]
				if !exists && !opts.keyCaseSensitive {
					// Try case-insensitive match for keys
					for ak, av := range actualMap {
						if strings.EqualFold(ak, k) {
							actualVal = av
							exists = true
							break
						}
					}
				}
				if !exists {
					return false
				}
				// Apply except pattern
				actualVal = m.applyExcept(actualVal, opts.except, opts.caseSensitive)

				if opts.caseSensitive {
					if actualVal != expectedVal {
						return false
					}
				} else {
					if !strings.EqualFold(actualVal, expectedVal) {
						return false
					}
				}
			}
			return true
		}
	}

	return actual == expected
}

// jsonContains checks if actual JSON contains expected values (for equals predicate)
// Unlike deepEquals, this allows actual to have extra fields
func (m *Matcher) jsonContains(actual, expected interface{}, opts predicateOptions) bool {
	// Handle nil
	if expected == nil {
		return actual == nil
	}

	// Handle maps - actual must contain all expected keys with matching values
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		// Check if actual is an array - if so, check if ANY element matches
		if actualArr, ok := actual.([]interface{}); ok {
			for _, elem := range actualArr {
				if m.jsonContains(elem, expectedMap, opts) {
					return true
				}
			}
			return false
		}

		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		for k, ev := range expectedMap {
			av, exists := actualMap[k]
			if !exists {
				// Try case-insensitive if needed
				if !opts.keyCaseSensitive {
					for ak, akv := range actualMap {
						if strings.EqualFold(ak, k) {
							av = akv
							exists = true
							break
						}
					}
				}
			}
			if !exists {
				return false
			}
			if !m.jsonContains(av, ev, opts) {
				return false
			}
		}
		return true
	}

	// Handle arrays - for equals predicate, compare order-insensitively (like a set)
	if expectedArr, ok := expected.([]interface{}); ok {
		actualArr, ok := actual.([]interface{})
		if !ok {
			return false
		}
		if len(actualArr) != len(expectedArr) {
			return false
		}
		// For equals predicate with arrays, check if all expected elements exist in actual array
		// This enables order-insensitive matching (needed for XPath array predicates)
		for _, ev := range expectedArr {
			found := false
			for _, av := range actualArr {
				if m.jsonContains(av, ev, opts) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	// Handle case where expected is primitive but actual is array
	// In mountebank, this checks if ANY element in the array matches
	if actualArr, ok := actual.([]interface{}); ok {
		for _, elem := range actualArr {
			if m.jsonContains(elem, expected, opts) {
				return true
			}
		}
		return false
	}

	// Handle strings with case sensitivity
	if expectedStr, ok := expected.(string); ok {
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		// Apply except pattern to both strings
		actualStr = m.applyExcept(actualStr, opts.except, opts.caseSensitive)
		expectedStr = m.applyExcept(expectedStr, opts.except, opts.caseSensitive)

		if opts.caseSensitive {
			return actualStr == expectedStr
		}
		return strings.EqualFold(actualStr, expectedStr)
	}

	// For other primitives (numbers, bools), use reflect.DeepEqual
	return reflect.DeepEqual(actual, expected)
}

// jsonContainsString checks if JSON values contain expected strings
func (m *Matcher) jsonContainsString(actual, expected interface{}, opts predicateOptions) bool {
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		for k, ev := range expectedMap {
			av, exists := actualMap[k]
			if !exists && !opts.keyCaseSensitive {
				for ak, akv := range actualMap {
					if strings.EqualFold(ak, k) {
						av = akv
						exists = true
						break
					}
				}
			}
			if !exists {
				return false
			}

			// Check if value contains expected string
			evStr, evOk := toString(ev)
			avStr, avOk := toString(av)
			if !evOk || !avOk {
				return false
			}

			if opts.caseSensitive {
				if !strings.Contains(avStr, evStr) {
					return false
				}
			} else {
				if !strings.Contains(strings.ToLower(avStr), strings.ToLower(evStr)) {
					return false
				}
			}
		}
		return true
	}
	return false
}

// jsonStartsWith checks if JSON values start with expected strings
func (m *Matcher) jsonStartsWith(actual, expected interface{}, opts predicateOptions) bool {
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		for k, ev := range expectedMap {
			av, exists := actualMap[k]
			if !exists && !opts.keyCaseSensitive {
				for ak, akv := range actualMap {
					if strings.EqualFold(ak, k) {
						av = akv
						exists = true
						break
					}
				}
			}
			if !exists {
				return false
			}

			// Check if value starts with expected string
			evStr, evOk := toString(ev)
			avStr, avOk := toString(av)
			if !evOk || !avOk {
				return false
			}

			if opts.caseSensitive {
				if !strings.HasPrefix(avStr, evStr) {
					return false
				}
			} else {
				if !strings.HasPrefix(strings.ToLower(avStr), strings.ToLower(evStr)) {
					return false
				}
			}
		}
		return true
	}
	return false
}

// jsonEndsWith checks if JSON values end with expected strings
func (m *Matcher) jsonEndsWith(actual, expected interface{}, opts predicateOptions) bool {
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		for k, ev := range expectedMap {
			av, exists := actualMap[k]
			if !exists && !opts.keyCaseSensitive {
				for ak, akv := range actualMap {
					if strings.EqualFold(ak, k) {
						av = akv
						exists = true
						break
					}
				}
			}
			if !exists {
				return false
			}

			// Check if value ends with expected string
			evStr, evOk := toString(ev)
			avStr, avOk := toString(av)
			if !evOk || !avOk {
				return false
			}

			if opts.caseSensitive {
				if !strings.HasSuffix(avStr, evStr) {
					return false
				}
			} else {
				if !strings.HasSuffix(strings.ToLower(avStr), strings.ToLower(evStr)) {
					return false
				}
			}
		}
		return true
	}
	return false
}

// containsValue checks if actual contains expected
func (m *Matcher) containsValue(actual, expected interface{}, opts predicateOptions) bool {
	actualStr, actualIsStr := toString(actual)

	// Handle case where expected is a map (JSON body matching)
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		// actual should be a JSON string, parse it
		if actualIsStr {
			var actualParsed interface{}
			if err := json.Unmarshal([]byte(actualStr), &actualParsed); err == nil {
				return m.jsonContainsString(actualParsed, expectedMap, opts)
			}
		}
		return false
	}

	expectedStr, expectedIsStr := toString(expected)

	if actualIsStr && expectedIsStr {
		// Apply except pattern to strip matching portions
		actualStr = m.applyExcept(actualStr, opts.except, opts.caseSensitive)

		if opts.caseSensitive {
			return strings.Contains(actualStr, expectedStr)
		}
		return strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr))
	}

	return false
}

// startsWithValue checks if actual starts with expected
func (m *Matcher) startsWithValue(actual, expected interface{}, opts predicateOptions) bool {
	actualStr, actualIsStr := toString(actual)

	// Handle case where expected is a map (JSON body matching)
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		// actual should be a JSON string, parse it
		if actualIsStr {
			var actualParsed interface{}
			if err := json.Unmarshal([]byte(actualStr), &actualParsed); err == nil {
				return m.jsonStartsWith(actualParsed, expectedMap, opts)
			}
		}
		return false
	}

	expectedStr, expectedIsStr := toString(expected)

	if actualIsStr && expectedIsStr {
		// Apply except pattern to strip matching portions
		actualStr = m.applyExcept(actualStr, opts.except, opts.caseSensitive)

		if opts.caseSensitive {
			return strings.HasPrefix(actualStr, expectedStr)
		}
		return strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(expectedStr))
	}

	return false
}

// endsWithValue checks if actual ends with expected
func (m *Matcher) endsWithValue(actual, expected interface{}, opts predicateOptions) bool {
	actualStr, actualIsStr := toString(actual)

	// Handle case where expected is a map (JSON body matching)
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		// actual should be a JSON string, parse it
		if actualIsStr {
			var actualParsed interface{}
			if err := json.Unmarshal([]byte(actualStr), &actualParsed); err == nil {
				return m.jsonEndsWith(actualParsed, expectedMap, opts)
			}
		}
		return false
	}

	expectedStr, expectedIsStr := toString(expected)

	if actualIsStr && expectedIsStr {
		// Apply except pattern to strip matching portions
		actualStr = m.applyExcept(actualStr, opts.except, opts.caseSensitive)

		if opts.caseSensitive {
			return strings.HasSuffix(actualStr, expectedStr)
		}
		return strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(expectedStr))
	}

	return false
}

// matchesPattern checks if actual matches the regex pattern
func (m *Matcher) matchesPattern(actual, pattern interface{}, opts predicateOptions) bool {
	actualStr, actualIsStr := toString(actual)
	patternStr, patternIsStr := toString(pattern)

	// Handle case where pattern is a map (JSON body matching with regex)
	if patternMap, ok := pattern.(map[string]interface{}); ok {
		// actual should be a JSON string, parse it
		if actualIsStr {
			var actualParsed interface{}
			if err := json.Unmarshal([]byte(actualStr), &actualParsed); err == nil {
				return m.jsonMatchesPattern(actualParsed, patternMap, opts)
			}
		}
		return false
	}

	if !actualIsStr || !patternIsStr {
		return false
	}

	// Apply except pattern to strip matching portions
	actualStr = m.applyExcept(actualStr, opts.except, opts.caseSensitive)

	re, err := regexp.Compile(patternStr)
	if err != nil {
		return false
	}

	return re.MatchString(actualStr)
}

// jsonMatchesPattern checks if JSON values match regex patterns
func (m *Matcher) jsonMatchesPattern(actual interface{}, patternMap map[string]interface{}, opts predicateOptions) bool {
	actualMap, ok := actual.(map[string]interface{})
	if !ok {
		return false
	}

	for k, patternVal := range patternMap {
		av, exists := actualMap[k]
		if !exists {
			// Try case-insensitive if needed
			if !opts.keyCaseSensitive {
				for ak, akv := range actualMap {
					if strings.EqualFold(ak, k) {
						av = akv
						exists = true
						break
					}
				}
			}
		}
		if !exists {
			return false
		}

		// patternVal could be a string (regex) or nested map
		if nestedPattern, ok := patternVal.(map[string]interface{}); ok {
			if !m.jsonMatchesPattern(av, nestedPattern, opts) {
				return false
			}
		} else {
			// It's a regex pattern string
			patternStr, _ := toString(patternVal)
			actualValStr := ""
			switch v := av.(type) {
			case string:
				actualValStr = v
			case float64:
				actualValStr = strings.TrimRight(strings.TrimRight(
					strings.Replace(fmt.Sprintf("%f", v), ".", "", -1), "0"), ".")
				// Try the simpler approach
				actualValStr = fmt.Sprintf("%v", v)
			default:
				actualValStr = fmt.Sprintf("%v", v)
			}

			// Add case-insensitive flag if not case-sensitive
			if !opts.caseSensitive {
				patternStr = "(?i)" + patternStr
			}

			re, err := regexp.Compile(patternStr)
			if err != nil {
				return false
			}
			if !re.MatchString(actualValStr) {
				return false
			}
		}
	}

	return true
}

// toString converts a value to string
func toString(v interface{}) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case []byte:
		return string(val), true
	case nil:
		return "", true
	default:
		return "", false
	}
}
