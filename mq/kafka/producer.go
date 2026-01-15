package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/aisgo/ais-go-pkg/mq"
)

/* ========================================================================
 * Kafka Producer - Kafka 消息生产者
 * ========================================================================
 * 职责: 实现 mq.Producer 接口
 * 技术: IBM/sarama
 * ======================================================================== */

// =============================================================================
// 注册工厂
// =============================================================================

func init() {
	mq.RegisterProducerFactory(mq.TypeKafka, NewProducerAdapter)
}

// =============================================================================
// Producer 适配器
// =============================================================================

// ProducerAdapter Kafka 生产者适配器
type ProducerAdapter struct {
	syncProducer  sarama.SyncProducer
	asyncProducer sarama.AsyncProducer
	logger        *zap.Logger
	wg            sync.WaitGroup
	closed        bool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewProducerAdapter 创建 Kafka 生产者适配器
func NewProducerAdapter(cfg *mq.Config, logger *zap.Logger) (mq.Producer, error) {
	if cfg.Kafka == nil {
		return nil, fmt.Errorf("kafka config is required")
	}

	kafkaCfg := cfg.Kafka

	// 构建 Sarama 配置
	saramaCfg, err := buildSaramaConfig(kafkaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build sarama config: %w", err)
	}

	// 创建同步生产者
	syncProducer, err := sarama.NewSyncProducer(kafkaCfg.Brokers, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka sync producer: %w", err)
	}

	// 创建异步生产者
	asyncProducer, err := sarama.NewAsyncProducer(kafkaCfg.Brokers, saramaCfg)
	if err != nil {
		syncProducer.Close()
		return nil, fmt.Errorf("failed to create kafka async producer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &ProducerAdapter{
		syncProducer:  syncProducer,
		asyncProducer: asyncProducer,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
	}

	// 启动异步错误处理
	adapter.wg.Add(1)
	go adapter.handleAsyncErrors()

	logger.Info("Kafka producer started",
		zap.Strings("brokers", kafkaCfg.Brokers),
	)

	return adapter, nil
}

// handleAsyncErrors 处理异步发送错误
func (p *ProducerAdapter) handleAsyncErrors() {
	defer p.wg.Done()

	for {
		select {
		case err, ok := <-p.asyncProducer.Errors():
			if !ok {
				return
			}
			if cb, ok := err.Msg.Metadata.(mq.SendCallback); ok && cb != nil {
				cb(nil, err.Err)
			} else {
				p.logger.Error("async producer error",
					zap.String("topic", err.Msg.Topic),
					zap.Error(err.Err),
				)
			}
		case msg, ok := <-p.asyncProducer.Successes():
			if !ok {
				return
			}
			if cb, ok := msg.Metadata.(mq.SendCallback); ok && cb != nil {
				cb(&mq.SendResult{
					MsgID:     fmt.Sprintf("%s-%d-%d", msg.Topic, msg.Partition, msg.Offset),
					Topic:     msg.Topic,
					Partition: msg.Partition,
					Offset:    msg.Offset,
					Status:    mq.SendStatusOK,
				}, nil)
			} else {
				p.logger.Debug("async message sent",
					zap.String("topic", msg.Topic),
					zap.Int32("partition", msg.Partition),
					zap.Int64("offset", msg.Offset),
				)
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// SendSync 同步发送消息
func (p *ProducerAdapter) SendSync(ctx context.Context, msg *mq.Message) (*mq.SendResult, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	kafkaMsg := convertToKafkaMessage(msg)

	partition, offset, err := p.syncProducer.SendMessage(kafkaMsg)
	if err != nil {
		p.logger.Error("failed to send message",
			zap.String("topic", msg.Topic),
			zap.Error(err),
		)
		return nil, err
	}

	p.logger.Debug("message sent",
		zap.String("topic", msg.Topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)

	return &mq.SendResult{
		MsgID:     fmt.Sprintf("%s-%d-%d", msg.Topic, partition, offset),
		Topic:     msg.Topic,
		Partition: partition,
		Offset:    offset,
		Status:    mq.SendStatusOK,
	}, nil
}

// SendAsync 异步发送消息
func (p *ProducerAdapter) SendAsync(ctx context.Context, msg *mq.Message, callback mq.SendCallback) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	kafkaMsg := convertToKafkaMessage(msg)
	kafkaMsg.Metadata = callback

	// 注意：Sarama 的异步 Producer 不支持单消息回调
	// 回调通过 Successes() 和 Errors() channel 处理（使用 ProducerMessage.Metadata 关联）
	select {
	case p.asyncProducer.Input() <- kafkaMsg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close 关闭生产者
func (p *ProducerAdapter) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	var errs []error

	if err := p.asyncProducer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("async producer close error: %w", err))
	}
	// 确保后台 goroutine 退出
	p.cancel()

	if err := p.syncProducer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("sync producer close error: %w", err))
	}

	p.wg.Wait()

	if len(errs) > 0 {
		p.logger.Error("failed to close producer", zap.Errors("errors", errs))
		return errs[0]
	}

	p.logger.Info("Kafka producer closed")
	return nil
}

// =============================================================================
// 辅助函数
// =============================================================================

func buildSaramaConfig(cfg *mq.KafkaConfig) (*sarama.Config, error) {
	saramaCfg := sarama.NewConfig()

	// 版本
	if cfg.Version != "" {
		version, err := sarama.ParseKafkaVersion(cfg.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid kafka version: %w", err)
		}
		saramaCfg.Version = version
	}

	// Producer 配置
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Producer.Retry.Max = cfg.Producer.RetryMax
	saramaCfg.Producer.Timeout = cfg.Producer.Timeout

	// ACKs
	switch cfg.Producer.RequiredAcks {
	case "none":
		saramaCfg.Producer.RequiredAcks = sarama.NoResponse
	case "leader":
		saramaCfg.Producer.RequiredAcks = sarama.WaitForLocal
	case "all":
		saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	default:
		saramaCfg.Producer.RequiredAcks = sarama.WaitForLocal
	}

	// 压缩
	switch cfg.Producer.Compression {
	case "gzip":
		saramaCfg.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaCfg.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaCfg.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaCfg.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaCfg.Producer.Compression = sarama.CompressionNone
	}

	// 幂等
	saramaCfg.Producer.Idempotent = cfg.Producer.Idempotent
	if cfg.Producer.Idempotent {
		saramaCfg.Net.MaxOpenRequests = 1
	}

	// 消息大小
	if cfg.Producer.MaxMessageBytes > 0 {
		saramaCfg.Producer.MaxMessageBytes = cfg.Producer.MaxMessageBytes
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

func buildTLSConfig(cfg mq.KafkaTLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
	}

	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load cert/key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func convertToKafkaMessage(msg *mq.Message) *sarama.ProducerMessage {
	kafkaMsg := &sarama.ProducerMessage{
		Topic:     msg.Topic,
		Value:     sarama.ByteEncoder(msg.Body),
		Timestamp: time.Now(),
	}

	// Key
	if msg.Key != "" {
		kafkaMsg.Key = sarama.StringEncoder(msg.Key)
	}

	// Headers (properties)
	if len(msg.Properties) > 0 {
		headers := make([]sarama.RecordHeader, 0, len(msg.Properties))
		for k, v := range msg.Properties {
			headers = append(headers, sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			})
		}
		kafkaMsg.Headers = headers
	}

	// Tag 作为 header
	if msg.Tag != "" {
		kafkaMsg.Headers = append(kafkaMsg.Headers, sarama.RecordHeader{
			Key:   []byte("X-Tag"),
			Value: []byte(msg.Tag),
		})
	}

	return kafkaMsg
}
