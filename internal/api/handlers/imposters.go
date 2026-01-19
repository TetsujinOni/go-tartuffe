package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/imposter"
	"github.com/TetsujinOni/go-tartuffe/internal/models"
	"github.com/TetsujinOni/go-tartuffe/internal/repository"
	"github.com/TetsujinOni/go-tartuffe/internal/response"
	"github.com/TetsujinOni/go-tartuffe/internal/web"
)

// ImpostersHandler handles imposter collection operations
type ImpostersHandler struct {
	repo    repository.Repository
	manager *imposter.Manager
	apiPort int
}

// NewImpostersHandler creates a new imposters handler
func NewImpostersHandler(repo repository.Repository, manager *imposter.Manager, apiPort int) *ImpostersHandler {
	return &ImpostersHandler{repo: repo, manager: manager, apiPort: apiPort}
}

// ImpostersResponse is the response for GET /imposters
type ImpostersResponse struct {
	Imposters []*models.Imposter `json:"imposters"`
}

// GetImposters handles GET /imposters
func (h *ImpostersHandler) GetImposters(w http.ResponseWriter, r *http.Request) {
	imposters, err := h.repo.All()
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	// Content negotiation: HTML for browsers, JSON for API clients
	if web.AcceptsHTML(r) {
		summaries := make([]web.ImposterSummary, len(imposters))
		for i, imp := range imposters {
			numReqs := 0
			if imp.NumberOfRequests != nil {
				numReqs = *imp.NumberOfRequests
			}
			summaries[i] = web.ImposterSummary{
				Port:             imp.Port,
				Protocol:         imp.Protocol,
				Name:             imp.Name,
				NumberOfRequests: numReqs,
				SelfHref:         fmt.Sprintf("/imposters/%d", imp.Port),
			}
		}
		data := web.ImpostersPageData{
			PageData: web.PageData{
				Title:       "running imposters",
				Description: "Placeholder description for imposters page.",
			},
			Imposters: summaries,
		}
		web.Render(w, "imposters.html", data)
		return
	}

	// Parse query options
	options := parseOptions(r)

	// Apply options to each imposter
	result := make([]*models.Imposter, len(imposters))
	for i, imp := range imposters {
		result[i] = applyOptionsWithRequest(imp, options, r)
	}

	response.WriteJSON(w, http.StatusOK, ImpostersResponse{Imposters: result})
}

// CreateImposter handles POST /imposters
func (h *ImpostersHandler) CreateImposter(w http.ResponseWriter, r *http.Request) {
	var imp models.Imposter
	if err := json.NewDecoder(r.Body).Decode(&imp); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeInvalidJSON, "Unable to parse body as JSON")
		return
	}

	// Validate required fields
	if imp.Protocol == "" {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "'protocol' is a required field")
		return
	}

	// Validate port (port=0 or missing means auto-assign, negative is invalid, >65535 is invalid)
	if imp.Port < 0 || imp.Port > 65535 {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "'port' must be a valid port number")
		return
	}

	// Check if port conflicts with API server (skip for auto-assign port=0)
	if imp.Port != 0 && imp.Port == h.apiPort {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeResourceConflict,
			"port "+strconv.Itoa(imp.Port)+" is already in use")
		return
	}

	// Validate protocol
	validProtocols := map[string]bool{"http": true, "https": true, "tcp": true, "smtp": true}
	if !validProtocols[imp.Protocol] {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "unsupported protocol: "+imp.Protocol)
		return
	}

	// Initialize stubs if nil
	if imp.Stubs == nil {
		imp.Stubs = []models.Stub{}
	}

	// Initialize request counter
	if imp.NumberOfRequests == nil {
		count := 0
		imp.NumberOfRequests = &count
	}

	// For TCP imposters, set defaults and validate
	if imp.Protocol == "tcp" {
		// Set default mode to "text" if not specified
		if imp.Mode == "" {
			imp.Mode = "text"
		}

		// Initialize requests array if nil (for API response consistency)
		if imp.Requests == nil {
			imp.Requests = []models.Request{}
		}

		// Validate matches predicate is not used in binary mode
		if imp.Mode == "binary" {
			for _, stub := range imp.Stubs {
				for _, pred := range stub.Predicates {
					if pred.Matches != nil {
						response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData,
							"the matches predicate is not allowed in binary mode")
						return
					}
				}
			}
		}

		// Validate proxy responses use TCP protocol
		for _, stub := range imp.Stubs {
			for _, resp := range stub.Responses {
				if resp.Proxy != nil && resp.Proxy.To != "" {
					// Validate that proxy.to uses tcp:// protocol
					proxyURL := resp.Proxy.To
					if !strings.HasPrefix(proxyURL, "tcp://") {
						response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData,
							"cannot proxy to "+proxyURL+" using tcp protocol")
						return
					}
				}
			}
		}
	}

	// For HTTPS imposters, extract certificate metadata
	if imp.Protocol == "https" {
		imp.ExtractCertMetadata()
	}

	// Start the imposter server first (HTTP, HTTPS, TCP, or SMTP)
	// This must happen before adding to repository so that auto-assigned port (port=0) is resolved
	if (imp.Protocol == "http" || imp.Protocol == "https" || imp.Protocol == "tcp" || imp.Protocol == "smtp") && h.manager != nil {
		if err := h.manager.Start(&imp); err != nil {
			response.WriteError(w, http.StatusBadRequest, response.ErrCodeResourceConflict,
				"cannot start server: "+err.Error())
			return
		}
	}

	// Now imp.Port has the actual port (auto-assigned if it was 0)
	// Add to repository with the actual port
	if err := h.repo.Add(&imp); err != nil {
		// Failed to add to repository, stop the server we just started
		if h.manager != nil {
			h.manager.Stop(imp.Port)
		}
		if _, ok := err.(repository.ErrConflict); ok {
			response.WriteError(w, http.StatusBadRequest, response.ErrCodeResourceConflict,
				"port "+strconv.Itoa(imp.Port)+" is already in use")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	// Add location header
	baseURL := buildBaseURL(r)
	w.Header().Set("Location", baseURL+"/imposters/"+strconv.Itoa(imp.Port))

	// Return created imposter with links
	result := applyOptionsWithRequest(&imp, models.SerializeOptions{}, r)
	response.WriteJSON(w, http.StatusCreated, result)
}

// DeleteImposters handles DELETE /imposters
func (h *ImpostersHandler) DeleteImposters(w http.ResponseWriter, r *http.Request) {
	imposters, err := h.repo.DeleteAll()
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, response.ErrCodeBadData, err.Error())
		return
	}

	// Stop all imposter servers
	if h.manager != nil {
		h.manager.StopAll()
	}

	options := parseOptions(r)
	// Default to replayable mode for DELETE
	if r.URL.Query().Get("replayable") == "" {
		options.Replayable = true
	}

	result := make([]*models.Imposter, len(imposters))
	for i, imp := range imposters {
		result[i] = applyOptionsWithRequest(imp, options, r)
	}

	response.WriteJSON(w, http.StatusOK, ImpostersResponse{Imposters: result})
}

// ReplaceImposters handles PUT /imposters
func (h *ImpostersHandler) ReplaceImposters(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Imposters []models.Imposter `json:"imposters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeInvalidJSON, "Unable to parse body as JSON")
		return
	}

	if req.Imposters == nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "'imposters' is a required field")
		return
	}

	// Validate all imposters first
	for _, imp := range req.Imposters {
		if imp.Protocol == "" {
			response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "'protocol' is a required field")
			return
		}
		if imp.Port <= 0 || imp.Port > 65535 {
			response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, "'port' must be a valid port number")
			return
		}
	}

	// Delete all existing imposters and stop servers
	h.repo.DeleteAll()
	if h.manager != nil {
		h.manager.StopAll()
	}

	// Create new imposters
	result := make([]*models.Imposter, len(req.Imposters))
	for i := range req.Imposters {
		imp := &req.Imposters[i]
		if imp.Stubs == nil {
			imp.Stubs = []models.Stub{}
		}

		// Initialize request counter
		if imp.NumberOfRequests == nil {
			count := 0
			imp.NumberOfRequests = &count
		}

		// For HTTPS imposters, extract certificate metadata
		if imp.Protocol == "https" {
			imp.ExtractCertMetadata()
		}

		if err := h.repo.Add(imp); err != nil {
			response.WriteError(w, http.StatusBadRequest, response.ErrCodeBadData, err.Error())
			return
		}

		// Start imposter server (HTTP, HTTPS, TCP, or SMTP)
		if (imp.Protocol == "http" || imp.Protocol == "https" || imp.Protocol == "tcp" || imp.Protocol == "smtp") && h.manager != nil {
			h.manager.Start(imp)
		}

		result[i] = applyOptionsWithRequest(imp, models.SerializeOptions{}, r)
	}

	response.WriteJSON(w, http.StatusOK, ImpostersResponse{Imposters: result})
}

// parseOptions extracts serialization options from query parameters
func parseOptions(r *http.Request) models.SerializeOptions {
	return models.SerializeOptions{
		Replayable:    r.URL.Query().Get("replayable") == "true",
		RemoveProxies: r.URL.Query().Get("removeProxies") == "true",
	}
}

// buildBaseURL constructs the base URL from the request
func buildBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header for proxy scenarios
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	if host == "" {
		host = "localhost:2525"
	}

	return scheme + "://" + host
}

// applyOptions creates a copy of the imposter with options applied
func applyOptions(imp *models.Imposter, options models.SerializeOptions) *models.Imposter {
	return applyOptionsWithRequest(imp, options, nil)
}

// applyOptionsWithRequest creates a copy of the imposter with options applied, using request for absolute URLs
func applyOptionsWithRequest(imp *models.Imposter, options models.SerializeOptions, r *http.Request) *models.Imposter {
	// Create a shallow copy
	result := *imp

	// Build base URL if request is provided
	baseURL := ""
	if r != nil {
		baseURL = buildBaseURL(r)
	}

	// In replayable mode, exclude requests, links, and request counter
	if options.Replayable {
		result.Requests = nil
		result.TCPRequests = nil
		result.SMTPRequests = nil
		result.GRPCRequests = nil
		result.Links = nil
		result.NumberOfRequests = nil
	} else {
		// Add links only in non-replayable mode
		result.Links = &models.Links{
			Self:  &models.Link{Href: baseURL + "/imposters/" + strconv.Itoa(imp.Port)},
			Stubs: &models.Link{Href: baseURL + "/imposters/" + strconv.Itoa(imp.Port) + "/stubs"},
		}

		// Ensure requests array exists in non-replayable mode (even if empty)
		// This is required for mountebank compatibility
		// The MarshalJSON will serialize the appropriate field as "requests" for each protocol
		switch result.Protocol {
		case "tcp":
			if result.TCPRequests == nil {
				result.TCPRequests = []models.TCPRequest{}
			}
		case "smtp":
			if result.SMTPRequests == nil {
				result.SMTPRequests = []models.SMTPRequest{}
			}
		case "grpc":
			if result.GRPCRequests == nil {
				result.GRPCRequests = []models.GRPCRequest{}
			}
		default:
			if result.Requests == nil {
				result.Requests = []models.Request{}
			}
		}
	}

	// Remove proxy responses if requested (but keep stubs with non-proxy responses)
	if options.RemoveProxies && len(result.Stubs) > 0 {
		filtered := make([]models.Stub, 0, len(result.Stubs))
		for _, stub := range result.Stubs {
			// Filter out proxy responses from this stub
			nonProxyResponses := make([]models.Response, 0, len(stub.Responses))
			for _, resp := range stub.Responses {
				if resp.Proxy == nil {
					nonProxyResponses = append(nonProxyResponses, resp)
				}
			}
			// Only keep stub if it has non-proxy responses
			if len(nonProxyResponses) > 0 {
				stubCopy := stub
				stubCopy.Responses = nonProxyResponses
				filtered = append(filtered, stubCopy)
			}
		}
		result.Stubs = filtered
	}

	// Add links to each stub (skip in replayable mode)
	if !options.Replayable && len(result.Stubs) > 0 {
		stubsWithLinks := make([]models.Stub, len(result.Stubs))
		for i, stub := range result.Stubs {
			stubsWithLinks[i] = stub
			stubsWithLinks[i].Links = &models.StubLinks{
				Self: &models.Link{Href: baseURL + "/imposters/" + strconv.Itoa(imp.Port) + "/stubs/" + strconv.Itoa(i)},
			}
		}
		result.Stubs = stubsWithLinks
	}

	// For HTTPS imposters, never return the private key in API responses
	// Keep certificate metadata for transparency
	if result.Protocol == "https" {
		result.Key = "" // Never expose private key material
	}

	return &result
}

// getParam retrieves a path parameter from the request
// Parameters are stored as _param_name in query string by router
func getParam(r *http.Request, name string) string {
	return r.URL.Query().Get("_param_" + name)
}
