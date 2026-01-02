package rocketmq

import (
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

/* ========================================================================
 * RocketMQ Message Options
 * ========================================================================
 * 职责: 提供消息选项设置
 * ======================================================================== */

// MessageOption 消息选项函数
type MessageOption func(*primitive.Message)

// WithKeys 设置消息 Keys（用于消息查询和去重）
func WithKeys(keys []string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithKeys(keys)
	}
}

// WithKey 设置单个消息 Key
func WithKey(key string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithKeys([]string{key})
	}
}

// WithTag 设置消息 Tag（用于消息过滤）
func WithTag(tag string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithTag(tag)
	}
}

// WithProperty 设置消息属性
func WithProperty(key, value string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithProperty(key, value)
	}
}

// WithProperties 批量设置消息属性
func WithProperties(props map[string]string) MessageOption {
	return func(msg *primitive.Message) {
		for k, v := range props {
			msg.WithProperty(k, v)
		}
	}
}

// WithDelayTimeLevel 设置延迟消息级别
// Level: 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h
// 对应级别: 1  2  3   4   5  6  7  8  9  10 11 12 13 14  15  16  17 18
func WithDelayTimeLevel(level int) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithDelayTimeLevel(level)
	}
}

// WithShardingKey 设置分区键（顺序消息）
func WithShardingKey(key string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithShardingKey(key)
	}
}

// WithTraceID 设置追踪 ID
func WithTraceID(traceID string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithProperty("TRACE_ID", traceID)
	}
}

// WithTenantID 设置租户 ID
func WithTenantID(tenantID string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithProperty("TENANT_ID", tenantID)
	}
}

// WithEventType 设置事件类型
func WithEventType(eventType string) MessageOption {
	return func(msg *primitive.Message) {
		msg.WithProperty("EVENT_TYPE", eventType)
	}
}
