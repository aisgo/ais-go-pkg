# AIS Go Pkg - 公共 Go Web 开发组件

> **定位**：团队公共 Go Web 开发组件库，沉淀最佳实践
> **设计原则**：复用优先、接口统一、配置驱动、零侵入

## 架构概览

```
ais-go-pkg/
├── go.mod                    # 模块定义
├── CLAUDE.md                 # 架构文档
│
├── logger/                   # 日志组件 (Zap)
│   └── logger.go                 # 结构化日志，支持 JSON/Console 格式
│
├── conf/                     # 配置组件 (Viper)
│   └── config.go                 # 配置加载器，支持 YAML + 环境变量
│
├── errors/                   # 错误处理
│   └── errors.go                 # 统一错误码，gRPC/HTTP 转换
│
├── database/                 # 数据库组件
│   ├── logger.go                 # GORM 日志适配器 (Zap)
│   ├── types.go                  # JSONB 等公共类型
│   └── postgres/
│       └── postgres.go           # PostgreSQL 连接池
│
├── cache/                    # 缓存组件
│   └── redis/
│       ├── client.go             # Redis 客户端 (go-redis/v9)
│       └── lock.go               # 分布式锁 (Redlock 简化版)
│
├── metrics/                  # 可观测性组件
│   └── metrics.go                # Prometheus 指标注册
│
├── middleware/               # HTTP 中间件
│   ├── apikey.go                 # API Key 认证
│   ├── auth_header.go            # Auth Header 统一认证/透传
│   ├── error.go                  # Fiber 统一错误处理
│   └── ratelimit.go              # 请求限流
│
├── transport/                # 传输层
│   ├── http/
│   │   └── server.go             # Fiber HTTP 服务器 + 健康检查
│   └── grpc/
│       └── server.go             # gRPC 服务器 (TCP + BufConn)
│
└── mq/                       # 消息队列抽象层
    ├── mq.go                     # 核心接口 (Producer, Consumer)
    ├── config.go                 # 统一配置
    ├── factory.go                # 工厂模式
    ├── fx.go                     # Fx 依赖注入模块
    ├── kafka/
    │   ├── producer.go           # Kafka 生产者适配器
    │   ├── consumer.go           # Kafka 消费者适配器
    │   └── scram.go              # SCRAM 认证
    └── rocketmq/
        ├── adapter.go            # RocketMQ 适配器
        ├── config.go             # 配置结构
        ├── producer.go           # 生产者封装
        ├── consumer.go           # 消费者封装
        ├── options.go            # 消息选项
        ├── transaction.go        # 事务消息
        └── fx.go                 # Fx 模块
```

## 使用方式

### 方式一：go mod replace（本地开发）

```go
// go.mod
replace github.com/aisgo/ais-go-pkg => ../ais-go-pkg

require github.com/aisgo/ais-go-pkg v0.0.0
```

### 方式二：Git Tag（正式发布后）

```go
// go.mod
require github.com/wangxiaomei/ais-go-pkg v1.0.0
```

## 组件说明

### Logger - 结构化日志
```go
import "github.com/aisgo/ais-go-pkg/logger"

log := logger.NewLogger(logger.Config{
    Level:  "info",
    Format: "json", // json | console
})
log.Info("hello", zap.String("key", "value"))
```

### Database - PostgreSQL + GORM
```go
import "github.com/aisgo/ais-go-pkg/database/postgres"

db, err := postgres.NewDB(postgres.Config{
    Host: "localhost", Port: 5432,
    User: "user", Password: "pass", DBName: "db",
}, log)
```

### Cache - Redis
```go
import "github.com/aisgo/ais-go-pkg/cache/redis"

client := redis.NewClient(redis.ClientParams{...})
client.Set(ctx, "key", "value", 1*time.Hour)

// 分布式锁
lock := client.NewLock("resource:123")
lock.Acquire(ctx)
defer lock.Release(ctx)
```

### MQ - 消息队列抽象
```go
import "github.com/aisgo/ais-go-pkg/mq"
import _ "github.com/aisgo/ais-go-pkg/mq/kafka"     // 注册 Kafka
import _ "github.com/aisgo/ais-go-pkg/mq/rocketmq" // 注册 RocketMQ

// 根据配置自动选择实现
cfg := &mq.Config{Type: mq.TypeKafka, Kafka: ...}
producer, _ := mq.NewProducer(cfg, logger)

msg := mq.NewMessage("topic", []byte("body")).WithKey("key")
producer.SendSync(ctx, msg)
```

### Transport - HTTP/gRPC
```go
import "github.com/aisgo/ais-go-pkg/transport/http"
import "github.com/aisgo/ais-go-pkg/transport/grpc"

// HTTP (Fiber)
app := http.NewHTTPServer(http.ServerParams{...})

// gRPC (支持 TCP 和 BufConn 模式)
server := grpc.NewServer(grpc.ServerParams{...})
```

## 依赖

| 组件 | 依赖库 |
|------|--------|
| Logger | go.uber.org/zap |
| Config | github.com/spf13/viper |
| Database | gorm.io/gorm, gorm.io/driver/postgres |
| Cache | github.com/redis/go-redis/v9 |
| HTTP | github.com/gofiber/fiber/v3 |
| gRPC | google.golang.org/grpc |
| Metrics | github.com/prometheus/client_golang |
| MQ | github.com/IBM/sarama, github.com/apache/rocketmq-client-go/v2 |
| DI | go.uber.org/fx |

## 开发规范

1. **接口优先**：所有组件提供清晰的接口定义
2. **配置驱动**：通过 YAML/环境变量控制行为
3. **零侵入**：不强制使用特定框架或模式
4. **可测试**：支持 Mock 和测试替身
5. **文档完备**：每个公共 API 都有清晰注释
