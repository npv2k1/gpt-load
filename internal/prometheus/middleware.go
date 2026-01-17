// Package prometheus provides middleware for Prometheus metrics collection
package prometheus

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin middleware that collects HTTP metrics for Prometheus
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Get request size
		reqSize := computeApproximateRequestSize(c.Request)
		
		// Process request
		c.Next()
		
		// Calculate duration
		duration := time.Since(start).Seconds()
		
		// Get response size from the response writer
		respSize := int64(c.Writer.Size())
		
		// Normalize endpoint path for metrics (remove path parameters)
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
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

// computeApproximateRequestSize approximates the size of the HTTP request
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

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += r.ContentLength
	}
	return s
}
