package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// SaveOptions contains options for saving configuration
type SaveOptions struct {
	SaveFile      string
	RemoveProxies bool
	Replayable    bool
}

// Saver handles saving imposter configurations to files
type Saver struct {
	options SaveOptions
}

// NewSaver creates a new configuration saver
func NewSaver(options SaveOptions) *Saver {
	if options.SaveFile == "" {
		options.SaveFile = "mb.json"
	}
	return &Saver{options: options}
}

// Save writes imposters to a file
func (s *Saver) Save(imposters []*models.Imposter) error {
	// Process imposters according to options
	processed := make([]models.Imposter, 0, len(imposters))

	for _, imp := range imposters {
		impCopy := *imp

		// In replayable mode, exclude requests
		if s.options.Replayable {
			impCopy.Requests = nil
			impCopy.NumberOfRequests = nil
		}

		// Remove proxy responses if requested
		if s.options.RemoveProxies && len(impCopy.Stubs) > 0 {
			filteredStubs := make([]models.Stub, 0, len(impCopy.Stubs))
			for _, stub := range impCopy.Stubs {
				// Filter out proxy responses from this stub
				nonProxyResponses := make([]models.Response, 0, len(stub.Responses))
				for _, resp := range stub.Responses {
					if resp.Proxy == nil {
						nonProxyResponses = append(nonProxyResponses, resp)
					}
				}
				// Only keep stub if it has non-proxy responses
				if len(nonProxyResponses) > 0 {
					stubCopy := stub
					stubCopy.Responses = nonProxyResponses
					stubCopy.Links = nil // Don't include links in saved file
					filteredStubs = append(filteredStubs, stubCopy)
				}
			}
			impCopy.Stubs = filteredStubs
		} else {
			// Still remove links from stubs
			if len(impCopy.Stubs) > 0 {
				cleanStubs := make([]models.Stub, len(impCopy.Stubs))
				for i, stub := range impCopy.Stubs {
					cleanStubs[i] = stub
					cleanStubs[i].Links = nil
				}
				impCopy.Stubs = cleanStubs
			}
		}

		// Remove hypermedia links
		impCopy.Links = nil

		processed = append(processed, impCopy)
	}

	// Create config structure
	config := Config{
		Imposters: processed,
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(s.options.SaveFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveFile is a convenience function to save imposters to a file
func SaveFile(imposters []*models.Imposter, filename string, removeProxies, replayable bool) error {
	saver := NewSaver(SaveOptions{
		SaveFile:      filename,
		RemoveProxies: removeProxies,
		Replayable:    replayable,
	})
	return saver.Save(imposters)
}
