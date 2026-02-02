package imposter

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/dop251/goja"
)

// ScriptCache caches compiled Goja programs to avoid re-parsing scripts.
// Compiled programs are VM-agnostic and can be reused across any goja.Runtime.
type ScriptCache struct {
	cache sync.Map // map[string]*goja.Program (key = script hash)
}

// NewScriptCache creates a new script compilation cache.
func NewScriptCache() *ScriptCache {
	return &ScriptCache{}
}

// GetOrCompile returns a compiled program for the given script.
// If the script has been compiled before, it returns the cached program.
// Otherwise, it compiles the script, caches it, and returns the program.
func (c *ScriptCache) GetOrCompile(script string) (*goja.Program, error) {
	// Compute hash of the script for cache key
	key := hashScript(script)

	// Check cache first (lock-free read)
	if cached, ok := c.cache.Load(key); ok {
		return cached.(*goja.Program), nil
	}

	// Compile the script
	program, err := goja.Compile("", script, true)
	if err != nil {
		return nil, err
	}

	// Store in cache (may race with another goroutine, but that's fine)
	c.cache.Store(key, program)

	return program, nil
}

// hashScript computes a SHA-256 hash of the script for use as cache key.
func hashScript(script string) string {
	hash := sha256.Sum256([]byte(script))
	return hex.EncodeToString(hash[:])
}
