package api

import (
	"net/http"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/response"
)

// Router is a simple HTTP router with path parameter support
type Router struct {
	routes []route
}

type route struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{}
}

// Handle registers a route
func (rt *Router) Handle(method, pattern string, handler http.HandlerFunc) {
	rt.routes = append(rt.routes, route{method, pattern, handler})
}

// GET registers a GET route
func (rt *Router) GET(pattern string, handler http.HandlerFunc) {
	rt.Handle("GET", pattern, handler)
}

// POST registers a POST route
func (rt *Router) POST(pattern string, handler http.HandlerFunc) {
	rt.Handle("POST", pattern, handler)
}

// PUT registers a PUT route
func (rt *Router) PUT(pattern string, handler http.HandlerFunc) {
	rt.Handle("PUT", pattern, handler)
}

// DELETE registers a DELETE route
func (rt *Router) DELETE(pattern string, handler http.HandlerFunc) {
	rt.Handle("DELETE", pattern, handler)
}

// ServeHTTP implements http.Handler
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range rt.routes {
		if route.method != r.Method {
			continue
		}

		params, ok := match(route.pattern, r.URL.Path)
		if !ok {
			continue
		}

		// Store params in request context via query params hack
		// (In a real implementation we'd use context.WithValue)
		q := r.URL.Query()
		for k, v := range params {
			q.Set("_param_"+k, v)
		}
		r.URL.RawQuery = q.Encode()

		route.handler(w, r)
		return
	}

	// No route matched
	response.WriteError(w, http.StatusNotFound, response.ErrCodeNoSuchResource, "resource not found")
}

// match checks if a path matches a pattern and extracts parameters
// Pattern format: /imposters/{id}/stubs/{stubIndex}
func match(pattern, path string) (map[string]string, bool) {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := make(map[string]string)

	for i, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			// This is a parameter
			paramName := part[1 : len(part)-1]
			params[paramName] = pathParts[i]
		} else if part != pathParts[i] {
			return nil, false
		}
	}

	return params, true
}

// GetParam retrieves a path parameter from the request
func GetParam(r *http.Request, name string) string {
	return r.URL.Query().Get("_param_" + name)
}
