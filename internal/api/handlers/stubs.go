package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
	"github.com/TetsujinOni/go-tartuffe/internal/response"
)

// StubsHandler handles stub management operations
type StubsHandler struct {
	repo repository.Repository
}

// NewStubsHandler creates a new stubs handler
func NewStubsHandler(repo repository.Repository) *StubsHandler {
	return &StubsHandler{repo: repo}
}

// ReplaceStubs handles PUT /imposters/{id}/stubs
func (h *StubsHandler) ReplaceStubs(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(getParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid port number")
		return
	}

	var req struct {
		Stubs []models.Stub `json:"stubs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeInvalidJSON, "unable to parse body as JSON")
		return
	}

	if req.Stubs == nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "'stubs' is a required field")
		return
	}

	if err := h.repo.UpdateStubs(port, req.Stubs); err != nil {
		if _, ok := err.(repository.ErrNotFound); ok {
			response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource,
				"imposter on port "+strconv.Itoa(port)+" does not exist")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	imp, _ := h.repo.Get(port)
	result := applyOptions(imp, models.SerializeOptions{})

	response.WriteJSON(w, http.StatusOK, result)
}

// AddStub handles POST /imposters/{id}/stubs
func (h *StubsHandler) AddStub(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(getParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid port number")
		return
	}

	var req struct {
		Stub  models.Stub `json:"stub"`
		Index *int        `json:"index,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeInvalidJSON, "unable to parse body as JSON")
		return
	}

	// Determine index
	index := -1
	if req.Index != nil {
		index = *req.Index
	}

	if err := h.repo.AddStub(port, req.Stub, index); err != nil {
		if _, ok := err.(repository.ErrNotFound); ok {
			response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource,
				"imposter on port "+strconv.Itoa(port)+" does not exist")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	imp, _ := h.repo.Get(port)
	result := applyOptions(imp, models.SerializeOptions{})

	response.WriteJSON(w, http.StatusOK, result)
}

// ReplaceStub handles PUT /imposters/{id}/stubs/{stubIndex}
func (h *StubsHandler) ReplaceStub(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(getParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid port number")
		return
	}

	stubIndex, err := strconv.Atoi(getParam(r, "stubIndex"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid stub index")
		return
	}

	var stub models.Stub
	if err := json.NewDecoder(r.Body).Decode(&stub); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeInvalidJSON, "unable to parse body as JSON")
		return
	}

	// Get current imposter to validate index
	imp, err := h.repo.Get(port)
	if err != nil {
		if _, ok := err.(repository.ErrNotFound); ok {
			response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource,
				"imposter on port "+strconv.Itoa(port)+" does not exist")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	if stubIndex < 0 || stubIndex >= len(imp.Stubs) {
		response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource,
			"stub index "+strconv.Itoa(stubIndex)+" does not exist")
		return
	}

	// Replace by deleting and adding
	_ = h.repo.DeleteStub(port, stubIndex)
	_ = h.repo.AddStub(port, stub, stubIndex)

	imp, _ = h.repo.Get(port)
	result := applyOptions(imp, models.SerializeOptions{})

	response.WriteJSON(w, http.StatusOK, result)
}

// DeleteStub handles DELETE /imposters/{id}/stubs/{stubIndex}
func (h *StubsHandler) DeleteStub(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(getParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid port number")
		return
	}

	stubIndex, err := strconv.Atoi(getParam(r, "stubIndex"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid stub index")
		return
	}

	if err := h.repo.DeleteStub(port, stubIndex); err != nil {
		if _, ok := err.(repository.ErrNotFound); ok {
			response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource,
				"imposter on port "+strconv.Itoa(port)+" does not exist")
			return
		}
		if _, ok := err.(repository.ErrInvalidIndex); ok {
			response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource,
				"stub index "+strconv.Itoa(stubIndex)+" does not exist")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	imp, _ := h.repo.Get(port)
	result := applyOptions(imp, models.SerializeOptions{})

	response.WriteJSON(w, http.StatusOK, result)
}
