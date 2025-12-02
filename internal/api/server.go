package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/api/handlers"
	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin"
	"github.com/TetsujinOni/go-tartuffe/internal/plugin/builtin"
	pluginrepo "github.com/TetsujinOni/go-tartuffe/internal/plugin/repository"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server is the main API server
type Server struct {
	httpServer      *http.Server
	repo            repository.Repository
	imposterManager *imposter.Manager
	pluginRegistry  *plugin.Registry
	startTime       time.Time
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port                int
	Host                string
	AllowInjection      bool
	LocalOnly           bool
	Debug               bool
	IPWhitelist         string
	Origin              string
	APIKey              string
	DataDir             string // If set, use filesystem-backed repository
	ProtoFile           string // Path to protocols.json for custom protocols
	PluginsDir          string // Directory containing Go plugin .so files
	ImpostersRepository string // Repository connection string (e.g., redis://localhost:6379)
}

// NewServer creates a new API server
func NewServer(cfg ServerConfig) *Server {
	imposterMgr := imposter.NewManager()
	startTime := time.Now()

	// Create plugin registry and register built-in protocols and repositories
	registry := plugin.NewRegistry()
	if err := builtin.RegisterAll(registry); err != nil {
		log.Fatalf("failed to register built-in protocols: %v", err)
	}

	// Initialize repository based on configuration
	var repo repository.Repository
	var err error

	if cfg.ImpostersRepository != "" {
		// Use plugin-based repository from connection string
		repoConfig, err := pluginrepo.ParseConnectionString(cfg.ImpostersRepository)
		if err != nil {
			log.Fatalf("failed to parse repository connection string: %v", err)
		}

		factory, ok := registry.GetRepositoryFactory(repoConfig.Scheme)
		if !ok {
			log.Fatalf("unknown repository scheme: %s", repoConfig.Scheme)
		}

		pluginRepo, err := factory(*repoConfig)
		if err != nil {
			log.Fatalf("failed to create repository: %v", err)
		}
		repo = pluginRepo
		log.Printf("using %s repository", repoConfig.Scheme)
	} else if cfg.DataDir != "" {
		// Legacy: use filesystem repository if datadir is specified
		repo, err = repository.NewFilesystem(cfg.DataDir)
		if err != nil {
			log.Fatalf("failed to create filesystem repository: %v", err)
		}
		log.Printf("using filesystem repository at %s", cfg.DataDir)
	} else {
		// Default: in-memory repository
		repo = repository.NewInMemory()
	}

	// Load custom protocols from protofile if specified
	if cfg.ProtoFile != "" {
		if err := registry.LoadProtocolsFile(cfg.ProtoFile); err != nil {
			log.Fatalf("failed to load protocols file: %v", err)
		}
		log.Printf("loaded custom protocols from %s", cfg.ProtoFile)
	}

	// Load Go plugins from plugins directory if specified
	if cfg.PluginsDir != "" {
		loader := plugin.NewPluginLoader(registry)
		if err := loader.LoadDirectory(cfg.PluginsDir); err != nil {
			log.Fatalf("failed to load plugins from %s: %v", cfg.PluginsDir, err)
		}
		log.Printf("loaded plugins from %s", cfg.PluginsDir)
	}

	// Create callback handler for out-of-process plugins
	baseURL := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	if cfg.Host == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
	}
	callbackHandler := plugin.NewCallbackHandler(repo, baseURL)

	// Create handlers
	impostersHandler := handlers.NewImpostersHandler(repo, imposterMgr, cfg.Port)
	imposterHandler := handlers.NewImposterHandler(repo, imposterMgr)
	stubsHandler := handlers.NewStubsHandler(repo)
	configHandler := handlers.NewConfigHandler(cfg.Port, cfg.Host, cfg.AllowInjection, cfg.LocalOnly, cfg.Debug, cfg.IPWhitelist, cfg.Origin, startTime.Unix())
	logsHandler := handlers.NewLogsHandler()

	// Create router
	router := NewRouter()

	// Register routes
	// Home
	router.GET("/", handlers.Home)

	// Imposters collection
	router.GET("/imposters", impostersHandler.GetImposters)
	router.POST("/imposters", impostersHandler.CreateImposter)
	router.DELETE("/imposters", impostersHandler.DeleteImposters)
	router.PUT("/imposters", impostersHandler.ReplaceImposters)

	// Individual imposter
	router.GET("/imposters/{id}", imposterHandler.GetImposter)
	router.DELETE("/imposters/{id}", imposterHandler.DeleteImposter)

	// Imposter requests/proxies
	router.DELETE("/imposters/{id}/savedRequests", imposterHandler.ResetRequests)
	router.DELETE("/imposters/{id}/savedProxyResponses", imposterHandler.ResetRequests) // Same handler

	// Stubs
	router.PUT("/imposters/{id}/stubs", stubsHandler.ReplaceStubs)
	router.POST("/imposters/{id}/stubs", stubsHandler.AddStub)
	router.PUT("/imposters/{id}/stubs/{stubIndex}", stubsHandler.ReplaceStub)
	router.DELETE("/imposters/{id}/stubs/{stubIndex}", stubsHandler.DeleteStub)

	// Plugin callback endpoint (for out-of-process protocol plugins)
	router.POST("/imposters/{id}/_requests", callbackHandler.HandleCallback)

	// Config and logs
	router.GET("/config", configHandler.GetConfig)
	router.GET("/logs", logsHandler.GetLogs)

	// Prometheus metrics endpoint
	router.GET("/metrics", promhttp.Handler().ServeHTTP)

	// Apply middleware chain
	handler := Logger(
		CORSWithOrigin(cfg.Origin)(
			APIKeyAuth(cfg.APIKey)(
				IPWhitelist(cfg.IPWhitelist)(
					LocalOnly(cfg.LocalOnly)(
						JSONBody(router))))))

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		repo:            repo,
		imposterManager: imposterMgr,
		pluginRegistry:  registry,
		startTime:       startTime,
	}
}

// Start starts the server
func (s *Server) Start() error {
	log.Printf("mountebank (go-tartuffe) running on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop all imposter servers first
	if s.imposterManager != nil {
		s.imposterManager.StopAll()
	}

	return s.httpServer.Shutdown(ctx)
}

// GetRepository returns the repository (for testing)
func (s *Server) GetRepository() repository.Repository {
	return s.repo
}

// GetImposterManager returns the imposter manager (for testing)
func (s *Server) GetImposterManager() *imposter.Manager {
	return s.imposterManager
}

// GetPluginRegistry returns the plugin registry (for testing)
func (s *Server) GetPluginRegistry() *plugin.Registry {
	return s.pluginRegistry
}

// LoadImposters loads imposters from a configuration
func (s *Server) LoadImposters(imposters []models.Imposter) error {
	for i := range imposters {
		imp := &imposters[i]

		// Initialize stubs if nil
		if imp.Stubs == nil {
			imp.Stubs = []models.Stub{}
		}

		// Add to repository
		if err := s.repo.Add(imp); err != nil {
			return fmt.Errorf("failed to add imposter on port %d: %w", imp.Port, err)
		}

		// Start imposter server for HTTP protocol
		if imp.Protocol == "http" && s.imposterManager != nil {
			if err := s.imposterManager.Start(imp); err != nil {
				// Remove from repository if failed to start
				s.repo.Delete(imp.Port)
				return fmt.Errorf("failed to start imposter on port %d: %w", imp.Port, err)
			}
		}
	}
	return nil
}

// SaveImposters returns all imposters for saving
func (s *Server) SaveImposters() ([]*models.Imposter, error) {
	return s.repo.All()
}

// LoadPersistedImposters loads imposters from the filesystem repository
// This is called at startup when using --datadir
func (s *Server) LoadPersistedImposters() error {
	// Check if this is a filesystem repository
	fsRepo, ok := s.repo.(*repository.FilesystemRepository)
	if !ok {
		return nil // Not a filesystem repository, nothing to load
	}

	imposters, err := fsRepo.LoadAll()
	if err != nil {
		return fmt.Errorf("failed to load persisted imposters: %w", err)
	}

	// Start imposter servers for loaded imposters
	for _, imp := range imposters {
		if imp.Protocol == "http" && s.imposterManager != nil {
			if err := s.imposterManager.Start(imp); err != nil {
				log.Printf("warning: failed to start persisted imposter on port %d: %v", imp.Port, err)
			} else {
				log.Printf("restored imposter %s on port %d", imp.Name, imp.Port)
			}
		}
	}

	return nil
}
