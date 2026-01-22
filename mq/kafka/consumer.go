package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/aisgo/ais-go-pkg/mq"
)

/* ========================================================================
 * Kafka Consumer - Kafka 消息消费者
 * ========================================================================
 * 职责: 实现 mq.Consumer 接口
 * 技术: IBM/sarama
 * ======================================================================== */

// 消费者配置常量
const (
	defaultMaxRetries     = 3                      // 默认最大重试次数
	defaultRetryBaseDelay = 100 * time.Millisecond // 默认重试基础延迟
)

// =============================================================================
// 注册工厂
// =============================================================================

func init() {
	mq.RegisterConsumerFactory(mq.TypeKafka, NewConsumerAdapter)
}

// =============================================================================
// Consumer 适配器
// =============================================================================

// ConsumerAdapter Kafka 消费者适配器
type ConsumerAdapter struct {
	client    sarama.ConsumerGroup
	logger    *zap.Logger
	config    *mq.KafkaConfig
	handlers  map[string]mq.MessageHandler
	topics    []string
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	ready     chan struct{}
	readyOnce sync.Once
}

// NewConsumerAdapter 创建 Kafka 消费者适配器
func NewConsumerAdapter(cfg *mq.Config, logger *zap.Logger) (mq.Consumer, error) {
	if cfg.Kafka == nil {
		return nil, fmt.Errorf("kafka config is required")
	}

	kafkaCfg := cfg.Kafka

	// 构建 Sarama 配置
	saramaCfg, err := buildConsumerConfig(kafkaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build sarama config: %w", err)
	}

	// 创建消费者组
	client, err := sarama.NewConsumerGroup(kafkaCfg.Brokers, kafkaCfg.Consumer.GroupID, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer group: %w", err)
	}

	logger.Info("Kafka consumer created",
		zap.String("group", kafkaCfg.Consumer.GroupID),
		zap.Strings("brokers", kafkaCfg.Brokers),
	)

	return &ConsumerAdapter{
		client:   client,
		logger:   logger,
		config:   kafkaCfg,
		handlers: make(map[string]mq.MessageHandler),
		topics:   make([]string, 0),
		ready:    make(chan struct{}),
	}, nil
}

// Subscribe 订阅主题
func (c *ConsumerAdapter) Subscribe(topic string, handler mq.MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.handlers[topic]; !exists {
		c.topics = append(c.topics, topic)
	}
	c.handlers[topic] = handler

	c.logger.Info("subscribed to topic", zap.String("topic", topic))
	return nil
}

func (c *ConsumerAdapter) signalReady() {
	c.readyOnce.Do(func() {
		close(c.ready)
	})
}

// Start 启动消费者
func (c *ConsumerAdapter) Start() error {
	c.mu.Lock()
	if c.cancel != nil {
		c.mu.Unlock()
		return fmt.Errorf("consumer already started")
	}
	topics := append([]string(nil), c.topics...)
	c.ready = make(chan struct{})
	c.readyOnce = sync.Once{}
	readyCh := c.ready
	c.mu.Unlock()

	if len(topics) == 0 {
		return fmt.Errorf("no topics subscribed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	startErr := make(chan error, 1)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			// 每次 Consume 都使用新的 handler；rebalance 会导致 Consume 返回并重入循环
			handler := &consumerGroupHandler{adapter: c}

			// `Consume` 会在 rebalance 后重新调用
			if err := c.client.Consume(ctx, topics, handler); err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("consumer error", zap.Error(err))

				// 防止 CPU 空转 (Busy Loop)
				// 如果是配置错误或 Broker 宕机，Consume 可能立即返回
				time.Sleep(2 * time.Second)

				select {
				case <-readyCh:
					// 已经启动完成：不影响启动流程
				default:
					// 启动阶段出错：尽快反馈给 Start()
					select {
					case startErr <- err:
					default:
					}
				}
			}

			// 检查上下文是否取消
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 等待消费者准备就绪
	select {
	case <-readyCh:
		c.logger.Info("Kafka consumer started", zap.Strings("topics", topics))
		return nil
	case err := <-startErr:
		cancel()
		c.wg.Wait()
		_ = c.client.Close()
		return err
	case <-time.After(30 * time.Second):
		cancel()
		c.wg.Wait()
		_ = c.client.Close()
		return fmt.Errorf("kafka consumer start timeout")
	}
}

// Close 关闭消费者
func (c *ConsumerAdapter) Close() error {
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	c.wg.Wait()

	if err := c.client.Close(); err != nil {
		c.logger.Error("failed to close consumer", zap.Error(err))
		return err
	}

	c.logger.Info("Kafka consumer closed")
	return nil
}

// =============================================================================
// ConsumerGroup Handler
// =============================================================================

type consumerGroupHandler struct {
	adapter *ConsumerAdapter
}

func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.adapter.signalReady()
	h.adapter.logger.Debug("consumer group setup",
		zap.Int32("generation_id", session.GenerationID()),
	)
	return nil
}

func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.adapter.logger.Debug("consumer group cleanup",
		zap.Int32("generation_id", session.GenerationID()),
	)
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	topic := claim.Topic()

	h.adapter.mu.RLock()
	handler, ok := h.adapter.handlers[topic]
	h.adapter.mu.RUnlock()

	if !ok {
		h.adapter.logger.Warn("no handler for topic", zap.String("topic", topic))
		return nil
	}

	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			// 转换消息
			convertedMsg := convertFromKafkaMessage(msg)

			// 带重试的消息处理
			var lastErr error
			var finalResult mq.ConsumeResult

			for retry := 0; retry < defaultMaxRetries; retry++ {
				result, err := handler(session.Context(), []*mq.ConsumedMessage{convertedMsg})
				if err == nil && result != mq.ConsumeRetryLater {
					finalResult = result
					lastErr = nil
					break
				}
				if err == nil {
					err = fmt.Errorf("consume retry later")
				}
				lastErr = err

				h.adapter.logger.Warn("message handling failed, retrying",
					zap.String("topic", topic),
					zap.Int32("partition", msg.Partition),
					zap.Int64("offset", msg.Offset),
					zap.Int("retry", retry+1),
					zap.Int("max_retries", defaultMaxRetries),
					zap.Error(err),
				)

				// 指数退避
				select {
				case <-session.Context().Done():
					return nil
				case <-time.After(defaultRetryBaseDelay * time.Duration(retry+1)):
				}
			}

			if lastErr != nil {
				h.adapter.logger.Error("message handling failed after all retries, stopping consumer to prevent data loss",
					zap.String("topic", topic),
					zap.Int32("partition", msg.Partition),
					zap.Int64("offset", msg.Offset),
					zap.Error(lastErr),
				)
				// 返回错误给 Sarama，这将停止当前分区的消费并触发重平衡
				// 确保 offset 不会被错误地提交
				return lastErr
			}

			// 只有成功处理才标记消息已消费
			session.MarkMessage(msg, "")
			if finalResult == mq.ConsumeCommit || !h.adapter.config.Consumer.AutoCommit {
				session.Commit()
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

// =============================================================================
// 辅助函数
// =============================================================================

func buildConsumerConfig(cfg *mq.KafkaConfig) (*sarama.Config, error) {
	saramaCfg := sarama.NewConfig()

	// 版本
	if cfg.Version != "" {
		version, err := sarama.ParseKafkaVersion(cfg.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid kafka version: %w", err)
		}
		saramaCfg.Version = version
	}

	// Consumer 配置
	saramaCfg.Consumer.Return.Errors = true

	// 初始偏移量
	switch cfg.Consumer.InitialOffset {
	case "oldest":
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	default:
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	// 自动提交
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = cfg.Consumer.AutoCommit
	if cfg.Consumer.AutoCommitInterval > 0 {
		saramaCfg.Consumer.Offsets.AutoCommit.Interval = cfg.Consumer.AutoCommitInterval
	}

	// 会话超时
	if cfg.Consumer.SessionTimeout > 0 {
		saramaCfg.Consumer.Group.Session.Timeout = cfg.Consumer.SessionTimeout
	}

	// 心跳间隔
	if cfg.Consumer.HeartbeatInterval > 0 {
		saramaCfg.Consumer.Group.Heartbeat.Interval = cfg.Consumer.HeartbeatInterval
	}

	// Fetch 配置
	if cfg.Consumer.FetchMin > 0 {
		saramaCfg.Consumer.Fetch.Min = cfg.Consumer.FetchMin
	}
	if cfg.Consumer.FetchMax > 0 {
		saramaCfg.Consumer.Fetch.Max = cfg.Consumer.FetchMax
	}
	if cfg.Consumer.FetchDefault > 0 {
		saramaCfg.Consumer.Fetch.Default = cfg.Consumer.FetchDefault
	}
	if cfg.Consumer.MaxWaitTime > 0 {
		saramaCfg.Consumer.MaxWaitTime = cfg.Consumer.MaxWaitTime
	}
	if cfg.Consumer.MaxProcessingTime > 0 {
		saramaCfg.Consumer.MaxProcessingTime = cfg.Consumer.MaxProcessingTime
	}

	// SASL
	if cfg.SASL.Enable {
		saramaCfg.Net.SASL.Enable = true
		saramaCfg.Net.SASL.User = cfg.SASL.Username
		saramaCfg.Net.SASL.Password = cfg.SASL.Password

		switch cfg.SASL.Mechanism {
		case "SCRAM-SHA-256":
			saramaCfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			}
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			saramaCfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
			}
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		default:
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	// TLS
	if cfg.TLS.Enable {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		saramaCfg.Net.TLS.Enable = true
		saramaCfg.Net.TLS.Config = tlsConfig
	}

	return saramaCfg, nil
}

func convertFromKafkaMessage(msg *sarama.ConsumerMessage) *mq.ConsumedMessage {
	result := &mq.ConsumedMessage{
		Topic:      msg.Topic,
		Body:       msg.Value,
		MsgID:      fmt.Sprintf("%s-%d-%d", msg.Topic, msg.Partition, msg.Offset),
		Offset:     msg.Offset,
		Partition:  msg.Partition,
		BornTime:   msg.Timestamp,
		Properties: make(map[string]string),
	}

	if msg.Key != nil {
		result.Key = string(msg.Key)
	}

	// Headers -> Properties
	for _, header := range msg.Headers {
		key := string(header.Key)
		value := string(header.Value)

		if key == "X-Tag" {
			result.Tag = value
		} else {
			result.Properties[key] = value
		}
	}

	return result
}
