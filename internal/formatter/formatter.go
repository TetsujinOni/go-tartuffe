package formatter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TetsujinOni/go-tartuffe/internal/config"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// Options holds formatter options from CLI
type Options struct {
	ConfigFile    string
	SaveFile      string
	NoParse       bool
	RemoveProxies bool
	// Additional custom options can be added via map
	Custom map[string]string
}

// Formatter defines the interface for config file formatters
type Formatter interface {
	// Load reads a config file and returns the parsed configuration
	Load(options Options) (*config.Config, error)

	// Save writes imposters to a file
	Save(options Options, imposters *ImpostersWrapper) error
}

// ImpostersWrapper wraps imposters for JSON serialization
type ImpostersWrapper struct {
	Imposters []models.Imposter `json:"imposters"`
}

// JSONFormatter is the default JSON formatter
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Load reads a JSON/EJS config file
func (f *JSONFormatter) Load(options Options) (*config.Config, error) {
	return config.LoadFile(options.ConfigFile, options.NoParse)
}

// Save writes imposters to a JSON file
func (f *JSONFormatter) Save(options Options, imposters *ImpostersWrapper) error {
	// Optionally remove proxy responses
	if options.RemoveProxies {
		for i := range imposters.Imposters {
			imposters.Imposters[i].Stubs = removeProxyStubs(imposters.Imposters[i].Stubs)
		}
	}

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(imposters, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal imposters: %w", err)
	}

	// Write to file
	if err := os.WriteFile(options.SaveFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}

	return nil
}

// removeProxyStubs filters out stubs that only contain proxy responses
func removeProxyStubs(stubs []models.Stub) []models.Stub {
	result := make([]models.Stub, 0)
	for _, stub := range stubs {
		// Filter out proxy responses
		nonProxyResponses := make([]models.Response, 0)
		for _, resp := range stub.Responses {
			if resp.Proxy == nil {
				nonProxyResponses = append(nonProxyResponses, resp)
			}
		}

		// Only keep stub if it has non-proxy responses
		if len(nonProxyResponses) > 0 {
			stub.Responses = nonProxyResponses
			result = append(result, stub)
		}
	}
	return result
}

// DefaultFormatter returns the default JSON formatter
func DefaultFormatter() Formatter {
	return NewJSONFormatter()
}
