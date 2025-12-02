package imposter

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// ProxyHandler handles proxy responses
type ProxyHandler struct {
	client *http.Client
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects - return them to the client
				return http.ErrUseLastResponse
			},
		},
	}
}

// getClient returns the appropriate HTTP client for the proxy request
// If mTLS options are specified, creates a new client with custom TLS config
func (h *ProxyHandler) getClient(proxy *models.ProxyResponse) *http.Client {
	// Use default client if no mTLS options
	if proxy.Cert == "" && proxy.Key == "" && proxy.SecureProtocol == "" {
		return h.client
	}

	// Create custom TLS config
	tlsConfig := &tls.Config{}

	// Load client certificate if provided
	if proxy.Cert != "" && proxy.Key != "" {
		cert, err := tls.X509KeyPair([]byte(proxy.Cert), []byte(proxy.Key))
		if err == nil {
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	// Set TLS version based on secureProtocol
	switch strings.ToLower(proxy.SecureProtocol) {
	case "tlsv1":
		tlsConfig.MinVersion = tls.VersionTLS10
		tlsConfig.MaxVersion = tls.VersionTLS10
	case "tlsv1.1", "tlsv1_1":
		tlsConfig.MinVersion = tls.VersionTLS11
		tlsConfig.MaxVersion = tls.VersionTLS11
	case "tlsv1.2", "tlsv1_2":
		tlsConfig.MinVersion = tls.VersionTLS12
		tlsConfig.MaxVersion = tls.VersionTLS12
	case "tlsv1.3", "tlsv1_3":
		tlsConfig.MinVersion = tls.VersionTLS13
		tlsConfig.MaxVersion = tls.VersionTLS13
	default:
		// Use Go defaults (TLS 1.2+)
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	// Create transport with custom TLS config
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ProxyResult contains the result of a proxy operation
type ProxyResult struct {
	Response      *models.IsResponse
	GeneratedStub *models.Stub
	ShouldRecord  bool
}

// Execute proxies a request and returns the response
func (h *ProxyHandler) Execute(req *models.Request, proxy *models.ProxyResponse, originalReq *http.Request) (*ProxyResult, error) {
	// Build target URL
	targetURL, err := h.buildTargetURL(proxy.To, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build target URL: %w", err)
	}

	// Handle request body - decode if binary mode
	var bodyReader io.Reader
	if req.Mode == "binary" {
		decoded, err := base64.StdEncoding.DecodeString(req.Body)
		if err == nil {
			bodyReader = bytes.NewReader(decoded)
		} else {
			bodyReader = strings.NewReader(req.Body)
		}
	} else {
		bodyReader = strings.NewReader(req.Body)
	}

	// Create proxy request
	proxyReq, err := http.NewRequest(req.Method, targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy request: %w", err)
	}

	// Copy headers from original request
	for k, v := range req.Headers {
		proxyReq.Header.Set(k, v)
	}

	// Add injected headers
	for k, v := range proxy.InjectHeaders {
		proxyReq.Header.Set(k, v)
	}

	// Remove hop-by-hop headers
	proxyReq.Header.Del("connection")
	proxyReq.Header.Del("keep-alive")
	proxyReq.Header.Del("proxy-authenticate")
	proxyReq.Header.Del("proxy-authorization")
	proxyReq.Header.Del("te")
	proxyReq.Header.Del("trailers")
	proxyReq.Header.Del("transfer-encoding")
	proxyReq.Header.Del("upgrade")

	// Get the appropriate client (default or mTLS-configured)
	client := h.getClient(proxy)

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("proxy request failed: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(startTime)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	// Convert response headers
	headers := make(map[string]interface{})
	for k, v := range resp.Header {
		if len(v) > 0 {
			// Skip hop-by-hop headers
			lk := strings.ToLower(k)
			if lk == "connection" || lk == "keep-alive" || lk == "transfer-encoding" {
				continue
			}
			// Support multi-value headers
			if len(v) == 1 {
				headers[k] = v[0]
			} else {
				headers[k] = v
			}
		}
	}

	// Build response - handle binary content
	isResp := &models.IsResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
	}

	// Check if response is binary
	contentType := resp.Header.Get("Content-Type")
	if isBinaryResponseContent(contentType, respBody) {
		isResp.Body = base64.StdEncoding.EncodeToString(respBody)
		isResp.Mode = "binary"
	} else {
		isResp.Body = string(respBody)
	}

	result := &ProxyResult{
		Response: isResp,
	}

	// Determine if we should record based on mode
	mode := proxy.Mode
	if mode == "" {
		mode = "proxyOnce"
	}

	switch mode {
	case "proxyOnce":
		result.ShouldRecord = true
		result.GeneratedStub = h.generateStub(req, isResp, proxy, elapsed)
	case "proxyAlways":
		result.ShouldRecord = true
		result.GeneratedStub = h.generateStub(req, isResp, proxy, elapsed)
	case "proxyTransparent":
		result.ShouldRecord = false
	}

	return result, nil
}

// buildTargetURL constructs the target URL for proxying
func (h *ProxyHandler) buildTargetURL(to string, req *models.Request) (string, error) {
	targetBase, err := url.Parse(to)
	if err != nil {
		return "", err
	}

	// Append path from request
	targetURL := targetBase.ResolveReference(&url.URL{Path: req.Path})

	// Add query parameters
	if len(req.Query) > 0 {
		q := targetURL.Query()
		for k, v := range req.Query {
			q.Set(k, v)
		}
		targetURL.RawQuery = q.Encode()
	}

	return targetURL.String(), nil
}

// generateStub creates a new stub from the proxied request/response
func (h *ProxyHandler) generateStub(req *models.Request, resp *models.IsResponse, proxy *models.ProxyResponse, elapsed time.Duration) *models.Stub {
	stub := &models.Stub{
		Responses: []models.Response{
			{
				Is: resp,
			},
		},
	}

	// Add wait behavior if configured
	if proxy.AddWaitBehavior {
		stub.Responses[0].Behaviors = append(stub.Responses[0].Behaviors, models.Behavior{
			Wait: int(elapsed.Milliseconds()),
		})
	}

	// Add decorate behavior if configured
	if proxy.AddDecorateBehavior != "" {
		stub.Responses[0].Behaviors = append(stub.Responses[0].Behaviors, models.Behavior{
			Decorate: proxy.AddDecorateBehavior,
		})
	}

	// Generate predicates
	if len(proxy.PredicateGenerators) > 0 {
		stub.Predicates = h.generatePredicates(req, proxy.PredicateGenerators)
	} else {
		// Default: match on path and method
		stub.Predicates = []models.Predicate{
			{
				Equals: map[string]interface{}{
					"method": req.Method,
					"path":   req.Path,
				},
			},
		}
	}

	return stub
}

// generatePredicates creates predicates from generators
func (h *ProxyHandler) generatePredicates(req *models.Request, generators []models.PredicateGen) []models.Predicate {
	var predicates []models.Predicate

	for _, gen := range generators {
		pred := h.generatePredicate(req, &gen)
		if pred != nil {
			predicates = append(predicates, *pred)
		}
	}

	return predicates
}

// generatePredicate creates a single predicate from a generator
func (h *ProxyHandler) generatePredicate(req *models.Request, gen *models.PredicateGen) *models.Predicate {
	if gen.Matches == nil {
		return nil
	}

	matchesMap, ok := gen.Matches.(map[string]interface{})
	if !ok {
		return nil
	}

	predicate := &models.Predicate{
		CaseSensitive: gen.CaseSensitive,
	}

	equalsMap := make(map[string]interface{})

	for field, pattern := range matchesMap {
		value := h.getFieldValue(req, field)
		if value == nil {
			continue
		}

		// If pattern is true, include the whole field
		if patternBool, ok := pattern.(bool); ok && patternBool {
			equalsMap[field] = value
			continue
		}

		// If pattern is a string, it's a regex to extract the matching part
		if patternStr, ok := pattern.(string); ok {
			extracted := h.extractWithPattern(value, patternStr)
			if extracted != nil {
				equalsMap[field] = extracted
			}
		}

		// If pattern is a map, process each nested field
		if patternMap, ok := pattern.(map[string]interface{}); ok {
			if valueMap, ok := value.(map[string]string); ok {
				nestedEquals := make(map[string]interface{})
				for nestedField, nestedPattern := range patternMap {
					if nestedVal, exists := valueMap[nestedField]; exists {
						if nestedPatternBool, ok := nestedPattern.(bool); ok && nestedPatternBool {
							nestedEquals[nestedField] = nestedVal
						}
					}
				}
				if len(nestedEquals) > 0 {
					equalsMap[field] = nestedEquals
				}
			}
		}
	}

	if len(equalsMap) > 0 {
		predicate.Equals = equalsMap
		return predicate
	}

	return nil
}

// getFieldValue gets a field value from the request
func (h *ProxyHandler) getFieldValue(req *models.Request, field string) interface{} {
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
	default:
		return nil
	}
}

// extractWithPattern extracts a value using a regex pattern
func (h *ProxyHandler) extractWithPattern(value interface{}, pattern string) interface{} {
	str, ok := value.(string)
	if !ok {
		return nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	match := re.FindString(str)
	if match != "" {
		return match
	}

	return nil
}

// ToJSON serializes a value to JSON for body comparison
func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// BuildProxyRequest creates an HTTP request for proxying
func BuildProxyRequest(method, targetURL string, body []byte, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// isBinaryResponseContent determines if response content should be treated as binary
func isBinaryResponseContent(contentType string, data []byte) bool {
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
