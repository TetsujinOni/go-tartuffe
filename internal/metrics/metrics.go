package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal tracks total requests per imposter
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mountebank",
			Name:      "requests_total",
			Help:      "Total number of requests received by imposters",
		},
		[]string{"port", "protocol"},
	)

	// ResponseDuration tracks response generation duration
	ResponseDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mountebank",
			Name:      "response_duration_seconds",
			Help:      "Response generation duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"port"},
	)

	// ProxyDuration tracks proxy request duration
	ProxyDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mountebank",
			Name:      "proxy_duration_seconds",
			Help:      "Proxy request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"port"},
	)

	// NoMatchTotal tracks requests with no matching stub
	NoMatchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mountebank",
			Name:      "no_match_total",
			Help:      "Total number of requests with no matching stub",
		},
		[]string{"port"},
	)

	// ImpostersTotal tracks the current number of imposters
	ImpostersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "mountebank",
			Name:      "imposters_total",
			Help:      "Current number of active imposters",
		},
	)

	// StubsTotal tracks the total number of stubs across all imposters
	StubsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "mountebank",
			Name:      "stubs_total",
			Help:      "Total number of stubs per imposter",
		},
		[]string{"port"},
	)
)

// RecordRequest records a request to an imposter
func RecordRequest(port, protocol string) {
	RequestsTotal.WithLabelValues(port, protocol).Inc()
}

// RecordResponseDuration records the time taken to generate a response
func RecordResponseDuration(port string, duration float64) {
	ResponseDuration.WithLabelValues(port).Observe(duration)
}

// RecordProxyDuration records the time taken for a proxy request
func RecordProxyDuration(port string, duration float64) {
	ProxyDuration.WithLabelValues(port).Observe(duration)
}

// RecordNoMatch records a request with no matching stub
func RecordNoMatch(port string) {
	NoMatchTotal.WithLabelValues(port).Inc()
}

// SetImpostersCount sets the current number of imposters
func SetImpostersCount(count int) {
	ImpostersTotal.Set(float64(count))
}

// SetStubsCount sets the number of stubs for an imposter
func SetStubsCount(port string, count int) {
	StubsTotal.WithLabelValues(port).Set(float64(count))
}

// RemoveImposterMetrics removes metrics for a deleted imposter
func RemoveImposterMetrics(port string) {
	StubsTotal.DeleteLabelValues(port)
}
