//go:build integration

package redis

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestCacheSetGetDel(t *testing.T) {
	ctx := context.Background()

	rc, err := redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("redis container: %v", err)
	}
	t.Cleanup(func() { _ = rc.Terminate(ctx) })

	host, err := rc.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	port, err := rc.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("port: %v", err)
	}

	c := NewCache(net.JoinHostPort(host, port.Port()), time.Minute)
	defer func() { _ = c.Close() }()

	if err := c.Set(ctx, "k", []byte("v1"), time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}

	v, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || string(v) != "v1" {
		t.Fatalf("get: err=%v ok=%v v=%q", err, ok, v)
	}

	if _, ok, err := c.Get(ctx, "nao-existe"); err != nil || ok {
		t.Fatalf("miss: err=%v ok=%v", err, ok)
	}

	if err := c.Del(ctx, "k"); err != nil {
		t.Fatalf("del: %v", err)
	}
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Fatal("chave deveria ter sido removida")
	}
}
