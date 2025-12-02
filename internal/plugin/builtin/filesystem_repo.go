package builtin

import (
	"fmt"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	pluginrepo "github.com/TetsujinOni/go-tartuffe/internal/plugin/repository"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
)

// FilesystemRepositoryPlugin wraps the filesystem repository as a plugin
type FilesystemRepositoryPlugin struct {
	repo *repository.FilesystemRepository
}

// NewFilesystemRepositoryPlugin creates a new filesystem repository plugin
func NewFilesystemRepositoryPlugin(dataDir string) (*FilesystemRepositoryPlugin, error) {
	repo, err := repository.NewFilesystem(dataDir)
	if err != nil {
		return nil, err
	}
	return &FilesystemRepositoryPlugin{
		repo: repo,
	}, nil
}

// Name returns the plugin name
func (f *FilesystemRepositoryPlugin) Name() string {
	return "file"
}

// Initialize initializes the repository with the given config
func (f *FilesystemRepositoryPlugin) Initialize(config pluginrepo.Config) error {
	// Already initialized in constructor
	return nil
}

// Close closes the repository
func (f *FilesystemRepositoryPlugin) Close() error {
	// Filesystem repository doesn't need cleanup
	return nil
}

// HealthCheck checks if the repository is healthy
func (f *FilesystemRepositoryPlugin) HealthCheck() error {
	// Try to list all imposters as a health check
	_, err := f.repo.All()
	return err
}

// Repository interface implementation - delegate to underlying repo

func (f *FilesystemRepositoryPlugin) Add(imp *models.Imposter) error {
	return f.repo.Add(imp)
}

func (f *FilesystemRepositoryPlugin) Get(port int) (*models.Imposter, error) {
	return f.repo.Get(port)
}

func (f *FilesystemRepositoryPlugin) All() ([]*models.Imposter, error) {
	return f.repo.All()
}

func (f *FilesystemRepositoryPlugin) Exists(port int) bool {
	return f.repo.Exists(port)
}

func (f *FilesystemRepositoryPlugin) Delete(port int) (*models.Imposter, error) {
	return f.repo.Delete(port)
}

func (f *FilesystemRepositoryPlugin) DeleteAll() ([]*models.Imposter, error) {
	return f.repo.DeleteAll()
}

func (f *FilesystemRepositoryPlugin) UpdateStubs(port int, stubs []models.Stub) error {
	return f.repo.UpdateStubs(port, stubs)
}

func (f *FilesystemRepositoryPlugin) AddStub(port int, stub models.Stub, index int) error {
	return f.repo.AddStub(port, stub, index)
}

func (f *FilesystemRepositoryPlugin) DeleteStub(port int, index int) error {
	return f.repo.DeleteStub(port, index)
}

func (f *FilesystemRepositoryPlugin) ClearRequests(port int) error {
	return f.repo.ClearRequests(port)
}

func (f *FilesystemRepositoryPlugin) AddRequest(port int, req models.Request) error {
	return f.repo.AddRequest(port, req)
}

// FilesystemRepositoryFactory creates filesystem repository plugins
func FilesystemRepositoryFactory(config pluginrepo.Config) (pluginrepo.RepositoryPlugin, error) {
	// Get data directory from connection string or options
	dataDir := config.ConnectionString
	if dataDir == "" {
		if dir, ok := config.Options["path"].(string); ok {
			dataDir = dir
		}
	}
	if dataDir == "" {
		return nil, fmt.Errorf("filesystem repository requires a data directory")
	}

	return NewFilesystemRepositoryPlugin(dataDir)
}
