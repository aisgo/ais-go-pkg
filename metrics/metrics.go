package metrics

import (
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

/* ========================================================================
 * Prometheus Metrics - 可观测性指标
 * ========================================================================
 * 职责: 提供 Prometheus 指标注册和暴露
 * ======================================================================== */

var (
	// HTTPRequestDuration HTTP 请求延迟
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "app",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestTotal HTTP 请求总数
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "app",
			Subsystem: "http",
			Name:      "request_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// GRPCRequestDuration gRPC 请求延迟
	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "app",
			Subsystem: "grpc",
			Name:      "request_duration_seconds",
			Help:      "gRPC request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	// GRPCRequestTotal gRPC 请求总数
	GRPCRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "app",
			Subsystem: "grpc",
			Name:      "request_total",
			Help:      "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	// DBQueryDuration 数据库查询延迟
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "app",
			Subsystem: "db",
			Name:      "query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// CacheHitTotal 缓存命中次数
	CacheHitTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "app",
			Subsystem: "cache",
			Name:      "hit_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache_name", "hit"}, // hit: true, false
	)
)

// RegisterMetricsEndpoint 注册 /metrics 端点
func RegisterMetricsEndpoint(app *fiber.App) {
	// 使用 fasthttpadaptor 将 promhttp.Handler 适配到 Fiber
	handler := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
	app.Get("/metrics", func(c fiber.Ctx) error {
		handler(c.RequestCtx())
		return nil
	})
}

// NewCounter 创建自定义 Counter
func NewCounter(namespace, subsystem, name, help string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

// NewGauge 创建自定义 Gauge
func NewGauge(namespace, subsystem, name, help string, labels []string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

// NewHistogram 创建自定义 Histogram
func NewHistogram(namespace, subsystem, name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	if buckets == nil {
		buckets = prometheus.DefBuckets
	}
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)
}
