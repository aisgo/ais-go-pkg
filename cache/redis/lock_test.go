package redis

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLockAcquireRelease(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	lock := client.NewLock("resource", LockOption{TTL: 200 * time.Millisecond, RetryTimes: 1, RetryDelay: 10 * time.Millisecond})
	if err := lock.Acquire(ctx); err != nil {
		t.Fatalf("acquire lock: %v", err)
	}

	lock2 := client.NewLock("resource", LockOption{TTL: 200 * time.Millisecond, RetryTimes: 1, RetryDelay: 10 * time.Millisecond})
	if err := lock2.Acquire(ctx); !errors.Is(err, ErrLockFailed) {
		t.Fatalf("expected ErrLockFailed, got: %v", err)
	}

	if err := lock.Release(ctx); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	if err := lock2.Acquire(ctx); err != nil {
		t.Fatalf("acquire lock after release: %v", err)
	}
}

func TestLockAutoExtendIgnoresParentCancel(t *testing.T) {
	client := newTestClient(t)
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lock := client.NewLock("auto", LockOption{TTL: 120 * time.Millisecond, RetryTimes: 1, AutoExtend: true, ExtendFactor: 0.5, IgnoreParentCancel: true})
	if err := lock.Acquire(parentCtx); err != nil {
		t.Fatalf("acquire lock: %v", err)
	}

	cancel()
	// 等待超过 TTL，若续期生效则锁仍存在
	time.Sleep(300 * time.Millisecond)

	exists, err := client.Exists(context.Background(), lock.key)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists == 0 {
		t.Fatalf("expected lock to be extended and still exist")
	}

	if err := lock.Release(context.Background()); err != nil {
		t.Fatalf("release lock: %v", err)
	}
}
