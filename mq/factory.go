package mq

import (
	"fmt"

	"go.uber.org/zap"
)

/* ========================================================================
 * MQ 工厂 - 根据配置创建对应实现
 * ========================================================================
 * 职责: 提供统一的工厂方法创建 Producer / Consumer
 * ======================================================================== */

// ProducerFactory 生产者工厂函数类型
type ProducerFactory func(cfg *Config, logger *zap.Logger) (Producer, error)

// ConsumerFactory 消费者工厂函数类型
type ConsumerFactory func(cfg *Config, logger *zap.Logger) (Consumer, error)

// 全局工厂注册表
var (
	producerFactories = make(map[Type]ProducerFactory)
	consumerFactories = make(map[Type]ConsumerFactory)
)

// RegisterProducerFactory 注册生产者工厂
func RegisterProducerFactory(mqType Type, factory ProducerFactory) {
	producerFactories[mqType] = factory
}

// RegisterConsumerFactory 注册消费者工厂
func RegisterConsumerFactory(mqType Type, factory ConsumerFactory) {
	consumerFactories[mqType] = factory
}

// NewProducer 创建生产者
func NewProducer(cfg *Config, logger *zap.Logger) (Producer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	factory, ok := producerFactories[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported MQ type: %s, available: rocketmq, kafka", cfg.Type)
	}

	logger.Info("creating MQ producer",
		zap.String("type", string(cfg.Type)),
	)

	return factory(cfg, logger)
}

// NewConsumer 创建消费者
func NewConsumer(cfg *Config, logger *zap.Logger) (Consumer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	factory, ok := consumerFactories[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported MQ type: %s, available: rocketmq, kafka", cfg.Type)
	}

	logger.Info("creating MQ consumer",
		zap.String("type", string(cfg.Type)),
	)

	return factory(cfg, logger)
}

// AvailableTypes 返回可用的 MQ 类型
func AvailableTypes() []Type {
	types := make([]Type, 0, len(producerFactories))
	for t := range producerFactories {
		types = append(types, t)
	}
	return types
}
