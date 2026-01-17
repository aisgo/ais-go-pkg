//go:build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"
	"github.com/aisgo/ais-go-pkg/mq"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestKafkaProducerConsumerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()

	container, err := kafka.Run(ctx, "confluentinc/cp-kafka:7.5.0", kafka.WithClusterID("ais-test"))
	if err != nil {
		t.Fatalf("start kafka container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	brokers, err := container.Brokers(ctx)
	if err != nil {
		t.Fatalf("brokers: %v", err)
	}

	kafkaCfg := mq.DefaultKafkaConfig()
	kafkaCfg.Brokers = brokers
	kafkaCfg.Version = "2.8.0"
	kafkaCfg.Consumer.GroupID = "group-" + uuid.NewString()
	kafkaCfg.Consumer.InitialOffset = "oldest"

	fullCfg := &mq.Config{Type: mq.TypeKafka, Kafka: kafkaCfg}

	// 创建 topic
	adminCfg, err := buildSaramaConfig(kafkaCfg)
	if err != nil {
		t.Fatalf("sarama config: %v", err)
	}
	admin, err := sarama.NewClusterAdmin(brokers, adminCfg)
	if err != nil {
		t.Fatalf("new cluster admin: %v", err)
	}
	defer admin.Close()

	topic := "topic-" + uuid.NewString()
	err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
	if err != nil && !errors.Is(err, sarama.ErrTopicAlreadyExists) {
		t.Fatalf("create topic: %v", err)
	}

	consumer, err := NewConsumerAdapter(fullCfg, logger.NewNop())
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}
	received := make(chan string, 1)
	if err := consumer.Subscribe(topic, func(ctx context.Context, msgs []*mq.ConsumedMessage) (mq.ConsumeResult, error) {
		if len(msgs) == 0 {
			return mq.ConsumeRetryLater, fmt.Errorf("empty message")
		}
		received <- string(msgs[0].Body)
		return mq.ConsumeSuccess, nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := consumer.Start(); err != nil {
		_ = consumer.Close()
		t.Fatalf("start consumer: %v", err)
	}
	t.Cleanup(func() {
		_ = consumer.Close()
	})

	producer, err := NewProducerAdapter(fullCfg, logger.NewNop())
	if err != nil {
		t.Fatalf("new producer: %v", err)
	}
	t.Cleanup(func() {
		_ = producer.Close()
	})

	msg := mq.NewMessage(topic, []byte("hello"))
	if _, err := producer.SendSync(ctx, msg); err != nil {
		t.Fatalf("send sync: %v", err)
	}

	select {
	case got := <-received:
		if got != "hello" {
			t.Fatalf("unexpected payload: %s", got)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("timeout waiting for message")
	}
}
