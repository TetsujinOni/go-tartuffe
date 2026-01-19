package imposter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/antchfx/xmlquery"
)

// SelectorEvaluator handles JSONPath and XPath selectors
type SelectorEvaluator struct{}

// NewSelectorEvaluator creates a new selector evaluator
func NewSelectorEvaluator() *SelectorEvaluator {
	return &SelectorEvaluator{}
}

// ApplySelector applies a selector to extract values from the request body
// Returns the extracted value(s) as a string for comparison
func (e *SelectorEvaluator) ApplySelector(body string, selector *models.Selector, selectorType string) (string, error) {
	return e.ApplySelectorWithOptions(body, selector, selectorType, false)
}

// ApplySelectorWithOptions applies a selector with case sensitivity option
func (e *SelectorEvaluator) ApplySelectorWithOptions(body string, selector *models.Selector, selectorType string, keyCaseSensitive bool) (string, error) {
	if selector == nil || selector.Selector == "" {
		return body, nil
	}

	switch selectorType {
	case "jsonpath":
		return e.applyJSONPathWithOptions(body, selector.Selector, keyCaseSensitive)
	case "xpath":
		return e.applyXPath(body, selector.Selector, selector.Namespaces)
	default:
		return body, nil
	}
}

// applyJSONPathWithOptions extracts values using JSONPath with case sensitivity option
func (e *SelectorEvaluator) applyJSONPathWithOptions(body, path string, keyCaseSensitive bool) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "", fmt.Errorf("invalid JSON body: %w", err)
	}

	result := e.evaluateJSONPathWithOptions(data, path, keyCaseSensitive)
	return result, nil
}

// evaluateJSONPathWithOptions evaluates a JSONPath expression with case sensitivity option
func (e *SelectorEvaluator) evaluateJSONPathWithOptions(data interface{}, path string, keyCaseSensitive bool) string {
	// Remove leading $ if present
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")

	if path == "" {
		return e.valueToString(data)
	}

	// Handle recursive descent (..)
	if strings.HasPrefix(path, ".") {
		path = strings.TrimPrefix(path, ".")
		return e.recursiveSearchWithOptions(data, path, keyCaseSensitive)
	}

	return e.navigatePathWithOptions(data, path, keyCaseSensitive)
}

// navigatePathWithOptions navigates through the data structure with case sensitivity option
func (e *SelectorEvaluator) navigatePathWithOptions(data interface{}, path string, keyCaseSensitive bool) string {
	if path == "" {
		return e.valueToString(data)
	}

	// Parse next segment
	segment, rest := e.parsePathSegment(path)

	switch d := data.(type) {
	case map[string]interface{}:
		// Try exact match first (works for both case-sensitive and case-insensitive modes)
		if val, ok := d[segment]; ok {
			return e.navigatePathWithOptions(val, rest, keyCaseSensitive)
		}

		// If case-insensitive mode and exact match failed, try case-insensitive match
		if !keyCaseSensitive {
			for k, v := range d {
				if strings.EqualFold(k, segment) {
					return e.navigatePathWithOptions(v, rest, keyCaseSensitive)
				}
			}
		}

		// If segment is bracket notation, handle it
		if strings.HasPrefix(segment, "[") {
			key := e.extractBracketKey(segment)
			if val, ok := d[key]; ok {
				return e.navigatePathWithOptions(val, rest, keyCaseSensitive)
			}
			// Try case-insensitive for bracket notation too
			if !keyCaseSensitive {
				for k, v := range d {
					if strings.EqualFold(k, key) {
						return e.navigatePathWithOptions(v, rest, keyCaseSensitive)
					}
				}
			}
			return ""
		}

		// Handle wildcard
		if segment == "*" {
			var results []string
			for _, v := range d {
				result := e.navigatePathWithOptions(v, rest, keyCaseSensitive)
				if result != "" {
					results = append(results, result)
				}
			}
			if len(results) == 1 {
				return results[0]
			}
			return strings.Join(results, ",")
		}

	case []interface{}:
		// Handle array index: [0], [1], etc.
		if strings.HasPrefix(segment, "[") {
			indexStr := strings.Trim(segment, "[]")

			// Handle wildcard [*]
			if indexStr == "*" {
				var results []string
				for _, item := range d {
					result := e.navigatePathWithOptions(item, rest, keyCaseSensitive)
					if result != "" {
						results = append(results, result)
					}
				}
				if len(results) == 1 {
					return results[0]
				}
				return strings.Join(results, ",")
			}

			// Handle negative index
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return ""
			}
			if index < 0 {
				index = len(d) + index
			}
			if index >= 0 && index < len(d) {
				return e.navigatePathWithOptions(d[index], rest, keyCaseSensitive)
			}
			return ""
		}

		// If segment is a field name, apply to all array elements
		var results []string
		for _, item := range d {
			result := e.navigatePathWithOptions(item, segment+rest, keyCaseSensitive)
			if result != "" {
				results = append(results, result)
			}
		}
		if len(results) == 1 {
			return results[0]
		}
		return strings.Join(results, ",")
	}

	return ""
}

// parsePathSegment extracts the next path segment
func (e *SelectorEvaluator) parsePathSegment(path string) (string, string) {
	if path == "" {
		return "", ""
	}

	// Handle bracket notation at start
	if strings.HasPrefix(path, "[") {
		// Find matching closing bracket
		depth := 0
		for i, ch := range path {
			if ch == '[' {
				depth++
			} else if ch == ']' {
				depth--
				if depth == 0 {
					segment := path[:i+1]
					rest := strings.TrimPrefix(path[i+1:], ".")
					return segment, rest
				}
			}
		}
		return path, ""
	}

	// Find next separator (. or [)
	dotIdx := strings.Index(path, ".")
	bracketIdx := strings.Index(path, "[")

	if dotIdx == -1 && bracketIdx == -1 {
		return path, ""
	}

	if dotIdx == -1 {
		return path[:bracketIdx], path[bracketIdx:]
	}

	if bracketIdx == -1 {
		return path[:dotIdx], path[dotIdx+1:]
	}

	if dotIdx < bracketIdx {
		return path[:dotIdx], path[dotIdx+1:]
	}

	return path[:bracketIdx], path[bracketIdx:]
}

// extractBracketKey extracts a key from bracket notation
func (e *SelectorEvaluator) extractBracketKey(segment string) string {
	// Remove brackets
	key := strings.Trim(segment, "[]")
	// Remove quotes
	key = strings.Trim(key, "'\"")
	return key
}

// recursiveSearchWithOptions searches recursively for a key with case sensitivity option
func (e *SelectorEvaluator) recursiveSearchWithOptions(data interface{}, path string, keyCaseSensitive bool) string {
	segment, rest := e.parsePathSegment(path)

	switch d := data.(type) {
	case map[string]interface{}:
		// Check current level with exact match
		if val, ok := d[segment]; ok {
			result := e.navigatePathWithOptions(val, rest, keyCaseSensitive)
			if result != "" {
				return result
			}
		}

		// If case-insensitive mode, try case-insensitive match
		if !keyCaseSensitive {
			for k, v := range d {
				if strings.EqualFold(k, segment) {
					result := e.navigatePathWithOptions(v, rest, keyCaseSensitive)
					if result != "" {
						return result
					}
				}
			}
		}

		// Search deeper
		for _, v := range d {
			result := e.recursiveSearchWithOptions(v, path, keyCaseSensitive)
			if result != "" {
				return result
			}
		}

	case []interface{}:
		for _, item := range d {
			result := e.recursiveSearchWithOptions(item, path, keyCaseSensitive)
			if result != "" {
				return result
			}
		}
	}

	return ""
}

// valueToString converts a JSON value to string
func (e *SelectorEvaluator) valueToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	case []interface{}:
		// Return JSON array representation
		b, _ := json.Marshal(val)
		return string(b)
	case map[string]interface{}:
		// Return JSON object representation
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// applyXPath extracts values using XPath
func (e *SelectorEvaluator) applyXPath(body, xpath string, namespaces map[string]string) (string, error) {
	// Parse XML
	doc, err := xmlquery.Parse(strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("invalid XML body: %w", err)
	}

	// Find nodes matching XPath
	nodes, err := xmlquery.QueryAll(doc, xpath)
	if err != nil {
		return "", fmt.Errorf("invalid XPath expression: %w", err)
	}

	if len(nodes) == 0 {
		return "", nil
	}

	// Return first match for simple cases
	if len(nodes) == 1 {
		return e.getXMLNodeValue(nodes[0]), nil
	}

	// Return all matches as JSON array (mountebank compatibility)
	var results []string
	for _, node := range nodes {
		results = append(results, e.getXMLNodeValue(node))
	}
	// Return as JSON array to enable array predicate matching
	jsonArray, _ := json.Marshal(results)
	return string(jsonArray), nil
}

// getXMLNodeValue extracts the text value from an XML node
func (e *SelectorEvaluator) getXMLNodeValue(node *xmlquery.Node) string {
	if node == nil {
		return ""
	}

	// For attribute nodes, return the attribute value
	if node.Type == xmlquery.AttributeNode {
		return node.InnerText()
	}

	// For element nodes, return inner text
	return strings.TrimSpace(node.InnerText())
}

// ExtractWithSelector is a helper that applies selector and returns the value
func ExtractWithSelector(body string, pred *models.Predicate) (string, error) {
	evaluator := NewSelectorEvaluator()

	if pred.JSONPath != nil {
		return evaluator.ApplySelector(body, pred.JSONPath, "jsonpath")
	}

	if pred.XPath != nil {
		return evaluator.ApplySelector(body, pred.XPath, "xpath")
	}

	return body, nil
}

// ApplyRegexSelector applies a regex pattern to extract values
func ApplyRegexSelector(value, pattern string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return value
	}

	matches := re.FindStringSubmatch(value)
	if len(matches) == 0 {
		return ""
	}

	// Return first capture group if exists, otherwise full match
	if len(matches) > 1 {
		return matches[1]
	}
	return matches[0]
}
