package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/response"
)

// Logger middleware logs requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for static assets
		if strings.HasPrefix(r.URL.Path, "/public/") ||
			strings.HasPrefix(r.URL.Path, "/node_modules/") {
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// CORS middleware adds CORS headers
func CORS(next http.Handler) http.Handler {
	return CORSWithOrigin("*")(next)
}

// CORSWithOrigin middleware adds CORS headers with a specific origin
func CORSWithOrigin(origin string) func(http.Handler) http.Handler {
	if origin == "" {
		origin = "*"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyAuth middleware validates API key
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			providedKey := r.Header.Get("X-Api-Key")
			if providedKey == "" {
				providedKey = r.URL.Query().Get("apikey")
			}

			if providedKey != apiKey {
				response.WriteError(w, http.StatusUnauthorized, "unauthorized", "API key required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPWhitelist middleware validates client IP against whitelist
func IPWhitelist(whitelist string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Parse whitelist
		allowedIPs := strings.Split(whitelist, "|")
		allowAll := false
		for _, ip := range allowedIPs {
			if ip == "*" {
				allowAll = true
				break
			}
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allowAll {
				next.ServeHTTP(w, r)
				return
			}

			// Get client IP
			clientIP := r.RemoteAddr
			if host, _, err := net.SplitHostPort(clientIP); err == nil {
				clientIP = host
			}

			// Check if allowed
			for _, ip := range allowedIPs {
				if ip == clientIP {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.WriteError(w, http.StatusForbidden, "forbidden", "IP not allowed")
		})
	}
}

// LocalOnly middleware only allows localhost connections
func LocalOnly(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := r.RemoteAddr
			if host, _, err := net.SplitHostPort(clientIP); err == nil {
				clientIP = host
			}

			// Check if localhost
			if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
				next.ServeHTTP(w, r)
				return
			}

			response.WriteError(w, http.StatusForbidden, "forbidden", "only localhost connections allowed")
		})
	}
}

// JSONBody middleware parses JSON request bodies
func JSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only process for methods that typically have bodies
		if r.Method == "POST" || r.Method == "PUT" {
			contentType := r.Header.Get("Content-Type")
			if strings.HasPrefix(contentType, "application/json") || contentType == "" {
				// Read body
				body, err := io.ReadAll(r.Body)
				r.Body.Close()
				if err != nil {
					response.WriteError(w, http.StatusBadRequest, response.ErrCodeInvalidJSON, "error reading request body")
					return
				}

				// Validate JSON if body is not empty
				if len(body) > 0 {
					if !json.Valid(body) {
						response.WriteError(w, http.StatusBadRequest, response.ErrCodeInvalidJSON, "unable to parse body as JSON")
						return
					}
				}

				// Put body back for handlers to read
				r.Body = io.NopCloser(bytes.NewReader(body))
			}
		}

		next.ServeHTTP(w, r)
	})
}
