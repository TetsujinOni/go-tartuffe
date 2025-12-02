package integration

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

// SMTP protocol tests

func TestSMTP_BasicServer(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "smtp",
		"port":           5800,
		"recordRequests": true,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Connect to SMTP server
	conn, err := net.Dial("tcp", "localhost:5800")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read greeting
	greeting, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read greeting: %v", err)
	}
	if !strings.HasPrefix(greeting, "220") {
		t.Errorf("expected 220 greeting, got '%s'", greeting)
	}

	// Send EHLO
	conn.Write([]byte("EHLO localhost\r\n"))
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("failed to read EHLO response: %v", err)
		}
		if strings.HasPrefix(line, "250 ") {
			break
		}
	}

	// Send QUIT
	conn.Write([]byte("QUIT\r\n"))
	quitResp, _ := reader.ReadString('\n')
	if !strings.HasPrefix(quitResp, "221") {
		t.Errorf("expected 221 bye, got '%s'", quitResp)
	}
}

func TestSMTP_RecordEmail(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "smtp",
		"port":           5801,
		"recordRequests": true,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Send an email
	err = sendTestEmail("localhost:5801", "sender@test.com", []string{"recipient@test.com"},
		"Subject: Test Email\r\nFrom: sender@test.com\r\nTo: recipient@test.com\r\n\r\nHello, this is a test email.")
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}

	// Check recorded requests
	getResp, body, err := get("/imposters/5801")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	smtpRequests, ok := body["smtpRequests"].([]interface{})
	if !ok {
		t.Fatal("expected smtpRequests array")
	}

	if len(smtpRequests) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(smtpRequests))
	}

	req := smtpRequests[0].(map[string]interface{})

	// Verify envelope
	if req["envelopeFrom"] != "sender@test.com" {
		t.Errorf("expected envelopeFrom 'sender@test.com', got '%v'", req["envelopeFrom"])
	}

	envelopeTo, ok := req["envelopeTo"].([]interface{})
	if !ok || len(envelopeTo) != 1 || envelopeTo[0] != "recipient@test.com" {
		t.Errorf("expected envelopeTo ['recipient@test.com'], got '%v'", req["envelopeTo"])
	}

	// Verify subject
	if req["subject"] != "Test Email" {
		t.Errorf("expected subject 'Test Email', got '%v'", req["subject"])
	}

	// Verify text body
	if req["text"] != "Hello, this is a test email." {
		t.Errorf("expected text 'Hello, this is a test email.', got '%v'", req["text"])
	}
}

func TestSMTP_MultipleRecipients(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "smtp",
		"port":           5802,
		"recordRequests": true,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Send email to multiple recipients
	err = sendTestEmail("localhost:5802", "sender@test.com",
		[]string{"alice@test.com", "bob@test.com", "charlie@test.com"},
		"Subject: Group Email\r\nFrom: sender@test.com\r\nTo: alice@test.com, bob@test.com\r\nCc: charlie@test.com\r\n\r\nHello everyone!")
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}

	// Check recorded requests
	getResp, body, err := get("/imposters/5802")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	smtpRequests := body["smtpRequests"].([]interface{})
	req := smtpRequests[0].(map[string]interface{})

	envelopeTo, ok := req["envelopeTo"].([]interface{})
	if !ok || len(envelopeTo) != 3 {
		t.Errorf("expected 3 envelope recipients, got %d", len(envelopeTo))
	}
}

func TestSMTP_RequestCount(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "smtp",
		"port":           5803,
		"recordRequests": true,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Send 3 emails
	for i := 0; i < 3; i++ {
		err = sendTestEmail("localhost:5803", "sender@test.com", []string{"recipient@test.com"},
			"Subject: Test\r\n\r\nBody")
		if err != nil {
			t.Fatalf("failed to send email %d: %v", i+1, err)
		}
	}

	// Check request count
	getResp, body, err := get("/imposters/5803")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	count, ok := body["numberOfRequests"].(float64)
	if !ok || int(count) != 3 {
		t.Errorf("expected numberOfRequests 3, got %v", body["numberOfRequests"])
	}

	smtpRequests, ok := body["smtpRequests"].([]interface{})
	if !ok || len(smtpRequests) != 3 {
		t.Errorf("expected 3 recorded requests, got %d", len(smtpRequests))
	}
}

func TestSMTP_HTMLEmail(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "smtp",
		"port":           5804,
		"recordRequests": true,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Send HTML email
	htmlContent := "Subject: HTML Email\r\nFrom: sender@test.com\r\nTo: recipient@test.com\r\nContent-Type: text/html\r\n\r\n<html><body><h1>Hello</h1></body></html>"
	err = sendTestEmail("localhost:5804", "sender@test.com", []string{"recipient@test.com"}, htmlContent)
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}

	// Check recorded requests
	getResp, body, err := get("/imposters/5804")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	smtpRequests := body["smtpRequests"].([]interface{})
	req := smtpRequests[0].(map[string]interface{})

	// HTML content should be in html field
	if req["html"] != "<html><body><h1>Hello</h1></body></html>" {
		t.Errorf("expected html body, got '%v'", req["html"])
	}
}

func TestSMTP_HELO(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "smtp",
		"port":     5805,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:5805")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read greeting
	reader.ReadString('\n')

	// Send HELO (not EHLO)
	conn.Write([]byte("HELO localhost\r\n"))
	heloResp, _ := reader.ReadString('\n')
	if !strings.HasPrefix(heloResp, "250") {
		t.Errorf("expected 250 response to HELO, got '%s'", heloResp)
	}
}

func TestSMTP_RSET(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "smtp",
		"port":           5806,
		"recordRequests": true,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:5806")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read greeting
	reader.ReadString('\n')

	// EHLO
	conn.Write([]byte("EHLO localhost\r\n"))
	for {
		line, _ := reader.ReadString('\n')
		if strings.HasPrefix(line, "250 ") {
			break
		}
	}

	// Start mail transaction
	conn.Write([]byte("MAIL FROM:<sender@test.com>\r\n"))
	reader.ReadString('\n')

	// Reset
	conn.Write([]byte("RSET\r\n"))
	rsetResp, _ := reader.ReadString('\n')
	if !strings.HasPrefix(rsetResp, "250") {
		t.Errorf("expected 250 response to RSET, got '%s'", rsetResp)
	}

	// Verify we can start a new transaction
	conn.Write([]byte("MAIL FROM:<newsender@test.com>\r\n"))
	mailResp, _ := reader.ReadString('\n')
	if !strings.HasPrefix(mailResp, "250") {
		t.Errorf("expected 250 response to MAIL FROM, got '%s'", mailResp)
	}
}

func TestSMTP_NOOP(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "smtp",
		"port":     5807,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:5807")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read greeting
	reader.ReadString('\n')

	// Send NOOP
	conn.Write([]byte("NOOP\r\n"))
	noopResp, _ := reader.ReadString('\n')
	if !strings.HasPrefix(noopResp, "250") {
		t.Errorf("expected 250 response to NOOP, got '%s'", noopResp)
	}
}

func TestSMTP_BadSequence(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol": "smtp",
		"port":     5808,
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:5808")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read greeting
	reader.ReadString('\n')

	// EHLO
	conn.Write([]byte("EHLO localhost\r\n"))
	for {
		line, _ := reader.ReadString('\n')
		if strings.HasPrefix(line, "250 ") {
			break
		}
	}

	// Try DATA without MAIL FROM or RCPT TO
	conn.Write([]byte("DATA\r\n"))
	dataResp, _ := reader.ReadString('\n')
	if !strings.HasPrefix(dataResp, "503") {
		t.Errorf("expected 503 bad sequence, got '%s'", dataResp)
	}
}

func TestSMTP_PredicateMatching(t *testing.T) {
	defer cleanup(t)

	resp, _, err := post("/imposters", map[string]interface{}{
		"protocol":       "smtp",
		"port":           5809,
		"recordRequests": true,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{"contains": map[string]interface{}{"subject": "urgent"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create imposter: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// Send email with "urgent" in subject
	err = sendTestEmail("localhost:5809", "sender@test.com", []string{"recipient@test.com"},
		"Subject: urgent request\r\n\r\nPlease respond ASAP")
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}

	// The email should be recorded
	getResp, body, err := get("/imposters/5809")
	if err != nil {
		t.Fatalf("failed to get imposter: %v", err)
	}
	if getResp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	smtpRequests := body["smtpRequests"].([]interface{})
	if len(smtpRequests) != 1 {
		t.Errorf("expected 1 recorded request, got %d", len(smtpRequests))
	}
}

// sendTestEmail sends an email via SMTP
func sendTestEmail(addr, from string, to []string, data string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Read greeting
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	// EHLO
	conn.Write([]byte("EHLO localhost\r\n"))
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "250 ") {
			break
		}
	}

	// MAIL FROM
	conn.Write([]byte("MAIL FROM:<" + from + ">\r\n"))
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	// RCPT TO (for each recipient)
	for _, rcpt := range to {
		conn.Write([]byte("RCPT TO:<" + rcpt + ">\r\n"))
		_, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
	}

	// DATA
	conn.Write([]byte("DATA\r\n"))
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	// Send email content
	conn.Write([]byte(data + "\r\n.\r\n"))
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	// QUIT
	conn.Write([]byte("QUIT\r\n"))
	reader.ReadString('\n')

	return nil
}
