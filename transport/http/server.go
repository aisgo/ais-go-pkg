package http

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"ais.local/ais-go-pkg/logger"
	"ais.local/ais-go-pkg/metrics"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

/* ========================================================================
 * HTTP Server - Fiber v3 HTTP 服务器
 * ========================================================================
 * 职责: 提供 HTTP 服务，健康检查，指标暴露
 * 技术: Fiber v3
 * ======================================================================== */

type Config struct {
	Port         int           `yaml:"port"`
	AppName      string        `yaml:"app_name"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type ServerParams struct {
	fx.In
	Lc     fx.Lifecycle
	Config Config
	Logger *logger.Logger
	DB     *gorm.DB `optional:"true"` // 用于健康检查，可选
}

// NewHTTPServer 创建 HTTP 服务器并注册生命周期
func NewHTTPServer(p ServerParams) *fiber.App {
	// 应用默认值
	readTimeout := p.Config.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	writeTimeout := p.Config.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}
	idleTimeout := p.Config.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = 120 * time.Second
	}
	appName := p.Config.AppName
	if appName == "" {
		appName = "AIS Go App"
	}

	app := fiber.New(fiber.Config{
		AppName:      appName,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	})

	// 注册健康检查端点
	registerHealthEndpoints(app, p.DB)

	// 注册 Prometheus 指标端点
	metrics.RegisterMetricsEndpoint(app)

	p.Lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 创建 channel 用于传递启动错误
			errChan := make(chan error, 1)

			go func() {
				addr := fmt.Sprintf(":%d", p.Config.Port)
				p.Logger.Info("Starting HTTP Server", zap.String("addr", addr))
				if err := app.Listen(addr); err != nil {
					p.Logger.Error("HTTP Server failed to start", zap.Error(err))
					errChan <- err
				}
			}()

			// 等待一小段时间，确保服务器成功启动
			select {
			case err := <-errChan:
				// 服务器立即启动失败
				return err
			case <-time.After(100 * time.Millisecond):
				// 服务器似乎成功启动
				return nil
			case <-ctx.Done():
				// 上下文被取消
				return ctx.Err()
			}
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("Stopping HTTP Server")
			return app.ShutdownWithContext(ctx)
		},
	})

	return app
}

/* ========================================================================
 * Health Check Endpoints
 * ========================================================================
 * /healthz - 存活探针 (Liveness Probe)
 *   - 用于 K8s 判断容器是否存活
 *   - 只要进程能响应就返回 200
 *
 * /readyz - 就绪探针 (Readiness Probe)
 *   - 用于 K8s 判断容器是否可以接收流量
 *   - 需要检查数据库等依赖是否就绪
 * ======================================================================== */

func registerHealthEndpoints(app *fiber.App, db *gorm.DB) {
	// 存活探针 - 简单返回 OK
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 就绪探针 - 检查依赖
	app.Get("/readyz", func(c fiber.Ctx) error {
		checks := make(map[string]string)
		healthy := true

		// 检查数据库连接
		if db != nil {
			sqlDB, err := db.DB()
			if err != nil {
				checks["database"] = "error: " + err.Error()
				healthy = false
			} else if err := sqlDB.Ping(); err != nil {
				checks["database"] = "error: " + err.Error()
				healthy = false
			} else {
				checks["database"] = "ok"
			}
		}

		// 内存使用情况
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		checks["memory_alloc_mb"] = fmt.Sprintf("%.2f", float64(m.Alloc)/1024/1024)
		checks["goroutines"] = fmt.Sprintf("%d", runtime.NumGoroutine())

		status := "ok"
		statusCode := fiber.StatusOK
		if !healthy {
			status = "unhealthy"
			statusCode = fiber.StatusServiceUnavailable
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status": status,
			"time":   time.Now().Format(time.RFC3339),
			"checks": checks,
		})
	})
}
