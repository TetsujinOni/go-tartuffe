package repository

import (
	"sync"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// InMemory implements Repository with in-memory storage
type InMemory struct {
	imposters map[int]*models.Imposter
	mu        sync.RWMutex
}

// NewInMemory creates a new in-memory repository
func NewInMemory() *InMemory {
	return &InMemory{
		imposters: make(map[int]*models.Imposter),
	}
}

// Add stores a new imposter
func (r *InMemory) Add(imp *models.Imposter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.imposters[imp.Port]; exists {
		return ErrConflict{Port: imp.Port}
	}

	r.imposters[imp.Port] = imp
	return nil
}

// Get retrieves an imposter by port
func (r *InMemory) Get(port int) (*models.Imposter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	imp, ok := r.imposters[port]
	if !ok {
		return nil, ErrNotFound{Port: port}
	}

	return imp, nil
}

// All returns all imposters
func (r *InMemory) All() ([]*models.Imposter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*models.Imposter, 0, len(r.imposters))
	for _, imp := range r.imposters {
		result = append(result, imp)
	}

	return result, nil
}

// Exists checks if an imposter exists at the given port
func (r *InMemory) Exists(port int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.imposters[port]
	return exists
}

// Delete removes an imposter by port
func (r *InMemory) Delete(port int) (*models.Imposter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imposters[port]
	if !ok {
		return nil, ErrNotFound{Port: port}
	}

	delete(r.imposters, port)
	return imp, nil
}

// DeleteAll removes all imposters
func (r *InMemory) DeleteAll() ([]*models.Imposter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]*models.Imposter, 0, len(r.imposters))
	for _, imp := range r.imposters {
		result = append(result, imp)
	}

	r.imposters = make(map[int]*models.Imposter)
	return result, nil
}

// UpdateStubs replaces all stubs for an imposter
func (r *InMemory) UpdateStubs(port int, stubs []models.Stub) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imposters[port]
	if !ok {
		return ErrNotFound{Port: port}
	}

	imp.Stubs = stubs
	return nil
}

// AddStub adds a stub at the given index (or appends if index < 0)
func (r *InMemory) AddStub(port int, stub models.Stub, index int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imposters[port]
	if !ok {
		return ErrNotFound{Port: port}
	}

	if index < 0 || index >= len(imp.Stubs) {
		// Append to end
		imp.Stubs = append(imp.Stubs, stub)
	} else {
		// Insert at index
		imp.Stubs = append(imp.Stubs[:index], append([]models.Stub{stub}, imp.Stubs[index:]...)...)
	}

	return nil
}

// DeleteStub removes a stub at the given index
func (r *InMemory) DeleteStub(port int, index int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imposters[port]
	if !ok {
		return ErrNotFound{Port: port}
	}

	if index < 0 || index >= len(imp.Stubs) {
		return ErrInvalidIndex{Index: index, Max: len(imp.Stubs) - 1}
	}

	imp.Stubs = append(imp.Stubs[:index], imp.Stubs[index+1:]...)
	return nil
}

// ClearRequests clears recorded requests for an imposter (HTTP and TCP)
func (r *InMemory) ClearRequests(port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imposters[port]
	if !ok {
		return ErrNotFound{Port: port}
	}

	imp.Requests = nil
	imp.TCPRequests = nil
	count := 0
	imp.NumberOfRequests = &count
	return nil
}

// AddRequest records a request for an imposter
func (r *InMemory) AddRequest(port int, req models.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imposters[port]
	if !ok {
		return ErrNotFound{Port: port}
	}

	if imp.RecordRequests {
		imp.Requests = append(imp.Requests, req)
	}
	// Increment request counter
	if imp.NumberOfRequests == nil {
		count := 1
		imp.NumberOfRequests = &count
	} else {
		*imp.NumberOfRequests++
	}

	return nil
}
