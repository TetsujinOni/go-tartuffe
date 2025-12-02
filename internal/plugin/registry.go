package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/repository"
)

// RepositoryFactory is a function that creates repository plugins
type RepositoryFactory func(config repository.Config) (repository.RepositoryPlugin, error)

// Registry manages all loaded plugins (both in-process and out-of-process)
type Registry struct {
	// In-process protocol plugins
	protocols map[string]protocol.ProtocolPlugin

	// Out-of-process protocol configurations
	outOfProcess map[string]*protocol.OutOfProcessConfig

	// Repository plugin factories
	repositories map[string]RepositoryFactory

	mu sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		protocols:    make(map[string]protocol.ProtocolPlugin),
		outOfProcess: make(map[string]*protocol.OutOfProcessConfig),
		repositories: make(map[string]RepositoryFactory),
	}
}

// RegisterProtocol registers an in-process protocol plugin.
// Returns an error if a protocol with the same name is already registered.
func (r *Registry) RegisterProtocol(p protocol.ProtocolPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.protocols[name]; exists {
		return fmt.Errorf("protocol %q already registered", name)
	}
	if _, exists := r.outOfProcess[name]; exists {
		return fmt.Errorf("protocol %q already registered as out-of-process", name)
	}

	r.protocols[name] = p
	return nil
}

// RegisterOutOfProcessProtocol registers an out-of-process protocol configuration.
// Returns an error if a protocol with the same name is already registered.
func (r *Registry) RegisterOutOfProcessProtocol(config *protocol.OutOfProcessConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := config.Name
	if _, exists := r.protocols[name]; exists {
		return fmt.Errorf("protocol %q already registered as in-process", name)
	}
	if _, exists := r.outOfProcess[name]; exists {
		return fmt.Errorf("protocol %q already registered", name)
	}

	r.outOfProcess[name] = config
	return nil
}

// GetProtocol returns a protocol plugin by name.
// For out-of-process protocols, this returns a bridge wrapper.
// Returns nil and false if the protocol is not found.
func (r *Registry) GetProtocol(name string) (protocol.ProtocolPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check in-process first
	if p, ok := r.protocols[name]; ok {
		return p, true
	}

	// Check out-of-process
	if cfg, ok := r.outOfProcess[name]; ok {
		// Return a bridge that wraps the out-of-process protocol
		return NewOutOfProcessBridge(cfg), true
	}

	return nil, false
}

// HasProtocol checks if a protocol is registered (either in-process or out-of-process)
func (r *Registry) HasProtocol(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.protocols[name]; ok {
		return true
	}
	if _, ok := r.outOfProcess[name]; ok {
		return true
	}
	return false
}

// ListProtocols returns all registered protocol names
func (r *Registry) ListProtocols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.protocols)+len(r.outOfProcess))
	for name := range r.protocols {
		names = append(names, name)
	}
	for name := range r.outOfProcess {
		names = append(names, name)
	}
	return names
}

// IsOutOfProcess returns true if the protocol is an out-of-process plugin
func (r *Registry) IsOutOfProcess(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.outOfProcess[name]
	return ok
}

// LoadProtocolsFile loads protocol definitions from a protocols.json file.
// This is compatible with mountebank's protocols.json format.
func (r *Registry) LoadProtocolsFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read protocols file: %w", err)
	}

	var configs map[string]*protocol.OutOfProcessConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse protocols file: %w", err)
	}

	for name, config := range configs {
		config.Name = name
		if err := r.RegisterOutOfProcessProtocol(config); err != nil {
			return fmt.Errorf("failed to register protocol %q: %w", name, err)
		}
	}

	return nil
}

// RegisterRepositoryFactory registers a repository plugin factory.
// The scheme is used to match connection strings (e.g., "redis" for "redis://...")
func (r *Registry) RegisterRepositoryFactory(scheme string, factory RepositoryFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.repositories[scheme] = factory
}

// GetRepositoryFactory returns a repository factory by scheme.
// Returns nil and false if the scheme is not found.
func (r *Registry) GetRepositoryFactory(scheme string) (RepositoryFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.repositories[scheme]
	return factory, ok
}

// HasRepositoryScheme checks if a repository scheme is registered
func (r *Registry) HasRepositoryScheme(scheme string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.repositories[scheme]
	return ok
}

// ListRepositorySchemes returns all registered repository schemes
func (r *Registry) ListRepositorySchemes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemes := make([]string, 0, len(r.repositories))
	for scheme := range r.repositories {
		schemes = append(schemes, scheme)
	}
	return schemes
}
