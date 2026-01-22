package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMiddlewareConfig configures the HTTP metrics middleware.
type HTTPMiddlewareConfig struct {
	// RequestTotal uses labels: method, path, status.
	RequestTotal *prometheus.CounterVec

	// RequestDuration uses labels: method, path, status.
	RequestDuration *prometheus.HistogramVec

	// Skipper allows skipping metrics for specific requests.
	Skipper func(fiber.Ctx) bool

	// DisableRoutePath disables Fiber route path and uses raw path instead.
	DisableRoutePath bool

	// NormalizePath optionally normalizes the final path label.
	NormalizePath func(string) string
}

// HTTPMetricsMiddleware records HTTP request metrics.
func HTTPMetricsMiddleware(cfg *HTTPMiddlewareConfig) fiber.Handler {
	config := &HTTPMiddlewareConfig{}
	if cfg != nil {
		*config = *cfg
	}

	requestTotal := config.RequestTotal
	if requestTotal == nil {
		requestTotal = HTTPRequestTotal
	}
	requestDuration := config.RequestDuration
	if requestDuration == nil {
		requestDuration = HTTPRequestDuration
	}

	return func(c fiber.Ctx) error {
		if config.Skipper != nil && config.Skipper(c) {
			return c.Next()
		}

		start := time.Now()
		err := c.Next()

		if requestTotal == nil && requestDuration == nil {
			return err
		}

		status := c.Response().StatusCode()
		statusLabel := strconv.Itoa(status)
		method := c.Method()
		path := ""
		if !config.DisableRoutePath {
			if route := c.Route(); route != nil {
				path = route.Path
			}
		}
		if path == "" || path == "/" {
			path = c.Path()
		}
		if config.NormalizePath != nil {
			path = config.NormalizePath(path)
		}

		if requestTotal != nil {
			requestTotal.WithLabelValues(method, path, statusLabel).Inc()
		}
		if requestDuration != nil {
			requestDuration.WithLabelValues(method, path, statusLabel).Observe(time.Since(start).Seconds())
		}

		return err
	}
}
