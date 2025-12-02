package repository

import "github.com/TetsujinOni/go-tartuffe/internal/models"

// Repository defines the interface for imposter storage
type Repository interface {
	// Add stores a new imposter
	Add(imp *models.Imposter) error

	// Get retrieves an imposter by port
	Get(port int) (*models.Imposter, error)

	// All returns all imposters
	All() ([]*models.Imposter, error)

	// Exists checks if an imposter exists at the given port
	Exists(port int) bool

	// Delete removes an imposter by port
	Delete(port int) (*models.Imposter, error)

	// DeleteAll removes all imposters
	DeleteAll() ([]*models.Imposter, error)

	// UpdateStubs replaces all stubs for an imposter
	UpdateStubs(port int, stubs []models.Stub) error

	// AddStub adds a stub at the given index (or appends if index < 0)
	AddStub(port int, stub models.Stub, index int) error

	// DeleteStub removes a stub at the given index
	DeleteStub(port int, index int) error

	// ClearRequests clears recorded requests for an imposter
	ClearRequests(port int) error

	// AddRequest records a request for an imposter
	AddRequest(port int, req models.Request) error
}

// ErrNotFound is returned when an imposter doesn't exist
type ErrNotFound struct {
	Port int
}

func (e ErrNotFound) Error() string {
	return "no such resource"
}

// ErrConflict is returned when a port is already in use
type ErrConflict struct {
	Port int
}

func (e ErrConflict) Error() string {
	return "resource conflict"
}

// ErrInvalidIndex is returned when a stub index is out of bounds
type ErrInvalidIndex struct {
	Index int
	Max   int
}

func (e ErrInvalidIndex) Error() string {
	return "bad data"
}
