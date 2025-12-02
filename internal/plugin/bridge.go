package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/protocol"
)

// OutOfProcessBridge implements ProtocolPlugin for out-of-process protocols.
// It creates servers that spawn subprocesses to handle the protocol.
type OutOfProcessBridge struct {
	config *protocol.OutOfProcessConfig
}

// NewOutOfProcessBridge creates a new bridge for an out-of-process protocol
func NewOutOfProcessBridge(config *protocol.OutOfProcessConfig) *OutOfProcessBridge {
	return &OutOfProcessBridge{config: config}
}

// Name returns the protocol name
func (b *OutOfProcessBridge) Name() string {
	return b.config.Name
}

// CreateServer creates a new out-of-process server
func (b *OutOfProcessBridge) CreateServer(imposter *models.Imposter, callback protocol.CallbackClient) (protocol.ProtocolServer, error) {
	return NewOutOfProcessServer(b.config, imposter, callback)
}

// ValidateConfig validates the imposter configuration
func (b *OutOfProcessBridge) ValidateConfig(imposter *models.Imposter) error {
	// Out-of-process validation is deferred to the subprocess
	return nil
}

// DefaultPort returns the default port (0 = no default)
func (b *OutOfProcessBridge) DefaultPort() int {
	return 0
}

// OutOfProcessServer manages an out-of-process protocol server subprocess
type OutOfProcessServer struct {
	config   *protocol.OutOfProcessConfig
	imposter *models.Imposter
	callback protocol.CallbackClient
	cmd      *exec.Cmd
	port     int
	pid      int
	started  bool
	mu       sync.RWMutex
}

// NewOutOfProcessServer creates a new out-of-process server
func NewOutOfProcessServer(config *protocol.OutOfProcessConfig, imposter *models.Imposter, callback protocol.CallbackClient) (*OutOfProcessServer, error) {
	return &OutOfProcessServer{
		config:   config,
		imposter: imposter,
		callback: callback,
		port:     imposter.Port,
	}, nil
}

// Start starts the out-of-process server by spawning a subprocess
func (s *OutOfProcessServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	// Build protocol config to pass to subprocess
	cfg := protocol.PluginConfig{
		Port:            s.imposter.Port,
		CallbackURL:     s.callback.GetCallbackURL(s.imposter.Port),
		Stubs:           s.imposter.Stubs,
		DefaultResponse: s.imposter.DefaultResponse,
		RecordRequests:  s.imposter.RecordRequests,
	}

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Parse create command and append config
	parts := strings.Fields(s.config.CreateCommand)
	if len(parts) == 0 {
		return fmt.Errorf("empty createCommand")
	}

	args := append(parts[1:], string(configJSON))
	s.cmd = exec.Command(parts[0], args...)
	s.cmd.Env = os.Environ()

	// Capture stdout for startup message
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr for logging
	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	s.pid = s.cmd.Process.Pid

	// Read startup message with timeout
	startupCh := make(chan *protocol.PluginStartupMessage, 1)
	errCh := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(stdout)
		line, err := reader.ReadString('\n')
		if err != nil {
			errCh <- fmt.Errorf("failed to read startup message: %w", err)
			return
		}

		var startupMsg protocol.PluginStartupMessage
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &startupMsg); err != nil {
			errCh <- fmt.Errorf("failed to parse startup message: %w", err)
			return
		}
		startupCh <- &startupMsg

		// Continue reading stdout for logging
		io.Copy(os.Stdout, reader)
	}()

	// Forward stderr to our stderr
	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	// Wait for startup with timeout
	select {
	case startupMsg := <-startupCh:
		if startupMsg.Port > 0 {
			s.port = startupMsg.Port
		}
		s.started = true
		return nil

	case err := <-errCh:
		s.cmd.Process.Kill()
		return err

	case <-time.After(30 * time.Second):
		s.cmd.Process.Kill()
		return fmt.Errorf("timeout waiting for plugin startup")
	}
}

// Stop stops the out-of-process server
func (s *OutOfProcessServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	s.started = false

	// Try graceful shutdown with SIGINT first
	if err := s.cmd.Process.Signal(syscall.SIGINT); err != nil {
		// Process might already be dead
		return nil
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// Force kill
		s.cmd.Process.Kill()
		return ctx.Err()
	}
}

// GetImposter returns the imposter configuration
func (s *OutOfProcessServer) GetImposter() *models.Imposter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.imposter
}

// UpdateStubs updates the stubs for this server
func (s *OutOfProcessServer) UpdateStubs(stubs []models.Stub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imposter.Stubs = stubs
	// Note: For out-of-process, the plugin will re-fetch stubs via callback
	// or we could implement a separate update endpoint
}

// Port returns the actual port the server is listening on
func (s *OutOfProcessServer) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.port
}
