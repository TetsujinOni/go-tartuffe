package models

// SMTPRequest represents a captured SMTP email request
type SMTPRequest struct {
	RequestFrom  string           `json:"requestFrom,omitempty"`  // Client socket address
	IP           string           `json:"ip,omitempty"`           // Client IP
	EnvelopeFrom string           `json:"envelopeFrom,omitempty"` // MAIL FROM address
	EnvelopeTo   []string         `json:"envelopeTo,omitempty"`   // RCPT TO addresses
	From         *EmailAddress    `json:"from,omitempty"`         // From header
	To           []EmailAddress   `json:"to"`                     // To header recipients (always include, even if empty)
	Cc           []EmailAddress   `json:"cc"`                     // CC recipients (always include, even if empty)
	Bcc          []EmailAddress   `json:"bcc"`                    // BCC recipients (always include, even if empty)
	Subject      string           `json:"subject,omitempty"`      // Email subject
	Priority     string           `json:"priority,omitempty"`     // Message priority
	References   []string         `json:"references"`             // Email references (always include, even if empty)
	InReplyTo    []string         `json:"inReplyTo"`              // In-Reply-To values (always include, even if empty)
	Text         string           `json:"text,omitempty"`         // Plain text body
	Html         string           `json:"html,omitempty"`         // HTML body
	Attachments  []SMTPAttachment `json:"attachments"`            // Email attachments (always include, even if empty)
	Timestamp    string           `json:"timestamp,omitempty"`    // When received
}

// EmailAddress represents an email address with optional name
type EmailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

// SMTPAttachment represents an email attachment
type SMTPAttachment struct {
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Size        int    `json:"size,omitempty"`
	Content     string `json:"content,omitempty"` // Base64 encoded
}

// ToMap converts SMTPRequest to a map for predicate matching
func (r *SMTPRequest) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	if r.RequestFrom != "" {
		result["requestFrom"] = r.RequestFrom
	}
	if r.IP != "" {
		result["ip"] = r.IP
	}
	if r.EnvelopeFrom != "" {
		result["envelopeFrom"] = r.EnvelopeFrom
	}
	if len(r.EnvelopeTo) > 0 {
		result["envelopeTo"] = r.EnvelopeTo
	}
	if r.From != nil {
		result["from"] = r.From
	}
	if len(r.To) > 0 {
		result["to"] = r.To
	}
	if len(r.Cc) > 0 {
		result["cc"] = r.Cc
	}
	if len(r.Bcc) > 0 {
		result["bcc"] = r.Bcc
	}
	if r.Subject != "" {
		result["subject"] = r.Subject
	}
	if r.Priority != "" {
		result["priority"] = r.Priority
	}
	if len(r.References) > 0 {
		result["references"] = r.References
	}
	if len(r.InReplyTo) > 0 {
		result["inReplyTo"] = r.InReplyTo
	}
	if r.Text != "" {
		result["text"] = r.Text
	}
	if r.Html != "" {
		result["html"] = r.Html
	}
	if len(r.Attachments) > 0 {
		result["attachments"] = r.Attachments
	}

	return result
}
