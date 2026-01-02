package mq

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

/* ========================================================================
 * Fx 模块 - 统一 MQ 依赖注入
 * ========================================================================
 * 职责: 提供 Fx 依赖注入支持
 * ======================================================================== */

// Module Fx 模块（根据配置自动选择 RocketMQ 或 Kafka）
var Module = fx.Module("mq",
	fx.Provide(
		ProvideProducer,
		ProvideConsumer,
	),
)

// ProducerParams Producer 依赖参数
type ProducerParams struct {
	fx.In

	Config *Config
	Logger *zap.Logger
}

// ProducerResult Producer 返回结果
type ProducerResult struct {
	fx.Out

	Producer Producer
}

// ProvideProducer 提供 Producer（用于 Fx）
func ProvideProducer(lc fx.Lifecycle, params ProducerParams) (ProducerResult, error) {
	producer, err := NewProducer(params.Config, params.Logger)
	if err != nil {
		return ProducerResult{}, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return producer.Close()
		},
	})

	return ProducerResult{Producer: producer}, nil
}

// ConsumerParams Consumer 依赖参数
type ConsumerParams struct {
	fx.In

	Config *Config
	Logger *zap.Logger
}

// ConsumerResult Consumer 返回结果
type ConsumerResult struct {
	fx.Out

	Consumer Consumer
}

// ProvideConsumer 提供 Consumer（用于 Fx）
func ProvideConsumer(lc fx.Lifecycle, params ConsumerParams) (ConsumerResult, error) {
	consumer, err := NewConsumer(params.Config, params.Logger)
	if err != nil {
		return ConsumerResult{}, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return consumer.Start()
		},
		OnStop: func(ctx context.Context) error {
			return consumer.Close()
		},
	})

	return ConsumerResult{Consumer: consumer}, nil
}

// =============================================================================
// 单独导入模块（不自动启动 Consumer）
// =============================================================================

// ProducerOnlyModule 仅提供 Producer 的模块
var ProducerOnlyModule = fx.Module("mq-producer",
	fx.Provide(ProvideProducer),
)

// ConsumerOnlyModule 仅提供 Consumer 的模块
var ConsumerOnlyModule = fx.Module("mq-consumer",
	fx.Provide(ProvideConsumer),
)
