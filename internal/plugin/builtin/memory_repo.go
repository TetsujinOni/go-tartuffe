package builtin

import (
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	pluginrepo "github.com/TetsujinOni/go-tartuffe/internal/plugin/repository"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
)

// MemoryRepositoryPlugin wraps the in-memory repository as a plugin
type MemoryRepositoryPlugin struct {
	repo *repository.InMemory
}

// NewMemoryRepositoryPlugin creates a new in-memory repository plugin
func NewMemoryRepositoryPlugin() *MemoryRepositoryPlugin {
	return &MemoryRepositoryPlugin{
		repo: repository.NewInMemory(),
	}
}

// Name returns the plugin name
func (m *MemoryRepositoryPlugin) Name() string {
	return "memory"
}

// Initialize initializes the repository with the given config
func (m *MemoryRepositoryPlugin) Initialize(config pluginrepo.Config) error {
	// In-memory repository doesn't need initialization
	return nil
}

// Close closes the repository
func (m *MemoryRepositoryPlugin) Close() error {
	// In-memory repository doesn't need cleanup
	return nil
}

// HealthCheck checks if the repository is healthy
func (m *MemoryRepositoryPlugin) HealthCheck() error {
	// In-memory repository is always healthy
	return nil
}

// Repository interface implementation - delegate to underlying repo

func (m *MemoryRepositoryPlugin) Add(imp *models.Imposter) error {
	return m.repo.Add(imp)
}

func (m *MemoryRepositoryPlugin) Get(port int) (*models.Imposter, error) {
	return m.repo.Get(port)
}

func (m *MemoryRepositoryPlugin) All() ([]*models.Imposter, error) {
	return m.repo.All()
}

func (m *MemoryRepositoryPlugin) Exists(port int) bool {
	return m.repo.Exists(port)
}

func (m *MemoryRepositoryPlugin) Delete(port int) (*models.Imposter, error) {
	return m.repo.Delete(port)
}

func (m *MemoryRepositoryPlugin) DeleteAll() ([]*models.Imposter, error) {
	return m.repo.DeleteAll()
}

func (m *MemoryRepositoryPlugin) UpdateStubs(port int, stubs []models.Stub) error {
	return m.repo.UpdateStubs(port, stubs)
}

func (m *MemoryRepositoryPlugin) AddStub(port int, stub models.Stub, index int) error {
	return m.repo.AddStub(port, stub, index)
}

func (m *MemoryRepositoryPlugin) DeleteStub(port int, index int) error {
	return m.repo.DeleteStub(port, index)
}

func (m *MemoryRepositoryPlugin) ClearRequests(port int) error {
	return m.repo.ClearRequests(port)
}

func (m *MemoryRepositoryPlugin) AddRequest(port int, req models.Request) error {
	return m.repo.AddRequest(port, req)
}

// MemoryRepositoryFactory creates memory repository plugins
func MemoryRepositoryFactory(config pluginrepo.Config) (pluginrepo.RepositoryPlugin, error) {
	plugin := NewMemoryRepositoryPlugin()
	if err := plugin.Initialize(config); err != nil {
		return nil, err
	}
	return plugin, nil
}
