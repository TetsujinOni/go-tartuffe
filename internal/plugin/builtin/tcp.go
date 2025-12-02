package builtin

import (
	"context"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

// TCPProtocol wraps the existing TCP server as a plugin
type TCPProtocol struct{}

// NewTCPProtocol creates a new TCP protocol plugin
func NewTCPProtocol() *TCPProtocol {
	return &TCPProtocol{}
}

// Name returns the protocol name
func (p *TCPProtocol) Name() string {
	return "tcp"
}

// CreateServer creates a new TCP server
func (p *TCPProtocol) CreateServer(imp *models.Imposter, callback protocol.CallbackClient) (protocol.ProtocolServer, error) {
	srv, err := imposter.NewTCPServer(imp)
	if err != nil {
		return nil, err
	}
	return &TCPServerAdapter{TCPServer: srv, imp: imp}, nil
}

// ValidateConfig validates the imposter configuration
func (p *TCPProtocol) ValidateConfig(imp *models.Imposter) error {
	return nil
}

// DefaultPort returns the default port (0 = no default)
func (p *TCPProtocol) DefaultPort() int {
	return 0
}

// TCPServerAdapter adapts the existing TCPServer to ProtocolServer interface
type TCPServerAdapter struct {
	*imposter.TCPServer
	imp *models.Imposter
}

// Port returns the port the server is listening on
func (a *TCPServerAdapter) Port() int {
	return a.imp.Port
}

// Stop implements graceful shutdown
func (a *TCPServerAdapter) Stop(ctx context.Context) error {
	return a.TCPServer.Stop(ctx)
}

// Ensure the adapter implements the interface
var _ protocol.ProtocolServer = (*TCPServerAdapter)(nil)
