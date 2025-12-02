package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EJSRenderer renders EJS templates with mountebank-compatible functions
type EJSRenderer struct {
	basePath string
	data     map[string]interface{}
}

// NewEJSRenderer creates a new EJS renderer
func NewEJSRenderer(basePath string) *EJSRenderer {
	return &EJSRenderer{
		basePath: basePath,
		data:     make(map[string]interface{}),
	}
}

// SetData sets the data context for template rendering
func (r *EJSRenderer) SetData(data map[string]interface{}) {
	r.data = data
}

// Render processes an EJS template and returns the result
func (r *EJSRenderer) Render(content string) (string, error) {
	result := content

	// Process includes: <%- include('filename') -%> or <%- include('filename') %>
	includeRegex := regexp.MustCompile(`<%-\s*include\s*\(\s*['"]([^'"]+)['"]\s*\)\s*-?%>`)
	for {
		matches := includeRegex.FindStringSubmatchIndex(result)
		if matches == nil {
			break
		}

		filename := result[matches[2]:matches[3]]
		includePath := filepath.Join(r.basePath, filename)

		includeContent, err := os.ReadFile(includePath)
		if err != nil {
			return "", fmt.Errorf("failed to include file %s: %w", filename, err)
		}

		// Recursively render included content
		subRenderer := NewEJSRenderer(filepath.Dir(includePath))
		subRenderer.SetData(r.data)
		renderedInclude, err := subRenderer.Render(string(includeContent))
		if err != nil {
			return "", fmt.Errorf("failed to render included file %s: %w", filename, err)
		}

		result = result[:matches[0]] + renderedInclude + result[matches[1]:]
	}

	// Process stringify: <%- stringify(filename, 'path/to/file') %> or <%- stringify(filename, 'path/to/file', {data}) %>
	// This reads a file and JSON-escapes its contents for embedding in a string
	// The third parameter (data) is optional and provides data context for nested templates
	stringifyRegex := regexp.MustCompile(`<%-\s*stringify\s*\(\s*filename\s*,\s*['"]([^'"]+)['"]\s*(?:,\s*(\{[^}]*\}))?\s*\)\s*%>`)
	for {
		matches := stringifyRegex.FindStringSubmatchIndex(result)
		if matches == nil {
			break
		}

		filename := result[matches[2]:matches[3]]
		filePath := filepath.Join(r.basePath, filename)

		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to stringify file %s: %w", filename, err)
		}

		// Check if there's data context specified
		var localData map[string]interface{}
		if matches[4] != -1 && matches[5] != -1 {
			dataStr := result[matches[4]:matches[5]]
			// Convert JavaScript object notation to JSON
			jsonStr := jsObjectToJSON(dataStr)
			if err := json.Unmarshal([]byte(jsonStr), &localData); err != nil {
				return "", fmt.Errorf("invalid data context in stringify '%s' (converted to '%s'): %w", dataStr, jsonStr, err)
			}
		}

		// If file is .ejs or needs EJS processing, render it first
		contentStr := string(fileContent)
		if strings.HasSuffix(filename, ".ejs") || strings.Contains(contentStr, "<%-") {
			subRenderer := NewEJSRenderer(filepath.Dir(filePath))
			if localData != nil {
				subRenderer.SetData(localData)
			} else {
				subRenderer.SetData(r.data)
			}
			contentStr, err = subRenderer.Render(contentStr)
			if err != nil {
				return "", fmt.Errorf("failed to render stringify file %s: %w", filename, err)
			}
		}

		// JSON-escape the content for embedding in a JSON string
		escaped, err := json.Marshal(contentStr)
		if err != nil {
			return "", fmt.Errorf("failed to JSON-escape content: %w", err)
		}
		// Remove the surrounding quotes from json.Marshal
		escapedStr := string(escaped[1 : len(escaped)-1])

		result = result[:matches[0]] + escapedStr + result[matches[1]:]
	}

	// Process inject: <%- inject(filename, 'path/to/file') %>
	// This is similar to stringify but for JavaScript injection
	injectRegex := regexp.MustCompile(`<%-\s*inject\s*\(\s*filename\s*,\s*['"]([^'"]+)['"]\s*\)\s*%>`)
	for {
		matches := injectRegex.FindStringSubmatchIndex(result)
		if matches == nil {
			break
		}

		filename := result[matches[2]:matches[3]]
		filePath := filepath.Join(r.basePath, filename)

		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to inject file %s: %w", filename, err)
		}

		// If file is .ejs, render it first
		contentStr := string(fileContent)
		if strings.HasSuffix(filename, ".ejs") {
			subRenderer := NewEJSRenderer(filepath.Dir(filePath))
			subRenderer.SetData(r.data)
			contentStr, err = subRenderer.Render(contentStr)
			if err != nil {
				return "", fmt.Errorf("failed to render inject file %s: %w", filename, err)
			}
		}

		// JSON-escape for embedding
		escaped, err := json.Marshal(contentStr)
		if err != nil {
			return "", fmt.Errorf("failed to JSON-escape inject content: %w", err)
		}
		escapedStr := string(escaped[1 : len(escaped)-1])

		result = result[:matches[0]] + escapedStr + result[matches[1]:]
	}

	// Process data variable access: <%- data.varName %>
	dataRegex := regexp.MustCompile(`<%-\s*data\.(\w+)\s*%>`)
	result = dataRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := dataRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := submatches[1]
		if val, ok := r.data[varName]; ok {
			return fmt.Sprintf("%v", val)
		}
		return ""
	})

	return result, nil
}

// RenderFile reads and renders an EJS file
func (r *EJSRenderer) RenderFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	r.basePath = filepath.Dir(filename)
	return r.Render(string(content))
}

// jsObjectToJSON converts JavaScript object notation to valid JSON
// Handles unquoted keys and single-quoted values
func jsObjectToJSON(jsObj string) string {
	result := jsObj

	// Replace single quotes with double quotes for values
	// Match: key: 'value' and convert to key: "value"
	singleQuoteRegex := regexp.MustCompile(`:\s*'([^']*)'`)
	result = singleQuoteRegex.ReplaceAllString(result, `: "$1"`)

	// Quote unquoted keys
	// Match: { key: or , key: and convert to { "key": or , "key":
	unquotedKeyRegex := regexp.MustCompile(`([{,])\s*(\w+)\s*:`)
	result = unquotedKeyRegex.ReplaceAllString(result, `$1 "$2":`)

	return result
}
