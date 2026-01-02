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
 * RocketMQ Transaction Producer - 事务消息生产者
 * ========================================================================
 * 职责: 提供 RocketMQ 事务消息能力
 * 技术: apache/rocketmq-client-go/v2
 * ======================================================================== */

// TransactionListener 事务监听器
type TransactionListener interface {
	// ExecuteLocalTransaction 执行本地事务
	ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState

	// CheckLocalTransaction 检查本地事务状态（用于事务回查）
	CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState
}

// TransactionProducer 事务消息生产者
type TransactionProducer struct {
	producer rocketmq.TransactionProducer
	logger   *zap.Logger
	config   *Config
}

// NewTransactionProducer 创建事务消息生产者
func NewTransactionProducer(cfg *Config, listener TransactionListener, logger *zap.Logger) (*TransactionProducer, error) {
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

	// 创建事务监听器适配器
	txListener := &transactionListenerAdapter{
		listener: listener,
		logger:   logger,
	}

	// 创建事务生产者
	p, err := rocketmq.NewTransactionProducer(txListener, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction producer: %w", err)
	}

	// 启动生产者
	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("failed to start transaction producer: %w", err)
	}

	logger.Info("RocketMQ transaction producer started",
		zap.String("group", cfg.Producer.GroupName),
		zap.Strings("name_servers", nameServers),
	)

	return &TransactionProducer{
		producer: p,
		logger:   logger,
		config:   cfg,
	}, nil
}

// SendMessageInTransaction 发送事务消息
func (p *TransactionProducer) SendMessageInTransaction(ctx context.Context, topic string, body []byte, opts ...MessageOption) (*primitive.TransactionSendResult, error) {
	msg := primitive.NewMessage(topic, body)

	// 应用选项
	for _, opt := range opts {
		opt(msg)
	}

	result, err := p.producer.SendMessageInTransaction(ctx, msg)
	if err != nil {
		p.logger.Error("failed to send transaction message",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return nil, err
	}

	p.logger.Debug("transaction message sent",
		zap.String("topic", topic),
		zap.String("msg_id", result.MsgID),
		zap.Int("status", int(result.Status)),
		zap.Int("tx_state", int(result.State)),
	)

	return result, nil
}

// Shutdown 关闭事务生产者
func (p *TransactionProducer) Shutdown() error {
	if err := p.producer.Shutdown(); err != nil {
		p.logger.Error("failed to shutdown transaction producer", zap.Error(err))
		return err
	}
	p.logger.Info("RocketMQ transaction producer shutdown")
	return nil
}

// transactionListenerAdapter 事务监听器适配器
type transactionListenerAdapter struct {
	listener TransactionListener
	logger   *zap.Logger
}

func (a *transactionListenerAdapter) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	state := a.listener.ExecuteLocalTransaction(msg)
	a.logger.Debug("execute local transaction",
		zap.String("msg_id", msg.GetProperty(primitive.PropertyUniqueClientMessageIdKeyIndex)),
		zap.Int("state", int(state)),
	)
	return state
}

func (a *transactionListenerAdapter) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
	state := a.listener.CheckLocalTransaction(msg)
	a.logger.Debug("check local transaction",
		zap.String("msg_id", msg.MsgId),
		zap.Int("state", int(state)),
	)
	return state
}
