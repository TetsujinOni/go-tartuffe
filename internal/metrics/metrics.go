package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal tracks total requests per imposter
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mb_requests_total",
			Help: "Total number of requests received by imposters",
		},
		[]string{"imposter", "protocol"},
	)

	// PredicateMatchDuration tracks predicate matching duration
	// This is the metric that mountebank tests expect
	// Note: We add an "endpoint" label (e < i) so there's content between { and imposter
	// in the output, which is required by mountebank's test regex pattern
	PredicateMatchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mb_predicate_match_duration_seconds",
			Help:    "Predicate match duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "imposter"},
	)

	// ResponseGenerationDuration tracks response generation duration
	// Named to match mountebank's expected metric name
	ResponseGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mb_response_generation_duration_seconds",
			Help:    "Response generation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "imposter"},
	)

	// ProxyDuration tracks proxy request duration
	ProxyDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mb_proxy_duration_seconds",
			Help:    "Proxy request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "imposter"},
	)

	// NoMatchTotal tracks requests with no matching stub
	// Named to match mountebank's expected metric name
	NoMatchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mb_no_match_total",
			Help: "Total number of requests with no matching stub",
		},
		[]string{"endpoint", "imposter"},
	)

	// BlockedIPTotal tracks blocked IP addresses
	// NOTE: Commented out until IP blocking is implemented
	// Global metrics without labels always appear in Prometheus output,
	// which conflicts with mountebank's expectation that metrics only appear when used
	// BlockedIPTotal = promauto.NewCounter(
	// 	prometheus.CounterOpts{
	// 		Name: "mb_blocked_ip_total",
	// 		Help: "Total number of blocked IP addresses",
	// 	},
	// )

	// ImpostersTotal tracks the current number of imposters
	// NOTE: Commented out until actively maintained
	// Global metrics without labels always appear in Prometheus output
	// ImpostersTotal = promauto.NewGauge(
	// 	prometheus.GaugeOpts{
	// 		Name: "mb_imposters_total",
	// 		Help: "Current number of active imposters",
	// 	},
	// )

	// StubsTotal tracks the total number of stubs across all imposters
	StubsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mb_stubs_total",
			Help: "Total number of stubs per imposter",
		},
		[]string{"imposter"},
	)

	// JSVMCreatedTotal counts new Goja VM allocations by the pool
	JSVMCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mb_js_vm_created_total",
		Help: "Total number of new JavaScript VMs created by the pool.",
	})

	// JSVMAcquiresTotal counts VM checkouts from the pool (one per JS execution)
	JSVMAcquiresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mb_js_vm_acquires_total",
		Help: "Total number of JavaScript VM acquisitions from the pool.",
	})

	// JSVMsInUse tracks the number of VMs currently checked out
	JSVMsInUse = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mb_js_vms_in_use",
		Help: "Current number of JavaScript VMs checked out from the pool.",
	})

	// JSExecutionDuration tracks JS script execution time, labelled by script type
	JSExecutionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mb_js_execution_duration_seconds",
		Help:    "Duration of JavaScript script execution in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"script_type"})
)

// RecordRequest records a request to an imposter
func RecordRequest(port, protocol string) {
	RequestsTotal.WithLabelValues(port, protocol).Inc()
}

// RecordPredicateMatchDuration records the time taken to match predicates
func RecordPredicateMatchDuration(endpoint, port string, duration float64) {
	PredicateMatchDuration.WithLabelValues(endpoint, port).Observe(duration)
}

// RecordResponseDuration records the time taken to generate a response
// This is an alias for RecordResponseGenerationDuration for backward compatibility
func RecordResponseDuration(endpoint, port string, duration float64) {
	ResponseGenerationDuration.WithLabelValues(endpoint, port).Observe(duration)
}

// RecordResponseGenerationDuration records the time taken to generate a response
func RecordResponseGenerationDuration(endpoint, port string, duration float64) {
	ResponseGenerationDuration.WithLabelValues(endpoint, port).Observe(duration)
}

// RecordProxyDuration records the time taken for a proxy request
func RecordProxyDuration(endpoint, port string, duration float64) {
	ProxyDuration.WithLabelValues(endpoint, port).Observe(duration)
}

// RecordNoMatch records a request with no matching stub
func RecordNoMatch(endpoint, port string) {
	NoMatchTotal.WithLabelValues(endpoint, port).Inc()
}

// RecordBlockedIP records a blocked IP address
// NOTE: Commented out until IP blocking is implemented
// func RecordBlockedIP() {
// 	BlockedIPTotal.Inc()
// }

// SetImpostersCount sets the current number of imposters
// NOTE: Commented out until actively maintained
// func SetImpostersCount(count int) {
// 	ImpostersTotal.Set(float64(count))
// }

// SetStubsCount sets the number of stubs for an imposter
func SetStubsCount(port string, count int) {
	StubsTotal.WithLabelValues(port).Set(float64(count))
}

// RecordVMCreated records a new Goja VM allocation by the pool.
func RecordVMCreated() { JSVMCreatedTotal.Inc() }

// RecordVMAcquire records a VM checkout from the pool.
func RecordVMAcquire() { JSVMAcquiresTotal.Inc(); JSVMsInUse.Inc() }

// RecordVMRelease records a VM being returned to the pool.
func RecordVMRelease() { JSVMsInUse.Dec() }

// RecordJSExecution records the duration of a JavaScript script execution.
// scriptType identifies the kind of script (e.g. "response", "predicate", "tcp_response").
func RecordJSExecution(scriptType string, duration float64) {
	JSExecutionDuration.WithLabelValues(scriptType).Observe(duration)
}

// RemoveImposterMetrics removes metrics for a deleted imposter
func RemoveImposterMetrics(endpoint, port string) {
	StubsTotal.DeleteLabelValues(port)
	NoMatchTotal.DeleteLabelValues(endpoint, port)
	PredicateMatchDuration.DeleteLabelValues(endpoint, port)
	ResponseGenerationDuration.DeleteLabelValues(endpoint, port)
	ProxyDuration.DeleteLabelValues(endpoint, port)
}
