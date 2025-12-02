package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	goplugin "plugin"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/repository"
)

// PluginLoader loads Go plugins (.so files) from disk
type PluginLoader struct {
	registry *Registry
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(registry *Registry) *PluginLoader {
	return &PluginLoader{
		registry: registry,
	}
}

// LoadDirectory loads all .so plugins from a directory
func (l *PluginLoader) LoadDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".so") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := l.LoadPlugin(path); err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", name, err)
		}
	}

	return nil
}

// LoadPlugin loads a single .so plugin file
func (l *PluginLoader) LoadPlugin(path string) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Try to load as protocol plugin
	if err := l.tryLoadProtocolPlugin(p, path); err == nil {
		return nil
	}

	// Try to load as repository plugin factory
	if err := l.tryLoadRepositoryPlugin(p, path); err == nil {
		return nil
	}

	return fmt.Errorf("plugin does not export ProtocolPlugin or RepositoryPluginFactory symbol")
}

// tryLoadProtocolPlugin attempts to load a protocol plugin
func (l *PluginLoader) tryLoadProtocolPlugin(p *goplugin.Plugin, path string) error {
	// Look for "ProtocolPlugin" symbol
	sym, err := p.Lookup("ProtocolPlugin")
	if err != nil {
		// Also try "Plugin" as a shorter alternative
		sym, err = p.Lookup("Plugin")
		if err != nil {
			return fmt.Errorf("symbol not found")
		}
	}

	// The symbol should be a pointer to a ProtocolPlugin implementation
	plugin, ok := sym.(protocol.ProtocolPlugin)
	if !ok {
		// Try pointer variant
		pluginPtr, ok := sym.(*protocol.ProtocolPlugin)
		if !ok {
			return fmt.Errorf("symbol is not a ProtocolPlugin")
		}
		plugin = *pluginPtr
	}

	if err := l.registry.RegisterProtocol(plugin); err != nil {
		return fmt.Errorf("failed to register protocol: %w", err)
	}

	return nil
}

// tryLoadRepositoryPlugin attempts to load a repository plugin factory
func (l *PluginLoader) tryLoadRepositoryPlugin(p *goplugin.Plugin, path string) error {
	// Look for "RepositoryPluginFactory" symbol
	sym, err := p.Lookup("RepositoryPluginFactory")
	if err != nil {
		// Also try "RepositoryFactory" as alternative
		sym, err = p.Lookup("RepositoryFactory")
		if err != nil {
			return fmt.Errorf("symbol not found")
		}
	}

	// The symbol should be a function that creates repository plugins
	factory, ok := sym.(func(config repository.Config) (repository.RepositoryPlugin, error))
	if !ok {
		return fmt.Errorf("symbol is not a RepositoryPluginFactory function")
	}

	// Get the plugin name - look for "Name" symbol
	nameSym, err := p.Lookup("Name")
	if err != nil {
		return fmt.Errorf("repository plugin missing Name symbol")
	}

	name, ok := nameSym.(*string)
	if !ok {
		nameStr, ok := nameSym.(string)
		if !ok {
			return fmt.Errorf("Name symbol is not a string")
		}
		name = &nameStr
	}

	l.registry.RegisterRepositoryFactory(*name, factory)
	return nil
}

// ProtocolPluginExport is the expected structure for protocol plugins
// Plugin .so files should export a variable of this type named "ProtocolPlugin"
type ProtocolPluginExport = protocol.ProtocolPlugin

// RepositoryFactoryFunc is the function signature for repository plugin factories
// Plugin .so files should export a function of this type named "RepositoryPluginFactory"
type RepositoryFactoryFunc = func(config repository.Config) (repository.RepositoryPlugin, error)
