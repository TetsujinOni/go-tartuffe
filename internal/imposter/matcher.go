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
	imposter *models.Imposter
}

// NewMatcher creates a new matcher for an imposter
func NewMatcher(imp *models.Imposter) *Matcher {
	return &Matcher{imposter: imp}
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
	if m.imposter.DefaultResponse != nil {
		if m.imposter.DefaultResponse.Is != nil {
			return &MatchResult{Response: m.imposter.DefaultResponse.Is}
		}
		if m.imposter.DefaultResponse.Proxy != nil {
			return &MatchResult{Proxy: m.imposter.DefaultResponse.Proxy}
		}
		if m.imposter.DefaultResponse.Inject != "" {
			return &MatchResult{Inject: m.imposter.DefaultResponse.Inject}
		}
	}

	return &MatchResult{Response: &models.IsResponse{StatusCode: 200}}
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
		result.Response = resp.Is
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
	opts := predicateOptions{
		caseSensitive:    pred.CaseSensitive,
		keyCaseSensitive: pred.KeyCaseSensitive,
		except:           pred.Except,
	}

	// Apply selector to extract value from body if specified
	effectiveReq := req
	if pred.JSONPath != nil || pred.XPath != nil {
		effectiveReq = m.applySelector(req, pred)
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
func (m *Matcher) applySelector(req *models.Request, pred *models.Predicate) *models.Request {
	evaluator := NewSelectorEvaluator()

	var extractedValue string
	var err error

	if pred.JSONPath != nil {
		extractedValue, err = evaluator.ApplySelector(req.Body, pred.JSONPath, "jsonpath")
	} else if pred.XPath != nil {
		extractedValue, err = evaluator.ApplySelector(req.Body, pred.XPath, "xpath")
	}

	if err != nil {
		// If extraction fails, return original request
		return req
	}

	// Create a modified request with extracted value as body
	modifiedReq := *req
	modifiedReq.Body = extractedValue
	return &modifiedReq
}

// evaluateInject executes a JavaScript predicate
func (m *Matcher) evaluateInject(script string, req *models.Request) bool {
	engine := NewJSEngine()
	result, err := engine.ExecutePredicate(script, req)
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

// deepCompareValues compares two values for deep equality
// This is stricter than compareValues - it requires exact match including no extra fields
func (m *Matcher) deepCompareValues(actual, expected interface{}, opts predicateOptions) bool {
	// Handle nil/null values
	if expected == nil {
		return actual == nil || actual == ""
	}

	// Handle string comparison
	actualStr, actualIsStr := toString(actual)
	expectedStr, expectedIsStr := toString(expected)

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
	if actualMap, ok := actual.(map[string]string); ok {
		if expectedMap, ok := expected.(map[string]interface{}); ok {
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
		if expectedMap, ok := expected.(map[string]interface{}); ok && len(expectedMap) == 0 {
			return len(actualMap) == 0
		}
	}

	// Handle nil expected with map actual
	if expected == nil {
		if actualMap, ok := actual.(map[string]string); ok {
			return len(actualMap) == 0
		}
	}

	return reflect.DeepEqual(actual, expected)
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

	// Handle arrays
	if expectedArr, ok := expected.([]interface{}); ok {
		actualArr, ok := actual.([]interface{})
		if !ok {
			return false
		}
		if len(actualArr) != len(expectedArr) {
			return false
		}
		for i, ev := range expectedArr {
			if !m.deepEqualJSON(actualArr[i], ev, opts) {
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
		return req.Query != nil && len(req.Query) > 0
	case "headers":
		return req.Headers != nil && len(req.Headers) > 0
	case "form":
		return req.Form != nil && len(req.Form) > 0
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

	// Handle arrays - must match exactly
	if expectedArr, ok := expected.([]interface{}); ok {
		actualArr, ok := actual.([]interface{})
		if !ok {
			return false
		}
		if len(actualArr) != len(expectedArr) {
			return false
		}
		for i, ev := range expectedArr {
			if !m.jsonContains(actualArr[i], ev, opts) {
				return false
			}
		}
		return true
	}

	// Handle case where expected is primitive but actual is array
	// In mountebank, this checks the first element
	if actualArr, ok := actual.([]interface{}); ok {
		if len(actualArr) > 0 {
			return m.jsonContains(actualArr[0], expected, opts)
		}
		return false
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

// containsValue checks if actual contains expected
func (m *Matcher) containsValue(actual, expected interface{}, opts predicateOptions) bool {
	actualStr, actualIsStr := toString(actual)
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
