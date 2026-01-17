package redis

import (
	"context"
	"testing"
	"time"
)

func TestClientCacheOps(t *testing.T) {
	client, server := newTestClientWithServer(t)
	ctx := context.Background()

	if err := client.Set(ctx, "k1", "v1", 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	val, err := client.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "v1" {
		t.Fatalf("unexpected value: %s", val)
	}

	exists, err := client.Exists(ctx, "k1")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists != 1 {
		t.Fatalf("unexpected exists: %d", exists)
	}

	if err := client.Expire(ctx, "k1", 2*time.Second); err != nil {
		t.Fatalf("expire: %v", err)
	}
	server.FastForward(3 * time.Second)

	exists, err = client.Exists(ctx, "k1")
	if err != nil {
		t.Fatalf("exists after expire: %v", err)
	}
	if exists != 0 {
		t.Fatalf("expected expired key")
	}
}

func TestClientHashOps(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	if err := client.HSet(ctx, "h1", "f1", "v1", "f2", "v2"); err != nil {
		t.Fatalf("hset: %v", err)
	}
	val, err := client.HGet(ctx, "h1", "f1")
	if err != nil {
		t.Fatalf("hget: %v", err)
	}
	if val != "v1" {
		t.Fatalf("unexpected hget value: %s", val)
	}

	all, err := client.HGetAll(ctx, "h1")
	if err != nil {
		t.Fatalf("hgetall: %v", err)
	}
	if all["f2"] != "v2" {
		t.Fatalf("unexpected hgetall value: %v", all)
	}

	if err := client.HDel(ctx, "h1", "f1"); err != nil {
		t.Fatalf("hdel: %v", err)
	}
}
