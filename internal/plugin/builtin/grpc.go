package builtin

import (
	"context"
	"fmt"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

// GRPCProtocol wraps the gRPC server as a plugin
type GRPCProtocol struct{}

// NewGRPCProtocol creates a new gRPC protocol plugin
func NewGRPCProtocol() *GRPCProtocol {
	return &GRPCProtocol{}
}

// Name returns the protocol name
func (p *GRPCProtocol) Name() string {
	return "grpc"
}

// CreateServer creates a new gRPC server
func (p *GRPCProtocol) CreateServer(imp *models.Imposter, callback protocol.CallbackClient) (protocol.ProtocolServer, error) {
	srv, err := imposter.NewGRPCServer(imp)
	if err != nil {
		return nil, err
	}
	return &GRPCServerAdapter{GRPCServer: srv, imp: imp}, nil
}

// ValidateConfig validates the imposter configuration
func (p *GRPCProtocol) ValidateConfig(imp *models.Imposter) error {
	if len(imp.ProtoFiles) == 0 {
		return fmt.Errorf("gRPC imposter requires protoFiles to be specified")
	}
	return nil
}

// DefaultPort returns the default port (0 = no default)
func (p *GRPCProtocol) DefaultPort() int {
	return 0
}

// GRPCServerAdapter adapts the GRPCServer to ProtocolServer interface
type GRPCServerAdapter struct {
	*imposter.GRPCServer
	imp *models.Imposter
}

// Port returns the port the server is listening on
func (a *GRPCServerAdapter) Port() int {
	return a.imp.Port
}

// Stop implements graceful shutdown
func (a *GRPCServerAdapter) Stop(ctx context.Context) error {
	return a.GRPCServer.Stop(ctx)
}

// Ensure the adapter implements the interface
var _ protocol.ProtocolServer = (*GRPCServerAdapter)(nil)
