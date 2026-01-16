package handlers

import (
	"net/http"
	"strconv"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
	"github.com/TetsujinOni/go-tartuffe/internal/response"
)

// ImposterHandler handles individual imposter operations
type ImposterHandler struct {
	repo    repository.Repository
	manager *imposter.Manager
}

// NewImposterHandler creates a new imposter handler
func NewImposterHandler(repo repository.Repository, manager *imposter.Manager) *ImposterHandler {
	return &ImposterHandler{repo: repo, manager: manager}
}

// GetImposter handles GET /imposters/{id}
func (h *ImposterHandler) GetImposter(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(getParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid port number")
		return
	}

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

	options := parseOptions(r)
	result := applyOptionsWithRequest(imp, options, r)

	response.WriteJSON(w, http.StatusOK, result)
}

// DeleteImposter handles DELETE /imposters/{id}
func (h *ImposterHandler) DeleteImposter(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(getParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid port number")
		return
	}

	imp, err := h.repo.Delete(port)
	if err != nil {
		if _, ok := err.(repository.ErrNotFound); ok {
			// DELETE is idempotent - return 200 with empty object for non-existent imposters
			response.WriteJSON(w, http.StatusOK, map[string]interface{}{})
			return
		}
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	// Stop the imposter server
	if h.manager != nil {
		h.manager.Stop(port)
	}

	options := parseOptions(r)
	if r.URL.Query().Get("replayable") == "" {
		options.Replayable = true
	}
	result := applyOptionsWithRequest(imp, options, r)

	response.WriteJSON(w, http.StatusOK, result)
}

// ResetRequests handles DELETE /imposters/{id}/savedRequests
func (h *ImposterHandler) ResetRequests(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(getParam(r, "id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "invalid port number")
		return
	}

	if err := h.repo.ClearRequests(port); err != nil {
		if _, ok := err.(repository.ErrNotFound); ok {
			response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource,
				"imposter on port "+strconv.Itoa(port)+" does not exist")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	// Also reset counter in the running server
	if h.manager != nil {
		if srv := h.manager.GetServer(port); srv != nil {
			srv.ResetRequestCount()
		}
	}

	imp, _ := h.repo.Get(port)
	result := applyOptionsWithRequest(imp, models.SerializeOptions{}, r)

	response.WriteJSON(w, http.StatusOK, result)
}
