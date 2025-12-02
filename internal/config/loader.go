package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// Config represents a mountebank configuration file structure
type Config struct {
	Imposters []models.Imposter `json:"imposters"`
}

// LoadOptions contains options for loading configuration
type LoadOptions struct {
	ConfigFile string
	NoParse    bool // If true, skip EJS rendering
}

// Loader handles loading imposter configurations from files
type Loader struct {
	options LoadOptions
}

// NewLoader creates a new configuration loader
func NewLoader(options LoadOptions) *Loader {
	return &Loader{options: options}
}

// Load reads and parses a configuration file
func (l *Loader) Load() (*Config, error) {
	if l.options.ConfigFile == "" {
		return nil, fmt.Errorf("no config file specified")
	}

	// Read the file
	content, err := os.ReadFile(l.options.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	contentStr := string(content)

	// Check if we need to render EJS
	if !l.options.NoParse && needsEJSRendering(l.options.ConfigFile, contentStr) {
		renderer := NewEJSRenderer(filepath.Dir(l.options.ConfigFile))
		contentStr, err = renderer.Render(contentStr)
		if err != nil {
			return nil, fmt.Errorf("failed to render EJS template: %w", err)
		}
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal([]byte(contentStr), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Validate imposters
	for i, imp := range config.Imposters {
		if imp.Protocol == "" {
			return nil, fmt.Errorf("imposter %d: 'protocol' is required", i)
		}
		if imp.Port <= 0 || imp.Port > 65535 {
			return nil, fmt.Errorf("imposter %d: invalid port %d", i, imp.Port)
		}
		// Initialize stubs if nil
		if imp.Stubs == nil {
			config.Imposters[i].Stubs = []models.Stub{}
		}
	}

	return &config, nil
}

// needsEJSRendering checks if a file needs EJS rendering
func needsEJSRendering(filename, content string) bool {
	// Check file extension
	if strings.HasSuffix(filename, ".ejs") {
		return true
	}

	// Check for EJS tags in content
	if strings.Contains(content, "<%") {
		return true
	}

	return false
}

// LoadFile is a convenience function to load a config file
func LoadFile(filename string, noParse bool) (*Config, error) {
	loader := NewLoader(LoadOptions{
		ConfigFile: filename,
		NoParse:    noParse,
	})
	return loader.Load()
}
