# AIS Go Pkg

[![Go Version](https://img.shields.io/badge/Go-1.25.5-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> 企业级 Go Web 开发公共组件库 - 沉淀最佳实践，加速业务开发

## ✨ 核心特性

- 🎯 **接口优先** - 清晰的抽象层，易于扩展和测试
- ⚙️ **配置驱动** - YAML + 环境变量，灵活适配多环境
- 🔌 **零侵入设计** - 不绑定特定框架，按需集成
- 🧪 **高可测试性** - 完整的 Mock 支持和测试工具
- 📦 **开箱即用** - 预配置最佳实践，减少重复工作
- 🚀 **生产就绪** - 经过实战验证的企业级组件

---

## 📦 组件清单

| 组件 | 功能 | 核心依赖 |
|------|------|---------|
| **logger** | 结构化日志 | zap |
| **conf** | 配置管理 | viper |
| **database** | 数据库连接池 | gorm, postgres |
| **cache** | Redis 客户端 + 分布式锁 | go-redis/v9 |
| **mq** | 消息队列抽象层 | Kafka, RocketMQ |
| **transport** | HTTP/gRPC 服务器 | Fiber v3, gRPC |
| **metrics** | Prometheus 监控 | prometheus/client_golang |
| **middleware** | HTTP 中间件 | API Key 认证、Auth Header 透传、错误处理、限流等 |
| **errors** | 统一错误处理 | gRPC/HTTP 错误转换 |
| **repository** | 数据仓储模式 | CRUD, 分页, 聚合 |
| **response** | 统一响应格式 | HTTP 响应封装 |
| **validator** | 数据验证 | validator/v10 |
| **shutdown** | 优雅关闭 | 分优先级资源清理 |
| **utils** | 工具集 | UUID, Snowflake 等 |

---

## 🚀 快速开始

### 安装

#### 方式一：本地开发（推荐）

```bash
# 在你的项目 go.mod 中添加
replace github.com/aisgo/ais-go-pkg => ../ais-go-pkg

require github.com/aisgo/ais-go-pkg v0.0.0
```

#### 方式二：Git 依赖（正式发布后）

```bash
go get github.com/your-org/ais-go-pkg@v1.0.0
```

### 基础示例

#### 方式一：直接调用（适合简单场景）

```go
package main

	import (
	    "github.com/aisgo/ais-go-pkg/logger"
	    "github.com/aisgo/ais-go-pkg/database/postgres"
	    "go.uber.org/zap"
	)

func main() {
    // ================================================================
    // 初始化日志
    // ================================================================
    log := logger.NewLogger(logger.Config{
        Level:  "info",
        Format: "json",
    })
    
    // ================================================================
    // 连接数据库
    // ================================================================
    db, err := postgres.NewDB(postgres.Config{
        Host:     "localhost",
        Port:     5432,
        User:     "user",
        Password: "pass",
        DBName:   "mydb",
    }, log)
    if err != nil {
        log.Fatal("failed to connect database", zap.Error(err))
    }
    
	    // 其他组件（Redis/MQ/Transport 等）请参考下方“组件详解”中的示例

	    log.Info("application started successfully")
	}
```

#### 方式二：使用 Fx 模块（推荐，适合复杂应用）

```go
package main

import (
    "github.com/aisgo/ais-go-pkg/cache"
    "github.com/aisgo/ais-go-pkg/cache/redis"
    "github.com/aisgo/ais-go-pkg/database/postgres"
    "github.com/aisgo/ais-go-pkg/logger"
    "github.com/aisgo/ais-go-pkg/mq"
    "github.com/aisgo/ais-go-pkg/transport/http"
    "github.com/gofiber/fiber/v3"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

func main() {
    app := fx.New(
        // ================================================================
        // 配置提供
        // ================================================================
        fx.Provide(
            func() logger.Config {
                return logger.Config{Level: "info", Format: "json"}
            },
            func() postgres.Config {
                return postgres.Config{
                    Host:     "localhost",
                    Port:     5432,
                    User:     "user",
                    Password: "pass",
                    DBName:   "mydb",
                }
            },
            func() redis.Config {
                return redis.Config{
                    Host: "localhost",
                    Port: 6379,
                }
            },
            func() *mq.Config {
                return &mq.Config{
                    Type: mq.TypeKafka,
                    Kafka: &mq.KafkaConfig{
                        Brokers: []string{"localhost:9092"},
                    },
                }
            },
            // HTTP Server 通过 NewHTTPServer 提供，无需 Module
            func() http.Config {
                return http.Config{Port: 8080}
            },
            http.NewHTTPServer,
        ),
        
        // ================================================================
        // 导入组件模块
        // ================================================================
        logger.Module,
        postgres.Module,
        cache.Module,
        mq.Module,
        
        // ================================================================
        // 业务逻辑
        // ================================================================
        fx.Invoke(func(
            log *logger.Logger,
            db *gorm.DB,
            fiberApp *fiber.App,
            // 其他依赖会自动注入
        ) {
            log.Info("application started successfully")
            // 使用 db、fiberApp 等组件...
        }),
    )
    
    app.Run()
}
```

---

## 📚 组件详解

### 🪵 Logger - 结构化日志

基于 Zap 的高性能日志组件，支持 JSON 和 Console 格式。

#### 直接使用

```go
import "github.com/aisgo/ais-go-pkg/logger"

log := logger.NewLogger(logger.Config{
    Level:      "info",        // debug, info, warn, error
    Format:     "json",        // json | console
    Output:     "app.log",     // 可选，默认 stdout
    MaxSize:    100,           // MB
    MaxBackups: 3,
    MaxAge:     28,            // days
})

// Compress 为 *bool，nil 表示默认 true；需要关闭时：
// b := false
// cfg.Compress = &b

log.Info("user login", 
    zap.String("user_id", "123"),
    zap.Duration("latency", time.Millisecond*50),
)
```

#### 使用 Fx 模块

```go
import (
    "github.com/aisgo/ais-go-pkg/logger"
    "go.uber.org/fx"
)

app := fx.New(
    fx.Provide(func() logger.Config {
        return logger.Config{Level: "info", Format: "json"}
    }),
    logger.Module,
    fx.Invoke(func(log *logger.Logger) {
        log.Info("application started")
    }),
)
```

### 🗄️ Database - PostgreSQL + GORM

预配置连接池和日志适配器。

#### 直接使用

```go
import "github.com/aisgo/ais-go-pkg/database/postgres"

db, err := postgres.NewDB(postgres.Config{
    Host:            "localhost",
    Port:            5432,
    User:            "postgres",
    Password:        "secret",
    DBName:          "mydb",
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: time.Hour,
}, log)

// 使用 GORM
type User struct {
    ID   uint
    Name string
}
db.AutoMigrate(&User{})
```

#### 使用 Fx 模块

```go
import (
    "github.com/aisgo/ais-go-pkg/database/postgres"
    "github.com/aisgo/ais-go-pkg/logger"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

app := fx.New(
    fx.Provide(
        func() logger.Config { return logger.Config{Level: "info"} },
        func() postgres.Config {
            return postgres.Config{
                Host:   "localhost",
                Port:   5432,
                User:   "postgres",
                DBName: "mydb",
            }
        },
    ),
    logger.Module,
    postgres.Module,
    fx.Invoke(func(db *gorm.DB) {
        // 使用 db...
    }),
)
```

### 💾 Cache - Redis 客户端

封装 go-redis/v9，提供分布式锁实现。

#### 直接使用

```go
import (
    "context"
    "time"
    "github.com/aisgo/ais-go-pkg/cache/redis"
    "github.com/aisgo/ais-go-pkg/logger"
)

log := logger.NewLogger(logger.Config{Level: "info"})
client := redis.NewClient(redis.Config{
    Host:         "localhost",
    Port:         6379,
    Password:     "",
    DB:           0,
    PoolSize:     10,
    MinIdleConns: 5,
}, log)

ctx := context.Background()
_ = client.Set(ctx, "key", "value", time.Hour)
_, _ = client.Get(ctx, "key")

// 分布式锁
lock := client.NewLock("resource:order:123")
if err := lock.Acquire(ctx); err == nil {
    defer lock.Release(ctx)
    // 临界区代码
}
```

#### 使用 Fx 模块

```go
import (
    "context"
    "time"
    "github.com/aisgo/ais-go-pkg/cache"
    "github.com/aisgo/ais-go-pkg/cache/redis"
    "github.com/aisgo/ais-go-pkg/logger"
    "go.uber.org/fx"
)

app := fx.New(
    fx.Provide(
        func() logger.Config { return logger.Config{Level: "info"} },
        func() redis.Config {
            return redis.Config{
                Host:         "localhost",
                Port:         6379,
                PoolSize:     10,
                MinIdleConns: 5,
            }
        },
    ),
    logger.Module,
    cache.Module,
    fx.Invoke(func(client redis.Clienter) {
        ctx := context.Background()
        _ = client.Set(ctx, "key", "value", time.Hour)
        
        // 分布式锁
        lock := client.NewLock("resource:order:123")
        if err := lock.Acquire(ctx); err == nil {
            defer lock.Release(ctx)
            // 临界区代码
        }
    }),
)
```

> ✅ 接口优先：通过 Fx 模块注入时推荐使用 `redis.Clienter` 接口，便于 mock/替换实现；同时仍保留 `*redis.Client` 注入。

### 📨 MQ - 消息队列抽象层

统一接口，支持 Kafka 和 RocketMQ 无缝切换。
Kafka Consumer 默认关闭 auto-commit，成功处理后会显式提交 offset；如需自动提交，请设置 `Consumer.AutoCommit=true`。

#### 直接使用

```go
import (
    "context"
    "github.com/aisgo/ais-go-pkg/mq"
    _ "github.com/aisgo/ais-go-pkg/mq/kafka"     // 注册 Kafka 实现
    _ "github.com/aisgo/ais-go-pkg/mq/rocketmq"  // 注册 RocketMQ 实现
    "go.uber.org/zap"
)

// 配置驱动 - 自动选择实现
cfg := &mq.Config{
    Type: mq.TypeKafka,
    Kafka: &mq.KafkaConfig{
        Brokers: []string{"localhost:9092"},
    },
}

producer, _ := mq.NewProducer(cfg, zap.NewNop())

// 发送消息
msg := mq.NewMessage("order-events", []byte(`{"order_id": 123}`)).
    WithKey("order-123").
    WithProperty("trace-id", "abc123")
_, _ = producer.SendSync(context.Background(), msg)

// 消费消息
consumer, _ := mq.NewConsumer(cfg, zap.NewNop())
_ = consumer.Subscribe("order-events", func(ctx context.Context, msgs []*mq.ConsumedMessage) (mq.ConsumeResult, error) {
    // TODO: 处理 msgs
    return mq.ConsumeSuccess, nil
})
_ = consumer.Start()
```

#### 使用 Fx 模块

```go
import (
    "context"
    "github.com/aisgo/ais-go-pkg/logger"
    "github.com/aisgo/ais-go-pkg/mq"
    _ "github.com/aisgo/ais-go-pkg/mq/kafka"
    "go.uber.org/fx"
)

app := fx.New(
    fx.Provide(
        func() logger.Config { return logger.Config{Level: "info"} },
        func() *mq.Config {
            return &mq.Config{
                Type: mq.TypeKafka,
                Kafka: &mq.KafkaConfig{
                    Brokers: []string{"localhost:9092"},
                },
            }
        },
    ),
    logger.Module,
    mq.Module, // 自动提供 Producer 和 Consumer
    fx.Invoke(func(producer mq.Producer, consumer mq.Consumer) {
        // Producer 和 Consumer 会自动注入
        // Consumer 会在应用启动时自动开始消费
    }),
)
```

### 🌐 Transport - HTTP/gRPC 服务器

#### HTTP Server (Fiber v3)

```go
import (
    aishttp "github.com/aisgo/ais-go-pkg/transport/http"
    "github.com/aisgo/ais-go-pkg/logger"
    "github.com/gofiber/fiber/v3"
    "go.uber.org/fx"
)

app := fx.New(
    fx.Provide(
        logger.NewNop,
        func() aishttp.Config { return aishttp.Config{Port: 8080} },
        aishttp.NewHTTPServer,
    ),
    fx.Invoke(func(fiberApp *fiber.App) {
        fiberApp.Get("/api/health", func(c fiber.Ctx) error {
            return c.JSON(fiber.Map{"status": "ok"})
        })
    }),
)
_ = app
```

#### gRPC Server

```go
import (
    aisgrpc "github.com/aisgo/ais-go-pkg/transport/grpc"
    "github.com/aisgo/ais-go-pkg/logger"
    "go.uber.org/fx"
    "google.golang.org/grpc"
)

app := fx.New(
    fx.Provide(
        logger.NewNop,
        func() aisgrpc.Config { return aisgrpc.Config{Port: 50051, Mode: "microservice"} },
        aisgrpc.NewInProcListener,
        aisgrpc.NewListener,
        aisgrpc.NewServer,
    ),
    fx.Invoke(func(s *grpc.Server) {
        // 注册服务
        // pb.RegisterYourServiceServer(s, &yourService{})
    }),
)
_ = app
```

> ✅ gRPC ClientFactory 支持 TLS：配置 `aisgrpc.Config.TLS`（例如 `enable/ca_file/cert_file/key_file/server_name`）即可启用安全连接。

### 📊 Metrics - Prometheus 监控

#### 直接使用

```go
import "github.com/aisgo/ais-go-pkg/metrics"

// 注册指标
requestCounter := metrics.NewCounter("http_requests_total", "Total HTTP requests")
requestDuration := metrics.NewHistogram("http_request_duration_seconds", "HTTP request latency")

// 使用
requestCounter.Inc()
requestDuration.Observe(0.05)
```

#### 使用 Fx 模块

```go
import (
    "github.com/aisgo/ais-go-pkg/metrics"
    "go.uber.org/fx"
)

app := fx.New(
    metrics.Module,
    fx.Invoke(func() {
        // 注册指标
        requestCounter := metrics.NewCounter("http_requests_total", "Total HTTP requests")
        requestCounter.Inc()
    }),
)
```

### 🗂️ Repository - 数据仓储模式

提供通用 CRUD、分页、聚合等数据访问模式。

```go
import "github.com/aisgo/ais-go-pkg/repository"

type UserRepository struct {
    repository.BaseRepository[User]
}

repo := &UserRepository{
    BaseRepository: repository.NewBaseRepository[User](db),
}

// CRUD 操作
user := &User{Name: "Alice"}
repo.Create(ctx, user)

// 分页查询
page := repo.Paginate(ctx, repository.PageRequest{
    Page:     1,
    PageSize: 10,
}, repository.WithCondition("age > ?", 18))
```

#### 多租户 (默认强制)

Repository 默认强制租户隔离，请在调用前将租户信息注入 context。

```go
ctx := repository.WithTenantContext(ctx, repository.TenantContext{
    TenantID: tenantID,
    DeptID:   deptID,
    IsAdmin:  false,
})

err := repo.Create(ctx, user)
```

如需对非多租户表关闭强制隔离，实现接口即可：

```go
type NonTenantModel struct {
    ID   string `gorm:"column:id;type:char(26);primaryKey"`
    Name string `gorm:"column:name"`
}

func (NonTenantModel) TenantIgnored() bool { return true }
```

#### 聚合返回值说明

`Max/Min/MaxWithCondition/MinWithCondition` 的返回值类型由数据库驱动决定（如 `int64/float64/string/[]byte/time.Time` 等），
无记录时返回 `nil`。调用方应按实际类型进行断言或转换。

### ✅ Validator - 数据验证

基于 validator/v10 的验证器封装。

#### 直接使用

```go
import "github.com/aisgo/ais-go-pkg/validator"

type CreateUserRequest struct {
    Email    string `validate:"required,email"`
    Age      int    `validate:"gte=0,lte=120"`
    Username string `validate:"required,min=3,max=20"`
}

v := validator.New()
req := &CreateUserRequest{Email: "invalid", Age: 200}

if err := v.Validate(req); err != nil {
    // 处理验证错误
}
```

#### 使用 Fx 模块

```go
import (
    "github.com/aisgo/ais-go-pkg/validator"
    "go.uber.org/fx"
)

app := fx.New(
    validator.Module,
    fx.Invoke(func(v *validator.Validator) {
        // 使用验证器...
    }),
)
```

### 🛑 Shutdown - 优雅关闭

分优先级管理资源清理顺序。

#### 直接使用

```go
import "github.com/aisgo/ais-go-pkg/shutdown"

manager := shutdown.NewManager(log)

// 注册清理函数（优先级：数字越小越先执行）
manager.Register(shutdown.PriorityHigh, func(ctx context.Context) error {
    return httpServer.Shutdown(ctx)
})

manager.Register(shutdown.PriorityMedium, func(ctx context.Context) error {
    return db.Close()
})

// 等待信号并执行清理
manager.Wait()
```

#### 使用 Fx 模块

```go
import (
    "github.com/aisgo/ais-go-pkg/shutdown"
    "go.uber.org/fx"
)

app := fx.New(
    shutdown.Module,
    fx.Invoke(func(manager *shutdown.Manager) {
        // 注册清理函数
        manager.Register(shutdown.PriorityHigh, func(ctx context.Context) error {
            return httpServer.Shutdown(ctx)
        })
    }),
)
```

---

## 🏗️ 架构设计

### 设计原则

1. **接口优先** - 所有组件基于接口设计，便于 Mock 和替换
2. **配置驱动** - 通过配置文件和环境变量控制行为
3. **依赖注入** - 支持 Uber Fx，也可独立使用
4. **错误透明** - 统一错误处理和转换机制
5. **可观测性** - 内置日志、指标、链路追踪支持

### 目录结构

```
ais-go-pkg/
├── cache/              # 缓存组件
│   └── redis/          # Redis 实现
├── conf/               # 配置加载
├── database/           # 数据库连接
│   └── postgres/       # PostgreSQL 实现
├── errors/             # 错误定义
├── logger/             # 日志组件
├── metrics/            # 监控指标
├── middleware/         # HTTP 中间件
├── mq/                 # 消息队列
│   ├── kafka/          # Kafka 适配器
│   └── rocketmq/       # RocketMQ 适配器
├── repository/         # 数据仓储
├── response/           # 响应封装
├── shutdown/           # 优雅关闭
├── transport/          # 传输层
│   ├── http/           # HTTP 服务器
│   └── grpc/           # gRPC 服务器
├── utils/              # 工具函数
└── validator/          # 数据验证
```

---

## 🔧 开发指南

### 添加新组件

1. 在对应目录创建包
2. 定义清晰的接口
3. 提供配置结构体
4. 实现 Fx 模块（可选）
5. 编写单元测试
6. 更新文档

### 测试

```bash
# 运行所有测试
go test ./...

# 带覆盖率
go test -cover ./...

# 指定包
go test ./logger -v
```

### 代码规范

- 遵循 [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- 所有公共 API 必须有注释
- 使用 ASCII 风格分块注释组织代码
- 错误处理必须明确，不吞噬错误

---

## 📋 依赖清单

### 核心依赖

| 库 | 版本 | 用途 |
|----|------|------|
| go.uber.org/zap | v1.27.1 | 结构化日志 |
| go.uber.org/fx | v1.24.0 | 依赖注入 |
| github.com/spf13/viper | v1.21.0 | 配置管理 |
| gorm.io/gorm | v1.31.1 | ORM 框架 |
| github.com/redis/go-redis/v9 | v9.17.2 | Redis 客户端 |
| github.com/gofiber/fiber/v3 | v3.0.0-rc.3 | HTTP 框架 |
| google.golang.org/grpc | v1.78.0 | gRPC 框架 |
| github.com/IBM/sarama | v1.46.3 | Kafka 客户端 |
| github.com/apache/rocketmq-client-go/v2 | v2.1.2 | RocketMQ 客户端 |
| github.com/prometheus/client_golang | v1.23.2 | Prometheus 客户端 |
| github.com/go-playground/validator/v10 | v10.30.1 | 数据验证 |

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 提交规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type:**
- `feat`: 新功能
- `fix`: 修复 Bug
- `docs`: 文档更新
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具链

---

## 📄 License

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 🔗 相关资源

- [CLAUDE.md](CLAUDE.md) - 详细架构文档
- [Go 官方文档](https://go.dev/doc/)
- [Uber Go Style Guide](https://github.com/uber-go/guide)

---

<div align="center">
Made with ❤️ by AIS Team
</div>
