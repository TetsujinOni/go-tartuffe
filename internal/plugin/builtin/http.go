package builtin

import (
	"context"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

// HTTPProtocol wraps the existing HTTP server as a plugin
type HTTPProtocol struct{}

// NewHTTPProtocol creates a new HTTP protocol plugin
func NewHTTPProtocol() *HTTPProtocol {
	return &HTTPProtocol{}
}

// Name returns the protocol name
func (p *HTTPProtocol) Name() string {
	return "http"
}

// CreateServer creates a new HTTP server
func (p *HTTPProtocol) CreateServer(imp *models.Imposter, callback protocol.CallbackClient) (protocol.ProtocolServer, error) {
	srv, err := imposter.NewServer(imp, false)
	if err != nil {
		return nil, err
	}
	return &HTTPServerAdapter{Server: srv, imp: imp}, nil
}

// ValidateConfig validates the imposter configuration
func (p *HTTPProtocol) ValidateConfig(imp *models.Imposter) error {
	return nil
}

// DefaultPort returns the default port (0 = no default)
func (p *HTTPProtocol) DefaultPort() int {
	return 0
}

// HTTPServerAdapter adapts the existing Server to ProtocolServer interface
type HTTPServerAdapter struct {
	*imposter.Server
	imp *models.Imposter
}

// Port returns the port the server is listening on
func (a *HTTPServerAdapter) Port() int {
	return a.imp.Port
}

// HTTPSProtocol wraps the existing HTTPS server as a plugin
type HTTPSProtocol struct{}

// NewHTTPSProtocol creates a new HTTPS protocol plugin
func NewHTTPSProtocol() *HTTPSProtocol {
	return &HTTPSProtocol{}
}

// Name returns the protocol name
func (p *HTTPSProtocol) Name() string {
	return "https"
}

// CreateServer creates a new HTTPS server
func (p *HTTPSProtocol) CreateServer(imp *models.Imposter, callback protocol.CallbackClient) (protocol.ProtocolServer, error) {
	srv, err := imposter.NewServer(imp, true)
	if err != nil {
		return nil, err
	}
	return &HTTPServerAdapter{Server: srv, imp: imp}, nil
}

// ValidateConfig validates the imposter configuration
func (p *HTTPSProtocol) ValidateConfig(imp *models.Imposter) error {
	return nil
}

// DefaultPort returns the default port (0 = no default)
func (p *HTTPSProtocol) DefaultPort() int {
	return 0
}

// Ensure the adapter implements the interface
var _ protocol.ProtocolServer = (*HTTPServerAdapter)(nil)

// Stop implements graceful shutdown
func (a *HTTPServerAdapter) Stop(ctx context.Context) error {
	return a.Server.Stop(ctx)
}
