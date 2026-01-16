package imposter

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/dop251/goja"
)

// TCPServer represents a TCP imposter server
type TCPServer struct {
	imposter *models.Imposter
	listener net.Listener
	matcher  *TCPMatcher
	jsEngine *JSEngine
	state    map[string]interface{} // Persistent state for injection scripts
	started  bool
	stopping bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// NewTCPServer creates a new TCP imposter server
func NewTCPServer(imp *models.Imposter) (*TCPServer, error) {
	return &TCPServer{
		imposter: imp,
		matcher:  NewTCPMatcher(imp),
		jsEngine: NewJSEngine(),
		state:    make(map[string]interface{}),
	}, nil
}

// Start starts the TCP server
func (s *TCPServer) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	s.started = true
	s.mu.Unlock()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.imposter.Host, s.imposter.Port))
	if err != nil {
		s.mu.Lock()
		s.started = false
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on %s:%d: %w", s.imposter.Host, s.imposter.Port, err)
	}
	s.listener = listener

	go s.acceptLoop()

	return nil
}

// Stop stops the TCP server
func (s *TCPServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.stopping = true
	s.started = false
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all connections to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// acceptLoop accepts incoming connections
func (s *TCPServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.RLock()
			stopping := s.stopping
			s.mu.RUnlock()
			if stopping {
				return
			}
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single TCP connection
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Read data from connection (potentially multiple packets with resolver)
	data, err := s.readRequest(conn)
	if err != nil {
		return
	}

	// Convert to string based on mode
	var dataStr string
	if s.imposter.Mode == "binary" {
		dataStr = base64.StdEncoding.EncodeToString(data)
	} else {
		dataStr = string(data)
	}

	// Record request if configured
	s.mu.Lock()
	if s.imposter.RecordRequests {
		tcpReq := models.TCPRequest{
			RequestFrom: conn.RemoteAddr().String(),
			Data:        dataStr,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		s.imposter.TCPRequests = append(s.imposter.TCPRequests, tcpReq)
	}
	// Increment request counter
	if s.imposter.NumberOfRequests == nil {
		count := 1
		s.imposter.NumberOfRequests = &count
	} else {
		*s.imposter.NumberOfRequests++
	}
	s.mu.Unlock()

	// Find matching stub
	match := s.matcher.Match(dataStr)

	// Check for proxy response first
	if match.RawResponse != nil && match.RawResponse.Proxy != nil {
		s.handleProxyRequest(conn, data, match.RawResponse.Proxy)
		return
	}

	// Determine response data
	var responseData string

	// Check for injection response
	if match.RawResponse != nil && match.RawResponse.Inject != "" {
		// Execute injection to get response
		s.mu.Lock()
		injectedData, err := s.jsEngine.ExecuteTCPResponse(match.RawResponse.Inject, dataStr, s.state)
		s.mu.Unlock()

		if err != nil {
			log.Printf("[ERROR] TCP response injection error: %v", err)
			// Fall through to use static response if available
			if match.Response != nil {
				responseData = match.Response.Data
			}
		} else {
			responseData = injectedData
		}
	} else if match.Response != nil {
		responseData = match.Response.Data
	}

	// Apply behaviors if present
	if match.RawResponse != nil && len(match.RawResponse.Behaviors) > 0 {
		responseData = s.applyTCPBehaviors(dataStr, responseData, match.RawResponse.Behaviors)
	}

	// Write response if we have data
	if responseData != "" {
		// Handle binary mode
		if s.imposter.Mode == "binary" || (match.Response != nil && match.Response.Mode == "binary") {
			decoded, err := base64.StdEncoding.DecodeString(responseData)
			if err == nil {
				conn.Write(decoded)
				return
			}
		}

		conn.Write([]byte(responseData))
	}
}

// applyTCPBehaviors applies behaviors to TCP response data
func (s *TCPServer) applyTCPBehaviors(requestData, responseData string, behaviors []models.Behavior) string {
	result := responseData

	// Create a simple request/response structure for behaviors
	// For TCP, request data goes in Body, response uses Data field
	req := &models.Request{
		Body: requestData,
	}
	resp := &models.IsResponse{
		Data: result,
	}

	// Use BehaviorExecutor for full behavior support
	behaviorExecutor := NewBehaviorExecutor(s.jsEngine)
	processedResp, err := behaviorExecutor.Execute(req, resp, behaviors)
	if err != nil {
		log.Printf("[ERROR] TCP behavior execution error: %v", err)
		return result
	}

	// Extract data from processed response
	if processedResp.Data != "" {
		result = processedResp.Data
	} else if bodyStr, ok := processedResp.Body.(string); ok {
		result = bodyStr
	}

	return result
}

// executeTCPDecorate executes a decorate behavior for TCP
func (s *TCPServer) executeTCPDecorate(requestData, responseData, script string) string {
	vm := goja.New()

	// Create request object
	reqObj := map[string]interface{}{
		"data": requestData,
	}

	// Create response object (mutable)
	respObj := map[string]interface{}{
		"data": responseData,
	}

	vm.Set("request", reqObj)
	vm.Set("response", respObj)

	// Wrap the script to execute the function
	wrappedScript := fmt.Sprintf(`
		(function() {
			var fn = %s;
			fn(request, response);
			return response;
		})()
	`, script)

	result, err := vm.RunString(wrappedScript)
	if err != nil {
		log.Printf("[ERROR] TCP decorate behavior error: %v", err)
		return responseData
	}

	// Extract data from the result
	exported := result.Export()
	if respMap, ok := exported.(map[string]interface{}); ok {
		if data, ok := respMap["data"]; ok {
			return fmt.Sprintf("%v", data)
		}
	}

	return responseData
}

// handleProxyRequest proxies the TCP request to the origin server
func (s *TCPServer) handleProxyRequest(clientConn net.Conn, requestData []byte, proxy *models.ProxyResponse) {
	// Parse the target URL
	targetURL := proxy.To
	// Remove tcp:// prefix if present
	target := strings.TrimPrefix(targetURL, "tcp://")

	// Connect to origin server
	originConn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Printf("[ERROR] TCP proxy connection failed to %s: %v", target, err)
		// Close client connection on proxy error
		return
	}
	defer originConn.Close()

	// Forward request to origin
	if _, err := originConn.Write(requestData); err != nil {
		log.Printf("[ERROR] TCP proxy write to origin failed: %v", err)
		return
	}

	// Read response from origin
	originConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	response := make([]byte, 4096)
	n, err := originConn.Read(response)
	if err != nil && err != io.EOF {
		log.Printf("[ERROR] TCP proxy read from origin failed: %v", err)
		return
	}

	// Forward response to client
	if n > 0 {
		clientConn.Write(response[:n])
	}
}

// readRequest reads request data from connection, using endOfRequestResolver if configured
func (s *TCPServer) readRequest(conn net.Conn) ([]byte, error) {
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Check if we have an endOfRequestResolver
	resolver := s.imposter.EndOfRequestResolver
	if resolver == nil || resolver.Inject == "" {
		// No resolver - single read (original behavior)
		buffer := make([]byte, 4096)
		n, err := conn.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		return buffer[:n], nil
	}

	// With resolver - buffer multiple reads until resolver returns true
	var accumulated []byte
	buffer := make([]byte, 4096)

	for {
		// Reset deadline for each read
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		n, err := conn.Read(buffer)
		if n > 0 {
			accumulated = append(accumulated, buffer[:n]...)

			// Check if request is complete using the resolver
			// Convert accumulated data to string for resolver
			var dataStr string
			if s.imposter.Mode == "binary" {
				dataStr = base64.StdEncoding.EncodeToString(accumulated)
			} else {
				dataStr = string(accumulated)
			}

			complete, resolverErr := s.jsEngine.ExecuteEndOfRequestResolver(resolver.Inject, dataStr)
			if resolverErr != nil {
				// Resolver error - treat current data as complete request
				return accumulated, nil
			}

			if complete {
				return accumulated, nil
			}
		}

		if err != nil {
			if err == io.EOF {
				// Connection closed - return what we have
				return accumulated, nil
			}
			// Check if it's a timeout
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout waiting for more data - return what we have
				if len(accumulated) > 0 {
					return accumulated, nil
				}
			}
			return accumulated, err
		}
	}
}

// GetImposter returns the imposter configuration
func (s *TCPServer) GetImposter() *models.Imposter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.imposter
}

// UpdateStubs updates the stubs for this imposter
func (s *TCPServer) UpdateStubs(stubs []models.Stub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imposter.Stubs = stubs
	s.matcher = NewTCPMatcher(s.imposter)
}

// TCPMatcher handles request matching for TCP protocol
type TCPMatcher struct {
	imposter *models.Imposter
	jsEngine *JSEngine
}

// NewTCPMatcher creates a new TCP matcher
func NewTCPMatcher(imp *models.Imposter) *TCPMatcher {
	return &TCPMatcher{
		imposter: imp,
		jsEngine: NewJSEngine(),
	}
}

// TCPMatchResult contains the result of matching a TCP request
type TCPMatchResult struct {
	Response    *models.IsResponse
	RawResponse *models.Response // For accessing Inject field
	Stub        *models.Stub
	StubIndex   int
}

// Match finds a matching stub for the given TCP data
func (m *TCPMatcher) Match(data string) *TCPMatchResult {
	for i := range m.imposter.Stubs {
		stub := &m.imposter.Stubs[i]
		if m.matchesAllPredicates(stub, data) {
			return m.getMatchResult(stub, i)
		}
	}

	// No match - return default response or empty
	if m.imposter.DefaultResponse != nil && m.imposter.DefaultResponse.Is != nil {
		return &TCPMatchResult{Response: m.imposter.DefaultResponse.Is}
	}

	return &TCPMatchResult{Response: &models.IsResponse{}}
}

// getMatchResult creates a TCPMatchResult from a stub
func (m *TCPMatcher) getMatchResult(stub *models.Stub, index int) *TCPMatchResult {
	if len(stub.Responses) == 0 {
		return &TCPMatchResult{
			Response:  &models.IsResponse{},
			Stub:      stub,
			StubIndex: index,
		}
	}

	resp := stub.NextResponse()
	if resp == nil {
		return &TCPMatchResult{
			Response:  &models.IsResponse{},
			Stub:      stub,
			StubIndex: index,
		}
	}

	return &TCPMatchResult{
		Response:    resp.Is,
		RawResponse: resp,
		Stub:        stub,
		StubIndex:   index,
	}
}

// matchesAllPredicates checks if data matches all predicates in a stub
func (m *TCPMatcher) matchesAllPredicates(stub *models.Stub, data string) bool {
	if len(stub.Predicates) == 0 {
		return true
	}

	for _, pred := range stub.Predicates {
		if !m.evaluatePredicate(&pred, data) {
			return false
		}
	}

	return true
}

// evaluatePredicate evaluates a single predicate against TCP data
func (m *TCPMatcher) evaluatePredicate(pred *models.Predicate, data string) bool {
	// Handle injection first
	if pred.Inject != "" {
		result, err := m.jsEngine.ExecuteTCPPredicate(pred.Inject, data)
		if err != nil {
			// Log error but don't match on injection error
			log.Printf("[WARN] TCP predicate injection error: %v", err)
			return false
		}
		return result
	}

	// Handle logical operators
	if pred.And != nil {
		for _, p := range pred.And {
			if !m.evaluatePredicate(&p, data) {
				return false
			}
		}
		return true
	}

	if pred.Or != nil {
		for _, p := range pred.Or {
			if m.evaluatePredicate(&p, data) {
				return true
			}
		}
		return false
	}

	if pred.Not != nil {
		return !m.evaluatePredicate(pred.Not, data)
	}

	// Handle comparison operators for TCP data
	if pred.Equals != nil {
		return m.evaluateEquals(pred.Equals, data, pred.CaseSensitive)
	}

	if pred.Contains != nil {
		return m.evaluateContains(pred.Contains, data, pred.CaseSensitive)
	}

	if pred.StartsWith != nil {
		return m.evaluateStartsWith(pred.StartsWith, data, pred.CaseSensitive)
	}

	if pred.EndsWith != nil {
		return m.evaluateEndsWith(pred.EndsWith, data, pred.CaseSensitive)
	}

	if pred.Matches != nil {
		return m.evaluateMatches(pred.Matches, data)
	}

	return true
}

// evaluateEquals checks if data equals the expected value
func (m *TCPMatcher) evaluateEquals(expected interface{}, data string, caseSensitive bool) bool {
	// Handle map format: {"data": "expected"}
	if predMap, ok := expected.(map[string]interface{}); ok {
		if expectedData, ok := predMap["data"]; ok {
			expectedStr, _ := expectedData.(string)
			if caseSensitive {
				return data == expectedStr
			}
			return strings.EqualFold(data, expectedStr)
		}
	}

	// Handle direct string
	if expectedStr, ok := expected.(string); ok {
		if caseSensitive {
			return data == expectedStr
		}
		return strings.EqualFold(data, expectedStr)
	}

	return false
}

// evaluateContains checks if data contains the expected value
func (m *TCPMatcher) evaluateContains(expected interface{}, data string, caseSensitive bool) bool {
	if predMap, ok := expected.(map[string]interface{}); ok {
		if expectedData, ok := predMap["data"]; ok {
			expectedStr, _ := expectedData.(string)
			if caseSensitive {
				return strings.Contains(data, expectedStr)
			}
			return strings.Contains(strings.ToLower(data), strings.ToLower(expectedStr))
		}
	}

	if expectedStr, ok := expected.(string); ok {
		if caseSensitive {
			return strings.Contains(data, expectedStr)
		}
		return strings.Contains(strings.ToLower(data), strings.ToLower(expectedStr))
	}

	return false
}

// evaluateStartsWith checks if data starts with the expected value
func (m *TCPMatcher) evaluateStartsWith(expected interface{}, data string, caseSensitive bool) bool {
	if predMap, ok := expected.(map[string]interface{}); ok {
		if expectedData, ok := predMap["data"]; ok {
			expectedStr, _ := expectedData.(string)
			if caseSensitive {
				return strings.HasPrefix(data, expectedStr)
			}
			return strings.HasPrefix(strings.ToLower(data), strings.ToLower(expectedStr))
		}
	}

	if expectedStr, ok := expected.(string); ok {
		if caseSensitive {
			return strings.HasPrefix(data, expectedStr)
		}
		return strings.HasPrefix(strings.ToLower(data), strings.ToLower(expectedStr))
	}

	return false
}

// evaluateEndsWith checks if data ends with the expected value
func (m *TCPMatcher) evaluateEndsWith(expected interface{}, data string, caseSensitive bool) bool {
	if predMap, ok := expected.(map[string]interface{}); ok {
		if expectedData, ok := predMap["data"]; ok {
			expectedStr, _ := expectedData.(string)
			if caseSensitive {
				return strings.HasSuffix(data, expectedStr)
			}
			return strings.HasSuffix(strings.ToLower(data), strings.ToLower(expectedStr))
		}
	}

	if expectedStr, ok := expected.(string); ok {
		if caseSensitive {
			return strings.HasSuffix(data, expectedStr)
		}
		return strings.HasSuffix(strings.ToLower(data), strings.ToLower(expectedStr))
	}

	return false
}

// evaluateMatches checks if data matches a regex pattern
func (m *TCPMatcher) evaluateMatches(expected interface{}, data string) bool {
	var pattern string

	if predMap, ok := expected.(map[string]interface{}); ok {
		if expectedData, ok := predMap["data"]; ok {
			pattern, _ = expectedData.(string)
		}
	} else if expectedStr, ok := expected.(string); ok {
		pattern = expectedStr
	}

	if pattern == "" {
		return false
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(data)
}
