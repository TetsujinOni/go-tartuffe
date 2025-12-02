package protocol

import (
	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// PluginStartupMessage is written to stdout by out-of-process plugins when ready.
// The plugin should write this as a single JSON line to stdout after starting.
type PluginStartupMessage struct {
	// Port is the actual port the plugin is listening on
	Port int `json:"port"`

	// Pid is the process ID (optional, for logging)
	Pid int `json:"pid,omitempty"`

	// Meta contains any protocol-specific metadata
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// PluginConfig is passed to out-of-process plugins via command line as JSON.
// This is the configuration the plugin needs to start serving requests.
type PluginConfig struct {
	// Port is the port to listen on
	Port int `json:"port"`

	// CallbackURL is the URL to POST requests to for stub matching
	// Format: http://localhost:2525/imposters/{port}/_requests
	CallbackURL string `json:"callbackURL"`

	// Stubs are the stub configurations for this imposter
	Stubs []models.Stub `json:"stubs,omitempty"`

	// DefaultResponse is the default response when no stub matches
	DefaultResponse *models.Response `json:"defaultResponse,omitempty"`

	// RecordRequests indicates whether to record incoming requests
	RecordRequests bool `json:"recordRequests,omitempty"`

	// Loglevel for plugin logging (debug, info, warn, error)
	Loglevel string `json:"loglevel,omitempty"`

	// AllowInjection indicates whether JavaScript injection is allowed
	AllowInjection bool `json:"allowInjection,omitempty"`

	// Options contains protocol-specific options
	Options map[string]interface{} `json:"options,omitempty"`
}

// CallbackRequest is sent to go-tartuffe from out-of-process plugins.
// Plugins POST this to the callback URL to get stub matches.
type CallbackRequest struct {
	// Request contains the protocol-specific request data
	Request map[string]interface{} `json:"request"`

	// RequestFrom is the client socket address (e.g., "127.0.0.1:54321")
	RequestFrom string `json:"requestFrom,omitempty"`

	// Timestamp is when the request was received (RFC3339 format)
	Timestamp string `json:"timestamp,omitempty"`
}

// CallbackResponse is returned from go-tartuffe to plugins.
// This contains the matched stub response.
type CallbackResponse struct {
	// Response is the matched stub response (nil if no match and no default)
	Response *models.IsResponse `json:"response,omitempty"`

	// Proxy is set if the matched stub requires proxying
	Proxy *models.ProxyResponse `json:"proxy,omitempty"`

	// StubIndex is the index of the matched stub (-1 if no match)
	StubIndex int `json:"stubIndex"`

	// Matched indicates whether a stub was matched
	Matched bool `json:"matched"`

	// Blocked indicates the request was blocked (e.g., IP whitelist)
	Blocked bool `json:"blocked,omitempty"`

	// BlockedReason explains why the request was blocked
	BlockedReason string `json:"blockedReason,omitempty"`
}

// OutOfProcessConfig defines an out-of-process protocol from protocols.json.
// This matches the mountebank protocols.json format.
type OutOfProcessConfig struct {
	// Name is the protocol name (key in protocols.json)
	Name string `json:"-"`

	// CreateCommand is the shell command to spawn the plugin
	// Config JSON will be appended as the last argument
	CreateCommand string `json:"createCommand"`

	// TestRequest is an example request for validation (optional)
	TestRequest map[string]interface{} `json:"testRequest,omitempty"`

	// TestProxyResponse is an example proxy response for validation (optional)
	TestProxyResponse map[string]interface{} `json:"testProxyResponse,omitempty"`
}
