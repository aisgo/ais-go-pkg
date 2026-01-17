package shutdown

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"
)

func TestShutdownHookTimeout(t *testing.T) {
	m := NewManager(ManagerParams{
		Logger: logger.NewNop(),
		Config: &Config{
			Timeout:     time.Second,
			HookTimeout: 50 * time.Millisecond,
		},
	})

	var fastCalled atomic.Bool

	m.RegisterHookWithPriority("slow", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, PriorityNormal)
	m.RegisterHookWithPriority("fast", func(ctx context.Context) error {
		fastCalled.Store(true)
		return nil
	}, PriorityNormal)

	start := time.Now()
	m.Shutdown(context.Background())
	elapsed := time.Since(start)

	if !fastCalled.Load() {
		t.Fatalf("fast hook not executed")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("shutdown took too long: %v", elapsed)
	}
}
