package handlers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler handles metrics operations
type MetricsHandler struct {
	handler http.Handler
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		handler: promhttp.Handler(),
	}
}

// GetMetrics handles GET /metrics
// Returns Prometheus-formatted metrics
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}
