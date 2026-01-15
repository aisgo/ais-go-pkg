# ais-go-pkg - Go Web 基础公共组件库
Go 1.25.5 + Fiber v3 + Fx + GORM + Zap + Viper + Prometheus + Redis(go-redis) + Kafka(sarama) + RocketMQ

<directory>
cache/ - Redis 客户端 + 分布式锁（1 child: redis/...)
conf/ - 配置加载（viper + env placeholder）
database/ - GORM 日志适配 + 公共 DB 类型 + 驱动封装（2 children: mysql/, postgres/...)
errors/ - 统一业务错误模型 + HTTP/gRPC 映射
logger/ - Zap 日志封装
metrics/ - Prometheus 指标注册 + /metrics 暴露（fasthttp adaptor）
middleware/ - Fiber 中间件（API Key 认证等）
mq/ - MQ 抽象 + 工厂注册 + Fx 注入（2 children: kafka/, rocketmq/...)
repository/ - GORM 泛型仓储（CRUD/query/page/aggregate/tx/base model）
response/ - Fiber 统一 JSON 响应封装
shutdown/ - 优雅关停管理（优先级钩子 + Fx 模块）
transport/ - HTTP/Fiber + gRPC 服务器封装（2 children: http/, grpc/...)
utils/ - 工具集（1 child: id-generator/...)
validator/ - 结构体验证（error_msg 自定义消息 + 递归校验）
</directory>

<config>
GO_RULES.md - Go 代码规范与架构标准
FIBER_RULES.md - Fiber v3/fasthttp 语义规则
go.mod - 模块依赖与 Go toolchain 版本
README.md - 组件清单与使用示例
</config>

Principles: minimal · stable · navigable · version-accurate

