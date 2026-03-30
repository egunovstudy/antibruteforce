package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func TestRedisLimiter_AllowAndReset(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR is not set")
	}

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush db: %v", err)
	}

	limiter := NewRedisLimiter(client, Config{
		LoginLimit:    2,
		PasswordLimit: 100,
		IPLimit:       100,
		BucketTTL:     time.Minute,
		RefillPeriod:  time.Minute,
	})

	ok, err := limiter.Allow(ctx, "alice", "secret", "192.168.1.10")
	if err != nil || !ok {
		t.Fatalf("first allow failed, ok=%v err=%v", ok, err)
	}

	ok, err = limiter.Allow(ctx, "alice", "secret", "192.168.1.10")
	if err != nil || !ok {
		t.Fatalf("second allow failed, ok=%v err=%v", ok, err)
	}

	ok, err = limiter.Allow(ctx, "alice", "secret", "192.168.1.10")
	if err != nil {
		t.Fatalf("third allow error: %v", err)
	}
	if ok {
		t.Fatal("expected third request to be blocked")
	}

	if err = limiter.Reset(ctx, "alice", "192.168.1.10"); err != nil {
		t.Fatalf("reset buckets: %v", err)
	}

	ok, err = limiter.Allow(ctx, "alice", "secret", "192.168.1.10")
	if err != nil || !ok {
		t.Fatalf("allow after reset failed, ok=%v err=%v", ok, err)
	}
}
