package imposter

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// GRPCMatcher handles request matching for gRPC protocol
type GRPCMatcher struct {
	imposter    *models.Imposter
	protoLoader *ProtoLoader
}

// NewGRPCMatcher creates a new gRPC matcher
func NewGRPCMatcher(imp *models.Imposter, loader *ProtoLoader) *GRPCMatcher {
	return &GRPCMatcher{
		imposter:    imp,
		protoLoader: loader,
	}
}

// GRPCMatchResult contains the result of matching a gRPC request
type GRPCMatchResult struct {
	Response  *models.IsResponse
	Stub      *models.Stub
	StubIndex int
	Behaviors []models.Behavior
}

// Match finds a matching stub for the given gRPC request
func (m *GRPCMatcher) Match(req *models.GRPCRequest, method protoreflect.MethodDescriptor) *GRPCMatchResult {
	for i := range m.imposter.Stubs {
		stub := &m.imposter.Stubs[i]
		if m.matchesAllPredicates(stub, req) {
			return m.getMatchResult(stub, i)
		}
	}

	// No match - return default response or nil
	if m.imposter.DefaultResponse != nil && m.imposter.DefaultResponse.Is != nil {
		return &GRPCMatchResult{Response: m.imposter.DefaultResponse.Is}
	}

	return &GRPCMatchResult{Response: nil}
}

// getMatchResult creates a GRPCMatchResult from a stub
func (m *GRPCMatcher) getMatchResult(stub *models.Stub, index int) *GRPCMatchResult {
	if len(stub.Responses) == 0 {
		return &GRPCMatchResult{
			Response:  &models.IsResponse{},
			Stub:      stub,
			StubIndex: index,
		}
	}

	resp := stub.NextResponse()
	if resp == nil {
		return &GRPCMatchResult{
			Response:  &models.IsResponse{},
			Stub:      stub,
			StubIndex: index,
		}
	}

	return &GRPCMatchResult{
		Response:  resp.Is,
		Stub:      stub,
		StubIndex: index,
		Behaviors: resp.Behaviors,
	}
}

// matchesAllPredicates checks if request matches all predicates in a stub
func (m *GRPCMatcher) matchesAllPredicates(stub *models.Stub, req *models.GRPCRequest) bool {
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

// evaluatePredicate evaluates a single predicate against a gRPC request
func (m *GRPCMatcher) evaluatePredicate(pred *models.Predicate, req *models.GRPCRequest) bool {
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

	// Handle comparison operators
	if pred.Equals != nil {
		return m.evaluateEquals(pred.Equals, req, pred.CaseSensitive)
	}

	if pred.DeepEquals != nil {
		return m.evaluateDeepEquals(pred.DeepEquals, req)
	}

	if pred.Contains != nil {
		return m.evaluateContains(pred.Contains, req, pred.CaseSensitive)
	}

	if pred.StartsWith != nil {
		return m.evaluateStartsWith(pred.StartsWith, req, pred.CaseSensitive)
	}

	if pred.EndsWith != nil {
		return m.evaluateEndsWith(pred.EndsWith, req, pred.CaseSensitive)
	}

	if pred.Matches != nil {
		return m.evaluateMatches(pred.Matches, req)
	}

	if pred.Exists != nil {
		return m.evaluateExists(pred.Exists, req)
	}

	return true
}

// evaluateEquals checks if request fields equal expected values
func (m *GRPCMatcher) evaluateEquals(expected interface{}, req *models.GRPCRequest, caseSensitive bool) bool {
	predMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expectedVal := range predMap {
		actualVal := m.getFieldValue(req, field)
		if !m.valuesEqual(actualVal, expectedVal, caseSensitive) {
			return false
		}
	}

	return true
}

// evaluateDeepEquals checks deep equality of fields
func (m *GRPCMatcher) evaluateDeepEquals(expected interface{}, req *models.GRPCRequest) bool {
	predMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expectedVal := range predMap {
		actualVal := m.getFieldValue(req, field)
		if !reflect.DeepEqual(actualVal, expectedVal) {
			return false
		}
	}

	return true
}

// evaluateContains checks if fields contain expected values
func (m *GRPCMatcher) evaluateContains(expected interface{}, req *models.GRPCRequest, caseSensitive bool) bool {
	predMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expectedVal := range predMap {
		actualVal := m.getFieldValue(req, field)
		if !m.valueContains(actualVal, expectedVal, caseSensitive) {
			return false
		}
	}

	return true
}

// evaluateStartsWith checks if fields start with expected values
func (m *GRPCMatcher) evaluateStartsWith(expected interface{}, req *models.GRPCRequest, caseSensitive bool) bool {
	predMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expectedVal := range predMap {
		actualVal := m.getFieldValue(req, field)
		if !m.valueStartsWith(actualVal, expectedVal, caseSensitive) {
			return false
		}
	}

	return true
}

// evaluateEndsWith checks if fields end with expected values
func (m *GRPCMatcher) evaluateEndsWith(expected interface{}, req *models.GRPCRequest, caseSensitive bool) bool {
	predMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expectedVal := range predMap {
		actualVal := m.getFieldValue(req, field)
		if !m.valueEndsWith(actualVal, expectedVal, caseSensitive) {
			return false
		}
	}

	return true
}

// evaluateMatches checks if fields match regex patterns
func (m *GRPCMatcher) evaluateMatches(expected interface{}, req *models.GRPCRequest) bool {
	predMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	for field, pattern := range predMap {
		actualVal := m.getFieldValue(req, field)
		if !m.valueMatches(actualVal, pattern) {
			return false
		}
	}

	return true
}

// evaluateExists checks if fields exist
func (m *GRPCMatcher) evaluateExists(expected interface{}, req *models.GRPCRequest) bool {
	predMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	for field, shouldExist := range predMap {
		exists := m.fieldExists(req, field)
		expectExists, _ := shouldExist.(bool)
		if exists != expectExists {
			return false
		}
	}

	return true
}

// getFieldValue extracts a field value from the request
func (m *GRPCMatcher) getFieldValue(req *models.GRPCRequest, field string) interface{} {
	switch field {
	case "service":
		return req.Service
	case "method":
		return req.Method
	case "message":
		return req.Message
	case "metadata":
		return req.Metadata
	default:
		// Check for nested message fields (e.g., "message.user.name")
		if strings.HasPrefix(field, "message.") {
			path := strings.TrimPrefix(field, "message.")
			return getNestedValue(req.Message, path)
		}
		// Check for metadata fields (e.g., "metadata.authorization")
		if strings.HasPrefix(field, "metadata.") {
			key := strings.TrimPrefix(field, "metadata.")
			if vals, ok := req.Metadata[key]; ok && len(vals) > 0 {
				if len(vals) == 1 {
					return vals[0]
				}
				return vals
			}
			return nil
		}
		// Check directly in message
		return getNestedValue(req.Message, field)
	}
}

// fieldExists checks if a field exists in the request
func (m *GRPCMatcher) fieldExists(req *models.GRPCRequest, field string) bool {
	val := m.getFieldValue(req, field)
	return val != nil
}

// getNestedValue extracts a nested value from a map using dot notation
func getNestedValue(data map[string]interface{}, path string) interface{} {
	if data == nil {
		return nil
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// valuesEqual compares two values for equality
func (m *GRPCMatcher) valuesEqual(actual, expected interface{}, caseSensitive bool) bool {
	// Convert both to strings for comparison if possible
	actualStr := grpcToString(actual)
	expectedStr := grpcToString(expected)

	if caseSensitive {
		return actualStr == expectedStr
	}
	return strings.EqualFold(actualStr, expectedStr)
}

// valueContains checks if actual contains expected
func (m *GRPCMatcher) valueContains(actual, expected interface{}, caseSensitive bool) bool {
	actualStr := grpcToString(actual)
	expectedStr := grpcToString(expected)

	if caseSensitive {
		return strings.Contains(actualStr, expectedStr)
	}
	return strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr))
}

// valueStartsWith checks if actual starts with expected
func (m *GRPCMatcher) valueStartsWith(actual, expected interface{}, caseSensitive bool) bool {
	actualStr := grpcToString(actual)
	expectedStr := grpcToString(expected)

	if caseSensitive {
		return strings.HasPrefix(actualStr, expectedStr)
	}
	return strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(expectedStr))
}

// valueEndsWith checks if actual ends with expected
func (m *GRPCMatcher) valueEndsWith(actual, expected interface{}, caseSensitive bool) bool {
	actualStr := grpcToString(actual)
	expectedStr := grpcToString(expected)

	if caseSensitive {
		return strings.HasSuffix(actualStr, expectedStr)
	}
	return strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(expectedStr))
}

// valueMatches checks if actual matches the regex pattern
func (m *GRPCMatcher) valueMatches(actual, pattern interface{}) bool {
	actualStr := grpcToString(actual)
	patternStr := grpcToString(pattern)

	if patternStr == "" {
		return false
	}

	re, err := regexp.Compile(patternStr)
	if err != nil {
		return false
	}

	return re.MatchString(actualStr)
}

// grpcToString converts a value to string for gRPC matching
func grpcToString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		// Convert to JSON for complex types
		bytes, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(bytes)
	}
}
