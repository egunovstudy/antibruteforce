package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMultiLimiter_AllowWithinLimit(t *testing.T) {
	t.Parallel()

	limiter := NewMultiLimiter(2, 10, 10, time.Minute, time.Hour)

	if !limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("first attempt must be allowed")
	}

	if !limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("second attempt must be allowed")
	}

	if limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("third attempt must be blocked")
	}
}

func TestMultiLimiter_Reset(t *testing.T) {
	t.Parallel()

	limiter := NewMultiLimiter(1, 10, 10, time.Minute, time.Hour)

	if !limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("first attempt must be allowed")
	}

	if limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("second attempt must be blocked before reset")
	}

	limiter.Reset("alice", "192.168.1.10")

	if !limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("attempt must be allowed after reset")
	}
}

func TestMultiLimiter_StartCleanup(t *testing.T) {
	limiter := NewMultiLimiter(1, 10, 10, 10*time.Millisecond, 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go limiter.StartCleanup(ctx)

	if !limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("first attempt must be allowed")
	}

	time.Sleep(40 * time.Millisecond)

	if !limiter.Allow("alice", "pwd", "192.168.1.10") {
		t.Fatal("bucket should be recreated after cleanup")
	}
}
