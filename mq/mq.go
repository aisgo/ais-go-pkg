package mq

import (
	"context"
	"time"
)

/* ========================================================================
 * MQ 抽象接口 - 支持 RocketMQ / Kafka 切换
 * ========================================================================
 * 职责: 定义统一的消息队列接口
 * 支持: RocketMQ, Kafka
 * ======================================================================== */

// Producer 消息生产者接口
type Producer interface {
	// SendSync 同步发送消息
	SendSync(ctx context.Context, msg *Message) (*SendResult, error)

	// SendAsync 异步发送消息
	SendAsync(ctx context.Context, msg *Message, callback SendCallback) error

	// Close 关闭生产者
	Close() error
}

// Consumer 消息消费者接口
type Consumer interface {
	// Subscribe 订阅主题
	Subscribe(topic string, handler MessageHandler) error

	// Start 启动消费者
	Start() error

	// Close 关闭消费者
	Close() error
}

// =============================================================================
// 消息模型
// =============================================================================

// Message 消息结构（MQ 无关）
type Message struct {
	Topic      string            // 主题
	Body       []byte            // 消息体
	Key        string            // 消息键（用于分区/顺序）
	Tag        string            // 标签（RocketMQ 特有，Kafka 忽略）
	Properties map[string]string // 自定义属性
	DelayLevel int               // 延迟级别（RocketMQ 特有）
	DelayTime  time.Duration     // 延迟时间（Kafka 可通过 header 实现）
}

// NewMessage 创建消息
func NewMessage(topic string, body []byte) *Message {
	return &Message{
		Topic:      topic,
		Body:       body,
		Properties: make(map[string]string),
	}
}

// WithKey 设置消息键
func (m *Message) WithKey(key string) *Message {
	m.Key = key
	return m
}

// WithTag 设置标签
func (m *Message) WithTag(tag string) *Message {
	m.Tag = tag
	return m
}

// WithProperty 设置属性
func (m *Message) WithProperty(key, value string) *Message {
	if m.Properties == nil {
		m.Properties = make(map[string]string)
	}
	m.Properties[key] = value
	return m
}

// WithProperties 批量设置属性
func (m *Message) WithProperties(props map[string]string) *Message {
	if m.Properties == nil {
		m.Properties = make(map[string]string)
	}
	for k, v := range props {
		m.Properties[k] = v
	}
	return m
}

// WithDelayLevel 设置延迟级别（RocketMQ）
func (m *Message) WithDelayLevel(level int) *Message {
	m.DelayLevel = level
	return m
}

// WithDelayTime 设置延迟时间
func (m *Message) WithDelayTime(d time.Duration) *Message {
	m.DelayTime = d
	return m
}

// =============================================================================
// 消费消息模型
// =============================================================================

// ConsumedMessage 已消费的消息
type ConsumedMessage struct {
	Topic        string            // 主题
	Body         []byte            // 消息体
	Key          string            // 消息键
	Tag          string            // 标签
	Properties   map[string]string // 属性
	MsgID        string            // 消息 ID
	Offset       int64             // 偏移量（Kafka）
	Partition    int32             // 分区（Kafka）
	BornTime     time.Time         // 消息产生时间
	ReconsumeCnt int32             // 重试次数
}

// =============================================================================
// 回调与结果
// =============================================================================

// SendResult 发送结果
type SendResult struct {
	MsgID     string // 消息 ID
	Topic     string // 主题
	Partition int32  // 分区（Kafka）
	Offset    int64  // 偏移量（Kafka）
	Status    SendStatus
}

// SendStatus 发送状态
type SendStatus int

const (
	SendStatusOK SendStatus = iota
	SendStatusFlushDiskTimeout
	SendStatusFlushSlaveTimeout
	SendStatusSlaveNotAvailable
	SendStatusUnknownError
)

// SendCallback 异步发送回调
type SendCallback func(result *SendResult, err error)

// =============================================================================
// 消费处理
// =============================================================================

// ConsumeResult 消费结果
type ConsumeResult int

const (
	ConsumeSuccess    ConsumeResult = iota // 消费成功
	ConsumeRetryLater                      // 稍后重试
	ConsumeCommit                          // 提交（Kafka）
)

// MessageHandler 消息处理函数
type MessageHandler func(ctx context.Context, msgs []*ConsumedMessage) (ConsumeResult, error)

// =============================================================================
// MQ 类型
// =============================================================================

// Type MQ 类型
type Type string

const (
	TypeRocketMQ Type = "rocketmq"
	TypeKafka    Type = "kafka"
)
