# HTTP Transport Module

基于 Fiber v3 的 HTTP 服务器模块，提供健康检查、Prometheus 指标暴露等功能。

## 功能特性

- ✅ 基于 Fiber v3 的高性能 HTTP 服务器
- ✅ 内置健康检查端点（`/healthz`, `/readyz`）
- ✅ Prometheus 指标暴露
- ✅ 灵活的配置系统（YAML + 代码）
- ✅ 完整的生命周期管理（基于 fx）
- ✅ 优雅关闭支持

## 配置方式

该模块提供了两种配置方式，以满足不同场景的需求：

### 1. YAML 配置（推荐用于常规场景）

适用于大部分可序列化的配置项，可以通过配置文件进行设置。

```yaml
http:
  port: 8080
  app_name: "My Application"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  
  # ListenConfig 配置
  listen:
    # 基础配置
    enable_prefork: false          # 是否启用多进程模式（生产环境可设为 true）
    disable_startup_message: false # 是否禁用启动消息
    enable_print_routes: false     # 是否打印所有路由
    listener_network: "tcp4"       # 监听网络类型: tcp, tcp4, tcp6, unix
    
    # TLS 配置
    cert_file: ""                  # TLS 证书文件路径
    cert_key_file: ""              # TLS 证书私钥文件路径
    cert_client_file: ""           # mTLS 客户端证书文件路径
    tls_min_version: 771           # TLS 最低版本: 771 (TLS 1.2), 772 (TLS 1.3)
    
    # 优雅关闭配置
    shutdown_timeout: 10s          # 优雅关闭超时时间
    
    # Unix Socket 配置
    unix_socket_file_mode: 0770    # Unix Socket 文件权限模式（八进制）
```

### 2. 代码自定义（用于高级场景）

对于无法通过 YAML 序列化的高级配置（如回调函数、context 等），可以通过 `ListenConfigCustomizer` 进行自定义。

```go
package main

import (
    "context"
    "crypto/tls"
    
    "github.com/aisgo/ais-go-pkg/transport/http"
    "github.com/gofiber/fiber/v3"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        // ... 其他模块
        
        // 提供 HTTP 服务器
        fx.Provide(http.NewHTTPServer),
        
        // 提供自定义的 ListenConfigCustomizer
        fx.Provide(func() http.ListenConfigCustomizer {
            return func(cfg *fiber.ListenConfig) {
                // 设置优雅关闭的 context
                cfg.GracefulContext = context.Background()
                
                // 自定义 TLS 配置
                cfg.TLSConfigFunc = func(tlsConfig *tls.Config) {
                    tlsConfig.MinVersion = tls.VersionTLS13
                    tlsConfig.CipherSuites = []uint16{
                        tls.TLS_AES_128_GCM_SHA256,
                        tls.TLS_AES_256_GCM_SHA384,
                    }
                }
                
                // 监听地址回调
                cfg.ListenerAddrFunc = func(addr net.Addr) {
                    log.Printf("Server is listening on: %s", addr.String())
                }
                
                // 服务启动前的回调
                cfg.BeforeServeFunc = func(app *fiber.App) error {
                    log.Println("Setting up routes...")
                    // 在这里可以注册额外的路由或中间件
                    return nil
                }
                
                // ACME 自动证书管理（Let's Encrypt）
                // cfg.AutoCertManager = &autocert.Manager{
                //     Prompt:      autocert.AcceptTOS,
                //     Cache:       autocert.DirCache("./certs"),
                //     HostPolicy:  autocert.HostWhitelist("example.com"),
                // }
            }
        }),
    ).Run()
}
```

## 使用示例

### 基础使用

```go
package main

import (
    "github.com/aisgo/ais-go-pkg/config"
    "github.com/aisgo/ais-go-pkg/logger"
    "github.com/aisgo/ais-go-pkg/transport/http"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

func main() {
    fx.New(
        // 提供配置
        fx.Provide(config.LoadConfig),
        
        // 提供 Logger
        fx.Provide(logger.NewLogger),
        
        // 提供数据库（可选，用于健康检查）
        fx.Provide(setupDatabase),
        
        // 提供 HTTP 服务器
        fx.Provide(http.NewHTTPServer),
        
        // 注册路由
        fx.Invoke(registerRoutes),
    ).Run()
}

func setupDatabase() (*gorm.DB, error) {
    // 数据库初始化逻辑
    return nil, nil
}

func registerRoutes(app *fiber.App) {
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendString("Hello, World!")
    })
    
    api := app.Group("/api/v1")
    api.Get("/users", handleGetUsers)
    api.Post("/users", handleCreateUser)
}
```

### 高级使用（结合 YAML + 代码自定义）

```go
package main

import (
    "context"
    "time"
    
    "github.com/aisgo/ais-go-pkg/config"
    "github.com/aisgo/ais-go-pkg/logger"
    "github.com/aisgo/ais-go-pkg/transport/http"
    "github.com/gofiber/fiber/v3"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        fx.Provide(config.LoadConfig),
        fx.Provide(logger.NewLogger),
        fx.Provide(http.NewHTTPServer),
        
        // 提供自定义的 ListenConfigCustomizer
        fx.Provide(NewListenConfigCustomizer),
        
        fx.Invoke(registerRoutes),
    ).Run()
}

// NewListenConfigCustomizer 创建自定义的 ListenConfig 配置函数
func NewListenConfigCustomizer(logger *logger.Logger) http.ListenConfigCustomizer {
    return func(cfg *fiber.ListenConfig) {
        // 设置优雅关闭的 context，超时 30 秒
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        cfg.GracefulContext = ctx
        
        // 在服务启动前执行一些初始化操作
        cfg.BeforeServeFunc = func(app *fiber.App) error {
            logger.Info("Performing pre-start initialization...")
            
            // 注册全局中间件
            app.Use(func(c fiber.Ctx) error {
                logger.Debug("Request received", 
                    zap.String("method", c.Method()),
                    zap.String("path", c.Path()),
                )
                return c.Next()
            })
            
            return nil
        }
        
        // 监听地址回调，记录实际监听的地址
        cfg.ListenerAddrFunc = func(addr net.Addr) {
            logger.Info("HTTP Server started", 
                zap.String("address", addr.String()),
            )
        }
    }
}

func registerRoutes(app *fiber.App) {
    app.Get("/api/hello", func(c fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "message": "Hello from AIS!",
        })
    })
}
```

## 健康检查端点

### 存活探针 - `/healthz`

用于 Kubernetes 判断容器是否存活，只要进程能响应就返回 200。

**响应示例：**
```json
{
  "status": "ok",
  "time": "2026-01-15T12:00:00+08:00"
}
```

### 就绪探针 - `/readyz`

用于 Kubernetes 判断容器是否可以接收流量，会检查数据库等依赖是否就绪。

**响应示例：**
```json
{
  "status": "ok",
  "time": "2026-01-15T12:00:00+08:00",
  "checks": {
    "database": "ok",
    "memory_alloc_mb": "128.45",
    "goroutines": "42"
  }
}
```

如果依赖不健康，返回 503：
```json
{
  "status": "unhealthy",
  "time": "2026-01-15T12:00:00+08:00",
  "checks": {
    "database": "error: connection refused",
    "memory_alloc_mb": "256.78",
    "goroutines": "89"
  }
}
```

## 配置字段说明

### Config 字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `port` | `int` | - | HTTP 服务器监听端口 |
| `app_name` | `string` | `"AIS Go App"` | 应用名称 |
| `read_timeout` | `time.Duration` | `30s` | 读取超时时间 |
| `write_timeout` | `time.Duration` | `30s` | 写入超时时间 |
| `idle_timeout` | `time.Duration` | `120s` | 空闲连接超时时间 |

### ListenOptions 字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enable_prefork` | `bool` | `false` | 是否启用 Prefork 模式（多进程） |
| `disable_startup_message` | `bool` | `false` | 是否禁用启动消息 |
| `enable_print_routes` | `bool` | `false` | 是否打印所有路由 |
| `listener_network` | `string` | `"tcp4"` | 监听网络类型（tcp/tcp4/tcp6/unix） |
| `cert_file` | `string` | `""` | TLS 证书文件路径 |
| `cert_key_file` | `string` | `""` | TLS 证书私钥文件路径 |
| `cert_client_file` | `string` | `""` | mTLS 客户端证书文件路径 |
| `shutdown_timeout` | `time.Duration` | Fiber 默认 `10s` | 优雅关闭超时时间 |
| `unix_socket_file_mode` | `uint32` | Fiber 默认 `0770` | Unix Socket 文件权限模式 |
| `tls_min_version` | `uint16` | Fiber 默认 TLS 1.2 | TLS 最低版本（771=TLS 1.2, 772=TLS 1.3） |

### ListenConfigCustomizer 可配置项

通过 `ListenConfigCustomizer` 可以设置以下高级选项：

- `GracefulContext`: 优雅关闭的 context
- `TLSConfigFunc`: 自定义 TLS 配置函数
- `ListenerAddrFunc`: 监听地址回调函数
- `BeforeServeFunc`: 服务启动前的回调函数
- `AutoCertManager`: ACME 自动证书管理器（Let's Encrypt）

## 最佳实践

1. **生产环境配置**
   - 建议启用 `enable_prefork` 以充分利用多核 CPU
   - 根据实际负载调整超时时间
   - 启用 TLS/mTLS 保护传输安全

2. **开发环境配置**
   - 保持 `enable_prefork: false` 以便调试
   - 设置 `enable_print_routes: true` 查看所有路由
   - 可以设置较短的超时时间以快速发现问题

3. **Kubernetes 部署**
   - 配置 liveness probe 指向 `/healthz`
   - 配置 readiness probe 指向 `/readyz`
   - 设置合适的 `shutdown_timeout` 确保优雅关闭

4. **扩展性**
   - 基础配置使用 YAML 文件管理
   - 高级配置使用 `ListenConfigCustomizer` 函数
   - 保持配置的清晰度和可维护性

## 注意事项

- ⚠️ 使用 Prefork 模式时，`listener_network` 只能选择 `tcp4` 或 `tcp6`
- ⚠️ TLS 1.0 和 TLS 1.1 不再支持，最低版本为 TLS 1.2
- ⚠️ `ListenConfigCustomizer` 是可选的，如果不需要高级配置可以不提供
- ⚠️ Unix Socket 模式下需要设置正确的文件权限
