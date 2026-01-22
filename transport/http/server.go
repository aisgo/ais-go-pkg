package http

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"
	"github.com/aisgo/ais-go-pkg/metrics"

	"github.com/gofiber/fiber/v3"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
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

// Config HTTP 服务器配置
type Config struct {
	Port               int           `yaml:"port"`
	Host               string        `yaml:"host"`
	AppName            string        `yaml:"app_name"`
	ReadTimeout        time.Duration `yaml:"read_timeout"`
	WriteTimeout       time.Duration `yaml:"write_timeout"`
	IdleTimeout        time.Duration `yaml:"idle_timeout"`
	HealthCheckTimeout time.Duration `yaml:"health_check_timeout"`

	// EnableRecover 是否启用 Panic 恢复中间件，默认 true（生产环境推荐）
	// 设为 false 可在开发/测试环境直接暴露 panic，便于问题定位
	EnableRecover *bool `yaml:"enable_recover"`

	// Listen 嵌套 ListenConfig 的可序列化配置项
	Listen ListenOptions `yaml:"listen"`
}

// ListenOptions 包含 Fiber ListenConfig 中可以通过 YAML 配置的字段
// 对于更高级的配置（如 TLSConfigFunc、BeforeServeFunc 等函数类型），
// 请使用 ServerParams 中的 ListenConfigCustomizer
type ListenOptions struct {
	// 是否启用 Prefork 模式（多进程），默认 false
	EnablePrefork bool `yaml:"enable_prefork"`

	// 是否禁用启动消息，默认 false
	DisableStartupMessage bool `yaml:"disable_startup_message"`

	// 是否打印所有路由，默认 false
	EnablePrintRoutes bool `yaml:"enable_print_routes"`

	// 监听网络类型（tcp, tcp4, tcp6, unix），默认 tcp4
	// 注意：使用 Prefork 时只能选择 tcp4 或 tcp6
	ListenerNetwork string `yaml:"listener_network"`

	// TLS 证书文件路径
	CertFile string `yaml:"cert_file"`

	// TLS 证书私钥文件路径
	CertKeyFile string `yaml:"cert_key_file"`

	// mTLS 客户端证书文件路径
	CertClientFile string `yaml:"cert_client_file"`

	// 优雅关闭超时时间，默认 10s
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`

	// Unix Socket 文件权限模式，默认 0770
	UnixSocketFileMode uint32 `yaml:"unix_socket_file_mode"`

	// TLS 最低版本，默认 TLS 1.2
	// 可选值: 771 (TLS 1.2), 772 (TLS 1.3)
	TLSMinVersion uint16 `yaml:"tls_min_version"`
}

// ListenConfigCustomizer 自定义 ListenConfig 的函数类型
// 用于配置那些无法通过 YAML 序列化的高级选项（如回调函数、context 等）
type ListenConfigCustomizer func(*fiber.ListenConfig)

// AppConfigCustomizer 自定义 Fiber Config
// 用于配置 Fiber ErrorHandler 或其他高级选项
type AppConfigCustomizer func(*fiber.Config)

type ServerParams struct {
	fx.In
	Lc     fx.Lifecycle
	Config Config
	Logger *logger.Logger
	DB     *gorm.DB `optional:"true"` // 用于健康检查，可选

	// ErrorHandler 可选的 Fiber ErrorHandler
	ErrorHandler fiber.ErrorHandler `optional:"true"`

	// ListenConfigCustomizer 可选的 ListenConfig 自定义函数
	// 使用此函数可以设置更高级的配置，如：
	//   - GracefulContext: 优雅关闭的 context
	//   - TLSConfigFunc: 自定义 TLS 配置函数
	//   - ListenerAddrFunc: 监听地址回调
	//   - BeforeServeFunc: 服务启动前的回调
	//   - AutoCertManager: ACME 自动证书管理器
	ListenConfigCustomizer ListenConfigCustomizer `optional:"true"`

	// AppConfigCustomizer 可选的 Fiber Config 自定义函数
	AppConfigCustomizer AppConfigCustomizer `optional:"true"`
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

	appConfig := fiber.Config{
		AppName:      appName,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	if p.AppConfigCustomizer != nil {
		p.AppConfigCustomizer(&appConfig)
	}
	if p.ErrorHandler != nil {
		appConfig.ErrorHandler = p.ErrorHandler
	}

	app := fiber.New(appConfig)

	// 默认启用 Recover 中间件（生产环境必备，防止 panic 导致服务崩溃）
	// 可通过配置 enable_recover: false 在测试环境禁用，便于问题暴露
	enableRecover := true
	if p.Config.EnableRecover != nil {
		enableRecover = *p.Config.EnableRecover
	}

	if enableRecover {
		app.Use(recoverer.New(recoverer.Config{
			EnableStackTrace: true,
			StackTraceHandler: func(c fiber.Ctx, e interface{}) {
				p.Logger.Error("Panic recovered",
					zap.Any("error", e),
					zap.String("path", c.Path()),
					zap.String("method", c.Method()),
					zap.String("ip", c.IP()),
				)
			},
		}))
	}

	// 注册健康检查端点
	healthCheckTimeout := p.Config.HealthCheckTimeout
	if healthCheckTimeout <= 0 {
		healthCheckTimeout = 2 * time.Second
	}
	registerHealthEndpoints(app, p.DB, healthCheckTimeout)

	// 注册 Prometheus 指标端点
	metrics.RegisterMetricsEndpoint(app)

	p.Lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf(":%d", p.Config.Port)
			if p.Config.Host != "" {
				addr = fmt.Sprintf("%s:%d", p.Config.Host, p.Config.Port)
			}

			// 预先创建 net.Listener，确保端口绑定成功
			listenConfig := buildListenConfig(p.Config.Listen)
			if p.ListenConfigCustomizer != nil {
				p.ListenConfigCustomizer(&listenConfig)
			}

			// 使用 Fiber 的 ListenConfig 创建 listener
			// 注意：Fiber v3 的 Listen 方法内部会创建 listener，我们需要使用 Listener 方法
			listener, err := createListener(addr, listenConfig)
			if err != nil {
				p.Logger.Error("Failed to create HTTP listener", zap.Error(err), zap.String("addr", addr))
				return fmt.Errorf("failed to bind to %s: %w", addr, err)
			}

			p.Logger.Info("HTTP Server listener created successfully", zap.String("addr", addr))

			// 创建 ready channel 用于确认服务器启动
			readyChan := make(chan struct{})
			errChan := make(chan error, 1)

			go func() {
				// 通知已准备就绪（listener 已创建）
				close(readyChan)

				p.Logger.Info("Starting HTTP Server", zap.String("addr", addr))
				if err := app.Listener(listener, listenConfig); err != nil {
					p.Logger.Error("HTTP Server failed", zap.Error(err))
					errChan <- err
				}
			}()

			// 等待 ready 信号或错误
			select {
			case <-readyChan:
				// 服务器已准备就绪（listener 已创建并绑定端口）
				return nil
			case err := <-errChan:
				// 服务器启动失败
				return err
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

// buildListenConfig 根据 ListenOptions 构建 Fiber ListenConfig，并应用默认值
func buildListenConfig(opts ListenOptions) fiber.ListenConfig {
	config := fiber.ListenConfig{
		EnablePrefork:         opts.EnablePrefork,
		DisableStartupMessage: opts.DisableStartupMessage,
		EnablePrintRoutes:     opts.EnablePrintRoutes,
		CertFile:              opts.CertFile,
		CertKeyFile:           opts.CertKeyFile,
		CertClientFile:        opts.CertClientFile,
	}

	// 应用默认值
	if opts.ListenerNetwork != "" {
		config.ListenerNetwork = opts.ListenerNetwork
	} else {
		config.ListenerNetwork = "tcp4" // 默认 tcp4
	}

	if opts.ShutdownTimeout > 0 {
		config.ShutdownTimeout = opts.ShutdownTimeout
	}
	// 注意：Fiber 默认的 ShutdownTimeout 是 10s，这里不设置则使用 Fiber 的默认值

	if opts.UnixSocketFileMode > 0 {
		config.UnixSocketFileMode = os.FileMode(opts.UnixSocketFileMode)
	}
	// 注意：Fiber 默认的 UnixSocketFileMode 是 0770

	if opts.TLSMinVersion > 0 {
		config.TLSMinVersion = opts.TLSMinVersion
	}
	// 注意：Fiber 默认的 TLSMinVersion 是 tls.VersionTLS12

	return config
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

func registerHealthEndpoints(app *fiber.App, db *gorm.DB, timeout time.Duration) {
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
			checkTimeout := timeout
			if checkTimeout <= 0 {
				checkTimeout = 2 * time.Second
			}
			sqlDB, err := db.DB()
			if err != nil {
				checks["database"] = "error: " + err.Error()
				healthy = false
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
				defer cancel()
				if err := sqlDB.PingContext(ctx); err != nil {
					checks["database"] = "error: " + err.Error()
					healthy = false
				} else {
					checks["database"] = "ok"
				}
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
