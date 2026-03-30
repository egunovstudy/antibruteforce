package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	LoginLimit    int
	PasswordLimit int
	IPLimit       int
	BucketTTL     time.Duration
	RefillPeriod  time.Duration
}

type Limiter interface {
	Allow(ctx context.Context, login, password, ip string) (bool, error)
	Reset(ctx context.Context, login, ip string) error
}

type RedisLimiter struct {
	client *redis.Client
	cfg    Config
	script *redis.Script
}

func NewRedisLimiter(client *redis.Client, cfg Config) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		cfg:    cfg,
		script: redis.NewScript(luaBucketScript),
	}
}

func (l *RedisLimiter) Allow(ctx context.Context, login, password, ip string) (bool, error) {
	keys := []string{
		bucketKey("login", login),
		bucketKey("password", password),
		bucketKey("ip", ip),
	}

	args := []any{
		time.Now().UnixMilli(),
		l.cfg.BucketTTL.Milliseconds(),
		l.cfg.RefillPeriod.Milliseconds(),
		l.cfg.LoginLimit,
		l.cfg.PasswordLimit,
		l.cfg.IPLimit,
	}

	result, err := l.script.Run(ctx, l.client, keys, args...).Int()
	if err != nil {
		return false, fmt.Errorf("run limiter script: %w", err)
	}

	return result == 1, nil
}

func (l *RedisLimiter) Reset(ctx context.Context, login, ip string) error {
	if err := l.client.Del(ctx, bucketKey("login", login), bucketKey("ip", ip)).Err(); err != nil {
		return fmt.Errorf("reset buckets: %w", err)
	}
	return nil
}

func bucketKey(kind, value string) string {
	return "antibf:bucket:" + kind + ":" + value
}

const luaBucketScript = `
local now = tonumber(ARGV[1])
local ttlMs = tonumber(ARGV[2])
local refillPeriodMs = tonumber(ARGV[3])

local capacities = {
  tonumber(ARGV[4]),
  tonumber(ARGV[5]),
  tonumber(ARGV[6]),
}

local states = {}

for i = 1, #KEYS do
  local key = KEYS[i]
  local data = redis.call('HMGET', key, 'tokens', 'ts')
  local tokens = tonumber(data[1])
  local ts = tonumber(data[2])

  if not tokens or not ts then
    tokens = capacities[i]
    ts = now
  else
    local refill = (now - ts) * capacities[i] / refillPeriodMs
    tokens = math.min(capacities[i], tokens + refill)
    ts = now
  end

  if tokens < 1 then
    return 0
  end

  states[i] = { key, tokens - 1, ts, capacities[i] }
end

for i = 1, #states do
  local state = states[i]
  redis.call('HSET', state[1], 'tokens', state[2], 'ts', state[3], 'capacity', state[4])
  redis.call('PEXPIRE', state[1], ttlMs)
end

return 1
`
