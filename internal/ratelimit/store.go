package ratelimit

import (
	"context"
	"sync"
	"time"
)

type keyKind string

const (
	kindLogin    keyKind = "login"
	kindPassword keyKind = "password"
	kindIP       keyKind = "ip"
)

type bucketEntry struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
	lastAccess time.Time
}

// MultiLimiter stores buckets for multiple dimensions.
type MultiLimiter struct {
	mu              sync.Mutex
	entries         map[string]*bucketEntry
	loginLimit      int
	passwordLimit   int
	ipLimit         int
	ttl             time.Duration
	cleanupInterval time.Duration
}

// NewMultiLimiter creates a limiter store.
func NewMultiLimiter(loginLimit, passwordLimit, ipLimit int, ttl, cleanupInterval time.Duration) *MultiLimiter {
	return &MultiLimiter{
		entries:         make(map[string]*bucketEntry),
		loginLimit:      loginLimit,
		passwordLimit:   passwordLimit,
		ipLimit:         ipLimit,
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
	}
}

// StartCleanup periodically removes inactive buckets.
func (m *MultiLimiter) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// Allow returns true only if all corresponding buckets allow the request.
func (m *MultiLimiter) Allow(login, password, ip string) bool {
	return m.allow(kindLogin, login) && m.allow(kindPassword, password) && m.allow(kindIP, ip)
}

// Reset removes login and ip buckets.
func (m *MultiLimiter) Reset(login, ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, makeKey(kindLogin, login))
	delete(m.entries, makeKey(kindIP, ip))
}

func (m *MultiLimiter) allow(kind keyKind, value string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	key := makeKey(kind, value)
	entry, ok := m.entries[key]
	if !ok {
		entry = newBucketEntry(m.limitFor(kind), now)
		m.entries[key] = entry
	}

	refill(entry, now)
	entry.lastAccess = now
	if entry.tokens < 1 {
		return false
	}

	entry.tokens--

	return true
}

func (m *MultiLimiter) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, entry := range m.entries {
		if now.Sub(entry.lastAccess) > m.ttl {
			delete(m.entries, key)
		}
	}
}

func (m *MultiLimiter) limitFor(kind keyKind) int {
	switch kind {
	case kindLogin:
		return m.loginLimit
	case kindPassword:
		return m.passwordLimit
	case kindIP:
		return m.ipLimit
	default:
		return 1
	}
}

func newBucketEntry(limit int, now time.Time) *bucketEntry {
	capacity := float64(limit)
	return &bucketEntry{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: capacity / 60.0,
		lastRefill: now,
		lastAccess: now,
	}
}

func refill(entry *bucketEntry, now time.Time) {
	elapsed := now.Sub(entry.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}

	entry.tokens += elapsed * entry.refillRate
	if entry.tokens > entry.capacity {
		entry.tokens = entry.capacity
	}
	entry.lastRefill = now
}

func makeKey(kind keyKind, value string) string {
	return string(kind) + ":" + value
}
