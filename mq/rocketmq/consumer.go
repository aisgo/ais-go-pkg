package rocketmq

import (
	"context"
	"fmt"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
)

/* ========================================================================
 * RocketMQ Consumer - 直接使用封装（向后兼容）
 * ========================================================================
 * 职责: 提供 RocketMQ 原生 Consumer API
 * 技术: apache/rocketmq-client-go/v2
 * ======================================================================== */

// MessageHandler 消息处理函数
type MessageHandler func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error)

// Consumer RocketMQ 消费者封装
type Consumer struct {
	consumer rocketmq.PushConsumer
	logger   *zap.Logger
	config   *Config
}

// NewConsumer 创建消费者
func NewConsumer(cfg *Config, logger *zap.Logger) (*Consumer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// 构建 NameServer 地址
	nameServers := make([]string, len(cfg.NameServers))
	copy(nameServers, cfg.NameServers)

	// 消费模式
	var consumeMode consumer.MessageModel
	if cfg.Consumer.Model == "Broadcasting" {
		consumeMode = consumer.BroadCasting
	} else {
		consumeMode = consumer.Clustering
	}

	// 消费位置
	var consumeFromWhere consumer.ConsumeFromWhere
	switch cfg.Consumer.ConsumeFromWhere {
	case "FirstOffset":
		consumeFromWhere = consumer.ConsumeFromFirstOffset
	case "Timestamp":
		consumeFromWhere = consumer.ConsumeFromTimestamp
	default:
		consumeFromWhere = consumer.ConsumeFromLastOffset
	}

	// 创建消费者选项
	opts := []consumer.Option{
		consumer.WithNameServer(nameServers),
		consumer.WithGroupName(cfg.Consumer.GroupName),
		consumer.WithConsumerModel(consumeMode),
		consumer.WithConsumeFromWhere(consumeFromWhere),
		consumer.WithConsumeMessageBatchMaxSize(cfg.Consumer.ConsumeMessageBatchMax),
		consumer.WithPullBatchSize(cfg.Consumer.PullBatchSize),
		consumer.WithPullInterval(cfg.Consumer.PullInterval),
		consumer.WithMaxReconsumeTimes(cfg.Consumer.MaxReconsumeTimes),
	}

	// 命名空间
	if cfg.Namespace != "" {
		opts = append(opts, consumer.WithNamespace(cfg.Namespace))
	}

	// 实例名称
	if cfg.InstanceName != "" {
		opts = append(opts, consumer.WithInstance(cfg.InstanceName))
	}

	// ACL 认证
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, consumer.WithCredentials(primitive.Credentials{
			AccessKey: cfg.AccessKey,
			SecretKey: cfg.SecretKey,
		}))
	}

	// 创建消费者实例
	c, err := rocketmq.NewPushConsumer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	logger.Info("RocketMQ consumer created",
		zap.String("group", cfg.Consumer.GroupName),
		zap.Strings("name_servers", nameServers),
	)

	return &Consumer{
		consumer: c,
		logger:   logger,
		config:   cfg,
	}, nil
}

// Subscribe 订阅主题
func (c *Consumer) Subscribe(topic string, selector consumer.MessageSelector, handler MessageHandler) error {
	err := c.consumer.Subscribe(topic, selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		c.logger.Debug("received messages",
			zap.String("topic", topic),
			zap.Int("count", len(msgs)),
		)

		result, err := handler(ctx, msgs...)
		if err != nil {
			c.logger.Error("failed to handle messages",
				zap.String("topic", topic),
				zap.Int("count", len(msgs)),
				zap.Error(err),
			)
			return consumer.ConsumeRetryLater, err
		}

		return result, nil
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe topic %s: %w", topic, err)
	}

	c.logger.Info("subscribed to topic",
		zap.String("topic", topic),
		zap.String("selector", selector.Expression),
	)

	return nil
}

// Start 启动消费者
func (c *Consumer) Start() error {
	if err := c.consumer.Start(); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}
	c.logger.Info("RocketMQ consumer started")
	return nil
}

// Shutdown 关闭消费者
func (c *Consumer) Shutdown() error {
	if err := c.consumer.Shutdown(); err != nil {
		c.logger.Error("failed to shutdown consumer", zap.Error(err))
		return err
	}
	c.logger.Info("RocketMQ consumer shutdown")
	return nil
}
