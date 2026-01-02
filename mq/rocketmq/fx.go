package rocketmq

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

/* ========================================================================
 * Fx 模块 - RocketMQ 依赖注入
 * ========================================================================
 * 职责: 提供 RocketMQ 的 Fx 依赖注入支持（向后兼容）
 * ======================================================================== */

// Module Fx 模块
var Module = fx.Module("rocketmq",
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

	Producer *Producer
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

	Consumer *Consumer
}

// ProvideProducer 提供 Producer（用于 Fx）
func ProvideProducer(lc fx.Lifecycle, params ProducerParams) (ProducerResult, error) {
	producer, err := NewProducer(params.Config, params.Logger)
	if err != nil {
		return ProducerResult{}, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return producer.Shutdown()
		},
	})

	return ProducerResult{Producer: producer}, nil
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
			return consumer.Shutdown()
		},
	})

	return ConsumerResult{Consumer: consumer}, nil
}
