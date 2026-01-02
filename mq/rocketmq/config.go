package rocketmq

import "time"

/* ========================================================================
 * RocketMQ 配置
 * ========================================================================
 * 职责: 定义 RocketMQ 专用配置结构（向后兼容）
 * ======================================================================== */

// Config RocketMQ 配置
type Config struct {
	// NameServer 地址列表
	NameServers []string `yaml:"name_servers" mapstructure:"name_servers"`

	// Producer 配置
	Producer ProducerConfig `yaml:"producer" mapstructure:"producer"`

	// Consumer 配置
	Consumer ConsumerConfig `yaml:"consumer" mapstructure:"consumer"`

	// 通用配置
	Namespace      string        `yaml:"namespace" mapstructure:"namespace"`               // 命名空间
	InstanceName   string        `yaml:"instance_name" mapstructure:"instance_name"`       // 实例名称
	RetryTimes     int           `yaml:"retry_times" mapstructure:"retry_times"`           // 重试次数
	SendMsgTimeout time.Duration `yaml:"send_msg_timeout" mapstructure:"send_msg_timeout"` // 发送超时
	EnableTrace    bool          `yaml:"enable_trace" mapstructure:"enable_trace"`         // 是否启用消息轨迹
	AccessKey      string        `yaml:"access_key" mapstructure:"access_key"`             // AccessKey (ACL)
	SecretKey      string        `yaml:"secret_key" mapstructure:"secret_key"`             // SecretKey (ACL)
}

// ProducerConfig Producer 配置
type ProducerConfig struct {
	GroupName          string        `yaml:"group_name" mapstructure:"group_name"`                       // 生产者组名
	MaxMessageSize     int           `yaml:"max_message_size" mapstructure:"max_message_size"`           // 最大消息大小 (默认 4MB)
	CompressLevel      int           `yaml:"compress_level" mapstructure:"compress_level"`               // 压缩级别 (0-9)
	SendMsgTimeout     time.Duration `yaml:"send_msg_timeout" mapstructure:"send_msg_timeout"`           // 发送超时
	RetryTimesOnFailed int           `yaml:"retry_times_on_failed" mapstructure:"retry_times_on_failed"` // 发送失败重试次数
}

// ConsumerConfig Consumer 配置
type ConsumerConfig struct {
	GroupName              string        `yaml:"group_name" mapstructure:"group_name"`                               // 消费者组名
	Model                  string        `yaml:"model" mapstructure:"model"`                                         // 消费模式: Clustering / Broadcasting
	ConsumeFromWhere       string        `yaml:"consume_from_where" mapstructure:"consume_from_where"`               // 消费位置: FirstOffset / LastOffset / Timestamp
	ConsumeMessageBatchMax int           `yaml:"consume_message_batch_max" mapstructure:"consume_message_batch_max"` // 批量消费最大消息数
	PullBatchSize          int32         `yaml:"pull_batch_size" mapstructure:"pull_batch_size"`                     // 拉取批量大小
	PullInterval           time.Duration `yaml:"pull_interval" mapstructure:"pull_interval"`                         // 拉取间隔
	MaxReconsumeTimes      int32         `yaml:"max_reconsume_times" mapstructure:"max_reconsume_times"`             // 最大重试次数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		NameServers: []string{"127.0.0.1:9876"},
		Producer: ProducerConfig{
			GroupName:          "default_producer_group",
			MaxMessageSize:     4 * 1024 * 1024, // 4MB
			CompressLevel:      5,
			SendMsgTimeout:     3 * time.Second,
			RetryTimesOnFailed: 2,
		},
		Consumer: ConsumerConfig{
			GroupName:              "default_consumer_group",
			Model:                  "Clustering",
			ConsumeFromWhere:       "LastOffset",
			ConsumeMessageBatchMax: 1,
			PullBatchSize:          32,
			PullInterval:           0,
			MaxReconsumeTimes:      16,
		},
		Namespace:      "",
		InstanceName:   "DEFAULT",
		RetryTimes:     2,
		SendMsgTimeout: 3 * time.Second,
		EnableTrace:    true,
	}
}
