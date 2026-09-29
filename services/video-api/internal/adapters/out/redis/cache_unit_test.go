package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func newTestCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	c := NewCache(mr.Addr(), time.Minute)
	t.Cleanup(func() { _ = c.Close() })

	return c, mr
}

func TestCacheSetAndGet(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "key", []byte("value"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, ok, err := c.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("esperava key existente")
	}
	if string(val) != "value" {
		t.Fatalf("valor = %q, want %q", string(val), "value")
	}
}

func TestCacheGetMiss(t *testing.T) {
	c, _ := newTestCache(t)

	_, ok, err := c.Get(context.Background(), "inexistente")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Fatal("esperava miss (ok=false)")
	}
}

func TestCacheDel(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "key", []byte("value"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Del(ctx, "key"); err != nil {
		t.Fatalf("Del: %v", err)
	}

	if _, ok, _ := c.Get(ctx, "key"); ok {
		t.Fatal("esperava miss após Del")
	}
}

func TestCacheExpiration(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "key", []byte("value"), 100*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}

	mr.FastForward(200 * time.Millisecond)

	if _, ok, _ := c.Get(ctx, "key"); ok {
		t.Fatal("esperava expiração da key")
	}
}
