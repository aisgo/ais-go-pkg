package rocketmq

import (
	"context"
	"fmt"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"
)

/* ========================================================================
 * RocketMQ Producer - 直接使用封装（向后兼容）
 * ========================================================================
 * 职责: 提供 RocketMQ 原生 Producer API
 * 技术: apache/rocketmq-client-go/v2
 * ======================================================================== */

// Producer RocketMQ 生产者封装
type Producer struct {
	producer rocketmq.Producer
	logger   *zap.Logger
	config   *Config
}

// NewProducer 创建生产者
func NewProducer(cfg *Config, logger *zap.Logger) (*Producer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// 构建 NameServer 地址
	nameServers := make([]string, len(cfg.NameServers))
	copy(nameServers, cfg.NameServers)

	// 创建生产者选项
	opts := []producer.Option{
		producer.WithNameServer(nameServers),
		producer.WithGroupName(cfg.Producer.GroupName),
		producer.WithRetry(cfg.Producer.RetryTimesOnFailed),
		producer.WithSendMsgTimeout(cfg.Producer.SendMsgTimeout),
	}

	// 命名空间
	if cfg.Namespace != "" {
		opts = append(opts, producer.WithNamespace(cfg.Namespace))
	}

	// 实例名称
	if cfg.InstanceName != "" {
		opts = append(opts, producer.WithInstanceName(cfg.InstanceName))
	}

	// ACL 认证
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, producer.WithCredentials(primitive.Credentials{
			AccessKey: cfg.AccessKey,
			SecretKey: cfg.SecretKey,
		}))
	}

	// 创建生产者实例
	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	// 启动生产者
	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("failed to start producer: %w", err)
	}

	logger.Info("RocketMQ producer started",
		zap.String("group", cfg.Producer.GroupName),
		zap.Strings("name_servers", nameServers),
	)

	return &Producer{
		producer: p,
		logger:   logger,
		config:   cfg,
	}, nil
}

// SendSync 同步发送消息
func (p *Producer) SendSync(ctx context.Context, topic string, body []byte, opts ...MessageOption) (*primitive.SendResult, error) {
	// 检查消息大小
	if err := p.checkMessageSize(len(body)); err != nil {
		return nil, err
	}

	msg := primitive.NewMessage(topic, body)

	// 应用选项
	for _, opt := range opts {
		opt(msg)
	}

	result, err := p.producer.SendSync(ctx, msg)
	if err != nil {
		p.logger.Error("failed to send message",
			zap.String("topic", topic),
			zap.Int("body_size", len(body)),
			zap.Error(err),
		)
		return nil, err
	}

	p.logger.Debug("message sent",
		zap.String("topic", topic),
		zap.String("msg_id", result.MsgID),
		zap.Int("status", int(result.Status)),
	)

	return result, nil
}

// SendAsync 异步发送消息
func (p *Producer) SendAsync(ctx context.Context, topic string, body []byte, callback func(context.Context, *primitive.SendResult, error), opts ...MessageOption) error {
	// 检查消息大小
	if err := p.checkMessageSize(len(body)); err != nil {
		return err
	}

	msg := primitive.NewMessage(topic, body)

	// 应用选项
	for _, opt := range opts {
		opt(msg)
	}

	// 如果 callback 为 nil，提供默认回调
	if callback == nil {
		callback = func(ctx context.Context, result *primitive.SendResult, err error) {
			if err != nil {
				p.logger.Error("async message send failed",
					zap.String("topic", topic),
					zap.Error(err),
				)
			}
		}
	}

	err := p.producer.SendAsync(ctx, callback, msg)
	if err != nil {
		p.logger.Error("failed to send async message",
			zap.String("topic", topic),
			zap.Int("body_size", len(body)),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// SendOneWay 单向发送消息（不关心结果）
func (p *Producer) SendOneWay(ctx context.Context, topic string, body []byte, opts ...MessageOption) error {
	// 检查消息大小
	if err := p.checkMessageSize(len(body)); err != nil {
		return err
	}

	msg := primitive.NewMessage(topic, body)

	// 应用选项
	for _, opt := range opts {
		opt(msg)
	}

	err := p.producer.SendOneWay(ctx, msg)
	if err != nil {
		p.logger.Error("failed to send oneway message",
			zap.String("topic", topic),
			zap.Int("body_size", len(body)),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// Shutdown 关闭生产者
func (p *Producer) Shutdown() error {
	if err := p.producer.Shutdown(); err != nil {
		p.logger.Error("failed to shutdown producer", zap.Error(err))
		return err
	}
	p.logger.Info("RocketMQ producer shutdown")
	return nil
}

// checkMessageSize 检查消息大小是否超过限制
func (p *Producer) checkMessageSize(size int) error {
	maxSize := p.config.Producer.MaxMessageSize
	if maxSize <= 0 {
		maxSize = 4 * 1024 * 1024 // 默认 4MB
	}
	if size > maxSize {
		return fmt.Errorf("message size %d bytes exceeds limit %d bytes", size, maxSize)
	}
	return nil
}
