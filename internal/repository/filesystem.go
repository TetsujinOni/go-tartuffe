package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
)

// FilesystemRepository implements Repository with filesystem persistence
// Directory structure:
//
//	/{datadir}
//	  /{port}
//	    /imposter.json   - main imposter config (protocol, port, name)
//	    /stubs/
//	      /{timestamp-pid-counter}/
//	        /meta.json     - responseFiles array, nextIndex
//	        /responses/
//	          /{timestamp}.json
//	        /matches/
//	          /{timestamp}.json
//	    /requests/
//	      /{timestamp}.json
type FilesystemRepository struct {
	datadir string
	counter int64
	mu      sync.RWMutex
}

// NewFilesystem creates a new filesystem-backed repository
func NewFilesystem(datadir string) (*FilesystemRepository, error) {
	// Ensure the datadir exists
	if err := os.MkdirAll(datadir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create datadir: %w", err)
	}

	return &FilesystemRepository{
		datadir: datadir,
		counter: 0,
	}, nil
}

// filenameFor generates a unique filename based on timestamp, pid, and counter
func (r *FilesystemRepository) filenameFor() string {
	epoch := time.Now().UnixMilli()
	counter := atomic.AddInt64(&r.counter, 1)
	return fmt.Sprintf("%d-%d-%d", epoch, os.Getpid(), counter)
}

// imposterDir returns the directory for an imposter
func (r *FilesystemRepository) imposterDir(port int) string {
	return filepath.Join(r.datadir, strconv.Itoa(port))
}

// imposterFile returns the path to the imposter.json file
func (r *FilesystemRepository) imposterFile(port int) string {
	return filepath.Join(r.imposterDir(port), "imposter.json")
}

// stubsDir returns the stubs directory for an imposter
func (r *FilesystemRepository) stubsDir(port int) string {
	return filepath.Join(r.imposterDir(port), "stubs")
}

// requestsDir returns the requests directory for an imposter
func (r *FilesystemRepository) requestsDir(port int) string {
	return filepath.Join(r.imposterDir(port), "requests")
}

// readJSON reads and unmarshals a JSON file
func readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// writeJSON writes a JSON file atomically (write to temp, then rename)
func writeJSON(path string, v interface{}) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// stubMeta holds stub metadata stored in meta.json
type stubMeta struct {
	ResponseFiles    []string `json:"responseFiles"`
	OrderWithRepeats []int    `json:"orderWithRepeats"`
	NextIndex        int      `json:"nextIndex"`
}

// imposterHeader holds the imposter header stored in imposter.json
type imposterHeader struct {
	Protocol       string               `json:"protocol"`
	Port           int                  `json:"port"`
	Name           string               `json:"name,omitempty"`
	RecordRequests bool                 `json:"recordRequests,omitempty"`
	Stubs          []imposterStubHeader `json:"stubs"`
}

// imposterStubHeader holds stub header info (predicates + meta dir reference)
type imposterStubHeader struct {
	Predicates []models.Predicate `json:"predicates,omitempty"`
	Meta       struct {
		Dir string `json:"dir"`
	} `json:"meta"`
}

// Add stores a new imposter
func (r *FilesystemRepository) Add(imp *models.Imposter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	impDir := r.imposterDir(imp.Port)

	// Check if already exists
	if _, err := os.Stat(impDir); err == nil {
		return ErrConflict{Port: imp.Port}
	}

	// Create imposter directory
	if err := os.MkdirAll(impDir, 0755); err != nil {
		return err
	}

	// Create the header with stub references
	header := imposterHeader{
		Protocol:       imp.Protocol,
		Port:           imp.Port,
		Name:           imp.Name,
		RecordRequests: imp.RecordRequests,
		Stubs:          make([]imposterStubHeader, 0),
	}

	// Save each stub to disk
	for _, stub := range imp.Stubs {
		stubHeader, err := r.saveStub(imp.Port, stub)
		if err != nil {
			// Clean up on error
			os.RemoveAll(impDir)
			return err
		}
		header.Stubs = append(header.Stubs, stubHeader)
	}

	// Write the imposter header
	return writeJSON(r.imposterFile(imp.Port), header)
}

// saveStub saves a stub to disk and returns its header
func (r *FilesystemRepository) saveStub(port int, stub models.Stub) (imposterStubHeader, error) {
	stubDirName := r.filenameFor()
	stubDir := filepath.Join(r.stubsDir(port), stubDirName)

	// Create stub directory
	if err := os.MkdirAll(filepath.Join(stubDir, "responses"), 0755); err != nil {
		return imposterStubHeader{}, err
	}

	meta := stubMeta{
		ResponseFiles:    make([]string, 0),
		OrderWithRepeats: make([]int, 0),
		NextIndex:        0,
	}

	// Save each response
	for i, resp := range stub.Responses {
		respFileName := r.filenameFor() + ".json"
		respPath := filepath.Join(stubDir, "responses", respFileName)

		if err := writeJSON(respPath, resp); err != nil {
			return imposterStubHeader{}, err
		}

		meta.ResponseFiles = append(meta.ResponseFiles, "responses/"+respFileName)

		// Handle repeat behavior
		repeat := 1
		if resp.Repeat > 0 {
			repeat = resp.Repeat
		}
		for j := 0; j < repeat; j++ {
			meta.OrderWithRepeats = append(meta.OrderWithRepeats, i)
		}
	}

	// Save stub meta
	if err := writeJSON(filepath.Join(stubDir, "meta.json"), meta); err != nil {
		return imposterStubHeader{}, err
	}

	return imposterStubHeader{
		Predicates: stub.Predicates,
		Meta: struct {
			Dir string `json:"dir"`
		}{Dir: "stubs/" + stubDirName},
	}, nil
}

// Get retrieves an imposter by port
func (r *FilesystemRepository) Get(port int) (*models.Imposter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.getImposter(port)
}

// getImposter is the internal implementation (must be called with lock held)
func (r *FilesystemRepository) getImposter(port int) (*models.Imposter, error) {
	impFile := r.imposterFile(port)

	var header imposterHeader
	if err := readJSON(impFile, &header); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound{Port: port}
		}
		return nil, err
	}

	imp := &models.Imposter{
		Protocol:       header.Protocol,
		Port:           header.Port,
		Name:           header.Name,
		RecordRequests: header.RecordRequests,
		Stubs:          make([]models.Stub, 0),
		Requests:       make([]models.Request, 0),
	}

	// Load stubs
	for _, stubHeader := range header.Stubs {
		stub, err := r.loadStub(port, stubHeader)
		if err != nil {
			return nil, err
		}
		imp.Stubs = append(imp.Stubs, stub)
	}

	// Load requests
	requests, err := r.loadRequests(port)
	if err == nil {
		imp.Requests = requests
	}

	return imp, nil
}

// loadStub loads a stub from disk
func (r *FilesystemRepository) loadStub(port int, header imposterStubHeader) (models.Stub, error) {
	stubDir := filepath.Join(r.imposterDir(port), header.Meta.Dir)
	metaPath := filepath.Join(stubDir, "meta.json")

	var meta stubMeta
	if err := readJSON(metaPath, &meta); err != nil {
		return models.Stub{}, err
	}

	stub := models.Stub{
		Predicates: header.Predicates,
		Responses:  make([]models.Response, 0),
	}

	// Load responses
	for _, respFile := range meta.ResponseFiles {
		var resp models.Response
		if err := readJSON(filepath.Join(stubDir, respFile), &resp); err != nil {
			return models.Stub{}, err
		}
		stub.Responses = append(stub.Responses, resp)
	}

	return stub, nil
}

// loadRequests loads all requests from the requests directory
func (r *FilesystemRepository) loadRequests(port int) ([]models.Request, error) {
	reqDir := r.requestsDir(port)
	entries, err := os.ReadDir(reqDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Request{}, nil
		}
		return nil, err
	}

	// Sort by filename (which is timestamp-based)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	requests := make([]models.Request, 0)
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var req models.Request
		if err := readJSON(filepath.Join(reqDir, entry.Name()), &req); err != nil {
			continue
		}
		requests = append(requests, req)
	}

	return requests, nil
}

// All returns all imposters
func (r *FilesystemRepository) All() ([]*models.Imposter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.datadir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.Imposter{}, nil
		}
		return nil, err
	}

	imposters := make([]*models.Imposter, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		port, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Skip non-numeric directories
		}

		imp, err := r.getImposter(port)
		if err != nil {
			continue
		}
		imposters = append(imposters, imp)
	}

	// Sort by port
	sort.Slice(imposters, func(i, j int) bool {
		return imposters[i].Port < imposters[j].Port
	})

	return imposters, nil
}

// Exists checks if an imposter exists at the given port
func (r *FilesystemRepository) Exists(port int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, err := os.Stat(r.imposterFile(port))
	return err == nil
}

// Delete removes an imposter by port
func (r *FilesystemRepository) Delete(port int) (*models.Imposter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get imposter first
	imp, err := r.getImposter(port)
	if err != nil {
		return nil, err
	}

	// Remove the directory
	if err := os.RemoveAll(r.imposterDir(port)); err != nil {
		return nil, err
	}

	return imp, nil
}

// DeleteAll removes all imposters
func (r *FilesystemRepository) DeleteAll() ([]*models.Imposter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries, err := os.ReadDir(r.datadir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.Imposter{}, nil
		}
		return nil, err
	}

	imposters := make([]*models.Imposter, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		port, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		imp, err := r.getImposter(port)
		if err != nil {
			continue
		}
		imposters = append(imposters, imp)

		// Remove the directory
		os.RemoveAll(r.imposterDir(port))
	}

	return imposters, nil
}

// UpdateStubs replaces all stubs for an imposter
func (r *FilesystemRepository) UpdateStubs(port int, stubs []models.Stub) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	impFile := r.imposterFile(port)

	var header imposterHeader
	if err := readJSON(impFile, &header); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound{Port: port}
		}
		return err
	}

	// Remove old stubs directory
	os.RemoveAll(r.stubsDir(port))

	// Save new stubs
	header.Stubs = make([]imposterStubHeader, 0)
	for _, stub := range stubs {
		stubHeader, err := r.saveStub(port, stub)
		if err != nil {
			return err
		}
		header.Stubs = append(header.Stubs, stubHeader)
	}

	return writeJSON(impFile, header)
}

// AddStub adds a stub at the given index (or appends if index < 0)
func (r *FilesystemRepository) AddStub(port int, stub models.Stub, index int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	impFile := r.imposterFile(port)

	var header imposterHeader
	if err := readJSON(impFile, &header); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound{Port: port}
		}
		return err
	}

	// Save the new stub
	stubHeader, err := r.saveStub(port, stub)
	if err != nil {
		return err
	}

	// Insert at index or append
	if index < 0 || index >= len(header.Stubs) {
		header.Stubs = append(header.Stubs, stubHeader)
	} else {
		header.Stubs = append(header.Stubs[:index+1], header.Stubs[index:]...)
		header.Stubs[index] = stubHeader
	}

	return writeJSON(impFile, header)
}

// DeleteStub removes a stub at the given index
func (r *FilesystemRepository) DeleteStub(port int, index int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	impFile := r.imposterFile(port)

	var header imposterHeader
	if err := readJSON(impFile, &header); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound{Port: port}
		}
		return err
	}

	if index < 0 || index >= len(header.Stubs) {
		return ErrInvalidIndex{Index: index, Max: len(header.Stubs)}
	}

	// Get the stub directory to remove
	stubDir := filepath.Join(r.imposterDir(port), header.Stubs[index].Meta.Dir)

	// Remove from header
	header.Stubs = append(header.Stubs[:index], header.Stubs[index+1:]...)

	// Write updated header
	if err := writeJSON(impFile, header); err != nil {
		return err
	}

	// Remove stub directory
	os.RemoveAll(stubDir)

	return nil
}

// ClearRequests clears recorded requests for an imposter
func (r *FilesystemRepository) ClearRequests(port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := os.Stat(r.imposterFile(port)); os.IsNotExist(err) {
		return ErrNotFound{Port: port}
	}

	return os.RemoveAll(r.requestsDir(port))
}

// AddRequest records a request for an imposter
func (r *FilesystemRepository) AddRequest(port int, req models.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := os.Stat(r.imposterFile(port)); os.IsNotExist(err) {
		return ErrNotFound{Port: port}
	}

	reqDir := r.requestsDir(port)
	if err := os.MkdirAll(reqDir, 0755); err != nil {
		return err
	}

	// Add timestamp to request
	if req.Timestamp == "" {
		req.Timestamp = time.Now().Format(time.RFC3339Nano)
	}

	reqFile := filepath.Join(reqDir, r.filenameFor()+".json")
	return writeJSON(reqFile, req)
}

// LoadAll loads all existing imposters from the datadir
// This is called at startup to restore persisted imposters
func (r *FilesystemRepository) LoadAll() ([]*models.Imposter, error) {
	return r.All()
}
