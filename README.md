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
| **middleware** | HTTP 中间件 | API Key 认证等 |
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
replace ais.local/ais-go-pkg => ../ais-go-pkg

require ais.local/ais-go-pkg v0.0.0
```

#### 方式二：Git 依赖（正式发布后）

```bash
go get github.com/your-org/ais-go-pkg@v1.0.0
```

### 基础示例

```go
package main

import (
    "ais.local/ais-go-pkg/logger"
    "ais.local/ais-go-pkg/database/postgres"
    "ais.local/ais-go-pkg/cache/redis"
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
    
    // ================================================================
    // 初始化 Redis
    // ================================================================
    rdb := redis.NewClient(redis.ClientParams{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    
    log.Info("application started successfully")
}
```

---

## 📚 组件详解

### 🪵 Logger - 结构化日志

基于 Zap 的高性能日志组件，支持 JSON 和 Console 格式。

```go
import "ais.local/ais-go-pkg/logger"

log := logger.NewLogger(logger.Config{
    Level:      "info",        // debug, info, warn, error
    Format:     "json",        // json | console
    OutputPath: "app.log",     // 可选，默认 stdout
})

log.Info("user login", 
    zap.String("user_id", "123"),
    zap.Duration("latency", time.Millisecond*50),
)
```

### 🗄️ Database - PostgreSQL + GORM

预配置连接池和日志适配器。

```go
import "ais.local/ais-go-pkg/database/postgres"

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

### 💾 Cache - Redis 客户端

封装 go-redis/v9，提供分布式锁实现。

```go
import "ais.local/ais-go-pkg/cache/redis"

client := redis.NewClient(redis.ClientParams{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           0,
    PoolSize:     10,
    MinIdleConns: 5,
})

// 基础操作
client.Set(ctx, "key", "value", time.Hour)
val, _ := client.Get(ctx, "key")

// 分布式锁
lock := client.NewLock("resource:order:123")
if lock.Acquire(ctx) {
    defer lock.Release(ctx)
    // 临界区代码
}
```

### 📨 MQ - 消息队列抽象层

统一接口，支持 Kafka 和 RocketMQ 无缝切换。

```go
import (
    "ais.local/ais-go-pkg/mq"
    _ "ais.local/ais-go-pkg/mq/kafka"     // 注册 Kafka 实现
    _ "ais.local/ais-go-pkg/mq/rocketmq"  // 注册 RocketMQ 实现
)

// ================================================================
// 配置驱动 - 自动选择实现
// ================================================================
cfg := &mq.Config{
    Type: mq.TypeKafka,
    Kafka: &mq.KafkaConfig{
        Brokers: []string{"localhost:9092"},
    },
}

producer, _ := mq.NewProducer(cfg, logger)

// ================================================================
// 发送消息
// ================================================================
msg := mq.NewMessage("order-events", []byte(`{"order_id": 123}`)).
    WithKey("order-123").
    WithHeader("trace-id", "abc123")

err := producer.SendSync(ctx, msg)

// ================================================================
// 消费消息
// ================================================================
consumer, _ := mq.NewConsumer(cfg, logger)
consumer.Subscribe(ctx, []string{"order-events"}, func(msg *mq.Message) error {
    log.Info("received", zap.ByteString("body", msg.Body))
    return nil
})
```

### 🌐 Transport - HTTP/gRPC 服务器

#### HTTP Server (Fiber v3)

```go
import "ais.local/ais-go-pkg/transport/http"

server := http.NewHTTPServer(http.ServerParams{
    Port:   8080,
    Logger: log,
})

app := server.App()
app.Get("/api/health", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{"status": "ok"})
})

server.Start()
```

#### gRPC Server

```go
import "ais.local/ais-go-pkg/transport/grpc"

server := grpc.NewServer(grpc.ServerParams{
    Port:   50051,
    Logger: log,
})

// 注册服务
pb.RegisterYourServiceServer(server.Server(), &yourService{})

server.Start()
```

### 📊 Metrics - Prometheus 监控

```go
import "ais.local/ais-go-pkg/metrics"

// 注册指标
requestCounter := metrics.NewCounter("http_requests_total", "Total HTTP requests")
requestDuration := metrics.NewHistogram("http_request_duration_seconds", "HTTP request latency")

// 使用
requestCounter.Inc()
requestDuration.Observe(0.05)
```

### 🗂️ Repository - 数据仓储模式

提供通用 CRUD、分页、聚合等数据访问模式。

```go
import "ais.local/ais-go-pkg/repository"

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

### ✅ Validator - 数据验证

基于 validator/v10 的验证器封装。

```go
import "ais.local/ais-go-pkg/validator"

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

### 🛑 Shutdown - 优雅关闭

分优先级管理资源清理顺序。

```go
import "ais.local/ais-go-pkg/shutdown"

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
