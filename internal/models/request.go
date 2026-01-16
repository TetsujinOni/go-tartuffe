package models

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

// Request represents a simplified HTTP request for matching
type Request struct {
	RequestFrom string            `json:"requestFrom,omitempty"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Query       map[string]string `json:"query,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	Form        map[string]string `json:"form,omitempty"`
	IP          string            `json:"ip,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
	Mode        string            `json:"_mode,omitempty"`
}

// NewRequestFromHTTP creates a Request from an http.Request
func NewRequestFromHTTP(r *http.Request) (*Request, error) {
	// Read body
	var body string
	var mode string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		// Check if body is binary (non-UTF8 or binary content type)
		contentType := r.Header.Get("Content-Type")
		if isBinaryContent(contentType, bodyBytes) {
			body = base64.StdEncoding.EncodeToString(bodyBytes)
			mode = "binary"
		} else {
			body = string(bodyBytes)
		}
	}

	// Parse query parameters
	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	// Convert headers to simple map (first value only)
	// Preserve the canonical header name (Go canonicalizes to Title-Case)
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Extract IP
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	// Parse form data if content type is form-urlencoded or multipart
	var form map[string]string
	contentType := r.Header.Get("Content-Type")
	if body != "" && mode != "binary" {
		form = parseFormData(contentType, body)
	}

	return &Request{
		RequestFrom: r.RemoteAddr,
		Method:      r.Method,
		Path:        r.URL.Path,
		Query:       query,
		Headers:     headers,
		Body:        body,
		Form:        form,
		IP:          ip,
		Mode:        mode,
	}, nil
}

// parseFormData parses form data from the body based on content type
func parseFormData(contentType, body string) map[string]string {
	ct := strings.ToLower(contentType)

	// Handle application/x-www-form-urlencoded
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(body)
		if err != nil {
			return nil
		}
		form := make(map[string]string)
		for k, v := range values {
			if len(v) > 0 {
				form[k] = v[0]
			}
		}
		return form
	}

	// Handle multipart/form-data
	if strings.Contains(ct, "multipart/form-data") {
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
			return nil
		}
		boundary := params["boundary"]
		if boundary == "" {
			return nil
		}

		reader := multipart.NewReader(strings.NewReader(body), boundary)
		form := make(map[string]string)
		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}
			// Skip file uploads, only include text fields
			if part.FileName() != "" {
				part.Close()
				continue
			}
			name := part.FormName()
			if name == "" {
				part.Close()
				continue
			}
			value, err := io.ReadAll(part)
			part.Close()
			if err != nil {
				continue
			}
			form[name] = string(value)
		}
		if len(form) > 0 {
			return form
		}
	}

	return nil
}

// isBinaryContent determines if content should be treated as binary
func isBinaryContent(contentType string, data []byte) bool {
	// Check content type first
	ct := strings.ToLower(contentType)
	binaryTypes := []string{
		"application/octet-stream",
		"image/",
		"audio/",
		"video/",
		"application/pdf",
		"application/zip",
		"application/gzip",
		"application/x-tar",
	}

	for _, bt := range binaryTypes {
		if strings.Contains(ct, bt) {
			return true
		}
	}

	// If content type suggests text, don't treat as binary
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-www-form-urlencoded",
	}

	for _, tt := range textTypes {
		if strings.Contains(ct, tt) {
			return false
		}
	}

	// Check if data is valid UTF-8
	if len(data) > 0 && !utf8.Valid(data) {
		return true
	}

	return false
}
