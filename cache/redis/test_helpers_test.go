package redis

import (
	"testing"

	"github.com/aisgo/ais-go-pkg/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestClientWithServer(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(server.Close)

	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	return &Client{rdb: rdb, log: logger.NewNop()}, server
}

func newTestClient(t *testing.T) *Client {
	client, _ := newTestClientWithServer(t)
	return client
}
