// Package prometheus provides middleware for Prometheus metrics collection
package prometheus

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// unknownEndpoint is used as the endpoint label when no route is matched
	unknownEndpoint = "unknown"
)

// Middleware returns a Gin middleware that collects HTTP metrics for Prometheus
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Calculate duration
		duration := time.Since(start).Seconds()
		
		// Get request and response sizes
		reqSize := computeApproximateRequestSize(c.Request)
		
		// Get response size from the response writer
		// Size() returns -1 if no data has been written, normalize to 0
		respSize := max(0, int64(c.Writer.Size()))
		
		// Normalize endpoint path for metrics to avoid high cardinality
		// Use the matched route pattern from Gin, which handles path parameters
		endpoint := c.FullPath()
		if endpoint == "" {
			// If no route matched, use a generic label to avoid unbounded cardinality
			endpoint = unknownEndpoint
		}
		
		// Record metrics
		RecordHTTPRequest(
			c.Request.Method,
			endpoint,
			c.Writer.Status(),
			duration,
			reqSize,
			respSize,
		)
	}
}

// computeApproximateRequestSize returns an approximation of the HTTP request size in bytes.
// This includes the request line (method + URL + protocol), headers, host, and content body.
// Note: This is an approximation and may not match the exact wire format size because:
// - It doesn't account for HTTP/1.1 vs HTTP/2 formatting differences
// - Header names and values are counted as-is without HTTP wire format overhead
// - r.Form and r.MultipartForm fields are NOT included in this calculation
func computeApproximateRequestSize(r *http.Request) int64 {
	s := int64(0)
	if r.URL != nil {
		s = int64(len(r.URL.Path))
	}

	s += int64(len(r.Method))
	s += int64(len(r.Proto))
	for name, values := range r.Header {
		s += int64(len(name))
		for _, value := range values {
			s += int64(len(value))
		}
	}
	s += int64(len(r.Host))

	// Add content length if available
	if r.ContentLength != -1 {
		s += r.ContentLength
	}
	return s
}
