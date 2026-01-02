package mq

import "time"

/* ========================================================================
 * MQ 统一配置
 * ========================================================================
 * 职责: 定义 RocketMQ 和 Kafka 的统一配置结构
 * ======================================================================== */

// Config MQ 统一配置
type Config struct {
	// Type MQ 类型: rocketmq / kafka
	Type Type `yaml:"type" mapstructure:"type"`

	// RocketMQ 特有配置
	RocketMQ *RocketMQConfig `yaml:"rocketmq" mapstructure:"rocketmq"`

	// Kafka 特有配置
	Kafka *KafkaConfig `yaml:"kafka" mapstructure:"kafka"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Type:     TypeRocketMQ,
		RocketMQ: DefaultRocketMQConfig(),
		Kafka:    DefaultKafkaConfig(),
	}
}

// =============================================================================
// RocketMQ 配置
// =============================================================================

// RocketMQConfig RocketMQ 配置
type RocketMQConfig struct {
	NameServers  []string      `yaml:"name_servers" mapstructure:"name_servers"`
	Namespace    string        `yaml:"namespace" mapstructure:"namespace"`
	InstanceName string        `yaml:"instance_name" mapstructure:"instance_name"`
	AccessKey    string        `yaml:"access_key" mapstructure:"access_key"`
	SecretKey    string        `yaml:"secret_key" mapstructure:"secret_key"`

	Producer RocketMQProducerConfig `yaml:"producer" mapstructure:"producer"`
	Consumer RocketMQConsumerConfig `yaml:"consumer" mapstructure:"consumer"`
}

// RocketMQProducerConfig RocketMQ 生产者配置
type RocketMQProducerConfig struct {
	GroupName          string        `yaml:"group_name" mapstructure:"group_name"`
	SendMsgTimeout     time.Duration `yaml:"send_msg_timeout" mapstructure:"send_msg_timeout"`
	RetryTimesOnFailed int           `yaml:"retry_times_on_failed" mapstructure:"retry_times_on_failed"`
	MaxMessageSize     int           `yaml:"max_message_size" mapstructure:"max_message_size"`
	CompressLevel      int           `yaml:"compress_level" mapstructure:"compress_level"`
}

// RocketMQConsumerConfig RocketMQ 消费者配置
type RocketMQConsumerConfig struct {
	GroupName              string        `yaml:"group_name" mapstructure:"group_name"`
	Model                  string        `yaml:"model" mapstructure:"model"`                                           // Clustering / Broadcasting
	ConsumeFromWhere       string        `yaml:"consume_from_where" mapstructure:"consume_from_where"`                 // FirstOffset / LastOffset / Timestamp
	ConsumeMessageBatchMax int           `yaml:"consume_message_batch_max" mapstructure:"consume_message_batch_max"`
	PullBatchSize          int32         `yaml:"pull_batch_size" mapstructure:"pull_batch_size"`
	PullInterval           time.Duration `yaml:"pull_interval" mapstructure:"pull_interval"`
	MaxReconsumeTimes      int32         `yaml:"max_reconsume_times" mapstructure:"max_reconsume_times"`
}

// DefaultRocketMQConfig 返回 RocketMQ 默认配置
func DefaultRocketMQConfig() *RocketMQConfig {
	return &RocketMQConfig{
		NameServers:  []string{"127.0.0.1:9876"},
		InstanceName: "DEFAULT",
		Producer: RocketMQProducerConfig{
			GroupName:          "default_producer_group",
			SendMsgTimeout:     3 * time.Second,
			RetryTimesOnFailed: 2,
			MaxMessageSize:     4 * 1024 * 1024,
			CompressLevel:      5,
		},
		Consumer: RocketMQConsumerConfig{
			GroupName:              "default_consumer_group",
			Model:                  "Clustering",
			ConsumeFromWhere:       "LastOffset",
			ConsumeMessageBatchMax: 1,
			PullBatchSize:          32,
			PullInterval:           0,
			MaxReconsumeTimes:      16,
		},
	}
}

// =============================================================================
// Kafka 配置
// =============================================================================

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers []string `yaml:"brokers" mapstructure:"brokers"`
	Version string   `yaml:"version" mapstructure:"version"` // Kafka 版本

	// SASL 认证
	SASL KafkaSASLConfig `yaml:"sasl" mapstructure:"sasl"`

	// TLS 配置
	TLS KafkaTLSConfig `yaml:"tls" mapstructure:"tls"`

	Producer KafkaProducerConfig `yaml:"producer" mapstructure:"producer"`
	Consumer KafkaConsumerConfig `yaml:"consumer" mapstructure:"consumer"`
}

// KafkaSASLConfig Kafka SASL 认证配置
type KafkaSASLConfig struct {
	Enable    bool   `yaml:"enable" mapstructure:"enable"`
	Mechanism string `yaml:"mechanism" mapstructure:"mechanism"` // PLAIN / SCRAM-SHA-256 / SCRAM-SHA-512
	Username  string `yaml:"username" mapstructure:"username"`
	Password  string `yaml:"password" mapstructure:"password"`
}

// KafkaTLSConfig Kafka TLS 配置
type KafkaTLSConfig struct {
	Enable   bool   `yaml:"enable" mapstructure:"enable"`
	CertFile string `yaml:"cert_file" mapstructure:"cert_file"`
	KeyFile  string `yaml:"key_file" mapstructure:"key_file"`
	CAFile   string `yaml:"ca_file" mapstructure:"ca_file"`
	Insecure bool   `yaml:"insecure" mapstructure:"insecure"` // 跳过证书验证
}

// KafkaProducerConfig Kafka 生产者配置
type KafkaProducerConfig struct {
	RequiredAcks    string        `yaml:"required_acks" mapstructure:"required_acks"` // none / leader / all
	Timeout         time.Duration `yaml:"timeout" mapstructure:"timeout"`
	MaxMessageBytes int           `yaml:"max_message_bytes" mapstructure:"max_message_bytes"`
	Compression     string        `yaml:"compression" mapstructure:"compression"` // none / gzip / snappy / lz4 / zstd
	Idempotent      bool          `yaml:"idempotent" mapstructure:"idempotent"`
	RetryMax        int           `yaml:"retry_max" mapstructure:"retry_max"`
}

// KafkaConsumerConfig Kafka 消费者配置
type KafkaConsumerConfig struct {
	GroupID            string        `yaml:"group_id" mapstructure:"group_id"`
	InitialOffset      string        `yaml:"initial_offset" mapstructure:"initial_offset"` // newest / oldest
	AutoCommit         bool          `yaml:"auto_commit" mapstructure:"auto_commit"`
	AutoCommitInterval time.Duration `yaml:"auto_commit_interval" mapstructure:"auto_commit_interval"`
	SessionTimeout     time.Duration `yaml:"session_timeout" mapstructure:"session_timeout"`
	HeartbeatInterval  time.Duration `yaml:"heartbeat_interval" mapstructure:"heartbeat_interval"`
	MaxWaitTime        time.Duration `yaml:"max_wait_time" mapstructure:"max_wait_time"`
	MaxProcessingTime  time.Duration `yaml:"max_processing_time" mapstructure:"max_processing_time"`
	FetchMin           int32         `yaml:"fetch_min" mapstructure:"fetch_min"`
	FetchMax           int32         `yaml:"fetch_max" mapstructure:"fetch_max"`
	FetchDefault       int32         `yaml:"fetch_default" mapstructure:"fetch_default"`
}

// DefaultKafkaConfig 返回 Kafka 默认配置
func DefaultKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		Brokers: []string{"127.0.0.1:9092"},
		Version: "2.8.0",
		Producer: KafkaProducerConfig{
			RequiredAcks:    "leader",
			Timeout:         10 * time.Second,
			MaxMessageBytes: 1024 * 1024,
			Compression:     "none",
			Idempotent:      false,
			RetryMax:        3,
		},
		Consumer: KafkaConsumerConfig{
			GroupID:            "default_consumer_group",
			InitialOffset:      "newest",
			AutoCommit:         true,
			AutoCommitInterval: 1 * time.Second,
			SessionTimeout:     10 * time.Second,
			HeartbeatInterval:  3 * time.Second,
			MaxWaitTime:        250 * time.Millisecond,
			MaxProcessingTime:  100 * time.Millisecond,
			FetchMin:           1,
			FetchMax:           10485760,
			FetchDefault:       1048576,
		},
	}
}
