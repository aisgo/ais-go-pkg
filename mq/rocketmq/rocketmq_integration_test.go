//go:build integration

package rocketmq

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"
	"github.com/aisgo/ais-go-pkg/mq"

	"github.com/google/uuid"
)

func TestRocketMQProducerConsumerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}

	nameSrv := os.Getenv("ROCKETMQ_NAMESRV")
	if nameSrv == "" {
		t.Skip("set ROCKETMQ_NAMESRV to run RocketMQ integration test")
	}

	cfg := mq.DefaultRocketMQConfig()
	cfg.NameServers = strings.Split(nameSrv, ",")
	cfg.Producer.GroupName = "producer-" + uuid.NewString()
	cfg.Consumer.GroupName = "consumer-" + uuid.NewString()
	cfg.Consumer.ConsumeFromWhere = "LastOffset"

	fullCfg := &mq.Config{Type: mq.TypeRocketMQ, RocketMQ: cfg}

	consumer, err := NewConsumerAdapter(fullCfg, logger.NewNop())
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}

	topic := "topic-" + uuid.NewString()
	received := make(chan struct{}, 1)
	if err := consumer.Subscribe(topic, func(ctx context.Context, msgs []*mq.ConsumedMessage) (mq.ConsumeResult, error) {
		if len(msgs) > 0 {
			if string(msgs[0].Body) == "hello" {
				received <- struct{}{}
			}
		}
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
	if _, err := producer.SendSync(context.Background(), msg); err != nil {
		t.Fatalf("send sync: %v", err)
	}

	select {
	case <-received:
	case <-time.After(20 * time.Second):
		t.Fatalf("timeout waiting for message")
	}
}
