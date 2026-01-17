package mq

import (
	"context"
	"errors"
	"sort"
	"testing"

	"go.uber.org/zap"
)

type testProducer struct{}

type testConsumer struct{}

func (t testProducer) SendSync(ctx context.Context, msg *Message) (*SendResult, error) {
	return nil, nil
}
func (t testProducer) SendAsync(ctx context.Context, msg *Message, callback SendCallback) error {
	return nil
}
func (t testProducer) Close() error { return nil }

func (t testConsumer) Subscribe(topic string, handler MessageHandler) error { return nil }
func (t testConsumer) Start() error                                         { return nil }
func (t testConsumer) Close() error                                         { return nil }

func snapshotFactories() (map[Type]ProducerFactory, map[Type]ConsumerFactory) {
	factoryMu.RLock()
	defer factoryMu.RUnlock()

	p := make(map[Type]ProducerFactory, len(producerFactories))
	for k, v := range producerFactories {
		p[k] = v
	}
	c := make(map[Type]ConsumerFactory, len(consumerFactories))
	for k, v := range consumerFactories {
		c[k] = v
	}
	return p, c
}

func restoreFactories(p map[Type]ProducerFactory, c map[Type]ConsumerFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	producerFactories = p
	consumerFactories = c
}

func TestFactoryErrors(t *testing.T) {
	if _, err := NewProducer(nil, nil); err == nil {
		t.Fatalf("expected error for nil config")
	}
	if _, err := NewConsumer(nil, nil); err == nil {
		t.Fatalf("expected error for nil config")
	}

	p, c := snapshotFactories()
	restoreFactories(make(map[Type]ProducerFactory), make(map[Type]ConsumerFactory))
	t.Cleanup(func() { restoreFactories(p, c) })

	_, err := NewProducer(&Config{Type: "unknown"}, zap.NewNop())
	if err == nil {
		t.Fatalf("expected error for unsupported producer type")
	}
	_, err = NewConsumer(&Config{Type: "unknown"}, zap.NewNop())
	if err == nil {
		t.Fatalf("expected error for unsupported consumer type")
	}
}

func TestFactoryRegisterAndCreate(t *testing.T) {
	p, c := snapshotFactories()
	restoreFactories(make(map[Type]ProducerFactory), make(map[Type]ConsumerFactory))
	t.Cleanup(func() { restoreFactories(p, c) })

	RegisterProducerFactory(TypeKafka, func(cfg *Config, logger *zap.Logger) (Producer, error) {
		return testProducer{}, nil
	})
	RegisterConsumerFactory(TypeKafka, func(cfg *Config, logger *zap.Logger) (Consumer, error) {
		return testConsumer{}, nil
	})

	producer, err := NewProducer(&Config{Type: TypeKafka}, nil)
	if err != nil {
		t.Fatalf("new producer: %v", err)
	}
	if producer == nil {
		t.Fatalf("expected producer")
	}

	consumer, err := NewConsumer(&Config{Type: TypeKafka}, nil)
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}
	if consumer == nil {
		t.Fatalf("expected consumer")
	}
}

func TestAvailableTypes(t *testing.T) {
	p, c := snapshotFactories()
	restoreFactories(make(map[Type]ProducerFactory), make(map[Type]ConsumerFactory))
	t.Cleanup(func() { restoreFactories(p, c) })

	RegisterProducerFactory(TypeKafka, func(cfg *Config, logger *zap.Logger) (Producer, error) {
		return testProducer{}, nil
	})
	RegisterProducerFactory(TypeRocketMQ, func(cfg *Config, logger *zap.Logger) (Producer, error) {
		return testProducer{}, nil
	})

	types := AvailableTypes()
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	if len(types) != 2 || types[0] != TypeKafka || types[1] != TypeRocketMQ {
		t.Fatalf("unexpected types: %v", types)
	}
}

func TestFactoryPropagatesError(t *testing.T) {
	p, c := snapshotFactories()
	restoreFactories(make(map[Type]ProducerFactory), make(map[Type]ConsumerFactory))
	t.Cleanup(func() { restoreFactories(p, c) })

	expectedErr := errors.New("boom")
	RegisterProducerFactory(TypeKafka, func(cfg *Config, logger *zap.Logger) (Producer, error) {
		return nil, expectedErr
	})

	_, err := NewProducer(&Config{Type: TypeKafka}, zap.NewNop())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("unexpected error: %v", err)
	}
}
