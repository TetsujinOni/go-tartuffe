package builtin

import (
	"github.com/TetsujinOni/go-tartuffe/internal/plugin"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

// RegisterAll registers all built-in protocols and repositories with the given registry
func RegisterAll(registry *plugin.Registry) error {
	// Register protocol plugins
	protocols := []protocol.ProtocolPlugin{
		NewHTTPProtocol(),
		NewHTTPSProtocol(),
		NewTCPProtocol(),
		NewSMTPProtocol(),
		NewGRPCProtocol(),
	}

	for _, p := range protocols {
		if err := registry.RegisterProtocol(p); err != nil {
			return err
		}
	}

	// Register repository factories
	registry.RegisterRepositoryFactory("memory", MemoryRepositoryFactory)
	registry.RegisterRepositoryFactory("file", FilesystemRepositoryFactory)

	return nil
}
