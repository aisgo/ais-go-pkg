package rocketmq

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"

	"github.com/aisgo/ais-go-pkg/mq"
)

/* ========================================================================
 * RocketMQ Adapter - RocketMQ 适配器
 * ========================================================================
 * 职责: 实现 mq.Producer / mq.Consumer 接口
 * 技术: apache/rocketmq-client-go/v2
 * ======================================================================== */

// =============================================================================
// 注册工厂
// =============================================================================

func init() {
	mq.RegisterProducerFactory(mq.TypeRocketMQ, NewProducerAdapter)
	mq.RegisterConsumerFactory(mq.TypeRocketMQ, NewConsumerAdapter)
}

// =============================================================================
// Producer 适配器
// =============================================================================

// ProducerAdapter RocketMQ 生产者适配器
type ProducerAdapter struct {
	producer rocketmq.Producer
	logger   *zap.Logger
}

// NewProducerAdapter 创建 RocketMQ 生产者适配器
func NewProducerAdapter(cfg *mq.Config, logger *zap.Logger) (mq.Producer, error) {
	if cfg.RocketMQ == nil {
		return nil, fmt.Errorf("rocketmq config is required")
	}

	rmqCfg := cfg.RocketMQ

	// 创建生产者选项
	opts := []producer.Option{
		producer.WithNameServer(rmqCfg.NameServers),
		producer.WithGroupName(rmqCfg.Producer.GroupName),
		producer.WithRetry(rmqCfg.Producer.RetryTimesOnFailed),
		producer.WithSendMsgTimeout(rmqCfg.Producer.SendMsgTimeout),
	}

	if rmqCfg.Namespace != "" {
		opts = append(opts, producer.WithNamespace(rmqCfg.Namespace))
	}

	if rmqCfg.InstanceName != "" {
		opts = append(opts, producer.WithInstanceName(rmqCfg.InstanceName))
	}

	if rmqCfg.AccessKey != "" && rmqCfg.SecretKey != "" {
		opts = append(opts, producer.WithCredentials(primitive.Credentials{
			AccessKey: rmqCfg.AccessKey,
			SecretKey: rmqCfg.SecretKey,
		}))
	}

	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create rocketmq producer: %w", err)
	}

	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("failed to start rocketmq producer: %w", err)
	}

	logger.Info("RocketMQ producer started",
		zap.String("group", rmqCfg.Producer.GroupName),
		zap.Strings("name_servers", rmqCfg.NameServers),
	)

	return &ProducerAdapter{
		producer: p,
		logger:   logger,
	}, nil
}

// SendSync 同步发送消息
func (p *ProducerAdapter) SendSync(ctx context.Context, msg *mq.Message) (*mq.SendResult, error) {
	rmqMsg := convertToRocketMQMessage(msg)

	result, err := p.producer.SendSync(ctx, rmqMsg)
	if err != nil {
		p.logger.Error("failed to send message",
			zap.String("topic", msg.Topic),
			zap.Error(err),
		)
		return nil, err
	}

	p.logger.Debug("message sent",
		zap.String("topic", msg.Topic),
		zap.String("msg_id", result.MsgID),
	)

	return convertFromRocketMQSendResult(result), nil
}

// SendAsync 异步发送消息
func (p *ProducerAdapter) SendAsync(ctx context.Context, msg *mq.Message, callback mq.SendCallback) error {
	rmqMsg := convertToRocketMQMessage(msg)

	err := p.producer.SendAsync(ctx, func(ctx context.Context, result *primitive.SendResult, err error) {
		if callback != nil {
			callback(convertFromRocketMQSendResult(result), err)
		}
	}, rmqMsg)

	if err != nil {
		p.logger.Error("failed to send async message",
			zap.String("topic", msg.Topic),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// Close 关闭生产者
func (p *ProducerAdapter) Close() error {
	if err := p.producer.Shutdown(); err != nil {
		p.logger.Error("failed to shutdown producer", zap.Error(err))
		return err
	}
	p.logger.Info("RocketMQ producer closed")
	return nil
}

// =============================================================================
// Consumer 适配器
// =============================================================================

// ConsumerAdapter RocketMQ 消费者适配器
type ConsumerAdapter struct {
	consumer rocketmq.PushConsumer
	logger   *zap.Logger
}

// NewConsumerAdapter 创建 RocketMQ 消费者适配器
func NewConsumerAdapter(cfg *mq.Config, logger *zap.Logger) (mq.Consumer, error) {
	if cfg.RocketMQ == nil {
		return nil, fmt.Errorf("rocketmq config is required")
	}

	rmqCfg := cfg.RocketMQ

	// 消费模式
	var consumeMode consumer.MessageModel
	if rmqCfg.Consumer.Model == "Broadcasting" {
		consumeMode = consumer.BroadCasting
	} else {
		consumeMode = consumer.Clustering
	}

	// 消费位置
	var consumeFromWhere consumer.ConsumeFromWhere
	switch rmqCfg.Consumer.ConsumeFromWhere {
	case "FirstOffset":
		consumeFromWhere = consumer.ConsumeFromFirstOffset
	case "Timestamp":
		consumeFromWhere = consumer.ConsumeFromTimestamp
	default:
		consumeFromWhere = consumer.ConsumeFromLastOffset
	}

	opts := []consumer.Option{
		consumer.WithNameServer(rmqCfg.NameServers),
		consumer.WithGroupName(rmqCfg.Consumer.GroupName),
		consumer.WithConsumerModel(consumeMode),
		consumer.WithConsumeFromWhere(consumeFromWhere),
		consumer.WithConsumeMessageBatchMaxSize(rmqCfg.Consumer.ConsumeMessageBatchMax),
		consumer.WithPullBatchSize(rmqCfg.Consumer.PullBatchSize),
		consumer.WithPullInterval(rmqCfg.Consumer.PullInterval),
		consumer.WithMaxReconsumeTimes(rmqCfg.Consumer.MaxReconsumeTimes),
	}

	if rmqCfg.Namespace != "" {
		opts = append(opts, consumer.WithNamespace(rmqCfg.Namespace))
	}

	if rmqCfg.InstanceName != "" {
		opts = append(opts, consumer.WithInstance(rmqCfg.InstanceName))
	}

	if rmqCfg.AccessKey != "" && rmqCfg.SecretKey != "" {
		opts = append(opts, consumer.WithCredentials(primitive.Credentials{
			AccessKey: rmqCfg.AccessKey,
			SecretKey: rmqCfg.SecretKey,
		}))
	}

	c, err := rocketmq.NewPushConsumer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create rocketmq consumer: %w", err)
	}

	logger.Info("RocketMQ consumer created",
		zap.String("group", rmqCfg.Consumer.GroupName),
		zap.Strings("name_servers", rmqCfg.NameServers),
	)

	return &ConsumerAdapter{
		consumer: c,
		logger:   logger,
	}, nil
}

// Subscribe 订阅主题
func (c *ConsumerAdapter) Subscribe(topic string, handler mq.MessageHandler) error {
	err := c.consumer.Subscribe(topic, consumer.MessageSelector{}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		// 转换消息
		convertedMsgs := make([]*mq.ConsumedMessage, len(msgs))
		for i, msg := range msgs {
			convertedMsgs[i] = convertFromRocketMQMessageExt(msg)
		}

		result, err := handler(ctx, convertedMsgs)
		if err != nil {
			c.logger.Error("failed to handle messages",
				zap.String("topic", topic),
				zap.Int("count", len(msgs)),
				zap.Error(err),
			)
			return consumer.ConsumeRetryLater, err
		}

		return convertToRocketMQConsumeResult(result), nil
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe topic %s: %w", topic, err)
	}

	c.logger.Info("subscribed to topic", zap.String("topic", topic))
	return nil
}

// Start 启动消费者
func (c *ConsumerAdapter) Start() error {
	if err := c.consumer.Start(); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}
	c.logger.Info("RocketMQ consumer started")
	return nil
}

// Close 关闭消费者
func (c *ConsumerAdapter) Close() error {
	if err := c.consumer.Shutdown(); err != nil {
		c.logger.Error("failed to shutdown consumer", zap.Error(err))
		return err
	}
	c.logger.Info("RocketMQ consumer closed")
	return nil
}

// =============================================================================
// 转换函数
// =============================================================================

func convertToRocketMQMessage(msg *mq.Message) *primitive.Message {
	rmqMsg := primitive.NewMessage(msg.Topic, msg.Body)

	if msg.Key != "" {
		rmqMsg.WithKeys([]string{msg.Key})
		rmqMsg.WithShardingKey(msg.Key)
	}

	if msg.Tag != "" {
		rmqMsg.WithTag(msg.Tag)
	}

	for k, v := range msg.Properties {
		rmqMsg.WithProperty(k, v)
	}

	if msg.DelayLevel > 0 {
		rmqMsg.WithDelayTimeLevel(msg.DelayLevel)
	}

	return rmqMsg
}

func convertFromRocketMQSendResult(result *primitive.SendResult) *mq.SendResult {
	if result == nil {
		return nil
	}
	return &mq.SendResult{
		MsgID:  result.MsgID,
		Topic:  result.MessageQueue.Topic,
		Status: mq.SendStatus(result.Status),
	}
}

func convertFromRocketMQMessageExt(msg *primitive.MessageExt) *mq.ConsumedMessage {
	return &mq.ConsumedMessage{
		Topic:        msg.Topic,
		Body:         msg.Body,
		Key:          msg.GetKeys(),
		Tag:          msg.GetTags(),
		Properties:   msg.GetProperties(),
		MsgID:        msg.MsgId,
		BornTime:     time.UnixMilli(msg.BornTimestamp),
		ReconsumeCnt: msg.ReconsumeTimes,
	}
}

func convertToRocketMQConsumeResult(result mq.ConsumeResult) consumer.ConsumeResult {
	switch result {
	case mq.ConsumeSuccess:
		return consumer.ConsumeSuccess
	case mq.ConsumeRetryLater:
		return consumer.ConsumeRetryLater
	default:
		return consumer.ConsumeSuccess
	}
}
