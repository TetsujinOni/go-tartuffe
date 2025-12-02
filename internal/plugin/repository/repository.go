package repository

import (
	repo "github.com/TetsujinOni/go-tartuffe/internal/repository"
)

// RepositoryPlugin extends the base Repository interface with plugin lifecycle methods.
// This interface is used for custom persistence backends (e.g., Redis, MongoDB).
type RepositoryPlugin interface {
	// Embed the base Repository interface
	repo.Repository

	// Name returns the plugin name (e.g., "redis", "mongodb")
	Name() string

	// Initialize sets up the repository with the given configuration.
	// This is called once when the plugin is loaded.
	Initialize(config Config) error

	// Close cleans up any resources (connections, etc.).
	// This is called when go-tartuffe is shutting down.
	Close() error

	// HealthCheck verifies the repository is operational.
	// Returns an error if the repository is not healthy.
	HealthCheck() error
}

// Config contains configuration for repository plugins
type Config struct {
	// Scheme identifies the repository type (e.g., "memory", "file", "redis")
	Scheme string

	// ConnectionString is the connection URL/path
	// Format depends on the repository type:
	//   redis://localhost:6379/0?prefix=mb:
	//   postgres://user:pass@localhost/dbname
	//   mongodb://localhost:27017/mountebank
	ConnectionString string

	// Options contains plugin-specific options
	Options map[string]interface{}
}

// Factory creates repository instances.
// Each repository plugin must export a factory function.
type Factory func(config Config) (RepositoryPlugin, error)
