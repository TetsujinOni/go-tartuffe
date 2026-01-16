package imposter

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/metrics"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// ImposterServer is the interface that both HTTP and TCP servers implement
type ImposterServer interface {
	Start() error
	Stop(ctx context.Context) error
	GetImposter() *models.Imposter
	UpdateStubs(stubs []models.Stub)
}

// Manager manages the lifecycle of imposter servers (HTTP, TCP, SMTP, gRPC)
type Manager struct {
	servers     map[int]*Server     // HTTP servers
	tcpServers  map[int]*TCPServer  // TCP servers
	smtpServers map[int]*SMTPServer // SMTP servers
	grpcServers map[int]*GRPCServer // gRPC servers
	mu          sync.RWMutex
}

// NewManager creates a new imposter manager
func NewManager() *Manager {
	return &Manager{
		servers:     make(map[int]*Server),
		tcpServers:  make(map[int]*TCPServer),
		smtpServers: make(map[int]*SMTPServer),
		grpcServers: make(map[int]*GRPCServer),
	}
}

// Start starts a server for the given imposter (HTTP or TCP based on protocol)
func (m *Manager) Start(imp *models.Imposter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if port is already in use
	if _, exists := m.servers[imp.Port]; exists {
		return fmt.Errorf("server already running on port %d", imp.Port)
	}
	if _, exists := m.tcpServers[imp.Port]; exists {
		return fmt.Errorf("server already running on port %d", imp.Port)
	}
	if _, exists := m.smtpServers[imp.Port]; exists {
		return fmt.Errorf("server already running on port %d", imp.Port)
	}
	if _, exists := m.grpcServers[imp.Port]; exists {
		return fmt.Errorf("server already running on port %d", imp.Port)
	}

	// Start appropriate server based on protocol
	switch imp.Protocol {
	case "tcp":
		return m.startTCPServer(imp)
	case "https":
		return m.startHTTPSServer(imp)
	case "smtp":
		return m.startSMTPServer(imp)
	case "grpc":
		return m.startGRPCServer(imp)
	default:
		return m.startHTTPServer(imp)
	}
}

// startHTTPServer starts an HTTP server for the given imposter
func (m *Manager) startHTTPServer(imp *models.Imposter) error {
	srv, err := NewServer(imp, false)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	m.servers[imp.Port] = srv
	return nil
}

// startHTTPSServer starts an HTTPS server for the given imposter
func (m *Manager) startHTTPSServer(imp *models.Imposter) error {
	srv, err := NewServer(imp, true)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	m.servers[imp.Port] = srv
	return nil
}

// startTCPServer starts a TCP server for the given imposter
func (m *Manager) startTCPServer(imp *models.Imposter) error {
	srv, err := NewTCPServer(imp)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	m.tcpServers[imp.Port] = srv
	return nil
}

// startSMTPServer starts an SMTP server for the given imposter
func (m *Manager) startSMTPServer(imp *models.Imposter) error {
	srv, err := NewSMTPServer(imp)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	m.smtpServers[imp.Port] = srv
	return nil
}

// startGRPCServer starts a gRPC server for the given imposter
func (m *Manager) startGRPCServer(imp *models.Imposter) error {
	srv, err := NewGRPCServer(imp)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	m.grpcServers[imp.Port] = srv
	return nil
}

// Stop stops the server for the given port (HTTP, TCP, SMTP, or gRPC)
func (m *Manager) Stop(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try HTTP server first
	if srv, exists := m.servers[port]; exists {
		if err := srv.Stop(ctx); err != nil {
			return err
		}
		delete(m.servers, port)
		return nil
	}

	// Try TCP server
	if srv, exists := m.tcpServers[port]; exists {
		if err := srv.Stop(ctx); err != nil {
			return err
		}
		delete(m.tcpServers, port)
		return nil
	}

	// Try SMTP server
	if srv, exists := m.smtpServers[port]; exists {
		if err := srv.Stop(ctx); err != nil {
			return err
		}
		delete(m.smtpServers, port)
		return nil
	}

	// Try gRPC server
	if srv, exists := m.grpcServers[port]; exists {
		if err := srv.Stop(ctx); err != nil {
			return err
		}
		delete(m.grpcServers, port)
		return nil
	}

	return nil // Not running, nothing to stop
}

// StopAll stops all running imposter servers (HTTP, TCP, SMTP, and gRPC)
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop HTTP servers
	for port, srv := range m.servers {
		if err := srv.Stop(ctx); err != nil {
			lastErr = err
		}
		delete(m.servers, port)
	}

	// Stop TCP servers
	for port, srv := range m.tcpServers {
		if err := srv.Stop(ctx); err != nil {
			lastErr = err
		}
		delete(m.tcpServers, port)
	}

	// Stop SMTP servers
	for port, srv := range m.smtpServers {
		if err := srv.Stop(ctx); err != nil {
			lastErr = err
		}
		delete(m.smtpServers, port)
	}

	// Stop gRPC servers
	for port, srv := range m.grpcServers {
		if err := srv.Stop(ctx); err != nil {
			lastErr = err
		}
		delete(m.grpcServers, port)
	}

	return lastErr
}

// IsRunning checks if a server is running on the given port
func (m *Manager) IsRunning(port int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.servers[port]; exists {
		return true
	}
	if _, exists := m.tcpServers[port]; exists {
		return true
	}
	if _, exists := m.smtpServers[port]; exists {
		return true
	}
	if _, exists := m.grpcServers[port]; exists {
		return true
	}
	return false
}

// GetServer returns the HTTP server running on the given port
func (m *Manager) GetServer(port int) *Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.servers[port]
}

// GetTCPServer returns the TCP server running on the given port
func (m *Manager) GetTCPServer(port int) *TCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tcpServers[port]
}

// GetSMTPServer returns the SMTP server running on the given port
func (m *Manager) GetSMTPServer(port int) *SMTPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.smtpServers[port]
}

// GetGRPCServer returns the gRPC server running on the given port
func (m *Manager) GetGRPCServer(port int) *GRPCServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.grpcServers[port]
}

// GetImposterServer returns the imposter server interface for the given port
func (m *Manager) GetImposterServer(port int) ImposterServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if srv, exists := m.servers[port]; exists {
		return srv
	}
	if srv, exists := m.tcpServers[port]; exists {
		return srv
	}
	if srv, exists := m.smtpServers[port]; exists {
		return srv
	}
	if srv, exists := m.grpcServers[port]; exists {
		return srv
	}
	return nil
}

// Server represents an HTTP/HTTPS imposter server
type Server struct {
	imposter         *models.Imposter
	httpServer       *http.Server
	matcher          *Matcher
	proxyHandler     *ProxyHandler
	jsEngine         *JSEngine
	behaviorExecutor *BehaviorExecutor
	tlsConfig        *tls.Config
	useTLS           bool
	started          bool
	mu               sync.RWMutex
}

// NewServer creates a new imposter server (HTTP or HTTPS based on useTLS flag)
func NewServer(imp *models.Imposter, useTLS bool) (*Server, error) {
	jsEngine := NewJSEngine()
	srv := &Server{
		imposter:         imp,
		matcher:          NewMatcher(imp),
		proxyHandler:     NewProxyHandler(),
		jsEngine:         jsEngine,
		behaviorExecutor: NewBehaviorExecutor(jsEngine),
		useTLS:           useTLS,
	}

	srv.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", imp.Host, imp.Port),
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Configure TLS for HTTPS
	if useTLS {
		tlsConfig, err := srv.configureTLS(imp)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		srv.tlsConfig = tlsConfig
		srv.httpServer.TLSConfig = tlsConfig
	}

	return srv, nil
}

// configureTLS sets up TLS configuration for HTTPS servers
func (s *Server) configureTLS(imp *models.Imposter) (*tls.Config, error) {
	var cert tls.Certificate
	var err error

	// Use provided key and cert, or generate self-signed
	if imp.Key != "" && imp.Cert != "" {
		cert, err = tls.X509KeyPair([]byte(imp.Cert), []byte(imp.Key))
		if err != nil {
			return nil, fmt.Errorf("invalid certificate/key pair: %w", err)
		}
	} else {
		// Generate self-signed certificate
		cert, err = generateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}

		// Store generated cert in imposter for metadata extraction
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
		imp.Cert = string(certPEM)
		imp.ExtractCertMetadata()
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Configure mutual TLS if requested
	if imp.MutualAuth {
		tlsConfig.ClientAuth = tls.RequestClientCert
		if imp.RejectUnauthorized {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		// Add CA certificates for client verification
		if len(imp.Ca) > 0 {
			caPool := x509.NewCertPool()
			for _, ca := range imp.Ca {
				caPool.AppendCertsFromPEM([]byte(ca))
			}
			tlsConfig.ClientCAs = caPool
		}
	}

	// Configure ciphers if specified
	if imp.Ciphers != "" {
		// Go's tls package handles cipher suite selection automatically
		// Custom cipher configuration would require parsing the cipher string
	}

	return tlsConfig, nil
}

// generateSelfSignedCert creates a self-signed certificate for HTTPS
func generateSelfSignedCert() (tls.Certificate, error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"mountebank"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Encode private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Encode certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// Start starts the HTTP/HTTPS server
func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	s.started = true
	s.mu.Unlock()

	go func() {
		var err error
		if s.useTLS {
			// Start HTTPS server - use ListenAndServeTLS with empty strings
			// since TLSConfig is already set with the certificate
			err = s.httpServer.ListenAndServeTLS("", "")
		} else {
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			// Log error but don't crash
			fmt.Printf("imposter server error on port %d: %v\n", s.imposter.Port, err)
		}
	}()

	// Wait a bit for the server to start
	time.Sleep(50 * time.Millisecond)
	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.started = false
	return s.httpServer.Shutdown(ctx)
}

// ServeHTTP handles incoming requests to the imposter
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	portStr := strconv.Itoa(s.imposter.Port)

	// Record request metric
	metrics.RecordRequest(portStr, s.imposter.Protocol)

	// Handle CORS preflight requests if allowCORS is enabled
	if s.imposter.AllowCORS && r.Method == "OPTIONS" {
		if s.handleCORSPreflight(w, r) {
			return
		}
	}

	// Convert HTTP request to our Request model
	req, err := models.NewRequestFromHTTP(r)
	if err != nil {
		http.Error(w, "failed to parse request", http.StatusInternalServerError)
		return
	}

	// Record the request if configured
	s.mu.Lock()
	if s.imposter.RecordRequests {
		req.Timestamp = time.Now().Format(time.RFC3339)
		s.imposter.Requests = append(s.imposter.Requests, *req)
	}
	s.imposter.NumberOfRequests++
	s.mu.Unlock()

	// Find matching stub
	match := s.matcher.Match(req)

	// Record no-match if no stub matched
	if match.StubIndex < 0 {
		metrics.RecordNoMatch(portStr)
	}

	// Defer response duration recording
	defer func() {
		metrics.RecordResponseDuration(portStr, time.Since(startTime).Seconds())
	}()

	// Handle fault responses first (they hijack the connection)
	if match.Fault != "" {
		s.handleFault(w, match.Fault)
		return
	}

	var resp *models.IsResponse

	// Handle different response types
	if match.Proxy != nil {
		// Handle proxy response
		proxyResult, err := s.proxyHandler.Execute(req, match.Proxy, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
			return
		}

		resp = proxyResult.Response

		// Record generated stub if needed
		if proxyResult.ShouldRecord && proxyResult.GeneratedStub != nil {
			s.recordProxyStub(match, proxyResult.GeneratedStub)
		}
	} else if match.Inject != "" {
		// Handle inject response
		injResp, err := s.jsEngine.ExecuteResponse(match.Inject, req)
		if err != nil {
			http.Error(w, fmt.Sprintf("inject error: %v", err), http.StatusInternalServerError)
			return
		}
		resp = injResp
	} else {
		// Standard "is" response
		resp = match.Response
	}

	// Apply behaviors if any
	if len(match.Behaviors) > 0 && resp != nil {
		var err error
		resp, err = s.behaviorExecutor.Execute(req, resp, match.Behaviors)
		if err != nil {
			http.Error(w, fmt.Sprintf("behavior error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Merge with defaultResponse if configured
	if resp != nil && s.imposter.DefaultResponse != nil && s.imposter.DefaultResponse.Is != nil {
		resp = s.mergeWithDefault(resp, s.imposter.DefaultResponse.Is)
	}

	// Write response
	s.writeResponse(w, resp)
}

// mergeWithDefault merges a stub response with the default response
// Missing fields in the stub response are filled from defaultResponse
func (s *Server) mergeWithDefault(resp, defaultResp *models.IsResponse) *models.IsResponse {
	if defaultResp == nil {
		return resp
	}

	// Create a new response to avoid modifying the original
	merged := &models.IsResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
		Data:       resp.Data,
		Mode:       resp.Mode,
	}

	// Fill in missing fields from default
	// StatusCode is interface{} so we need to check if it's nil or zero
	if merged.StatusCode == nil && defaultResp.StatusCode != nil {
		merged.StatusCode = defaultResp.StatusCode
	} else if merged.StatusCode != nil {
		// Check if it's a zero value
		switch v := merged.StatusCode.(type) {
		case int:
			if v == 0 && defaultResp.StatusCode != nil {
				merged.StatusCode = defaultResp.StatusCode
			}
		case float64:
			if v == 0 && defaultResp.StatusCode != nil {
				merged.StatusCode = defaultResp.StatusCode
			}
		}
	}
	if merged.Body == nil && defaultResp.Body != nil {
		merged.Body = defaultResp.Body
	}
	if merged.Headers == nil && defaultResp.Headers != nil {
		merged.Headers = defaultResp.Headers
	}
	if merged.Data == "" && defaultResp.Data != "" {
		merged.Data = defaultResp.Data
	}
	if merged.Mode == "" && defaultResp.Mode != "" {
		merged.Mode = defaultResp.Mode
	}

	return merged
}

// handleFault handles fault injection responses
func (s *Server) handleFault(w http.ResponseWriter, fault string) {
	// Get the underlying connection using Hijacker
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		// If we can't hijack, just return an empty response
		w.WriteHeader(http.StatusOK)
		return
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch fault {
	case models.FaultConnectionResetByPeer:
		// Immediately close the connection with RST
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetLinger(0) // Send RST instead of FIN
		}
		conn.Close()

	case models.FaultRandomDataThenClose:
		// Write random garbage data then close
		garbage := make([]byte, 32)
		for i := range garbage {
			garbage[i] = byte(i * 17 % 256) // Pseudo-random but deterministic
		}
		conn.Write(garbage)
		conn.Close()

	default:
		// Unknown fault type - just close gracefully
		conn.Close()
	}
}

// recordProxyStub records a stub generated by proxy
func (s *Server) recordProxyStub(match *MatchResult, newStub *models.Stub) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get proxy mode
	mode := "proxyOnce"
	if match.Proxy != nil && match.Proxy.Mode != "" {
		mode = match.Proxy.Mode
	}

	switch mode {
	case "proxyOnce":
		// Insert new stub before the proxy stub
		if match.StubIndex >= 0 && match.StubIndex < len(s.imposter.Stubs) {
			// Insert at the current position
			stubs := make([]models.Stub, 0, len(s.imposter.Stubs)+1)
			stubs = append(stubs, s.imposter.Stubs[:match.StubIndex]...)
			stubs = append(stubs, *newStub)
			stubs = append(stubs, s.imposter.Stubs[match.StubIndex:]...)
			s.imposter.Stubs = stubs
		} else {
			// Append to the beginning
			s.imposter.Stubs = append([]models.Stub{*newStub}, s.imposter.Stubs...)
		}
	case "proxyAlways":
		// Insert at the end of the stub list (before proxy stub is still matched)
		if match.StubIndex >= 0 && match.StubIndex < len(s.imposter.Stubs) {
			stubs := make([]models.Stub, 0, len(s.imposter.Stubs)+1)
			stubs = append(stubs, s.imposter.Stubs[:match.StubIndex]...)
			stubs = append(stubs, *newStub)
			stubs = append(stubs, s.imposter.Stubs[match.StubIndex:]...)
			s.imposter.Stubs = stubs
		} else {
			s.imposter.Stubs = append(s.imposter.Stubs, *newStub)
		}
	}

	// Update matcher with new stubs
	s.matcher = NewMatcher(s.imposter)
}

// writeResponse writes the response to the HTTP response writer
func (s *Server) writeResponse(w http.ResponseWriter, resp *models.IsResponse) {
	// Set default status code
	statusCode := 200
	if resp != nil && resp.StatusCode != nil {
		switch v := resp.StatusCode.(type) {
		case int:
			if v != 0 {
				statusCode = v
			}
		case float64:
			if v != 0 {
				statusCode = int(v)
			}
		case string:
			// If it's still a string token (not replaced by copy behavior), try to parse it
			if code, err := strconv.Atoi(v); err == nil && code != 0 {
				statusCode = code
			}
		}
	}

	// Set headers
	if resp != nil && resp.Headers != nil {
		for k, v := range resp.Headers {
			switch val := v.(type) {
			case string:
				w.Header().Set(k, val)
			case []interface{}:
				// Multi-value header (e.g., Set-Cookie)
				for _, item := range val {
					if str, ok := item.(string); ok {
						w.Header().Add(k, str)
					}
				}
			case []string:
				for _, str := range val {
					w.Header().Add(k, str)
				}
			}
		}
	}

	// Set content-type if not set and we have a body
	if w.Header().Get("Content-Type") == "" && resp != nil && resp.Body != nil {
		// Default to text/plain for string bodies, application/json for objects
		switch resp.Body.(type) {
		case string, []byte:
			w.Header().Set("Content-Type", "text/plain")
		default:
			w.Header().Set("Content-Type", "application/json")
		}
	}

	w.WriteHeader(statusCode)

	// Write body
	if resp != nil && resp.Body != nil {
		// Check if binary mode - decode base64
		if resp.Mode == "binary" {
			bodyStr, ok := resp.Body.(string)
			if ok {
				decoded, err := base64.StdEncoding.DecodeString(bodyStr)
				if err == nil {
					w.Write(decoded)
					return
				}
			}
		}

		switch body := resp.Body.(type) {
		case string:
			w.Write([]byte(body))
		case []byte:
			w.Write(body)
		default:
			// Try to marshal as JSON
			if jsonBody, err := models.MarshalBody(body); err == nil {
				w.Write(jsonBody)
			}
		}
	}
}

// handleCORSPreflight handles CORS preflight requests when allowCORS is enabled
// Returns true if this was a valid preflight request that was handled
func (s *Server) handleCORSPreflight(w http.ResponseWriter, r *http.Request) bool {
	// Check for required preflight headers
	origin := r.Header.Get("Origin")
	requestMethod := r.Header.Get("Access-Control-Request-Method")

	// If missing required preflight headers, this is not a valid preflight
	if origin == "" || requestMethod == "" {
		return false
	}

	// Set CORS response headers
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", requestMethod)

	// Echo back requested headers if any
	requestHeaders := r.Header.Get("Access-Control-Request-Headers")
	if requestHeaders != "" {
		w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
	}

	w.WriteHeader(http.StatusOK)
	return true
}

// GetImposter returns the imposter configuration
func (s *Server) GetImposter() *models.Imposter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.imposter
}

// ResetRequestCount resets the request counter and clears recorded requests
func (s *Server) ResetRequestCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imposter.NumberOfRequests = 0
	s.imposter.Requests = nil
}

// UpdateStubs updates the stubs for this imposter
func (s *Server) UpdateStubs(stubs []models.Stub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imposter.Stubs = stubs
	s.matcher = NewMatcher(s.imposter)
}
