// Package prometheus provides Prometheus metrics for monitoring the application.
package prometheus

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	// Application-specific metrics
	activeKeysTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gpt_load_active_keys_total",
			Help: "Total number of active API keys per group",
		},
		[]string{"group"},
	)

	invalidKeysTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gpt_load_invalid_keys_total",
			Help: "Total number of invalid API keys per group",
		},
		[]string{"group"},
	)

	proxyRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpt_load_proxy_requests_total",
			Help: "Total number of proxy requests per group",
		},
		[]string{"group", "status"},
	)

	proxyRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gpt_load_proxy_request_duration_seconds",
			Help:    "Proxy request duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"group"},
	)

	keyRotationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpt_load_key_rotations_total",
			Help: "Total number of key rotations per group",
		},
		[]string{"group"},
	)

	keyValidationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpt_load_key_validation_total",
			Help: "Total number of key validations",
		},
		[]string{"group", "result"},
	)
)

// Init initializes and registers all Prometheus metrics
func Init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestSize,
		httpResponseSize,
		activeKeysTotal,
		invalidKeysTotal,
		proxyRequestsTotal,
		proxyRequestDuration,
		keyRotationsTotal,
		keyValidationTotal,
	)
}

// Handler returns a Gin handler for the Prometheus metrics endpoint
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint string, status int, duration float64, reqSize, respSize int64) {
	statusStr := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(method, endpoint, statusStr).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint, statusStr).Observe(duration)
	if reqSize > 0 {
		httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(reqSize))
	}
	if respSize > 0 {
		httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(respSize))
	}
}

// SetActiveKeys sets the number of active keys for a group
func SetActiveKeys(group string, count float64) {
	activeKeysTotal.WithLabelValues(group).Set(count)
}

// SetInvalidKeys sets the number of invalid keys for a group
func SetInvalidKeys(group string, count float64) {
	invalidKeysTotal.WithLabelValues(group).Set(count)
}

// RecordProxyRequest records a proxy request
func RecordProxyRequest(group, status string, duration float64) {
	proxyRequestsTotal.WithLabelValues(group, status).Inc()
	proxyRequestDuration.WithLabelValues(group).Observe(duration)
}

// RecordKeyRotation records a key rotation event
func RecordKeyRotation(group string) {
	keyRotationsTotal.WithLabelValues(group).Inc()
}

// RecordKeyValidation records a key validation result
func RecordKeyValidation(group, result string) {
	keyValidationTotal.WithLabelValues(group, result).Inc()
}
