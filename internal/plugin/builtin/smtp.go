package builtin

import (
	"context"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

// SMTPProtocol wraps the existing SMTP server as a plugin
type SMTPProtocol struct{}

// NewSMTPProtocol creates a new SMTP protocol plugin
func NewSMTPProtocol() *SMTPProtocol {
	return &SMTPProtocol{}
}

// Name returns the protocol name
func (p *SMTPProtocol) Name() string {
	return "smtp"
}

// CreateServer creates a new SMTP server
func (p *SMTPProtocol) CreateServer(imp *models.Imposter, callback protocol.CallbackClient) (protocol.ProtocolServer, error) {
	srv, err := imposter.NewSMTPServer(imp)
	if err != nil {
		return nil, err
	}
	return &SMTPServerAdapter{SMTPServer: srv, imp: imp}, nil
}

// ValidateConfig validates the imposter configuration
func (p *SMTPProtocol) ValidateConfig(imp *models.Imposter) error {
	return nil
}

// DefaultPort returns the default port (0 = no default)
func (p *SMTPProtocol) DefaultPort() int {
	return 25
}

// SMTPServerAdapter adapts the existing SMTPServer to ProtocolServer interface
type SMTPServerAdapter struct {
	*imposter.SMTPServer
	imp *models.Imposter
}

// Port returns the port the server is listening on
func (a *SMTPServerAdapter) Port() int {
	return a.imp.Port
}

// Stop implements graceful shutdown
func (a *SMTPServerAdapter) Stop(ctx context.Context) error {
	return a.SMTPServer.Stop(ctx)
}

// Ensure the adapter implements the interface
var _ protocol.ProtocolServer = (*SMTPServerAdapter)(nil)
