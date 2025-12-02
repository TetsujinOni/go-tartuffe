package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

// PluginManager manages imposter servers using the plugin registry.
// This is the new unified manager that replaces protocol-specific handling.
type PluginManager struct {
	registry *Registry
	callback protocol.CallbackClient
	servers  map[int]protocol.ProtocolServer
	mu       sync.RWMutex
}

// NewPluginManager creates a new plugin-based imposter manager
func NewPluginManager(registry *Registry, callback protocol.CallbackClient) *PluginManager {
	return &PluginManager{
		registry: registry,
		callback: callback,
		servers:  make(map[int]protocol.ProtocolServer),
	}
}

// Start starts a server for the given imposter using the appropriate protocol plugin
func (m *PluginManager) Start(imp *models.Imposter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if port is already in use
	if _, exists := m.servers[imp.Port]; exists {
		return fmt.Errorf("server already running on port %d", imp.Port)
	}

	// Get protocol plugin from registry
	proto, ok := m.registry.GetProtocol(imp.Protocol)
	if !ok {
		return fmt.Errorf("unsupported protocol: %s", imp.Protocol)
	}

	// Validate configuration
	if err := proto.ValidateConfig(imp); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create server
	server, err := proto.CreateServer(imp, m.callback)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	m.servers[imp.Port] = server
	return nil
}

// Stop stops the server for the given port
func (m *PluginManager) Stop(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, exists := m.servers[port]
	if !exists {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		return err
	}

	delete(m.servers, port)
	return nil
}

// StopAll stops all running servers
func (m *PluginManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for port, server := range m.servers {
		if err := server.Stop(ctx); err != nil {
			lastErr = err
		}
		delete(m.servers, port)
	}

	return lastErr
}

// IsRunning checks if a server is running on the given port
func (m *PluginManager) IsRunning(port int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.servers[port]
	return exists
}

// GetServer returns the server running on the given port
func (m *PluginManager) GetServer(port int) protocol.ProtocolServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.servers[port]
}

// GetRegistry returns the plugin registry
func (m *PluginManager) GetRegistry() *Registry {
	return m.registry
}

// HasProtocol checks if a protocol is supported
func (m *PluginManager) HasProtocol(name string) bool {
	return m.registry.HasProtocol(name)
}
