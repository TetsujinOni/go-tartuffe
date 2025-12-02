package protocol

import (
	"context"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// ProtocolPlugin defines the interface all protocol plugins must implement.
// Both in-process Go plugins and out-of-process plugins implement this.
type ProtocolPlugin interface {
	// Name returns the protocol name (e.g., "http", "tcp", "grpc")
	Name() string

	// CreateServer creates a new imposter server for this protocol
	CreateServer(imposter *models.Imposter, callback CallbackClient) (ProtocolServer, error)

	// ValidateConfig validates protocol-specific imposter configuration
	ValidateConfig(imposter *models.Imposter) error

	// DefaultPort returns the default port for this protocol (0 for no default)
	DefaultPort() int
}

// ProtocolServer represents a running protocol server instance.
// This extends the existing ImposterServer concept with additional methods.
type ProtocolServer interface {
	// Start starts the server and begins accepting connections
	Start() error

	// Stop gracefully stops the server with context for timeout
	Stop(ctx context.Context) error

	// GetImposter returns the current imposter configuration
	GetImposter() *models.Imposter

	// UpdateStubs updates the stubs for this server at runtime
	UpdateStubs(stubs []models.Stub)

	// Port returns the actual port the server is listening on
	// This may differ from the configured port if port was auto-assigned
	Port() int
}

// CallbackClient provides access to go-tartuffe core services.
// This allows plugins to interact with the main system for stub matching
// and request recording.
type CallbackClient interface {
	// RecordRequest records a request for the imposter at the given port
	RecordRequest(port int, request interface{}) error

	// MatchStub finds a matching stub for a request and returns the response
	// The request can be any protocol-specific format that can be matched
	MatchStub(port int, request map[string]interface{}) (*MatchResult, error)

	// GetCallbackURL returns the callback URL for out-of-process plugins
	// to POST requests to for stub matching
	GetCallbackURL(port int) string
}

// MatchResult contains the result of stub matching
type MatchResult struct {
	// Response is the matched stub response (nil if no match)
	Response *models.Response

	// StubIndex is the index of the matched stub (-1 if no match)
	StubIndex int

	// Matched indicates whether a stub was matched
	Matched bool
}

// PluginMetadata contains information about a plugin
type PluginMetadata struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}
