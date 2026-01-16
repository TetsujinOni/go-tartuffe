package imposter

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// TestSMTPRequestFormat tests SMTP request recording format
func TestSMTPRequestFormat(t *testing.T) {
	port := 9300

	imp := &models.Imposter{
		Protocol:       "smtp",
		Port:           port,
		RecordRequests: true,
	}

	srv, err := NewSMTPServer(imp)
	if err != nil {
		t.Fatalf("NewSMTPServer() error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Send a simple email via SMTP
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Read greeting
	reader.ReadString('\n')

	// Send EHLO
	writer.WriteString("EHLO client.example.com\r\n")
	writer.Flush()
	// Read multi-line response
	for {
		line, _ := reader.ReadString('\n')
		if len(line) >= 4 && line[3] == ' ' {
			break
		}
	}

	// Send MAIL FROM
	writer.WriteString("MAIL FROM:<sender@example.com>\r\n")
	writer.Flush()
	reader.ReadString('\n')

	// Send RCPT TO
	writer.WriteString("RCPT TO:<recipient@example.com>\r\n")
	writer.Flush()
	reader.ReadString('\n')

	// Send DATA
	writer.WriteString("DATA\r\n")
	writer.Flush()
	reader.ReadString('\n')

	// Send email headers and body
	email := `From: "Sender Name" <sender@example.com>
To: "Recipient Name" <recipient@example.com>
Subject: Test Email
Content-Type: text/plain

This is a test email body.
`
	writer.WriteString(email)
	writer.WriteString("\r\n.\r\n")
	writer.Flush()
	reader.ReadString('\n')

	// Send QUIT
	writer.WriteString("QUIT\r\n")
	writer.Flush()

	// Give server time to process
	time.Sleep(100 * time.Millisecond)

	// Verify request was recorded
	storedImp := srv.GetImposter()
	if len(storedImp.SMTPRequests) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(storedImp.SMTPRequests))
	}

	req := storedImp.SMTPRequests[0]

	// Verify envelope addresses
	if req.EnvelopeFrom != "sender@example.com" {
		t.Errorf("EnvelopeFrom = %q, want %q", req.EnvelopeFrom, "sender@example.com")
	}

	if len(req.EnvelopeTo) != 1 || req.EnvelopeTo[0] != "recipient@example.com" {
		t.Errorf("EnvelopeTo = %v, want [recipient@example.com]", req.EnvelopeTo)
	}

	// Verify From header parsing
	if req.From == nil {
		t.Fatal("From is nil")
	}
	if req.From.Address != "sender@example.com" {
		t.Errorf("From.Address = %q, want %q", req.From.Address, "sender@example.com")
	}
	if req.From.Name != "Sender Name" {
		t.Errorf("From.Name = %q, want %q", req.From.Name, "Sender Name")
	}

	// Verify To header parsing
	if len(req.To) != 1 {
		t.Fatalf("To length = %d, want 1", len(req.To))
	}
	if req.To[0].Address != "recipient@example.com" {
		t.Errorf("To[0].Address = %q, want %q", req.To[0].Address, "recipient@example.com")
	}
	if req.To[0].Name != "Recipient Name" {
		t.Errorf("To[0].Name = %q, want %q", req.To[0].Name, "Recipient Name")
	}

	// Verify Subject
	if req.Subject != "Test Email" {
		t.Errorf("Subject = %q, want %q", req.Subject, "Test Email")
	}

	// Verify body
	if req.Text != "This is a test email body." {
		t.Errorf("Text = %q, want %q", req.Text, "This is a test email body.")
	}

	// Verify empty fields are initialized (NOT nil)
	// This is what mountebank expects - empty arrays, not null
	if req.Cc == nil {
		t.Error("Cc should be empty array, not nil")
	}
	if req.Bcc == nil {
		t.Error("Bcc should be empty array, not nil")
	}
	if req.References == nil {
		t.Error("References should be empty array, not nil")
	}
	if req.InReplyTo == nil {
		t.Error("InReplyTo should be empty array, not nil")
	}
	if req.Attachments == nil {
		t.Error("Attachments should be empty array, not nil")
	}

	// Test JSON marshaling - verify empty arrays appear in JSON
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// These fields should exist as empty arrays in JSON
	checkEmptyArray := func(field string) {
		val, exists := jsonMap[field]
		if !exists {
			t.Errorf("JSON missing field %q (should be empty array)", field)
			return
		}
		arr, ok := val.([]interface{})
		if !ok {
			t.Errorf("JSON field %q is not an array: %T", field, val)
			return
		}
		if len(arr) != 0 {
			t.Errorf("JSON field %q should be empty array, got length %d", field, len(arr))
		}
	}

	checkEmptyArray("cc")
	checkEmptyArray("bcc")
	checkEmptyArray("references")
	checkEmptyArray("inReplyTo")
	checkEmptyArray("attachments")
}

// TestSMTPWithCcAndBcc tests email with CC and BCC
func TestSMTPWithCcAndBcc(t *testing.T) {
	port := 9301

	imp := &models.Imposter{
		Protocol:       "smtp",
		Port:           port,
		RecordRequests: true,
	}

	srv, err := NewSMTPServer(imp)
	if err != nil {
		t.Fatalf("NewSMTPServer() error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Read greeting
	reader.ReadString('\n')

	// SMTP conversation
	writer.WriteString("EHLO client\r\n")
	writer.Flush()
	for {
		line, _ := reader.ReadString('\n')
		if len(line) >= 4 && line[3] == ' ' {
			break
		}
	}

	writer.WriteString("MAIL FROM:<sender@example.com>\r\n")
	writer.Flush()
	reader.ReadString('\n')

	writer.WriteString("RCPT TO:<to@example.com>\r\n")
	writer.Flush()
	reader.ReadString('\n')

	writer.WriteString("DATA\r\n")
	writer.Flush()
	reader.ReadString('\n')

	email := `From: sender@example.com
To: to@example.com
Cc: "CC Name" <cc@example.com>
Bcc: "BCC Name" <bcc@example.com>
Subject: Test with CC and BCC

Body text.
`
	writer.WriteString(email)
	writer.WriteString("\r\n.\r\n")
	writer.Flush()
	reader.ReadString('\n')

	writer.WriteString("QUIT\r\n")
	writer.Flush()

	time.Sleep(100 * time.Millisecond)

	storedImp := srv.GetImposter()
	if len(storedImp.SMTPRequests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(storedImp.SMTPRequests))
	}

	req := storedImp.SMTPRequests[0]

	// Verify CC
	if len(req.Cc) != 1 {
		t.Fatalf("Cc length = %d, want 1", len(req.Cc))
	}
	if req.Cc[0].Address != "cc@example.com" {
		t.Errorf("Cc[0].Address = %q, want %q", req.Cc[0].Address, "cc@example.com")
	}
	if req.Cc[0].Name != "CC Name" {
		t.Errorf("Cc[0].Name = %q, want %q", req.Cc[0].Name, "CC Name")
	}

	// Verify BCC
	if len(req.Bcc) != 1 {
		t.Fatalf("Bcc length = %d, want 1", len(req.Bcc))
	}
	if req.Bcc[0].Address != "bcc@example.com" {
		t.Errorf("Bcc[0].Address = %q, want %q", req.Bcc[0].Address, "bcc@example.com")
	}
	if req.Bcc[0].Name != "BCC Name" {
		t.Errorf("Bcc[0].Name = %q, want %q", req.Bcc[0].Name, "BCC Name")
	}
}

// TestSMTPHtmlEmail tests HTML email parsing
func TestSMTPHtmlEmail(t *testing.T) {
	port := 9302

	imp := &models.Imposter{
		Protocol:       "smtp",
		Port:           port,
		RecordRequests: true,
	}

	srv, err := NewSMTPServer(imp)
	if err != nil {
		t.Fatalf("NewSMTPServer() error = %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	reader.ReadString('\n')

	writer.WriteString("EHLO client\r\n")
	writer.Flush()
	for {
		line, _ := reader.ReadString('\n')
		if len(line) >= 4 && line[3] == ' ' {
			break
		}
	}

	writer.WriteString("MAIL FROM:<sender@example.com>\r\n")
	writer.Flush()
	reader.ReadString('\n')

	writer.WriteString("RCPT TO:<to@example.com>\r\n")
	writer.Flush()
	reader.ReadString('\n')

	writer.WriteString("DATA\r\n")
	writer.Flush()
	reader.ReadString('\n')

	email := `From: sender@example.com
To: to@example.com
Subject: HTML Email
Content-Type: text/html

<html><body><h1>Hello</h1></body></html>
`
	writer.WriteString(email)
	writer.WriteString("\r\n.\r\n")
	writer.Flush()
	reader.ReadString('\n')

	writer.WriteString("QUIT\r\n")
	writer.Flush()

	time.Sleep(100 * time.Millisecond)

	storedImp := srv.GetImposter()
	req := storedImp.SMTPRequests[0]

	// Verify HTML body
	if req.Html != "<html><body><h1>Hello</h1></body></html>" {
		t.Errorf("Html = %q, want HTML content", req.Html)
	}

	// Text should be empty
	if req.Text != "" {
		t.Errorf("Text should be empty for HTML-only email, got %q", req.Text)
	}
}
