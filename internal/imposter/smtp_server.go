package imposter

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// SMTPServer represents an SMTP imposter server
type SMTPServer struct {
	imposter *models.Imposter
	listener net.Listener
	matcher  *SMTPMatcher
	started  bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
	quit     chan struct{}
}

// NewSMTPServer creates a new SMTP imposter server
func NewSMTPServer(imp *models.Imposter) (*SMTPServer, error) {
	return &SMTPServer{
		imposter: imp,
		matcher:  NewSMTPMatcher(imp),
		quit:     make(chan struct{}),
	}, nil
}

// Start starts the SMTP server
func (s *SMTPServer) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.imposter.Host, s.imposter.Port))
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to start SMTP server on %s:%d: %w", s.imposter.Host, s.imposter.Port, err)
	}

	s.listener = listener
	s.started = true
	s.mu.Unlock()

	go s.acceptLoop()

	// Wait a bit for the server to start
	time.Sleep(50 * time.Millisecond)
	return nil
}

// Stop stops the SMTP server
func (s *SMTPServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	close(s.quit)
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all connections to finish
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

// GetImposter returns the imposter configuration
func (s *SMTPServer) GetImposter() *models.Imposter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.imposter
}

// UpdateStubs updates the stubs for this imposter
func (s *SMTPServer) UpdateStubs(stubs []models.Stub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imposter.Stubs = stubs
	s.matcher = NewSMTPMatcher(s.imposter)
}

// acceptLoop accepts incoming connections
func (s *SMTPServer) acceptLoop() {
	for {
		select {
		case <-s.quit:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				continue
			}
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

// handleConnection handles a single SMTP connection
func (s *SMTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set read/write timeout
	conn.SetDeadline(time.Now().Add(60 * time.Second))

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send greeting
	s.writeLine(writer, "220 localhost ESMTP mountebank")

	// Session state
	var mailFrom string
	var rcptTo []string
	var dataMode bool
	var dataBuffer strings.Builder

	clientAddr := conn.RemoteAddr().String()

	for {
		// Reset deadline on each command
		conn.SetDeadline(time.Now().Add(60 * time.Second))

		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				// Log error but continue
			}
			return
		}

		line = strings.TrimRight(line, "\r\n")

		if dataMode {
			// In DATA mode, collect email content until single "."
			if line == "." {
				dataMode = false

				// Parse and record the email
				smtpReq := s.parseEmail(clientAddr, mailFrom, rcptTo, dataBuffer.String())

				// Record request if configured
				s.mu.Lock()
				if s.imposter.RecordRequests {
					smtpReq.Timestamp = time.Now().Format(time.RFC3339)
					s.imposter.SMTPRequests = append(s.imposter.SMTPRequests, *smtpReq)
				}
				s.imposter.NumberOfRequests++
				s.mu.Unlock()

				// Match against stubs (for logging/tracking purposes)
				s.matcher.Match(smtpReq)

				s.writeLine(writer, "250 OK message queued")

				// Reset state for next message
				mailFrom = ""
				rcptTo = nil
				dataBuffer.Reset()
			} else {
				// Handle dot-stuffing (lines starting with . are escaped)
				if strings.HasPrefix(line, ".") {
					line = line[1:]
				}
				dataBuffer.WriteString(line)
				dataBuffer.WriteString("\r\n")
			}
			continue
		}

		// Parse command
		cmd := strings.ToUpper(line)
		if len(cmd) >= 4 {
			cmd = cmd[:4]
		}

		switch cmd {
		case "HELO":
			s.writeLine(writer, "250 localhost Hello")

		case "EHLO":
			// Extended SMTP greeting
			s.writeLine(writer, "250-localhost Hello")
			s.writeLine(writer, "250-SIZE 10485760")
			s.writeLine(writer, "250-8BITMIME")
			s.writeLine(writer, "250 OK")

		case "MAIL":
			// MAIL FROM:<address>
			mailFrom = extractAddress(line)
			if mailFrom != "" {
				s.writeLine(writer, "250 OK")
			} else {
				s.writeLine(writer, "501 Syntax error in parameters")
			}

		case "RCPT":
			// RCPT TO:<address>
			addr := extractAddress(line)
			if addr != "" {
				rcptTo = append(rcptTo, addr)
				s.writeLine(writer, "250 OK")
			} else {
				s.writeLine(writer, "501 Syntax error in parameters")
			}

		case "DATA":
			if mailFrom == "" || len(rcptTo) == 0 {
				s.writeLine(writer, "503 Bad sequence of commands")
			} else {
				s.writeLine(writer, "354 Start mail input; end with <CRLF>.<CRLF>")
				dataMode = true
			}

		case "RSET":
			mailFrom = ""
			rcptTo = nil
			dataBuffer.Reset()
			s.writeLine(writer, "250 OK")

		case "NOOP":
			s.writeLine(writer, "250 OK")

		case "QUIT":
			s.writeLine(writer, "221 Bye")
			return

		case "VRFY":
			s.writeLine(writer, "252 Cannot verify user")

		case "AUTH":
			// Accept any authentication
			s.writeLine(writer, "235 Authentication successful")

		default:
			s.writeLine(writer, "500 Command not recognized")
		}
	}
}

// writeLine writes a line to the SMTP connection
func (s *SMTPServer) writeLine(writer *bufio.Writer, line string) {
	writer.WriteString(line + "\r\n")
	writer.Flush()
}

// extractAddress extracts email address from MAIL FROM or RCPT TO command
func extractAddress(line string) string {
	// Find < and >
	start := strings.Index(line, "<")
	end := strings.Index(line, ">")

	if start >= 0 && end > start {
		return line[start+1 : end]
	}

	// Try to extract without angle brackets
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

// parseEmail parses raw email data into SMTPRequest
func (s *SMTPServer) parseEmail(clientAddr, mailFrom string, rcptTo []string, data string) *models.SMTPRequest {
	req := &models.SMTPRequest{
		RequestFrom:  clientAddr,
		IP:           extractIP(clientAddr),
		EnvelopeFrom: mailFrom,
		EnvelopeTo:   rcptTo,
		Priority:     "normal",
	}

	// Parse headers and body
	headers, body := parseEmailParts(data)

	// Extract From header
	if from := headers["from"]; from != "" {
		req.From = parseEmailAddress(from)
	}

	// Extract To header
	if to := headers["to"]; to != "" {
		req.To = parseEmailAddresses(to)
	}

	// Extract CC
	if cc := headers["cc"]; cc != "" {
		req.Cc = parseEmailAddresses(cc)
	}

	// Extract BCC (rarely in headers, but check)
	if bcc := headers["bcc"]; bcc != "" {
		req.Bcc = parseEmailAddresses(bcc)
	}

	// Extract Subject
	req.Subject = headers["subject"]

	// Extract Priority
	if priority := headers["x-priority"]; priority != "" {
		switch priority {
		case "1", "2":
			req.Priority = "high"
		case "4", "5":
			req.Priority = "low"
		default:
			req.Priority = "normal"
		}
	}

	// Extract References
	if refs := headers["references"]; refs != "" {
		req.References = strings.Fields(refs)
	}

	// Extract In-Reply-To
	if inReplyTo := headers["in-reply-to"]; inReplyTo != "" {
		req.InReplyTo = strings.Fields(inReplyTo)
	}

	// Determine content type and parse body
	contentType := headers["content-type"]
	if strings.Contains(strings.ToLower(contentType), "multipart") {
		// Parse multipart message
		boundary := extractBoundary(contentType)
		if boundary != "" {
			parts := parseMultipart(body, boundary)
			for _, part := range parts {
				partHeaders, partBody := parseEmailParts(part)
				partContentType := strings.ToLower(partHeaders["content-type"])

				if strings.Contains(partContentType, "text/plain") {
					req.Text = strings.TrimSpace(partBody)
				} else if strings.Contains(partContentType, "text/html") {
					req.Html = strings.TrimSpace(partBody)
				} else if partHeaders["content-disposition"] != "" {
					// Attachment
					attachment := models.SMTPAttachment{
						ContentType: partContentType,
						Content:     base64.StdEncoding.EncodeToString([]byte(partBody)),
						Size:        len(partBody),
					}
					if filename := extractFilename(partHeaders["content-disposition"]); filename != "" {
						attachment.Filename = filename
					}
					req.Attachments = append(req.Attachments, attachment)
				}
			}
		}
	} else if strings.Contains(strings.ToLower(contentType), "text/html") {
		req.Html = strings.TrimSpace(body)
	} else {
		// Default to plain text
		req.Text = strings.TrimSpace(body)
	}

	return req
}

// parseEmailParts splits email data into headers and body
func parseEmailParts(data string) (map[string]string, string) {
	headers := make(map[string]string)

	// Split by blank line (separates headers from body)
	parts := strings.SplitN(data, "\r\n\r\n", 2)
	if len(parts) < 2 {
		// Try with just \n\n
		parts = strings.SplitN(data, "\n\n", 2)
	}

	headerSection := ""
	body := ""
	if len(parts) >= 1 {
		headerSection = parts[0]
	}
	if len(parts) >= 2 {
		body = parts[1]
	}

	// Parse headers
	var currentKey string
	var currentValue strings.Builder

	for _, line := range strings.Split(headerSection, "\n") {
		line = strings.TrimRight(line, "\r")

		if line == "" {
			continue
		}

		// Check for continuation (starts with whitespace)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if currentKey != "" {
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSpace(line))
			}
			continue
		}

		// Save previous header
		if currentKey != "" {
			headers[strings.ToLower(currentKey)] = currentValue.String()
		}

		// Parse new header
		colonIdx := strings.Index(line, ":")
		if colonIdx > 0 {
			currentKey = line[:colonIdx]
			currentValue.Reset()
			currentValue.WriteString(strings.TrimSpace(line[colonIdx+1:]))
		}
	}

	// Save last header
	if currentKey != "" {
		headers[strings.ToLower(currentKey)] = currentValue.String()
	}

	return headers, body
}

// parseEmailAddress parses a single email address
func parseEmailAddress(addr string) *models.EmailAddress {
	addr = strings.TrimSpace(addr)

	// Check for "Name <email>" format
	if start := strings.Index(addr, "<"); start >= 0 {
		if end := strings.Index(addr, ">"); end > start {
			name := strings.TrimSpace(addr[:start])
			// Remove quotes from name
			name = strings.Trim(name, "\"")
			return &models.EmailAddress{
				Address: addr[start+1 : end],
				Name:    name,
			}
		}
	}

	return &models.EmailAddress{Address: addr}
}

// parseEmailAddresses parses multiple email addresses
func parseEmailAddresses(addrs string) []models.EmailAddress {
	var result []models.EmailAddress

	// Split by comma, but be careful of commas inside quotes
	parts := splitAddresses(addrs)
	for _, part := range parts {
		if addr := parseEmailAddress(part); addr != nil {
			result = append(result, *addr)
		}
	}

	return result
}

// splitAddresses splits address list by comma, respecting quotes
func splitAddresses(s string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	inAngle := false

	for _, ch := range s {
		switch ch {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(ch)
		case '<':
			inAngle = true
			current.WriteRune(ch)
		case '>':
			inAngle = false
			current.WriteRune(ch)
		case ',':
			if !inQuotes && !inAngle {
				if current.Len() > 0 {
					result = append(result, strings.TrimSpace(current.String()))
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

// extractBoundary extracts boundary from Content-Type header
func extractBoundary(contentType string) string {
	parts := strings.Split(contentType, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "boundary=") {
			boundary := part[9:]
			return strings.Trim(boundary, "\"")
		}
	}
	return ""
}

// parseMultipart parses multipart email body
func parseMultipart(body, boundary string) []string {
	var parts []string
	delimiter := "--" + boundary

	sections := strings.Split(body, delimiter)
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" || section == "--" {
			continue
		}
		// Remove trailing -- from last part
		section = strings.TrimSuffix(section, "--")
		section = strings.TrimSpace(section)
		if section != "" {
			parts = append(parts, section)
		}
	}

	return parts
}

// extractFilename extracts filename from Content-Disposition header
func extractFilename(disposition string) string {
	parts := strings.Split(disposition, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "filename=") {
			filename := part[9:]
			return strings.Trim(filename, "\"")
		}
	}
	return ""
}

// extractIP extracts IP from address:port format
func extractIP(addr string) string {
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[:idx]
	}
	return addr
}

// SMTPMatcher handles request matching for SMTP
type SMTPMatcher struct {
	imposter *models.Imposter
}

// NewSMTPMatcher creates a new SMTP matcher
func NewSMTPMatcher(imp *models.Imposter) *SMTPMatcher {
	return &SMTPMatcher{imposter: imp}
}

// Match finds a matching stub for an SMTP request
func (m *SMTPMatcher) Match(req *models.SMTPRequest) *SMTPMatchResult {
	for i := range m.imposter.Stubs {
		stub := &m.imposter.Stubs[i]
		if m.matchesAllPredicates(stub, req) {
			return &SMTPMatchResult{
				Stub:      stub,
				StubIndex: i,
				Matched:   true,
			}
		}
	}

	return &SMTPMatchResult{Matched: false}
}

// SMTPMatchResult contains the result of matching
type SMTPMatchResult struct {
	Stub      *models.Stub
	StubIndex int
	Matched   bool
}

// matchesAllPredicates checks if request matches all predicates
func (m *SMTPMatcher) matchesAllPredicates(stub *models.Stub, req *models.SMTPRequest) bool {
	if len(stub.Predicates) == 0 {
		return true
	}

	reqMap := req.ToMap()

	for _, pred := range stub.Predicates {
		if !m.evaluatePredicate(&pred, reqMap) {
			return false
		}
	}

	return true
}

// evaluatePredicate evaluates a single predicate
func (m *SMTPMatcher) evaluatePredicate(pred *models.Predicate, reqMap map[string]interface{}) bool {
	// Handle logical operators
	if pred.And != nil {
		for _, p := range pred.And {
			if !m.evaluatePredicate(&p, reqMap) {
				return false
			}
		}
		return true
	}

	if pred.Or != nil {
		for _, p := range pred.Or {
			if m.evaluatePredicate(&p, reqMap) {
				return true
			}
		}
		return false
	}

	if pred.Not != nil {
		return !m.evaluatePredicate(pred.Not, reqMap)
	}

	// Handle comparison operators
	if pred.Equals != nil {
		return m.evaluateEquals(pred.Equals, reqMap, pred.CaseSensitive)
	}

	if pred.Contains != nil {
		return m.evaluateContains(pred.Contains, reqMap, pred.CaseSensitive)
	}

	if pred.StartsWith != nil {
		return m.evaluateStartsWith(pred.StartsWith, reqMap, pred.CaseSensitive)
	}

	if pred.EndsWith != nil {
		return m.evaluateEndsWith(pred.EndsWith, reqMap, pred.CaseSensitive)
	}

	if pred.Matches != nil {
		return m.evaluateMatches(pred.Matches, reqMap)
	}

	if pred.Exists != nil {
		return m.evaluateExists(pred.Exists, reqMap)
	}

	return true
}

// evaluateEquals checks field equality
func (m *SMTPMatcher) evaluateEquals(value interface{}, reqMap map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := reqMap[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if actualStr != expectedStr {
				return false
			}
		} else {
			if !strings.EqualFold(actualStr, expectedStr) {
				return false
			}
		}
	}

	return true
}

// evaluateContains checks if field contains value
func (m *SMTPMatcher) evaluateContains(value interface{}, reqMap map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := reqMap[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if !strings.Contains(actualStr, expectedStr) {
				return false
			}
		} else {
			if !strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr)) {
				return false
			}
		}
	}

	return true
}

// evaluateStartsWith checks if field starts with value
func (m *SMTPMatcher) evaluateStartsWith(value interface{}, reqMap map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := reqMap[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if !strings.HasPrefix(actualStr, expectedStr) {
				return false
			}
		} else {
			if !strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(expectedStr)) {
				return false
			}
		}
	}

	return true
}

// evaluateEndsWith checks if field ends with value
func (m *SMTPMatcher) evaluateEndsWith(value interface{}, reqMap map[string]interface{}, caseSensitive bool) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, expected := range predMap {
		actual, exists := reqMap[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		expectedStr := fmt.Sprintf("%v", expected)

		if caseSensitive {
			if !strings.HasSuffix(actualStr, expectedStr) {
				return false
			}
		} else {
			if !strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(expectedStr)) {
				return false
			}
		}
	}

	return true
}

// evaluateMatches checks if field matches regex
func (m *SMTPMatcher) evaluateMatches(value interface{}, reqMap map[string]interface{}) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, pattern := range predMap {
		actual, exists := reqMap[field]
		if !exists {
			return false
		}

		actualStr := fmt.Sprintf("%v", actual)
		patternStr := fmt.Sprintf("%v", pattern)

		matched, err := regexpMatch(patternStr, actualStr)
		if err != nil || !matched {
			return false
		}
	}

	return true
}

// evaluateExists checks if field exists
func (m *SMTPMatcher) evaluateExists(value interface{}, reqMap map[string]interface{}) bool {
	predMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	for field, shouldExist := range predMap {
		_, exists := reqMap[field]
		expected, _ := shouldExist.(bool)

		if exists != expected {
			return false
		}
	}

	return true
}

// regexpMatch helper for regex matching
func regexpMatch(pattern, s string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(s), nil
}
